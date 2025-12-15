package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	mpgsclient "github.com/RBConsult-BH/pay/internal/clients/mpgs"
	"github.com/RBConsult-BH/pay/internal/config"
	"github.com/RBConsult-BH/pay/internal/store"
	"github.com/RBConsult-BH/pay/internal/web"
	"github.com/RBConsult-BH/pay/internal/web/middlewares"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
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
