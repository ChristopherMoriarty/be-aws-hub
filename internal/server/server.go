package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/auth"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/handler"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/logger"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/service"
)

func New(pool *pgxpool.Pool, log logger.Logger, authService *service.Auth) http.Handler {
	h := handler.New(pool, authService, log)

	r := chi.NewRouter()
	r.Use(
		middleware.RequestID,
		middleware.Recoverer,
		requestLogger(log),
	)
	r.Get("/health", h.Health)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", h.Login)
		r.Post("/auth/refresh", h.Refresh)
		r.Group(func(r chi.Router) {
			r.Use(auth.Require(authService.Principal))
			r.Get("/me", h.Me)
		})
	})

	return r
}

func requestLogger(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			started := time.Now()
			next.ServeHTTP(wrapped, r)
			log.Infow("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.Status(),
				"bytes", wrapped.BytesWritten(),
				"duration", time.Since(started).String(),
			)
		})
	}
}
