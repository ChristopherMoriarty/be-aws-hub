package response

import "time"

type Attempt struct {
	ID              string            `json:"id"`
	CertificationID string            `json:"certification_id"`
	TopicID         *string           `json:"topic_id"`
	Mode            string            `json:"mode"`
	QuestionCount   int               `json:"question_count"`
	OnlyMistakes    bool              `json:"only_mistakes"`
	Score           *int              `json:"score"`
	CorrectCount    *int              `json:"correct_count"`
	StartedAt       time.Time         `json:"started_at"`
	CompletedAt     *time.Time        `json:"completed_at"`
	DurationSeconds *int              `json:"duration_seconds"`
	Questions       []AttemptQuestion `json:"questions,omitempty"`
}

type AttemptQuestion struct {
	ID                string          `json:"id"`
	TopicID           string          `json:"topic_id"`
	Position          int             `json:"position"`
	Body              string          `json:"body"`
	Explanation       *string         `json:"explanation"`
	Difficulty        string          `json:"difficulty"`
	ImageURL          *string         `json:"image_url"`
	Options           []AttemptOption `json:"options"`
	SelectedOptionIDs []string        `json:"selected_option_ids"`
	Correct           *bool           `json:"is_correct"`
}

type AttemptOption struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	SortOrder int    `json:"sort_order"`
	IsCorrect *bool  `json:"is_correct"`
}

type TopicProgress struct {
	TopicID            string  `json:"topic_id"`
	Title              string  `json:"title"`
	SortOrder          int     `json:"sort_order"`
	TotalQuestions     int     `json:"total_questions"`
	AttemptedQuestions int     `json:"attempted_questions"`
	MasteredQuestions  int     `json:"mastered_questions"`
	TotalAttempts      int     `json:"total_attempts"`
	CorrectAnswers     int     `json:"correct_answers"`
	Accuracy           float64 `json:"accuracy"`
	Mastered           bool    `json:"mastered"`
}

type CertificationStats struct {
	TotalQuestions     int     `json:"total_questions"`
	AttemptedQuestions int     `json:"attempted_questions"`
	MasteredQuestions  int     `json:"mastered_questions"`
	NeedsReview        int     `json:"needs_review"`
	TotalAttempts      int     `json:"total_attempts"`
	CorrectAnswers     int     `json:"correct_answers"`
	ExamCount          int     `json:"exam_count"`
	Accuracy           float64 `json:"accuracy"`
}
