package middleware

import (
	"net/http"

	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
)

func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeAuthError(w, apperror.Unauthorized("unauthorized"))
				return
			}
			if !allowed[claims.Role] {
				writeAuthError(w, apperror.Forbidden("insufficient permissions"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
