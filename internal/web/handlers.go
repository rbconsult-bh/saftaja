package web

import (
	"context"
	"net/http"

	"github.com/RBConsult-BH/pay/internal/clients/mpgs"
	"github.com/RBConsult-BH/pay/internal/web/templfiles"
	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"
)

type handlers struct {
	mpgsBaseURL    string
	mpgsMerchantID string
}

func New(mpgsBaseURL, mpgsMerchantID string) Handlers {
	return &handlers{
		mpgsBaseURL:    mpgsBaseURL,
		mpgsMerchantID: mpgsMerchantID,
	}
}

func (h *handlers) CheckoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// TODO: get this from store in the future when payment links concept is made in the db
	// for now we are acting like this is a sessionID, but yea in the future it should be the payment id
	pid := chi.URLParam(r, "pid")

	if err := templfiles.CheckoutPage(h.mpgsBaseURL, mpgs.APIVersion, h.mpgsMerchantID, pid).Render(ctx, w); err != nil {
		log.Error().Err(err).Msg("failed to render checkout page")
	}
}
