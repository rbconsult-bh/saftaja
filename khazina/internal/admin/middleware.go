package admin

import (
	"crypto/subtle"
	"net/http"
)

func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Admin-Key")
			if key == "" {
				respondError(w, "missing X-Admin-Key header", http.StatusUnauthorized)
				return
			}
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(key)) == 0 {
				respondError(w, "invalid API key", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
