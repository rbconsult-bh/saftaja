package main

import (
	"net/http"
	"os"
	"time"

	mpgsclient "github.com/RBConsult-BH/pay/internal/clients/mpgs"
	"github.com/RBConsult-BH/pay/internal/config"
	"github.com/RBConsult-BH/pay/internal/web"
	"github.com/RBConsult-BH/pay/internal/web/middlewares"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
		w.TimeFormat = time.RFC3339
	})).With().Timestamp().Logger()

	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to loading config")
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middlewares.ZeroLogger)

	mpgsCli := mpgsclient.New(cfg.MPGSBaseURL, cfg.MPGSMerchantID, cfg.MPGSAPIPassword)

	h := web.New(cfg.MPGSBaseURL, cfg.MPGSMerchantID, mpgsCli)

	r.Get("/checkout/{pid}", h.CheckoutHandler)
	r.Post("/checkout/{pid}/initiate-auth/{sid}", h.CheckoutInitiateAuthHandler)
	r.Post("/checkout/{pid}/process-auth/{sid}/{txid}", h.CheckoutProcessAuthHandler)
	r.Post("/checkout/{pid}/pay/{sid}/{txid}", h.CheckoutPayHandler)

	log.Info().Msg("starting listener on port 8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal().Err(err).Msg("failed to listen on port 8080")
	}
}
