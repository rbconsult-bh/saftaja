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
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/checkout"
	saftamiddleware "github.com/rbconsult-bh/saftaja/khazina/internal/transport/middleware"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1/authpbv1connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/project/v1/projectpbv1connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/workspace/v1/workspacepbv1connect"
)

func BuildRouter(cfg *config.Config, deps *dependencies) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(saftamiddleware.ZeroLogger)

	mountHealthCheck(r, deps)

	r.Group(func(webRouter chi.Router) {
		webRouter.Use(saftamiddleware.DynamicCORS(deps.tenantSvc))
		webRouter.Use(saftamiddleware.TenantResolver(deps.tenantSvc))

		mountWebRoutes(webRouter, cfg, deps)
	})

	r.Group(func(dashRouter chi.Router) {
		dashRouter.Use(cors.AllowAll().Handler)

		mountDashboardRoutes(dashRouter, deps)
		mountReflection(dashRouter)
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
	h := checkout.New(deps.billingSvc, deps.tenantSvc, deps.gatewaySvc, cfg.VerifyDomainSecret)

	r.Get("/verify-domain", h.VerifyDomainHandler)
	r.Get("/checkout/{invoice_id}", h.CheckoutPageHandler)
	r.Post("/checkout/{invoice_id}/payment-intents", h.CreatePaymentIntentHandler)
	r.Route("/checkout/{invoice_id}/payment-intents/{payment_intent_id}", func(r chi.Router) {
		r.Route("/card-authentication", func(r chi.Router) {
			r.Post("/prepare", h.PrepareCardAuthenticationHandler)
			r.Post("/authenticate", h.AuthenticateCardholderHandler)
			r.Post("/return", h.CardAuthenticationReturnHandler)
			r.Post("/verify", h.VerifyCardAuthenticationHandler)
		})
		r.Post("/capture", h.CapturePaymentIntentHandler)
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

	dashboardProjectPath, dashboardProjectHandler := projectpbv1connect.NewProjectServiceHandler(
		deps.dashboardProjectSvc,
		connect.WithInterceptors(interceptors...),
	)
	r.Mount(dashboardProjectPath, dashboardProjectHandler)
}

func mountReflection(r chi.Router) {
	reflector := grpcreflect.NewStaticReflector(
		authpbv1connect.AuthServiceName,
		workspacepbv1connect.WorkspaceServiceName,
		projectpbv1connect.ProjectServiceName,
	)

	v1Path, v1Handler := grpcreflect.NewHandlerV1(reflector)
	r.Mount(v1Path, v1Handler)

	v1AlphaPath, v1AlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
	r.Mount(v1AlphaPath, v1AlphaHandler)

	log.Debug().Msg("reflection mounted on " + v1Path + " and " + v1AlphaPath)
}
