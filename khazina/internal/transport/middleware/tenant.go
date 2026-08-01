package middleware

import (
	"net/http"
	"strings"

	saftajacontext "github.com/rbconsult-bh/saftaja/khazina/internal/pkg/context"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
)

func TenantResolver(svc tenant.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host
			if idx := strings.Index(host, ":"); idx != -1 {
				host = host[:idx]
			}

			project, err := svc.GetProjectByDomain(r.Context(), host)
			if err == nil && project != nil {
				ctx := saftajacontext.WithProject(r.Context(), project)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}
