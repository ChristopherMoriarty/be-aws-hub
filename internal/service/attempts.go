package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/store/db"
)

const (
	modeExam     = "exam"
	modeLearning = "learning"
)

type Attempts struct {
	pool *pgxpool.Pool
}

func NewAttempts(pool *pgxpool.Pool) *Attempts {
	return &Attempts{pool: pool}
}

type StartAttemptInput struct {
	Mode          string
	TopicID       *uuid.UUID
	OnlyMistakes  bool
	QuestionCount *int
}

type AnswerInput struct {
	QuestionID uuid.UUID
	OptionIDs  []uuid.UUID
}

type SubmitInput struct {
	DurationSeconds *int
	Answers         []AnswerInput
}

type Attempt struct {
	ID              uuid.UUID
	CertificationID uuid.UUID
	TopicID         *uuid.UUID
	Mode            string
	QuestionCount   int
	OnlyMistakes    bool
	Score           *int
	CorrectCount    *int
	StartedAt       time.Time
	CompletedAt     *time.Time
	DurationSeconds *int
	Questions       []AttemptQuestion
}

type AttemptQuestion struct {
	ID                uuid.UUID
	TopicID           uuid.UUID
	Position          int
	Body              string
	Explanation       *string
	Difficulty        string
	ImageURL          *string
	Options           []AttemptOption
	SelectedOptionIDs []uuid.UUID
	Correct           *bool
}

type AttemptOption struct {
	ID        uuid.UUID
	Body      string
	SortOrder int
	IsCorrect *bool
}

type TopicProgress struct {
	TopicID            uuid.UUID
	Title              string
	SortOrder          int
	TotalQuestions     int
	AttemptedQuestions int
	MasteredQuestions  int
	TotalAttempts      int
	CorrectAnswers     int
	Accuracy           float64
	Mastered           bool
}

type CertificationStats struct {
	TotalQuestions     int
	AttemptedQuestions int
	MasteredQuestions  int
	NeedsReview        int
	TotalAttempts      int
	CorrectAnswers     int
	ExamCount          int
	Accuracy           float64
}

type gradedAnswer struct {
	question  db.Question
	position  int
	options   []db.QuestionOption
	optionIDs []uuid.UUID
	isCorrect bool
}

func (s *Attempts) queries() *db.Queries {
	return db.New(s.pool)
}

