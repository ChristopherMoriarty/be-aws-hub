package request

type StartAttempt struct {
	Mode          string  `json:"mode"`
	TopicID       *string `json:"topic_id"`
	OnlyMistakes  bool    `json:"only_mistakes"`
	QuestionCount *int    `json:"question_count"`
}

type SubmitAttempt struct {
	DurationSeconds *int           `json:"duration_seconds"`
	Answers         []SubmitAnswer `json:"answers"`
}

type SubmitAnswer struct {
	QuestionID string   `json:"question_id"`
	OptionIDs  []string `json:"option_ids"`
}
