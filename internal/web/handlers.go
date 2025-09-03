package web

import (
	"context"
	"net/http"

	"github.com/RBConsult-BH/pay/internal/web/templfiles"
	"github.com/rs/zerolog/log"
)

func CheckoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	if err := templfiles.CheckoutPage().Render(ctx, w); err != nil {
		log.Error().Err(err).Msg("failed to render checkout page")
	}
}
