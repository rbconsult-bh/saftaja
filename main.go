package main

import (
	"net/http"

	"github.com/RBConsult-BH/pay/internal/config"
	"github.com/RBConsult-BH/pay/internal/web"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to loading config")
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	h := web.New(cfg.MPGSBaseURL, cfg.MPGSMerchantID)

	r.Get("/checkout/{pid}", h.CheckoutHandler)

	log.Info().Msg("starting listener on port 8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal().Err(err).Msg("failed to listen on port 8080")
	}
}
