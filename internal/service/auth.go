package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/auth"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/store/db"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

type Auth struct {
	queries *db.Queries
	tokens  *auth.Tokens
}

func NewAuth(queries *db.Queries, tokens *auth.Tokens) *Auth {
	return &Auth{queries: queries, tokens: tokens}
}

func (s *Auth) Login(ctx context.Context, email, password string) (auth.TokenPair, error) {
	user, err := s.queries.GetUserByEmail(ctx, strings.TrimSpace(email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.TokenPair{}, ErrInvalidCredentials
		}
		return auth.TokenPair{}, fmt.Errorf("get user: %w", err)
	}
	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		return auth.TokenPair{}, ErrInvalidCredentials
	}
	return s.tokens.Issue(user.ID)
}

func (s *Auth) Refresh(ctx context.Context, refreshToken string) (auth.TokenPair, error) {
	principal, err := s.principal(ctx, refreshToken, auth.AudienceRefresh)
	if err != nil {
		return auth.TokenPair{}, err
	}
	return s.tokens.Issue(principal.ID)
}

func (s *Auth) Principal(ctx context.Context, accessToken string) (auth.Principal, error) {
	return s.principal(ctx, accessToken, auth.AudienceAccess)
}

func (s *Auth) principal(ctx context.Context, raw, audience string) (auth.Principal, error) {
	userID, err := s.tokens.Parse(raw, audience)
	if err != nil {
		return auth.Principal{}, ErrUnauthorized
	}
	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.Principal{}, ErrUnauthorized
		}
		return auth.Principal{}, fmt.Errorf("get user: %w", err)
	}
	return principalFrom(user), nil
}

func principalFrom(user db.User) auth.Principal {
	return auth.Principal{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}
}
