package main

import (
	"net/http"

	"github.com/RBConsult-BH/pay/internal/web"
	"github.com/rs/zerolog/log"
)

func main() {
	http.HandleFunc("/checkout", web.CheckoutHandler)

	log.Info().Msg("starting listener on port 8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal().Err(err).Msg("failed to listen on port 8080")
	}
}
