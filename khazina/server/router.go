package server

import (
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/khazina/internal/config"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/admin"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/middlewares"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1/authpbv1connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/workspace/v1/workspacepbv1connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/web"
)

func BuildRouter(cfg *config.Config, deps *dependencies) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middlewares.ZeroLogger)

	mountHealthCheck(r, deps)

	r.Group(func(webRouter chi.Router) {
		webRouter.Use(middlewares.DynamicCORS(deps.tenantSvc))
		webRouter.Use(middlewares.TenantResolver(deps.tenantSvc))

		mountWebRoutes(webRouter, cfg, deps)
	})

	r.Group(func(dashRouter chi.Router) {
		dashRouter.Use(cors.AllowAll().Handler)

		mountDashboardRoutes(dashRouter, deps)
		mountReflection(dashRouter)
	})

	r.Group(func(adminRouter chi.Router) {
		mountAdminRoutes(adminRouter, cfg, deps)
	})

	return r
}

func mountHealthCheck(r chi.Router, deps *dependencies) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.dbPool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func mountWebRoutes(r chi.Router, cfg *config.Config, deps *dependencies) {
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

func mountAdminRoutes(r chi.Router, cfg *config.Config, deps *dependencies) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(admin.APIKeyAuth(cfg.AdminAPIKey))
		deps.adminHandlers.RegisterRoutes(r)
	})
}

func mountDashboardRoutes(r chi.Router, deps *dependencies) {
	interceptors := []connect.Interceptor{
		deps.authInterceptor,
		validate.NewInterceptor(),
	}

	dashboardAuthPath, dashboardAuthHandler := authpbv1connect.NewAuthServiceHandler(
		deps.dashboardAuthSvc,
		connect.WithInterceptors(interceptors...),
	)
	r.Mount(dashboardAuthPath, dashboardAuthHandler)

	dashboardWorkspacePath, dashboardWorkspaceHandler := workspacepbv1connect.NewWorkspaceServiceHandler(
		deps.dashboardWorkspaceSvc,
		connect.WithInterceptors(interceptors...),
	)
	r.Mount(dashboardWorkspacePath, dashboardWorkspaceHandler)
}

func mountReflection(r chi.Router) {
	reflector := grpcreflect.NewStaticReflector(
		authpbv1connect.AuthServiceName,
		workspacepbv1connect.WorkspaceServiceName,
	)

	v1Path, v1Handler := grpcreflect.NewHandlerV1(reflector)
	r.Mount(v1Path, v1Handler)

	v1AlphaPath, v1AlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
	r.Mount(v1AlphaPath, v1AlphaHandler)

	log.Debug().Msg("reflection mounted on " + v1Path + " and " + v1AlphaPath)
}
