package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/resendlabs/resend-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/auth"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/payment"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
	"github.com/rbconsult-bh/saftaja/khazina/internal/clients/email"
	"github.com/rbconsult-bh/saftaja/khazina/internal/config"
	_ "github.com/rbconsult-bh/saftaja/khazina/internal/connectors/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/admin"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/dashboard/authv1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1/authpbv1connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/web"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/web/middlewares"
)

func main() {
	ctx := context.Background()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBDatabase)

	log.Info().Msg("running database migrations...")
	if err := runMigrations(dsn); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	log.Info().Msg("migrations completed")

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to parse db config")
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create connection pool")
	}
	defer dbPool.Close()

	queries := store.NewTransactionQuerier(dbPool)

	encryptionKey, err := cfg.GetEncryptionKey()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get encryption key")
	}

	tenantSvc := tenant.NewService(queries)
	paymentSvc := payment.NewService(dbPool, queries, encryptionKey)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middlewares.ZeroLogger)

	r.Use(middlewares.DynamicCORS(tenantSvc))
	r.Use(middlewares.TenantResolver(tenantSvc))

	h := web.New(paymentSvc, cfg.VerifyDomainSecret)

	// =========================================================================
	// ROUTING
	// =========================================================================
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := dbPool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Domain verification for Caddy/Traefik on-demand TLS
	r.Get("/verify-domain", h.VerifyDomainHandler)

	r.Get("/checkout/{invoice_id}", h.CheckoutPageHandler)
	r.Post("/checkout/{invoice_id}/initiate", h.InitiateSessionHandler)
	r.Route("/checkout/{invoice_id}/pay/card/{payment_session_id}", func(r chi.Router) {
		r.Post("/initiate-auth", h.CardInitiateAuthHandler)
		r.Post("/process-auth", h.CardProcessAuthHandler)
		r.Post("/finalize", h.CardFinalizeHandler)
	})

	// TODO: in the future, insha Allah :D
	// r.Route("/checkout/{invoice_id}/pay/wallet/{session_id}", func(r chi.Router) {
	// 	r.Post("/finalize", h.WalletPayHandler)
	// })

	adminSvc := admin.NewService(queries, encryptionKey)
	adminHandlers := admin.NewHandlers(adminSvc)
	r.Route("/admin", func(r chi.Router) {
		r.Use(admin.APIKeyAuth(cfg.AdminAPIKey))
		adminHandlers.RegisterRoutes(r)
	})

	var emailer email.Emailer
	switch cfg.Emailer {
	case string(email.EmailerName_Resend):
		resendCli := resend.NewClient(cfg.ResendAPIKey)
		emailer = email.NewResendEmailer(resendCli)
	case string(email.EmailerName_Stdout):
		emailer = email.NewStdoutEmailer()
	default:
		log.Fatal().Msgf("emailer is not expected: %v", cfg.Emailer)
	}

	emailTemplates := email.NewTemplates(cfg.Domain)

	authSvc := auth.New(dbPool, queries, emailer, emailTemplates)

	dashboardAuthSvc := authv1.New(authSvc)
	dashboardAuthPath, dashboardAuthHandler := authpbv1connect.NewAuthServiceHandler(
		dashboardAuthSvc,
		connect.WithInterceptors(validate.NewInterceptor()),
	)

	r.Mount(dashboardAuthPath, dashboardAuthHandler)

	reflector := grpcreflect.NewStaticReflector(
		authpbv1connect.AuthServiceName,
	)

	v1Path, v1Handler := grpcreflect.NewHandlerV1(reflector)
	r.Mount(v1Path, v1Handler)

	v1AlphaPath, v1AlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
	r.Mount(v1AlphaPath, v1AlphaHandler)

	log.Debug().Msg("reflection mounted on " + v1Path + " and " + v1AlphaPath)

	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)

	// Server with graceful shutdown
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		Protocols:    p,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().Msgf("starting listener on port: %d", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msgf("failed to listen on port: %d", cfg.Port)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	// Give outstanding requests 30 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server stopped")
}

func runMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("cannot connect to database: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("cannot create postgres driver: %w", err)
	}

	source, err := iofs.New(store.MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("cannot create iofs source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("cannot create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("cannot run migrations: %w", err)
	}

	return nil
}
