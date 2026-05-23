package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/auth"
)

type contextKey string

const UserClaimsKey contextKey = "user_claims"

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				writeAuthError(w, apperror.Unauthorized("missing or invalid authorization header"))
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			claims, err := auth.ParseToken(secret, token)
			if err != nil {
				writeAuthError(w, apperror.Unauthorized("invalid or expired token"))
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(auth.Claims)
	return claims, ok
}

func writeAuthError(w http.ResponseWriter, err *apperror.HTTPError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	_, _ = w.Write([]byte(`{"error":"` + err.Message + `"}`))
}
