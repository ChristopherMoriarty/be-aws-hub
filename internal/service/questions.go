package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/store/db"
)

type Option struct {
	ID        uuid.UUID
	Body      string
	IsCorrect bool
	SortOrder int
}

type OptionInput struct {
	Body      string
	IsCorrect bool
}

type Question struct {
	ID          uuid.UUID
	TopicID     uuid.UUID
	Body        string
	Explanation *string
	Difficulty  string
	ImageURL    *string
	Options     []Option
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type QuestionInput struct {
	Body        string
	Explanation *string
	Difficulty  string
	ImageURL    *string
	Options     []OptionInput
}

func (s *Catalog) ListQuestions(ctx context.Context, topicID uuid.UUID) ([]Question, error) {
	if _, err := s.queries().GetTopic(ctx, topicID); err != nil {
		return nil, readErr(err, "topic not found", "get topic")
	}
	rows, err := s.queries().ListQuestionsByTopic(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	return s.attachOptions(ctx, s.queries(), rows)
}

func (s *Catalog) GetQuestion(ctx context.Context, id uuid.UUID) (Question, error) {
	row, err := s.queries().GetQuestion(ctx, id)
	if err != nil {
		return Question{}, readErr(err, "question not found", "get question")
	}
	options, err := s.queries().ListOptionsByQuestion(ctx, id)
	if err != nil {
		return Question{}, fmt.Errorf("list options: %w", err)
	}
	return toQuestion(row, mapOptions(options)), nil
}

func (s *Catalog) CreateQuestion(ctx context.Context, topicID uuid.UUID, in QuestionInput) (Question, error) {
	if in.Options == nil {
		return Question{}, invalid("at least 2 options")
	}
	normalized, err := normalizeQuestion(in)
	if err != nil {
		return Question{}, err
	}
	var result Question
	err = s.withTx(ctx, func(q *db.Queries) error {
		if _, err := q.GetTopic(ctx, topicID); err != nil {
			return readErr(err, "topic not found", "get topic")
		}
		row, err := q.CreateQuestion(ctx, db.CreateQuestionParams{
			TopicID:     topicID,
			Body:        normalized.Body,
			Explanation: normalized.Explanation,
			Difficulty:  normalized.Difficulty,
			ImageUrl:    normalized.ImageURL,
		})
		if err != nil {
			return storeErr("create question", err)
		}
		options, err := insertOptions(ctx, q, row.ID, normalized.Options)
		if err != nil {
			return err
		}
		result = toQuestion(row, options)
		return nil
	})
	return result, err
}

func (s *Catalog) UpdateQuestion(ctx context.Context, id uuid.UUID, in QuestionInput) (Question, error) {
	normalized, err := normalizeQuestion(in)
	if err != nil {
		return Question{}, err
	}
	var result Question
	err = s.withTx(ctx, func(q *db.Queries) error {
		row, err := q.UpdateQuestion(ctx, db.UpdateQuestionParams{
			Body:        normalized.Body,
			Explanation: normalized.Explanation,
			Difficulty:  normalized.Difficulty,
			ImageUrl:    normalized.ImageURL,
			ID:          id,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return notFound("question not found")
			}
			return storeErr("update question", err)
		}
		var options []Option
		if normalized.Options == nil {
			stored, err := q.ListOptionsByQuestion(ctx, id)
			if err != nil {
				return fmt.Errorf("list options: %w", err)
			}
			options = mapOptions(stored)
		} else {
			if err := q.DeleteOptionsByQuestion(ctx, id); err != nil {
				return storeErr("delete options", err)
			}
			options, err = insertOptions(ctx, q, id, normalized.Options)
			if err != nil {
				return err
			}
		}
		result = toQuestion(row, options)
		return nil
	})
	return result, err
}

func (s *Catalog) DeleteQuestion(ctx context.Context, id uuid.UUID) error {
	tag, err := s.queries().DeleteQuestion(ctx, id)
	if err != nil {
		return storeErr("delete question", err)
	}
	return deleted(tag, "question not found")
}

func (s *Catalog) attachOptions(ctx context.Context, q *db.Queries, rows []db.Question) ([]Question, error) {
	questions := make([]Question, 0, len(rows))
	if len(rows) == 0 {
		return questions, nil
	}
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	optionRows, err := q.ListOptionsByQuestionIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list options: %w", err)
	}
	grouped := make(map[uuid.UUID][]Option, len(rows))
	for _, row := range optionRows {
		grouped[row.QuestionID] = append(grouped[row.QuestionID], toOption(row))
	}
	for _, row := range rows {
		options := grouped[row.ID]
		if options == nil {
			options = []Option{}
		}
		questions = append(questions, toQuestion(row, options))
	}
	return questions, nil
}

func insertOptions(ctx context.Context, q *db.Queries, questionID uuid.UUID, options []OptionInput) ([]Option, error) {
	stored := make([]Option, 0, len(options))
	for i, option := range options {
		row, err := q.CreateQuestionOption(ctx, db.CreateQuestionOptionParams{
			QuestionID: questionID,
			Body:       option.Body,
			IsCorrect:  option.IsCorrect,
			SortOrder:  int32(i),
		})
		if err != nil {
			return nil, storeErr("create option", err)
		}
		stored = append(stored, toOption(row))
	}
	return stored, nil
}

func normalizeQuestion(in QuestionInput) (QuestionInput, error) {
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" {
		return QuestionInput{}, invalid("body is required")
	}
	difficulty, err := normalizeDifficulty(in.Difficulty)
	if err != nil {
		return QuestionInput{}, err
	}
	in.Difficulty = difficulty
	in.Explanation = blankPtr(in.Explanation)
	in.ImageURL = blankPtr(in.ImageURL)
	if in.Options != nil {
		options, err := normalizeOptions(in.Options)
		if err != nil {
			return QuestionInput{}, err
		}
		in.Options = options
	}
	return in, nil
}

func normalizeOptions(options []OptionInput) ([]OptionInput, error) {
	if len(options) < 2 {
		return nil, invalid("at least 2 options")
	}
	normalized := make([]OptionInput, len(options))
	correct := 0
	for i, option := range options {
		body := strings.TrimSpace(option.Body)
		if body == "" {
			return nil, invalid("option body is required")
		}
		if option.IsCorrect {
			correct++
		}
		normalized[i] = OptionInput{Body: body, IsCorrect: option.IsCorrect}
	}
	if correct == 0 {
		return nil, invalid("at least one correct option")
	}
	return normalized, nil
}

func normalizeDifficulty(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case difficultyEasy, difficultyMedium, difficultyHard:
		return value, nil
	default:
		return "", invalid("difficulty must be easy, medium, or hard")
	}
}

func blankPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func toQuestion(row db.Question, options []Option) Question {
	if options == nil {
		options = []Option{}
	}
	return Question{
		ID:          row.ID,
		TopicID:     row.TopicID,
		Body:        row.Body,
		Explanation: row.Explanation,
		Difficulty:  row.Difficulty,
		ImageURL:    row.ImageUrl,
		Options:     options,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func toOption(row db.QuestionOption) Option {
	return Option{
		ID:        row.ID,
		Body:      row.Body,
		IsCorrect: row.IsCorrect,
		SortOrder: int(row.SortOrder),
	}
}

func mapOptions(rows []db.QuestionOption) []Option {
	options := make([]Option, 0, len(rows))
	for _, row := range rows {
		options = append(options, toOption(row))
	}
	return options
}
