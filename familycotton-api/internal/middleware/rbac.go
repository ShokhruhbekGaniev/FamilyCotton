package middleware

import (
	"net/http"
)

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r.Context())
			if !allowed[role] {
				jsonError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
