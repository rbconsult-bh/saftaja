package web

import (
	"net/http"

	mpgsclient "github.com/RBConsult-BH/pay/internal/clients/mpgs"
	"github.com/RBConsult-BH/pay/internal/utils"
	"github.com/RBConsult-BH/pay/internal/web/templfiles"
	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"
)

type handlers struct {
	mpgsBaseURL    string
	mpgsMerchantID string
	mpgsCli        mpgsclient.Client
}

func New(mpgsBaseURL, mpgsMerchantID string, mpgsCli mpgsclient.Client) Handlers {
	return &handlers{
		mpgsBaseURL:    mpgsBaseURL,
		mpgsMerchantID: mpgsMerchantID,
		mpgsCli:        mpgsCli,
	}
}

func (h *handlers) CheckoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: get this from store in the future when payment links concept is made in the db
	// for now we are acting like this is a sessionID, but yea in the future it should be the payment id
	pid := chi.URLParam(r, "pid")
	log.Ctx(ctx).Info().Str("pid", pid).Msg("got pid :D")

	resp, err := h.mpgsCli.CreateSession(ctx, &mpgsclient.CreateSessionRequest{
		Session: &mpgsclient.CreateSessionRequestSession{
			AuthenticationLimit: utils.Ptr[int32](25),
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create session with mpgs")
		w.WriteHeader(500)
		return
	}

	_, err = h.mpgsCli.UpdateSession(ctx, resp.Data.Session.ID, &mpgsclient.UpdateSessionRequest{
		Order: mpgsclient.UpdateSessionOrder{
			Amount:   "100", // TODO: this should be fetched from DB
			Currency: "BHD", // TODO: this should be fetched from DB
			ID:       pid,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update session with mpgs")
		w.WriteHeader(500)
		return
	}

	if err := templfiles.CheckoutPage(h.mpgsBaseURL, mpgsclient.APIVersion, h.mpgsMerchantID, resp.Data.Session.ID, pid).Render(ctx, w); err != nil {
		log.Error().Err(err).Msg("failed to render checkout page")
	}
}

func (h *handlers) CheckoutInitiateAuthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Ctx(ctx).Info().Msg("we are calling checkout initiate auth endpoint :D")

	w.WriteHeader(200)
}

func (h *handlers) CheckoutProcessAuthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Ctx(ctx).Info().Msg("we are calling checkout process auth endpoint :D")

	w.WriteHeader(200)
}

func (h *handlers) CheckoutPayHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Ctx(ctx).Info().Msg("we are calling checkout pay endpoint :D")

	w.WriteHeader(200)
}
