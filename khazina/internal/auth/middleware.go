package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
)

// GetUserIDFromContext retrieves the authenticated user ID from request context.
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return id, ok
}

// RequireAuth validates a Bearer JWT or sk_* API key and injects the user ID into context.
// Returns 401 if the token/key is missing or invalid.
func RequireAuth(svc Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w)
				return
			}

			var userID uuid.UUID
			var err error

			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				userID, err = svc.ValidateJWT(token)
			} else if strings.HasPrefix(authHeader, "sk_") {
				userID, err = svc.ValidateAPIKey(r.Context(), authHeader)
			} else {
				respondUnauthorized(w)
				return
			}

			if err != nil {
				respondUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized"}`))
}
