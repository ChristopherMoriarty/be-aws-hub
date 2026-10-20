package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/store/db"
)

const (
	difficultyEasy   = "easy"
	difficultyMedium = "medium"
	difficultyHard   = "hard"
)

type Catalog struct {
	pool *pgxpool.Pool
}

func NewCatalog(pool *pgxpool.Pool) *Catalog {
	return &Catalog{pool: pool}
}

type Certification struct {
	ID              uuid.UUID
	Code            string
	Title           string
	QuestionCount   int
	DurationSeconds int
	PassingPercent  int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CertificationInput struct {
	Code            string
	Title           string
	QuestionCount   int
	DurationSeconds int
	PassingPercent  int
}

type Topic struct {
	ID              uuid.UUID
	CertificationID uuid.UUID
	Title           string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type TopicInput struct {
	Title     string
	SortOrder *int
}

func (s *Catalog) queries() *db.Queries {
	return db.New(s.pool)
}

func (s *Catalog) withTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if err := fn(db.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *Catalog) ListCertifications(ctx context.Context) ([]Certification, error) {
	rows, err := s.queries().ListCertifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("list certifications: %w", err)
	}
	items := make([]Certification, 0, len(rows))
	for _, row := range rows {
		items = append(items, toCertification(row))
	}
	return items, nil
}

func (s *Catalog) GetCertification(ctx context.Context, id uuid.UUID) (Certification, error) {
	row, err := s.queries().GetCertification(ctx, id)
	if err != nil {
		return Certification{}, readErr(err, "certification not found", "get certification")
	}
	return toCertification(row), nil
}

func (s *Catalog) CreateCertification(ctx context.Context, in CertificationInput) (Certification, error) {
	params, err := certificationParams(in)
	if err != nil {
		return Certification{}, err
	}
	row, err := s.queries().CreateCertification(ctx, params)
	if err != nil {
		return Certification{}, storeErr("create certification", err)
	}
	return toCertification(row), nil
}

func (s *Catalog) UpdateCertification(ctx context.Context, id uuid.UUID, in CertificationInput) (Certification, error) {
	params, err := certificationParams(in)
	if err != nil {
		return Certification{}, err
	}
	row, err := s.queries().UpdateCertification(ctx, db.UpdateCertificationParams{
		Code:            params.Code,
		Title:           params.Title,
		QuestionCount:   params.QuestionCount,
		DurationSeconds: params.DurationSeconds,
		PassingPercent:  params.PassingPercent,
		ID:              id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Certification{}, notFound("certification not found")
		}
		return Certification{}, storeErr("update certification", err)
	}
	return toCertification(row), nil
}

func (s *Catalog) DeleteCertification(ctx context.Context, id uuid.UUID) error {
	tag, err := s.queries().DeleteCertification(ctx, id)
	if err != nil {
		return storeErr("delete certification", err)
	}
	return deleted(tag, "certification not found")
}

func (s *Catalog) ListTopics(ctx context.Context, certificationID uuid.UUID) ([]Topic, error) {
	if _, err := s.queries().GetCertification(ctx, certificationID); err != nil {
		return nil, readErr(err, "certification not found", "get certification")
	}
	rows, err := s.queries().ListTopicsByCertification(ctx, certificationID)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	items := make([]Topic, 0, len(rows))
	for _, row := range rows {
		items = append(items, toTopic(row))
	}
	return items, nil
}

func (s *Catalog) CreateTopic(ctx context.Context, certificationID uuid.UUID, in TopicInput) (Topic, error) {
	title, sortOrder, err := normalizeTopic(in)
	if err != nil {
		return Topic{}, err
	}
	if _, err := s.queries().GetCertification(ctx, certificationID); err != nil {
		return Topic{}, readErr(err, "certification not found", "get certification")
	}
	if sortOrder == nil {
		next, err := s.queries().NextTopicSortOrder(ctx, certificationID)
		if err != nil {
			return Topic{}, fmt.Errorf("topic sort: %w", err)
		}
		sortOrder = &next
	}
	row, err := s.queries().CreateTopic(ctx, db.CreateTopicParams{
		CertificationID: certificationID,
		Title:           title,
		SortOrder:       *sortOrder,
	})
	if err != nil {
		return Topic{}, storeErr("create topic", err)
	}
	return toTopic(row), nil
}

func (s *Catalog) UpdateTopic(ctx context.Context, id uuid.UUID, in TopicInput) (Topic, error) {
	title, sortOrder, err := normalizeTopic(in)
	if err != nil {
		return Topic{}, err
	}
	current, err := s.queries().GetTopic(ctx, id)
	if err != nil {
		return Topic{}, readErr(err, "topic not found", "get topic")
	}
	order := current.SortOrder
	if sortOrder != nil {
		order = *sortOrder
	}
	row, err := s.queries().UpdateTopic(ctx, db.UpdateTopicParams{
		Title:     title,
		SortOrder: order,
		ID:        id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Topic{}, notFound("topic not found")
		}
		return Topic{}, storeErr("update topic", err)
	}
	return toTopic(row), nil
}

func (s *Catalog) DeleteTopic(ctx context.Context, id uuid.UUID) error {
	tag, err := s.queries().DeleteTopic(ctx, id)
	if err != nil {
		return storeErr("delete topic", err)
	}
	return deleted(tag, "topic not found")
}

func certificationParams(in CertificationInput) (db.CreateCertificationParams, error) {
	code := strings.TrimSpace(in.Code)
	title := strings.TrimSpace(in.Title)
	if code == "" {
		return db.CreateCertificationParams{}, invalid("code is required")
	}
	if title == "" {
		return db.CreateCertificationParams{}, invalid("title is required")
	}
	questionCount, err := positiveInt32(in.QuestionCount, "question_count")
	if err != nil {
		return db.CreateCertificationParams{}, err
	}
	duration, err := positiveInt32(in.DurationSeconds, "duration_seconds")
	if err != nil {
		return db.CreateCertificationParams{}, err
	}
	if in.PassingPercent < 0 || in.PassingPercent > 100 {
		return db.CreateCertificationParams{}, invalid("passing_percent must be between 0 and 100")
	}
	return db.CreateCertificationParams{
		Code:            code,
		Title:           title,
		QuestionCount:   questionCount,
		DurationSeconds: duration,
		PassingPercent:  int32(in.PassingPercent),
	}, nil
}

func normalizeTopic(in TopicInput) (string, *int32, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return "", nil, invalid("title is required")
	}
	if in.SortOrder == nil {
		return title, nil, nil
	}
	if *in.SortOrder < math.MinInt32 || *in.SortOrder > math.MaxInt32 {
		return "", nil, invalid("sort_order is out of range")
	}
	sortOrder := int32(*in.SortOrder)
	return title, &sortOrder, nil
}

func positiveInt32(value int, field string) (int32, error) {
	if value <= 0 || value > math.MaxInt32 {
		return 0, invalid(field + " must be positive")
	}
	return int32(value), nil
}

func deleted(tag pgconn.CommandTag, message string) error {
	if tag.RowsAffected() == 0 {
		return notFound(message)
	}
	return nil
}

func toCertification(row db.Certification) Certification {
	return Certification{
		ID:              row.ID,
		Code:            row.Code,
		Title:           row.Title,
		QuestionCount:   int(row.QuestionCount),
		DurationSeconds: int(row.DurationSeconds),
		PassingPercent:  int(row.PassingPercent),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func toTopic(row db.Topic) Topic {
	return Topic{
		ID:              row.ID,
		CertificationID: row.CertificationID,
		Title:           row.Title,
		SortOrder:       int(row.SortOrder),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
