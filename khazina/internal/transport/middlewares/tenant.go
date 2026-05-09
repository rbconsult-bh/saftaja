package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
)

type contextKey string

const ProjectContextKey contextKey = "project"

func TenantResolver(svc tenant.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host
			if idx := strings.Index(host, ":"); idx != -1 {
				host = host[:idx]
			}

			project, err := svc.GetProjectByDomain(r.Context(), host)
			if err == nil && project != nil {
				ctx := context.WithValue(r.Context(), ProjectContextKey, project)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetProjectFromContext(ctx context.Context) *tenant.Project {
	if p, ok := ctx.Value(ProjectContextKey).(*tenant.Project); ok {
		return p
	}
	return nil
}
