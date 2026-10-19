package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccessTokenIsNotARefreshToken(t *testing.T) {
	t.Parallel()

	tokens := NewTokens("0123456789abcdef", time.Minute, time.Hour)
	userID := uuid.New()
	pair, err := tokens.Issue(userID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := tokens.Parse(pair.AccessToken, AudienceRefresh); err == nil {
		t.Fatal("access token was accepted as a refresh token")
	}

	got, err := tokens.Parse(pair.RefreshToken, AudienceRefresh)
	if err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
	if got != userID {
		t.Fatalf("subject = %s, want %s", got, userID)
	}
}
