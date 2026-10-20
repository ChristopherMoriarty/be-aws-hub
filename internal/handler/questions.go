package handler

import (
	"net/http"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/dto/request"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/dto/response"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/service"
)

func (h *Handler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "topicID")
	if !ok {
		return
	}
	items, err := h.catalog.ListQuestions(r.Context(), id)
	if err != nil {
		h.writeUseCaseError(w, err, "list questions")
		return
	}
	body := make([]response.Question, 0, len(items))
	for _, item := range items {
		body = append(body, questionResponse(item))
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) GetQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "questionID")
	if !ok {
		return
	}
	item, err := h.catalog.GetQuestion(r.Context(), id)
	if err != nil {
		h.writeUseCaseError(w, err, "get question")
		return
	}
	writeJSON(w, http.StatusOK, questionResponse(item))
}

func (h *Handler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "topicID")
	if !ok {
		return
	}
	var body request.Question
	if !h.readJSON(w, r, &body) {
		return
	}
	item, err := h.catalog.CreateQuestion(r.Context(), id, questionInput(body))
	if err != nil {
		h.writeUseCaseError(w, err, "create question")
		return
	}
	writeJSON(w, http.StatusCreated, questionResponse(item))
}

func (h *Handler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "questionID")
	if !ok {
		return
	}
	var body request.Question
	if !h.readJSON(w, r, &body) {
		return
	}
	item, err := h.catalog.UpdateQuestion(r.Context(), id, questionInput(body))
	if err != nil {
		h.writeUseCaseError(w, err, "update question")
		return
	}
	writeJSON(w, http.StatusOK, questionResponse(item))
}

func (h *Handler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "questionID")
	if !ok {
		return
	}
	if err := h.catalog.DeleteQuestion(r.Context(), id); err != nil {
		h.writeUseCaseError(w, err, "delete question")
		return
	}
	writeNoContent(w)
}

func questionInput(body request.Question) service.QuestionInput {
	options := make([]service.OptionInput, len(body.Options))
	for i, option := range body.Options {
		options[i] = service.OptionInput{Body: option.Body, IsCorrect: option.IsCorrect}
	}
	if body.Options == nil {
		options = nil
	}
	return service.QuestionInput{
		Body:        body.Body,
		Explanation: body.Explanation,
		Difficulty:  body.Difficulty,
		ImageURL:    body.ImageURL,
		Options:     options,
	}
}

func questionResponse(item service.Question) response.Question {
	options := make([]response.Option, 0, len(item.Options))
	for _, option := range item.Options {
		options = append(options, response.Option{
			ID:        option.ID.String(),
			Body:      option.Body,
			IsCorrect: option.IsCorrect,
			SortOrder: option.SortOrder,
		})
	}
	return response.Question{
		ID:          item.ID.String(),
		TopicID:     item.TopicID.String(),
		Body:        item.Body,
		Explanation: item.Explanation,
		Difficulty:  item.Difficulty,
		ImageURL:    item.ImageURL,
		Options:     options,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}
