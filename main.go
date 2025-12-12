package main

import (
	"context"
	"net/http"
	"os"
	"time"

	mpgsclient "github.com/RBConsult-BH/pay/internal/clients/mpgs"
	"github.com/RBConsult-BH/pay/internal/config"
	"github.com/RBConsult-BH/pay/internal/store"
	"github.com/RBConsult-BH/pay/internal/web"
	"github.com/RBConsult-BH/pay/internal/web/middlewares"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
		log.Fatal().Err(err).Msg("failed to loading config")
	}

	pgxConn, err := pgx.ConnectConfig(ctx, &pgx.ConnConfig{
		Config: pgconn.Config{
			Host:     cfg.DBHost,
			Port:     cfg.DBPort,
			Database: cfg.DBDatabase,
			User:     cfg.DBUser,
			Password: cfg.DBPassword,
		},
		Tracer: nil, // TODO: handle this or use it somehow :D
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect using pgx to db")
	}

	queries := store.New(pgxConn)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middlewares.ZeroLogger)

	mpgsCli := mpgsclient.New(cfg.MPGSBaseURL, cfg.MPGSMerchantID, cfg.MPGSAPIPassword)

	h := web.New(cfg.MPGSBaseURL, cfg.MPGSMerchantID, mpgsCli, *queries)

	r.Get("/checkout/{invoice_id}", h.CheckoutHandler)
	r.Post("/checkout/{invoice_id}/initiate-auth", h.CheckoutInitiateAuthHandler)
	r.Post("/checkout/{invoice_id}/process-auth", h.CheckoutProcessAuthHandler)
	r.Post("/checkout/{invoice_id}/pay", h.CheckoutPayHandler)

	log.Info().Msg("starting listener on port 8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal().Err(err).Msg("failed to listen on port 8080")
	}
}
