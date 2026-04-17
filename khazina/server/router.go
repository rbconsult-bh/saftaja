package server

import (
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/khazina/internal/config"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/admin"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1/authpbv1connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/web"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/web/middlewares"
)

func BuildRouter(cfg *config.Config, deps *dependencies) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middlewares.ZeroLogger)
	r.Use(middlewares.DynamicCORS(deps.tenantSvc))
	r.Use(middlewares.TenantResolver(deps.tenantSvc))

	mountHealthCheck(r, deps)
	mountWebRoutes(r, cfg, deps)
	mountAdminRoutes(r, cfg, deps)
	mountDashboardRoutes(r, deps)
	mountReflection(r, deps)

	return r
}

func mountHealthCheck(r *chi.Mux, deps *dependencies) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.dbPool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func mountWebRoutes(r *chi.Mux, cfg *config.Config, deps *dependencies) {
	h := web.New(deps.paymentSvc, cfg.VerifyDomainSecret)

	r.Get("/verify-domain", h.VerifyDomainHandler)
	r.Get("/checkout/{invoice_id}", h.CheckoutPageHandler)
	r.Post("/checkout/{invoice_id}/initiate", h.InitiateSessionHandler)
	r.Route("/checkout/{invoice_id}/pay/card/{payment_session_id}", func(r chi.Router) {
		r.Post("/initiate-auth", h.CardInitiateAuthHandler)
		r.Post("/process-auth", h.CardProcessAuthHandler)
		r.Post("/finalize", h.CardFinalizeHandler)
	})
}

func mountAdminRoutes(r *chi.Mux, cfg *config.Config, deps *dependencies) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(admin.APIKeyAuth(cfg.AdminAPIKey))
		deps.adminHandlers.RegisterRoutes(r)
	})
}

func mountDashboardRoutes(r *chi.Mux, deps *dependencies) {
	dashboardAuthPath, dashboardAuthHandler := authpbv1connect.NewAuthServiceHandler(
		deps.dashboardAuthSvc,
		connect.WithInterceptors(
			deps.authInterceptor,
			validate.NewInterceptor(),
		),
	)
	r.Mount(dashboardAuthPath, dashboardAuthHandler)
}

func mountReflection(r *chi.Mux, deps *dependencies) {
	reflector := grpcreflect.NewStaticReflector(
		authpbv1connect.AuthServiceName,
	)

	v1Path, v1Handler := grpcreflect.NewHandlerV1(reflector)
	r.Mount(v1Path, v1Handler)

	v1AlphaPath, v1AlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
	r.Mount(v1AlphaPath, v1AlphaHandler)

	log.Debug().Msg("reflection mounted on " + v1Path + " and " + v1AlphaPath)
}
