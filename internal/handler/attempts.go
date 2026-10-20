package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/dto/request"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/dto/response"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/service"
)

func (h *Handler) StartAttempt(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.principal(w, r)
	if !ok {
		return
	}
	certificationID, ok := h.pathID(w, r, "certificationID")
	if !ok {
		return
	}
	var body request.StartAttempt
	if !h.readJSON(w, r, &body) {
		return
	}
	topicID, ok := optionalID(w, body.TopicID)
	if !ok {
		return
	}
	item, err := h.attempts.Start(r.Context(), principal.ID, certificationID, service.StartAttemptInput{
		Mode:          body.Mode,
		TopicID:       topicID,
		OnlyMistakes:  body.OnlyMistakes,
		QuestionCount: body.QuestionCount,
	})
	if err != nil {
		h.writeUseCaseError(w, err, "start attempt")
		return
	}
	writeJSON(w, http.StatusCreated, attemptResponse(item))
}

func (h *Handler) SubmitAttempt(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.principal(w, r)
	if !ok {
		return
	}
	attemptID, ok := h.pathID(w, r, "attemptID")
	if !ok {
		return
	}
	var body request.SubmitAttempt
	if !h.readJSON(w, r, &body) {
		return
	}
	input, ok := submitInput(w, body)
	if !ok {
		return
	}
	item, err := h.attempts.Submit(r.Context(), principal.ID, attemptID, input)
	if err != nil {
		h.writeUseCaseError(w, err, "submit attempt")
		return
	}
	writeJSON(w, http.StatusOK, attemptResponse(item))
}

func (h *Handler) GetAttempt(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.principal(w, r)
	if !ok {
		return
	}
	attemptID, ok := h.pathID(w, r, "attemptID")
	if !ok {
		return
	}
	item, err := h.attempts.Get(r.Context(), principal.ID, attemptID)
	if err != nil {
		h.writeUseCaseError(w, err, "get attempt")
		return
	}
	writeJSON(w, http.StatusOK, attemptResponse(item))
}

func (h *Handler) ListAttempts(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.principal(w, r)
	if !ok {
		return
	}
	certificationID, ok := h.pathID(w, r, "certificationID")
	if !ok {
		return
	}
	items, err := h.attempts.List(r.Context(), principal.ID, certificationID)
	if err != nil {
		h.writeUseCaseError(w, err, "list attempts")
		return
	}
	body := make([]response.Attempt, 0, len(items))
	for _, item := range items {
		body = append(body, attemptResponse(item))
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) TopicProgress(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.principal(w, r)
	if !ok {
		return
	}
	certificationID, ok := h.pathID(w, r, "certificationID")
	if !ok {
		return
	}
	items, err := h.attempts.Progress(r.Context(), principal.ID, certificationID)
	if err != nil {
		h.writeUseCaseError(w, err, "topic progress")
		return
	}
	body := make([]response.TopicProgress, 0, len(items))
	for _, item := range items {
		body = append(body, response.TopicProgress{
			TopicID:            item.TopicID.String(),
			Title:              item.Title,
			SortOrder:          item.SortOrder,
			TotalQuestions:     item.TotalQuestions,
			AttemptedQuestions: item.AttemptedQuestions,
			MasteredQuestions:  item.MasteredQuestions,
			TotalAttempts:      item.TotalAttempts,
			CorrectAnswers:     item.CorrectAnswers,
			Accuracy:           item.Accuracy,
			Mastered:           item.Mastered,
		})
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) CertificationStats(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.principal(w, r)
	if !ok {
		return
	}
	certificationID, ok := h.pathID(w, r, "certificationID")
	if !ok {
		return
	}
	item, err := h.attempts.Stats(r.Context(), principal.ID, certificationID)
	if err != nil {
		h.writeUseCaseError(w, err, "certification stats")
		return
	}
	writeJSON(w, http.StatusOK, response.CertificationStats{
		TotalQuestions:     item.TotalQuestions,
		AttemptedQuestions: item.AttemptedQuestions,
		MasteredQuestions:  item.MasteredQuestions,
		NeedsReview:        item.NeedsReview,
		TotalAttempts:      item.TotalAttempts,
		CorrectAnswers:     item.CorrectAnswers,
		ExamCount:          item.ExamCount,
		Accuracy:           item.Accuracy,
	})
}

func optionalID(w http.ResponseWriter, raw *string) (*uuid.UUID, bool) {
	if raw == nil {
		return nil, true
	}
	id, err := uuid.Parse(*raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return nil, false
	}
	return &id, true
}

func submitInput(w http.ResponseWriter, body request.SubmitAttempt) (service.SubmitInput, bool) {
	answers := make([]service.AnswerInput, len(body.Answers))
	for i, answer := range body.Answers {
		questionID, err := uuid.Parse(answer.QuestionID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return service.SubmitInput{}, false
		}
		optionIDs := make([]uuid.UUID, len(answer.OptionIDs))
		for j, raw := range answer.OptionIDs {
			optionIDs[j], err = uuid.Parse(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return service.SubmitInput{}, false
			}
		}
		answers[i] = service.AnswerInput{QuestionID: questionID, OptionIDs: optionIDs}
	}
	return service.SubmitInput{DurationSeconds: body.DurationSeconds, Answers: answers}, true
}

func attemptResponse(item service.Attempt) response.Attempt {
	var topicID *string
	if item.TopicID != nil {
		value := item.TopicID.String()
		topicID = &value
	}
	var questions []response.AttemptQuestion
	if item.Questions != nil {
		questions = make([]response.AttemptQuestion, 0, len(item.Questions))
		for _, question := range item.Questions {
			questions = append(questions, attemptQuestionResponse(question))
		}
	}
	return response.Attempt{
		ID:              item.ID.String(),
		CertificationID: item.CertificationID.String(),
		TopicID:         topicID,
		Mode:            item.Mode,
		QuestionCount:   item.QuestionCount,
		OnlyMistakes:    item.OnlyMistakes,
		Score:           item.Score,
		CorrectCount:    item.CorrectCount,
		StartedAt:       item.StartedAt,
		CompletedAt:     item.CompletedAt,
		DurationSeconds: item.DurationSeconds,
		Questions:       questions,
	}
}

func attemptQuestionResponse(item service.AttemptQuestion) response.AttemptQuestion {
	options := make([]response.AttemptOption, 0, len(item.Options))
	for _, option := range item.Options {
		options = append(options, response.AttemptOption{
			ID:        option.ID.String(),
			Body:      option.Body,
			SortOrder: option.SortOrder,
			IsCorrect: option.IsCorrect,
		})
	}
	selected := make([]string, len(item.SelectedOptionIDs))
	for i, id := range item.SelectedOptionIDs {
		selected[i] = id.String()
	}
	if item.SelectedOptionIDs == nil {
		selected = nil
	}
	return response.AttemptQuestion{
		ID:                item.ID.String(),
		TopicID:           item.TopicID.String(),
		Position:          item.Position,
		Body:              item.Body,
		Explanation:       item.Explanation,
		Difficulty:        item.Difficulty,
		ImageURL:          item.ImageURL,
		Options:           options,
		SelectedOptionIDs: selected,
		Correct:           item.Correct,
	}
}
