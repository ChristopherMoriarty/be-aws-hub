package handler

import (
	"net/http"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/dto/response"
)

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK
	if err := h.pool.Ping(r.Context()); err != nil {
		status = "unavailable"
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, response.Health{Status: status})
}
