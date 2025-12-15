package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	mpgsclient "github.com/RBConsult-BH/pay/internal/clients/mpgs"
	"github.com/RBConsult-BH/pay/internal/config"
	"github.com/RBConsult-BH/pay/internal/store"
	"github.com/RBConsult-BH/pay/internal/web"
	"github.com/RBConsult-BH/pay/internal/web/middlewares"
)

func main() {
	ctx := context.Background()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
		w.TimeFormat = time.RFC3339
	})).With().Timestamp().Logger()

	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBDatabase)

	// ======== MIGRATIONS ========
	log.Info().Msg("running database migrations...")
	if err := runMigrations(dsn); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	log.Info().Msg("migrations completed")
	// ======== MIGRATIONS ========

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to parse db config")
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create connection pool")
	}
	defer dbPool.Close()

	queries := store.New(dbPool)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middlewares.ZeroLogger)

	mpgsCli := mpgsclient.New(cfg.MPGSBaseURL, cfg.MPGSMerchantID, cfg.MPGSAPIPassword)
	h := web.New(cfg.MPGSBaseURL, cfg.MPGSMerchantID, mpgsCli, queries)

	// =========================================================================
	// ROUTING
	// =========================================================================
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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

	log.Info().Msgf("starting listener on port: %d", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r); err != nil {
		log.Fatal().Err(err).Msgf("failed to listen on port: %d", cfg.Port)
	}
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
