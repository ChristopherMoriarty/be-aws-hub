package handler

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/logger"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/service"
)

type Handler struct {
	pool *pgxpool.Pool
	auth *service.Auth
	log  logger.Logger
}

func New(pool *pgxpool.Pool, auth *service.Auth, log logger.Logger) *Handler {
	return &Handler{pool: pool, auth: auth, log: log}
}