func (s *Attempts) withTx(ctx context.Context, fn func(*db.Queries) error) error {
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

func (s *Attempts) Start(ctx context.Context, userID, certificationID uuid.UUID, in StartAttemptInput) (Attempt, error) {
	mode, err := normalizeMode(in.Mode)
	if err != nil {
		return Attempt{}, err
	}
	var result Attempt
	err = s.withTx(ctx, func(q *db.Queries) error {
		cert, err := q.GetCertification(ctx, certificationID)
		if err != nil {
			return readErr(err, "certification not found", "get certification")
		}
		count, err := attemptQuestionCount(ctx, q, cert, in)
		if err != nil {
			return err
		}
		// Topic study skips questions that are already mastered. The exam keeps the full pool.
		onlyMistakes := in.OnlyMistakes || (in.TopicID != nil && mode == modeLearning)
		rows, err := q.ListQuestionsForAttempt(ctx, db.ListQuestionsForAttemptParams{
			UserID:          userID,
			CertificationID: certificationID,
			TopicID:         in.TopicID,
			OnlyMistakes:    onlyMistakes,
		})
		if err != nil {
			return fmt.Errorf("list questions: %w", err)
		}
		if len(rows) == 0 {
			return notFound("no questions found")
		}
		if count > len(rows) {
			count = len(rows)
		}
		picked := pickQuestions(rows, count)
		attempt, err := q.CreateAttempt(ctx, db.CreateAttemptParams{
			UserID:          userID,
			CertificationID: certificationID,
			TopicID:         in.TopicID,
			Mode:            mode,
			QuestionCount:   int32(len(picked)),
			OnlyMistakes:    onlyMistakes,
		})
		if err != nil {
			return storeErr("create attempt", err)
		}
		ids := make([]uuid.UUID, len(picked))
		for i, question := range picked {
			ids[i] = question.ID
			if err := q.InsertAttemptQuestion(ctx, db.InsertAttemptQuestionParams{
				AttemptID:  attempt.ID,
				QuestionID: question.ID,
				Position:   int32(i + 1),
			}); err != nil {
				return storeErr("insert attempt question", err)
			}
		}
		options, err := q.ListOptionsByQuestionIDs(ctx, ids)
		if err != nil {
			return fmt.Errorf("list options: %w", err)
		}
		result = openAttempt(attempt, picked, options)
		return nil
	})
	return result, err
}

func (s *Attempts) Submit(ctx context.Context, userID, attemptID uuid.UUID, in SubmitInput) (Attempt, error) {
	duration, err := normalizeDuration(in.DurationSeconds)
	if err != nil {
		return Attempt{}, err
	}
	var result Attempt
	err = s.withTx(ctx, func(q *db.Queries) error {
		attempt, err := q.GetAttemptForUpdate(ctx, db.GetAttemptForUpdateParams{
			ID:     attemptID,
			UserID: userID,
		})
		if err != nil {
			return readErr(err, "attempt not found", "get attempt")
		}
		if attempt.CompletedAt != nil {
			return conflict("attempt is already submitted")
		}
		links, err := q.ListAttemptQuestions(ctx, attempt.ID)
		if err != nil {
			return fmt.Errorf("list attempt questions: %w", err)
		}
		graded, err := grade(ctx, q, links, in.Answers)
		if err != nil {
			return err
		}
		correctCount := 0
		for _, answer := range graded {
			if answer.isCorrect {
				correctCount++
			}
			if err := q.InsertAttemptAnswer(ctx, db.InsertAttemptAnswerParams{
				AttemptID:  attempt.ID,
				QuestionID: answer.question.ID,
				IsCorrect:  answer.isCorrect,
			}); err != nil {
				return storeErr("insert answer", err)
			}
			for _, optionID := range answer.optionIDs {
				if err := q.InsertAttemptAnswerOption(ctx, db.InsertAttemptAnswerOptionParams{
					AttemptID:  attempt.ID,
					QuestionID: answer.question.ID,
					OptionID:   optionID,
				}); err != nil {
					return storeErr("insert answer option", err)
				}
			}
		}
		if err := saveStats(ctx, q, userID, graded); err != nil {
			return err
		}
		score := int32(percentScore(correctCount, len(graded)))
		correct := int32(correctCount)
		completed, err := q.CompleteAttempt(ctx, db.CompleteAttemptParams{
			Score:           &score,
			CorrectCount:    &correct,
			DurationSeconds: duration,
			ID:              attempt.ID,
			UserID:          userID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return conflict("attempt is already submitted")
			}
			return storeErr("complete attempt", err)
		}
		result = closedAttempt(completed, graded)
		return nil
	})
	return result, err
}

func (s *Attempts) Get(ctx context.Context, userID, attemptID uuid.UUID) (Attempt, error) {
	row, err := s.queries().GetAttempt(ctx, db.GetAttemptParams{ID: attemptID, UserID: userID})
	if err != nil {
		return Attempt{}, readErr(err, "attempt not found", "get attempt")
	}
	return s.loadAttempt(ctx, s.queries(), row)
}

func (s *Attempts) List(ctx context.Context, userID, certificationID uuid.UUID) ([]Attempt, error) {
	if _, err := s.queries().GetCertification(ctx, certificationID); err != nil {
		return nil, readErr(err, "certification not found", "get certification")
	}
	rows, err := s.queries().ListAttemptsByCertification(ctx, db.ListAttemptsByCertificationParams{
		UserID:          userID,
		CertificationID: certificationID,
	})
	if err != nil {
		return nil, fmt.Errorf("list attempts: %w", err)
	}
	items := make([]Attempt, 0, len(rows))
	for _, row := range rows {
		items = append(items, attemptHeader(row))
	}
	return items, nil
}

func (s *Attempts) Progress(ctx context.Context, userID, certificationID uuid.UUID) ([]TopicProgress, error) {
	if _, err := s.queries().GetCertification(ctx, certificationID); err != nil {
		return nil, readErr(err, "certification not found", "get certification")
	}
	rows, err := s.queries().ListTopicProgress(ctx, db.ListTopicProgressParams{
		UserID:          userID,
		CertificationID: certificationID,
	})
	if err != nil {
		return nil, fmt.Errorf("list topic progress: %w", err)
	}
	items := make([]TopicProgress, 0, len(rows))
	for _, row := range rows {
		total := int(row.TotalQuestions)
		attempted := int(row.AttemptedQuestions)
		mastered := int(row.MasteredQuestions)
		correct := int(row.CorrectAnswers)
		answers := int(row.TotalAttempts)
		items = append(items, TopicProgress{
			TopicID:            row.ID,
			Title:              row.Title,
			SortOrder:          int(row.SortOrder),
			TotalQuestions:     total,
			AttemptedQuestions: attempted,
			MasteredQuestions:  mastered,
			TotalAttempts:      answers,
			CorrectAnswers:     correct,
			Accuracy:           answerAccuracy(correct, answers),
			Mastered:           topicMastered(total, attempted, mastered),
		})
	}
	return items, nil
}

func (s *Attempts) Stats(ctx context.Context, userID, certificationID uuid.UUID) (CertificationStats, error) {
	if _, err := s.queries().GetCertification(ctx, certificationID); err != nil {
		return CertificationStats{}, readErr(err, "certification not found", "get certification")
	}
	row, err := s.queries().GetCertificationStats(ctx, db.GetCertificationStatsParams{
		UserID:          userID,
		CertificationID: certificationID,
	})
	if err != nil {
		return CertificationStats{}, fmt.Errorf("certification stats: %w", err)
	}
	exams, err := s.queries().CountCompletedExams(ctx, db.CountCompletedExamsParams{
		UserID:          userID,
		CertificationID: certificationID,
	})
	if err != nil {
		return CertificationStats{}, fmt.Errorf("count exams: %w", err)
	}
	correct := int(row.CorrectAnswers)
	answers := int(row.TotalAttempts)
	return CertificationStats{
		TotalQuestions:     int(row.TotalQuestions),
		AttemptedQuestions: int(row.AttemptedQuestions),
		MasteredQuestions:  int(row.MasteredQuestions),
		NeedsReview:        int(row.NeedsReview),
		TotalAttempts:      answers,
		CorrectAnswers:     correct,
		ExamCount:          int(exams),
		Accuracy:           answerAccuracy(correct, answers),
	}, nil
}

func (s *Attempts) loadAttempt(ctx context.Context, q *db.Queries, row db.Attempt) (Attempt, error) {
	links, err := q.ListAttemptQuestions(ctx, row.ID)
	if err != nil {
		return Attempt{}, fmt.Errorf("list attempt questions: %w", err)
	}
	ids := make([]uuid.UUID, len(links))
	for i, link := range links {
		ids[i] = link.QuestionID
	}
	questionRows, err := q.ListQuestionsByIDs(ctx, ids)
	if err != nil {
		return Attempt{}, fmt.Errorf("list questions: %w", err)
	}
	optionRows, err := q.ListOptionsByQuestionIDs(ctx, ids)
	if err != nil {
		return Attempt{}, fmt.Errorf("list options: %w", err)
	}
	questions := make(map[uuid.UUID]db.Question, len(questionRows))
	for _, question := range questionRows {
		questions[question.ID] = question
	}
	options := groupOptions(optionRows)
	if row.CompletedAt == nil {
		return openAttemptFromLinks(row, links, questions, options), nil
	}
	answerRows, err := q.ListAttemptAnswers(ctx, row.ID)
	if err != nil {
		return Attempt{}, fmt.Errorf("list answers: %w", err)
	}
	selectedRows, err := q.ListAttemptAnswerOptions(ctx, row.ID)
	if err != nil {
		return Attempt{}, fmt.Errorf("list answer options: %w", err)
	}
	return closedAttemptFromRows(row, links, questions, options, answerRows, selectedRows), nil
}

func attemptQuestionCount(ctx context.Context, q *db.Queries, cert db.Certification, in StartAttemptInput) (int, error) {
	if in.TopicID != nil {
		topic, err := q.GetTopic(ctx, *in.TopicID)
		if err != nil {
			return 0, readErr(err, "topic not found", "get topic")
		}
		if topic.CertificationID != cert.ID {
			return 0, invalid("topic does not belong to certification")
		}
	}
	if in.QuestionCount != nil {
		if *in.QuestionCount <= 0 || *in.QuestionCount > math.MaxInt32 {
			return 0, invalid("question_count must be positive")
		}
		return *in.QuestionCount, nil
	}
	if in.TopicID != nil {
		return topicTestQuestionCount, nil
	}
	return int(cert.QuestionCount), nil
}

func grade(ctx context.Context, q *db.Queries, links []db.AttemptQuestion, answers []AnswerInput) ([]gradedAnswer, error) {
	byID := make(map[uuid.UUID]AnswerInput, len(answers))
	for _, answer := range answers {
		if _, exists := byID[answer.QuestionID]; exists {
			return nil, invalid("duplicate question")
		}
		byID[answer.QuestionID] = answer
	}
	if len(byID) != len(links) {
		return nil, invalid("answers must include every question")
	}
	ids := make([]uuid.UUID, len(links))
	for i, link := range links {
		if _, ok := byID[link.QuestionID]; !ok {
			return nil, invalid("answers must include every question")
		}
		ids[i] = link.QuestionID
	}
	questionRows, err := q.ListQuestionsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	optionRows, err := q.ListOptionsByQuestionIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list options: %w", err)
	}
	questions := make(map[uuid.UUID]db.Question, len(questionRows))
	for _, question := range questionRows {
		questions[question.ID] = question
	}
	options := groupOptions(optionRows)
	graded := make([]gradedAnswer, 0, len(links))
	for _, link := range links {
		question, ok := questions[link.QuestionID]
		if !ok {
			return nil, notFound("question not found")
		}
		answer := byID[link.QuestionID]
		known := make(map[uuid.UUID]struct{}, len(options[link.QuestionID]))
		correctIDs := make([]uuid.UUID, 0, 1)
		for _, option := range options[link.QuestionID] {
			known[option.ID] = struct{}{}
			if option.IsCorrect {
				correctIDs = append(correctIDs, option.ID)
			}
		}
		seen := make(map[uuid.UUID]struct{}, len(answer.OptionIDs))
		for _, optionID := range answer.OptionIDs {
			if _, dup := seen[optionID]; dup {
				return nil, invalid("duplicate option")
			}
			seen[optionID] = struct{}{}
			if _, ok := known[optionID]; !ok {
				return nil, invalid("option does not belong to question")
			}
		}
		graded = append(graded, gradedAnswer{
			question:  question,
			position:  int(link.Position),
			options:   options[link.QuestionID],
			optionIDs: answer.OptionIDs,
			isCorrect: sameOptions(answer.OptionIDs, correctIDs),
		})
	}
	return graded, nil
}

