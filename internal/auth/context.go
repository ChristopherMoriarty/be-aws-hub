package auth

import (
	"context"
	"net/http"
	"strings"
)

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}

func Require(principal func(ctx context.Context, accessToken string) (Principal, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeUnauthorized(w)
				return
			}
			user, err := principal(r.Context(), token)
			if err != nil {
				writeUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), user)))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("{\"error\":\"unauthorized\"}\n"))
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "bearer") || token == "" {
		return "", false
	}
	return token, true
}
