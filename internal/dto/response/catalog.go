package response

import "time"

type Certification struct {
	ID              string    `json:"id"`
	Code            string    `json:"code"`
	Title           string    `json:"title"`
	QuestionCount   int       `json:"question_count"`
	DurationSeconds int       `json:"duration_seconds"`
	PassingPercent  int       `json:"passing_percent"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Topic struct {
	ID              string    `json:"id"`
	CertificationID string    `json:"certification_id"`
	Title           string    `json:"title"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Option struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	IsCorrect bool   `json:"is_correct"`
	SortOrder int    `json:"sort_order"`
}

type Question struct {
	ID          string    `json:"id"`
	TopicID     string    `json:"topic_id"`
	Body        string    `json:"body"`
	Explanation *string   `json:"explanation"`
	Difficulty  string    `json:"difficulty"`
	ImageURL    *string   `json:"image_url"`
	Options     []Option  `json:"options"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
