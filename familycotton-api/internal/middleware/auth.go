package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/service"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RoleKey   contextKey = "role"
)

func GetUserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(UserIDKey).(uuid.UUID)
	return id
}

func GetRole(ctx context.Context) string {
	role, _ := ctx.Value(RoleKey).(string)
	return role
}

func jsonError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"code":"%s","message":"%s"}}`, code, message)
}

func Auth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				jsonError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				jsonError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization format")
				return
			}

			userID, role, err := authService.ParseAccessToken(parts[1])
			if err != nil {
				jsonError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, RoleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
