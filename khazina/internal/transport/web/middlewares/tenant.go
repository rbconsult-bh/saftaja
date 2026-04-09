package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type contextKey string

// ProjectContextKey is the key for storing the resolved project in request context.
const ProjectContextKey contextKey = "project"

// TenantResolver resolves the project from the Host header via tenant service.
// If no project is found for the host, the request continues without tenant context
// (allows /health and /verify-domain to work).
func TenantResolver(svc tenant.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host
			// Strip port if present
			if idx := strings.Index(host, ":"); idx != -1 {
				host = host[:idx]
			}

			project, err := svc.GetProjectByDomain(r.Context(), host)
			if err == nil && project != nil {
				ctx := context.WithValue(r.Context(), ProjectContextKey, project)
				r = r.WithContext(ctx)
			}
			// If no project found, continue without tenant context

			next.ServeHTTP(w, r)
		})
	}
}

// GetProjectFromContext retrieves the resolved project from request context.
// Returns nil if no project was resolved (e.g., request via unknown domain).
func GetProjectFromContext(ctx context.Context) *store.Project {
	if p, ok := ctx.Value(ProjectContextKey).(*store.Project); ok {
		return p
	}
	return nil
}
