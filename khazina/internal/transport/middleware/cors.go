package middleware

import (
	"net/http"
	"strings"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
)

// DynamicCORS validates CORS origins against project custom_domains via tenant service.
// Only origins that match a registered custom_domain are allowed.
func DynamicCORS(svc tenant.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" {
				// Extract host from origin (remove protocol)
				originHost := strings.TrimPrefix(origin, "https://")
				originHost = strings.TrimPrefix(originHost, "http://")

				// Check if origin is a valid project custom_domain
				valid, _ := svc.IsDomainValid(r.Context(), originHost)
				if valid {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, Idempotency-Key")
					w.Header().Set("Access-Control-Max-Age", "86400")
				}
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