func saveStats(ctx context.Context, q *db.Queries, userID uuid.UUID, graded []gradedAnswer) error {
	ordered := slices.Clone(graded)
	slices.SortFunc(ordered, func(a, b gradedAnswer) int {
		return bytes.Compare(a.question.ID[:], b.question.ID[:])
	})
	for _, answer := range ordered {
		if err := saveStat(ctx, q, userID, answer.question.ID, answer.isCorrect); err != nil {
			return err
		}
	}
	return nil
}

func saveStat(ctx context.Context, q *db.Queries, userID, questionID uuid.UUID, correct bool) error {
	for try := 0; try < 2; try++ {
		row, err := q.GetQuestionStatForUpdate(ctx, db.GetQuestionStatForUpdateParams{
			UserID:     userID,
			QuestionID: questionID,
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("get question stat: %w", err)
		}
		next := applyAnswer(statSnapshot(row, err), correct)
		lastCorrect := next.LastCorrect
		params := statParams(userID, questionID, next, &lastCorrect)
		if errors.Is(err, pgx.ErrNoRows) {
			err = q.InsertQuestionStat(ctx, params.insert)
			if uniqueViolation(err) {
				continue
			}
			return storeErr("insert question stat", err)
		}
		return storeErr("update question stat", q.UpdateQuestionStat(ctx, params.update))
	}
	return conflict("conflict")
}

type statWrite struct {
	insert db.InsertQuestionStatParams
	update db.UpdateQuestionStatParams
}

func statParams(userID, questionID uuid.UUID, next answerStat, lastCorrect *bool) statWrite {
	return statWrite{
		insert: db.InsertQuestionStatParams{
			UserID:             userID,
			QuestionID:         questionID,
			TotalAttempts:      int32(next.TotalAttempts),
			CorrectAttempts:    int32(next.CorrectAttempts),
			ConsecutiveCorrect: int32(next.ConsecutiveCorrect),
			LearningScore:      int32(next.LearningScore),
			Weight:             next.Weight,
			LastCorrect:        lastCorrect,
		},
		update: db.UpdateQuestionStatParams{
			TotalAttempts:      int32(next.TotalAttempts),
			CorrectAttempts:    int32(next.CorrectAttempts),
			ConsecutiveCorrect: int32(next.ConsecutiveCorrect),
			LearningScore:      int32(next.LearningScore),
			Weight:             next.Weight,
			LastCorrect:        lastCorrect,
			UserID:             userID,
			QuestionID:         questionID,
		},
	}
}

func statSnapshot(row db.QuestionStat, err error) answerStat {
	if errors.Is(err, pgx.ErrNoRows) {
		return answerStat{}
	}
	lastCorrect := false
	if row.LastCorrect != nil {
		lastCorrect = *row.LastCorrect
	}
	return answerStat{
		TotalAttempts:      int(row.TotalAttempts),
		CorrectAttempts:    int(row.CorrectAttempts),
		ConsecutiveCorrect: int(row.ConsecutiveCorrect),
		LearningScore:      int(row.LearningScore),
		Weight:             row.Weight,
		LastCorrect:        lastCorrect,
		Exists:             true,
	}
}

func pickQuestions(rows []db.ListQuestionsForAttemptRow, count int) []db.ListQuestionsForAttemptRow {
	weights := make([]float64, len(rows))
	for i, row := range rows {
		weights[i] = row.Weight
	}
	indexes := selectWeighted(weights, count, rand.Float64)
	picked := make([]db.ListQuestionsForAttemptRow, len(indexes))
	for i, index := range indexes {
		picked[i] = rows[index]
	}
	return picked
}

func openAttempt(row db.Attempt, picked []db.ListQuestionsForAttemptRow, optionRows []db.QuestionOption) Attempt {
	options := groupOptions(optionRows)
	questions := make([]AttemptQuestion, 0, len(picked))
	for i, question := range picked {
		questions = append(questions, presentQuestion(db.Question{
			ID:          question.ID,
			TopicID:     question.TopicID,
			Body:        question.Body,
			Explanation: question.Explanation,
			Difficulty:  question.Difficulty,
			ImageUrl:    question.ImageUrl,
		}, options[question.ID], i+1, false, nil, false))
	}
	attempt := attemptHeader(row)
	attempt.Questions = questions
	return attempt
}

func openAttemptFromLinks(row db.Attempt, links []db.AttemptQuestion, questions map[uuid.UUID]db.Question, options map[uuid.UUID][]db.QuestionOption) Attempt {
	attempt := attemptHeader(row)
	attempt.Questions = make([]AttemptQuestion, 0, len(links))
	for _, link := range links {
		question := questions[link.QuestionID]
		attempt.Questions = append(attempt.Questions, presentQuestion(question, options[link.QuestionID], int(link.Position), false, nil, false))
	}
	return attempt
}

func closedAttempt(row db.Attempt, graded []gradedAnswer) Attempt {
	attempt := attemptHeader(row)
	attempt.Questions = make([]AttemptQuestion, 0, len(graded))
	for _, answer := range graded {
		attempt.Questions = append(attempt.Questions, presentQuestion(answer.question, answer.options, answer.position, true, answer.optionIDs, answer.isCorrect))
	}
	return attempt
}

func closedAttemptFromRows(row db.Attempt, links []db.AttemptQuestion, questions map[uuid.UUID]db.Question, options map[uuid.UUID][]db.QuestionOption, answers []db.AttemptAnswer, selected []db.AttemptAnswerOption) Attempt {
	correct := make(map[uuid.UUID]bool, len(answers))
	for _, answer := range answers {
		correct[answer.QuestionID] = answer.IsCorrect
	}
	chosen := make(map[uuid.UUID][]uuid.UUID, len(links))
	for _, option := range selected {
		chosen[option.QuestionID] = append(chosen[option.QuestionID], option.OptionID)
	}
	attempt := attemptHeader(row)
	attempt.Questions = make([]AttemptQuestion, 0, len(links))
	for _, link := range links {
		isCorrect := correct[link.QuestionID]
		optionIDs := chosen[link.QuestionID]
		if optionIDs == nil {
			optionIDs = []uuid.UUID{}
		}
		attempt.Questions = append(attempt.Questions, presentQuestion(questions[link.QuestionID], options[link.QuestionID], int(link.Position), true, optionIDs, isCorrect))
	}
	return attempt
}

func presentQuestion(question db.Question, options []db.QuestionOption, position int, reveal bool, selected []uuid.UUID, correct bool) AttemptQuestion {
	presented := make([]AttemptOption, 0, len(options))
	for _, option := range options {
		item := AttemptOption{ID: option.ID, Body: option.Body, SortOrder: int(option.SortOrder)}
		if reveal {
			isCorrect := option.IsCorrect
			item.IsCorrect = &isCorrect
		}
		presented = append(presented, item)
	}
	item := AttemptQuestion{
		ID:         question.ID,
		TopicID:    question.TopicID,
		Position:   position,
		Body:       question.Body,
		Difficulty: question.Difficulty,
		ImageURL:   question.ImageUrl,
		Options:    presented,
	}
	if reveal {
		if selected == nil {
			selected = []uuid.UUID{}
		}
		item.Explanation = question.Explanation
		item.SelectedOptionIDs = selected
		item.Correct = &correct
	}
	return item
}

func attemptHeader(row db.Attempt) Attempt {
	return Attempt{
		ID:              row.ID,
		CertificationID: row.CertificationID,
		TopicID:         row.TopicID,
		Mode:            row.Mode,
		QuestionCount:   int(row.QuestionCount),
		OnlyMistakes:    row.OnlyMistakes,
		Score:           intPtrFrom32(row.Score),
		CorrectCount:    intPtrFrom32(row.CorrectCount),
		StartedAt:       row.StartedAt,
		CompletedAt:     row.CompletedAt,
		DurationSeconds: intPtrFrom32(row.DurationSeconds),
	}
}

func groupOptions(rows []db.QuestionOption) map[uuid.UUID][]db.QuestionOption {
	grouped := make(map[uuid.UUID][]db.QuestionOption)
	for _, row := range rows {
		grouped[row.QuestionID] = append(grouped[row.QuestionID], row)
	}
	return grouped
}

func sameOptions(selected, correct []uuid.UUID) bool {
	if len(selected) != len(correct) {
		return false
	}
	want := make(map[uuid.UUID]struct{}, len(correct))
	for _, id := range correct {
		want[id] = struct{}{}
	}
	for _, id := range selected {
		if _, ok := want[id]; !ok {
			return false
		}
	}
	return true
}

func normalizeMode(mode string) (string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case modeExam, modeLearning:
		return mode, nil
	default:
		return "", invalid("mode must be exam or learning")
	}
}

func normalizeDuration(value *int) (*int32, error) {
	if value == nil {
		return nil, nil
	}
	if *value < 0 {
		return nil, invalid("duration_seconds must not be negative")
	}
	if *value > math.MaxInt32 {
		return nil, invalid("duration_seconds is out of range")
	}
	converted := int32(*value)
	return &converted, nil
}

func intPtrFrom32(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}
