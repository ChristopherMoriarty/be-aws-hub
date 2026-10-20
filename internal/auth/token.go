package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	AudienceAccess  = "access"
	AudienceRefresh = "refresh"

	RoleUser  = "user"
	RoleAdmin = "admin"
)

type Principal struct {
	ID    uuid.UUID
	Email string
	Role  string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type Tokens struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokens(secret string, accessTTL, refreshTTL time.Duration) *Tokens {
	return &Tokens{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (t *Tokens) Issue(userID uuid.UUID) (TokenPair, error) {
	access, err := t.sign(userID, AudienceAccess, t.accessTTL)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := t.sign(userID, AudienceRefresh, t.refreshTTL)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(t.accessTTL.Seconds()),
	}, nil
}

func (t *Tokens) Parse(raw, audience string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return t.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse token: %w", err)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("token subject: %w", err)
	}
	return userID, nil
}

func (t *Tokens) sign(userID uuid.UUID, audience string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		Audience:  jwt.ClaimStrings{audience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
