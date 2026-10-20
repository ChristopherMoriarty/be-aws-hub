package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/auth"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/dto/request"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/dto/response"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/service"
)

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body request.Login
	if !h.readJSON(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Email) == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	pair, err := h.auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		h.log.Errorw("login", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, response.Token{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    pair.ExpiresIn,
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body request.Refresh
	if !h.readJSON(w, r, &body) {
		return
	}
	if body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	pair, err := h.auth.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		h.log.Errorw("refresh", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, response.Token{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    pair.ExpiresIn,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, response.User{
		ID:    principal.ID.String(),
		Email: principal.Email,
		Role:  principal.Role,
	})
}
