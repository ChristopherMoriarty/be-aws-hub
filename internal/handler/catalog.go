package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/dto/request"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/dto/response"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/service"
)

func (h *Handler) ListCertifications(w http.ResponseWriter, r *http.Request) {
	items, err := h.catalog.ListCertifications(r.Context())
	if err != nil {
		h.writeUseCaseError(w, err, "list certifications")
		return
	}
	body := make([]response.Certification, 0, len(items))
	for _, item := range items {
		body = append(body, certificationResponse(item))
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) GetCertification(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "certificationID")
	if !ok {
		return
	}
	item, err := h.catalog.GetCertification(r.Context(), id)
	if err != nil {
		h.writeUseCaseError(w, err, "get certification")
		return
	}
	writeJSON(w, http.StatusOK, certificationResponse(item))
}

func (h *Handler) CreateCertification(w http.ResponseWriter, r *http.Request) {
	var body request.Certification
	if !h.readJSON(w, r, &body) {
		return
	}
	item, err := h.catalog.CreateCertification(r.Context(), certificationInput(body))
	if err != nil {
		h.writeUseCaseError(w, err, "create certification")
		return
	}
	writeJSON(w, http.StatusCreated, certificationResponse(item))
}

func (h *Handler) UpdateCertification(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "certificationID")
	if !ok {
		return
	}
	var body request.Certification
	if !h.readJSON(w, r, &body) {
		return
	}
	item, err := h.catalog.UpdateCertification(r.Context(), id, certificationInput(body))
	if err != nil {
		h.writeUseCaseError(w, err, "update certification")
		return
	}
	writeJSON(w, http.StatusOK, certificationResponse(item))
}

func (h *Handler) DeleteCertification(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "certificationID")
	if !ok {
		return
	}
	if err := h.catalog.DeleteCertification(r.Context(), id); err != nil {
		h.writeUseCaseError(w, err, "delete certification")
		return
	}
	writeNoContent(w)
}

func (h *Handler) ListTopics(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "certificationID")
	if !ok {
		return
	}
	items, err := h.catalog.ListTopics(r.Context(), id)
	if err != nil {
		h.writeUseCaseError(w, err, "list topics")
		return
	}
	body := make([]response.Topic, 0, len(items))
	for _, item := range items {
		body = append(body, topicResponse(item))
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "certificationID")
	if !ok {
		return
	}
	var body request.Topic
	if !h.readJSON(w, r, &body) {
		return
	}
	item, err := h.catalog.CreateTopic(r.Context(), id, topicInput(body))
	if err != nil {
		h.writeUseCaseError(w, err, "create topic")
		return
	}
	writeJSON(w, http.StatusCreated, topicResponse(item))
}

func (h *Handler) UpdateTopic(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "topicID")
	if !ok {
		return
	}
	var body request.Topic
	if !h.readJSON(w, r, &body) {
		return
	}
	item, err := h.catalog.UpdateTopic(r.Context(), id, topicInput(body))
	if err != nil {
		h.writeUseCaseError(w, err, "update topic")
		return
	}
	writeJSON(w, http.StatusOK, topicResponse(item))
}

func (h *Handler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, "topicID")
	if !ok {
		return
	}
	if err := h.catalog.DeleteTopic(r.Context(), id); err != nil {
		h.writeUseCaseError(w, err, "delete topic")
		return
	}
	writeNoContent(w)
}

func (h *Handler) pathID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) writeUseCaseError(w http.ResponseWriter, err error, op string) {
	var invalidErr *service.InvalidError
	if errors.As(err, &invalidErr) {
		writeError(w, http.StatusBadRequest, invalidErr.Message)
		return
	}
	var conflictErr *service.ConflictError
	if errors.As(err, &conflictErr) {
		writeError(w, http.StatusConflict, conflictErr.Message)
		return
	}
	var notFoundErr *service.NotFoundError
	if errors.As(err, &notFoundErr) {
		writeError(w, http.StatusNotFound, notFoundErr.Message)
		return
	}
	h.log.Errorw(op, "error", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func certificationInput(body request.Certification) service.CertificationInput {
	return service.CertificationInput{
		Code:            body.Code,
		Title:           body.Title,
		QuestionCount:   body.QuestionCount,
		DurationSeconds: body.DurationSeconds,
		PassingPercent:  body.PassingPercent,
	}
}

func topicInput(body request.Topic) service.TopicInput {
	return service.TopicInput{Title: body.Title, SortOrder: body.SortOrder}
}

func certificationResponse(item service.Certification) response.Certification {
	return response.Certification{
		ID:              item.ID.String(),
		Code:            item.Code,
		Title:           item.Title,
		QuestionCount:   item.QuestionCount,
		DurationSeconds: item.DurationSeconds,
		PassingPercent:  item.PassingPercent,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func topicResponse(item service.Topic) response.Topic {
	return response.Topic{
		ID:              item.ID.String(),
		CertificationID: item.CertificationID.String(),
		Title:           item.Title,
		SortOrder:       item.SortOrder,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}
