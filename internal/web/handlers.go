package web

import (
	"encoding/json"
	"fmt"
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
	if pid == "" {
		http.Error(w, "missing pid ", 400)
		return
	}

	log.Ctx(ctx).Info().Str("pid", pid).Msg("got pid :D")

	resp, err := h.mpgsCli.CreateSession(ctx, &mpgsclient.CreateSessionRequest{
		Session: &mpgsclient.CreateSessionRequestSession{
			AuthenticationLimit: utils.Ptr[int32](25),
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create session with mpgs")
		http.Error(w, "internal server error", 500)
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
		http.Error(w, "internal server error", 500)
		return
	}

	if err := templfiles.CheckoutPage(h.mpgsBaseURL, mpgsclient.APIVersion, h.mpgsMerchantID, resp.Data.Session.ID, pid).Render(ctx, w); err != nil {
		log.Error().Err(err).Msg("failed to render checkout page")
	}
}

func (h *handlers) CheckoutInitiateAuthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Ctx(ctx).Info().Msg("we are calling checkout initiate auth endpoint :D")

	pid := chi.URLParam(r, "pid")
	sid := chi.URLParam(r, "sid")
	if pid == "" || sid == "" {
		http.Error(w, "missing pid or session id", 400)
		return
	}
	log.Ctx(ctx).Info().Str("pid", pid).Msg("got pid :D")
	log.Ctx(ctx).Info().Str("sid", sid).Msg("got sid :D")

	txID := fmt.Sprintf("txn-init-%s", utils.GenerateRandomString(5))
	resp, err := h.mpgsCli.InitiateAuthentication(ctx, pid, txID, &mpgsclient.InitiateAuthenticationRequest{
		APIOperation: mpgsclient.OperationInitiateAuthentication,
		Authentication: mpgsclient.InitiateAuthenticationReqAuthentication{
			Channel: mpgsclient.ChannelPayerBrowser,
		},
		Order: mpgsclient.InitiateAuthenticationOrder{
			Currency: "BHD", // TODO: this should be fetched from DB
		},
		Session: mpgsclient.InitiateAuthenticationSession{ID: sid},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to initiate authentication with mpgs")
		http.Error(w, "internal server error", 500)
		return
	}

	var nextStep string
	switch resp.Data.Response.GatewayRecommendation {
	case mpgsclient.GatewayRecommendationProceed:
		nextStep = "authenticate"
	case mpgsclient.GatewayRecommendationDoNotProceed:
		nextStep = "cant_continue"
	case mpgsclient.GatewayRecommendationResubmitWithAltPay:
		nextStep = "another_payment_method"
	default:
		log.Ctx(ctx).Error().Str("gateway_recommendation_code", string(resp.Data.Response.GatewayRecommendation)).Msg("unexpected recommendation code")
		http.Error(w, "internal server error", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"next_step": nextStep,
		"tx_id":     txID,
	}); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode json")
		http.Error(w, "internal server error", 500)
	}
}

func (h *handlers) CheckoutProcessAuthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Ctx(ctx).Info().Msg("we are calling checkout process auth endpoint :D")

	pid := chi.URLParam(r, "pid")
	sid := chi.URLParam(r, "sid")
	txID := chi.URLParam(r, "txid")
	if pid == "" || sid == "" {
		http.Error(w, "missing pid or session id", 400)
		return
	}
	log.Ctx(ctx).Info().Str("pid", pid).Msg("got pid :D")
	log.Ctx(ctx).Info().Str("sid", sid).Msg("got sid :D")

	// {
	//   "apiOperation": "AUTHENTICATE_PAYER",
	//   "authentication": {
	//     "redirectResponseUrl": "https://rbconsult.bh"
	//   },
	//   "device": {
	//     "browser": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	//     "browserDetails": {
	//       "3DSecureChallengeWindowSize": "FULL_SCREEN",
	//       "acceptHeaders": "application/json",
	//       "colorDepth": 24,
	//       "javaEnabled": true,
	//       "language": "en-US",
	//       "screenHeight": 1080,
	//       "screenWidth": 1920,
	//       "timeZone": -180
	//     },
	//     "ipAddress": "127.0.0.1"
	//   },
	//   "order": {
	//     "amount": "{{amount}}",
	//     "currency": "{{currency}}"
	//   },
	//   "session": {
	//     "id": "{{sessionID}}"
	//   }
	// }

	resp, err := h.mpgsCli.AuthenticatePayer(ctx, pid, txID, &mpgsclient.AuthenticatePayerRequest{
		APIOperation: mpgsclient.OperationAuthenticatePayer,
		Authentication: mpgsclient.AuthenticatePayerReqAuthentication{
			RedirectResponseURL: "https://rbconsult.bh", // TODO: make it configurable, perhaps from DB even somehow :D
		},
		Device: mpgsclient.AuthenticatePayerReqDevice{
			Browser:        "",
			BrowserDetails: &mpgsclient.AuthenticatePayerReqBrowserDetails{},
			IPAddress:      "",
		},
		Order: mpgsclient.AuthenticatePayerReqOrder{
			Amount:   "100",
			Currency: "BHD",
		},
		Session: mpgsclient.AuthenticatePayerReqSession{ID: sid},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to authenticate payer with mpgs")
		http.Error(w, "internal server error", 500)
		return
	}

	var nextStep string
	var redirectHTML string

	switch resp.Data.Response.GatewayRecommendation {
	case mpgsclient.GatewayRecommendationProceed:
		if resp.Data.Authentication.Redirect.HTML != "" {
			nextStep = "3ds_challenge"
			redirectHTML = resp.Data.Authentication.Redirect.HTML
		} else {
			nextStep = "pay"
		}

	case mpgsclient.GatewayRecommendationDoNotProceed:
		nextStep = "cant_continue"

	case mpgsclient.GatewayRecommendationResubmitWithAltPay:
		nextStep = "another_payment_method"

	default:
		log.Ctx(ctx).Error().
			Str("gateway_recommendation", string(resp.Data.Response.GatewayRecommendation)).
			Msg("unexpected gateway recommendation")
		http.Error(w, "internal server error", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"next_step":     nextStep,
		"redirect_html": redirectHTML, // only if 3ds_challenge
		"tx_id":         txID,
	}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to encode json")
		http.Error(w, "internal server error", 500)
	}
}

func (h *handlers) CheckoutPayHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Ctx(ctx).Info().Msg("we are calling checkout pay endpoint :D")

	w.WriteHeader(200)
}
