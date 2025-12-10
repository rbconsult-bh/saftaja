package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
	log.Ctx(ctx).Info().Str("txID", sid).Msg("got txID :D")

	var browserDetails mpgsclient.AuthenticatePayerReqBrowserDetails
	if err := json.NewDecoder(r.Body).Decode(&browserDetails); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to decode body into browser details struct")
		http.Error(w, "internal server error", 500)
		return
	}
	if r.Header.Get("Accept") != "" {
		browserDetails.AcceptHeaders = r.Header.Get("Accept")
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("remote_addr", r.RemoteAddr).Msg("failed to split host port of remote addr")
		http.Error(w, "internal server error", 500)
		return
	}
	if ip == "::1" {
		ip = "0000:0000:0000:0000:0000:0000:0000:0001" // Full IPv6 loopback
	}

	// TODO: read the header of cloudflare, otherwise use remote addr if no cloudflare proxy in front of this.
	//
	// if isCloudflareIP(ip) {
	// 	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
	// 		ip = cfIP
	// 	}
	// }
	//
	// if net.ParseIP(ip) == nil {
	// 	log.Ctx(ctx).Error().Str("ip", ip).Msg("invalid IP address")
	// 	http.Error(w, "internal server error", 500)
	// 	return
	// }

	resp, err := h.mpgsCli.AuthenticatePayer(ctx, pid, txID, &mpgsclient.AuthenticatePayerRequest{
		APIOperation: mpgsclient.OperationAuthenticatePayer,
		Authentication: mpgsclient.AuthenticatePayerReqAuthentication{
			RedirectResponseURL: fmt.Sprintf("http://localhost:8080/checkout/%s/pay/%s/%s", pid, sid, txID), // TODO: make it configurable, perhaps from DB even somehow :D
		},
		// TODO: browser and ip address from server, but auth payer req browser details from client parse from body
		Device: mpgsclient.AuthenticatePayerReqDevice{
			Browser:        r.Header.Get("User-Agent"),
			BrowserDetails: &browserDetails,
			IPAddress:      ip,
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

	pid := chi.URLParam(r, "pid")
	sid := chi.URLParam(r, "sid")
	authTxID := chi.URLParam(r, "txid") // this is the AUTHENTICATION tx id

	if pid == "" || sid == "" || authTxID == "" {
		http.Error(w, "missing required parameters", 400)
		return
	}

	log.Ctx(ctx).Info().
		Str("pid", pid).
		Str("sid", sid).
		Str("auth_tx_id", authTxID).
		Msg("calling checkout pay endpoint")

	authTxResp, err := h.mpgsCli.RetrieveTransaction(ctx, pid, authTxID)
	if err != nil {
		var errResp mpgsclient.ErrorResponse
		if errors.As(err, &errResp) {
			log.Ctx(ctx).Error().Err(errResp).Msg("failed to retrieve auth transaction")
			http.Error(w, "failed to verify authentication", 500)
			return
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to retrieve auth transaction")
		http.Error(w, "failed to verify authentication", 500)
		return
	}

	if authTxResp.Data.Transaction.AuthenticationStatus != mpgsclient.AuthStatusSuccessful {
		log.Ctx(ctx).Warn().
			Str("auth_status", string(authTxResp.Data.Transaction.AuthenticationStatus)).
			Msg("authentication not successful")
		http.Error(w, "authentication not completed", 400)
		return
	}

	payTxID := fmt.Sprintf("txn-pay-%s", utils.GenerateRandomString(5))

	log.Ctx(ctx).Info().
		Str("pay_tx_id", payTxID).
		Str("auth_tx_id", authTxID).
		Msg("executing payment with new transaction id")

	resp, err := h.mpgsCli.ExecutePay(ctx, pid, payTxID, &mpgsclient.ExecutePayRequest{
		APIOperation: mpgsclient.OperationPay,
		Authentication: mpgsclient.ExecutePayReqAuthentication{
			TransactionID: authTxID,
		},
		Order: mpgsclient.ExecutePayReqOrder{
			Amount:    fmt.Sprintf("%.2f", authTxResp.Data.Order.Amount),
			Currency:  authTxResp.Data.Order.Currency,
			Reference: fmt.Sprintf("order-%s", pid),
		},
		Session: mpgsclient.ExecutePayReqSession{
			ID: sid,
		},
		Transaction: &mpgsclient.ExecutePayReqTransaction{
			Reference: pid,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to execute payment")
		http.Error(w, "payment failed", 500)
		return
	}

	var status string
	var message string
	switch resp.Data.Response.GatewayCode {
	case mpgsclient.CodeApproved:
		status = "success"
		message = "Payment successful!"
		log.Ctx(ctx).Info().
			Str("order_id", resp.Data.Order.ID).
			Str("tx_id", payTxID).
			Msg("payment approved")
		// TODO: Update invoice status in DB to "paid"

	case mpgsclient.CodeDeclined:
		status = "declined"
		message = "Payment declined by bank"
		log.Ctx(ctx).Warn().Msg("payment declined")

	// TODO: fix this thingy :d
	// case mpgsclient.CodeError:
	// 	status = "error"
	// 	message = "Payment error occurred"
	// 	log.Ctx(ctx).Error().Msg("payment error")

	default:
		log.Ctx(ctx).Error().
			Str("gateway_code", string(resp.Data.Response.GatewayCode)).
			Msg("unexpected gateway code")
		status = "unknown"
		message = "Unknown payment status"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     status,
		"message":    message,
		"pay_tx_id":  payTxID,
		"auth_tx_id": authTxID,
		"order_id":   resp.Data.Order.ID,
		"amount":     resp.Data.Order.Amount,
		"currency":   resp.Data.Order.Currency,
	})
}
