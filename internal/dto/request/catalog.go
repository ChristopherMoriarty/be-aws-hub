package request

type Certification struct {
	Code            string `json:"code"`
	Title           string `json:"title"`
	QuestionCount   int    `json:"question_count"`
	DurationSeconds int    `json:"duration_seconds"`
	PassingPercent  int    `json:"passing_percent"`
}

type Topic struct {
	Title     string `json:"title"`
	SortOrder *int   `json:"sort_order"`
}

type Option struct {
	Body      string `json:"body"`
	IsCorrect bool   `json:"is_correct"`
}

type Question struct {
	Body        string   `json:"body"`
	Explanation *string  `json:"explanation"`
	Difficulty  string   `json:"difficulty"`
	ImageURL    *string  `json:"image_url"`
	Options     []Option `json:"options"`
}
