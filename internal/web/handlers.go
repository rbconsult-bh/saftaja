package web

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	mpgsclient "github.com/RBConsult-BH/pay/internal/clients/mpgs"
	"github.com/RBConsult-BH/pay/internal/store"
	"github.com/RBConsult-BH/pay/internal/utils"
	"github.com/RBConsult-BH/pay/internal/web/templfiles"
)

type handlers struct {
	mpgsBaseURL    string
	mpgsMerchantID string
	mpgsCli        mpgsclient.Client
	queries        *store.Queries
}

func New(mpgsBaseURL, mpgsMerchantID string, mpgsCli mpgsclient.Client, queries *store.Queries) *handlers {
	return &handlers{
		mpgsBaseURL:    mpgsBaseURL,
		mpgsMerchantID: mpgsMerchantID,
		mpgsCli:        mpgsCli,
		queries:        queries,
	}
}

func (h *handlers) CheckoutPageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawInvoiceID := chi.URLParam(r, "invoice_id")

	invoiceID, err := uuid.Parse(rawInvoiceID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse invoice_id")
		http.Error(w, "invalid invoice_id", http.StatusBadRequest)
		return
	}

	invoice, err := h.queries.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Ctx(ctx).Info().Msg("invoice not found")
			http.Error(w, "invoice not found", http.StatusNotFound)
			return
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if invoice.Status == store.InvoiceStatusPaid {
		w.Write([]byte("Invoice already paid!"))
		return
	}

	accounts, err := h.queries.ListActiveGatewayAccounts(ctx, invoice.ProjectID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to fetch gateway accounts")
		http.Error(w, "configuration error", http.StatusInternalServerError)
		return
	}

	var options []templfiles.PaymentOption
	for _, acc := range accounts {
		var methods []string
		_ = json.Unmarshal(acc.PaymentMethods, &methods)

		for _, m := range methods {
			var label string
			switch m {
			case "card":
				label = "Credit / Debit Card"
			case "apple_pay":
				label = "Apple Pay"
			default:
				label = m
			}

			options = append(options, templfiles.PaymentOption{
				ID:    acc.ID.String(),
				Label: label,
				Type:  m,
			})
		}
	}

	if err := templfiles.CheckoutPage(h.mpgsBaseURL, mpgsclient.APIVersion, h.mpgsMerchantID, invoice.ID.String(), options).Render(ctx, w); err != nil {
		log.Error().Err(err).Msg("failed to render checkout page")
	}
}

func (h *handlers) InitiateSessionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	invoiceID, _ := uuid.Parse(chi.URLParam(r, "invoice_id"))

	var req struct {
		GatewayAccountID uuid.UUID                         `json:"gateway_account_id"`
		PaymentMethod    store.PaymentSessionPaymentMethod `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	invoice, _ := h.queries.GetInvoiceByID(ctx, invoiceID)
	account, err := h.queries.GetGatewayAccount(ctx, req.GatewayAccountID)
	if err != nil {
		http.Error(w, "invalid gateway account", http.StatusBadRequest)
		return
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("remote_addr", r.RemoteAddr).Msg("failed to split host port of remote addr")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if ip == "::1" {
		ip = "0000:0000:0000:0000:0000:0000:0000:0001" // Full IPv6 loopback
	}

	// TODO: read the header of cloudflare, otherwise use remote addr if no cloudflare proxy in front of this.
	// if isCloudflareIP(ip) {
	// 	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
	// 		ip = cfIP
	// 	}
	// }

	if net.ParseIP(ip) == nil {
		log.Ctx(ctx).Error().Str("ip", ip).Msg("invalid ip address")
		http.Error(w, "invalid ip address", http.StatusBadRequest)
		return
	}

	if req.PaymentMethod == store.PaymentSessionPaymentMethodCard || req.PaymentMethod == store.PaymentSessionPaymentMethodApplePay {
		resp, err := h.mpgsCli.CreateSession(ctx, &mpgsclient.CreateSessionRequest{
			Session: &mpgsclient.CreateSessionRequestSession{
				AuthenticationLimit: utils.Ptr[int32](25),
			},
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("mpgs create session failed")
			http.Error(w, "gateway error", http.StatusBadGateway)
			return
		}

		dbSession, err := h.queries.CreatePaymentSession(ctx, store.CreatePaymentSessionParams{
			InvoiceID:        invoice.ID,
			ProjectID:        invoice.ProjectID,
			GatewayAccountID: account.ID,
			GatewaySessionID: resp.Data.Session.ID,
			PaymentMethod:    req.PaymentMethod,
			PayerIp:          pgtype.Text{String: ip, Valid: true},
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("db session creation failed")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		_, err = h.mpgsCli.UpdateSession(ctx, resp.Data.Session.ID, &mpgsclient.UpdateSessionRequest{
			Order: mpgsclient.UpdateSessionOrder{
				Amount:   invoice.Amount.String(),
				Currency: invoice.Currency,
				ID:       invoice.ID.String(),
			},
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to update session at mpgs")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"action":             "render_embedded",
			"payment_session_id": dbSession.ID.String(),
			"mpgs_session_id":    resp.Data.Session.ID,
		})
		return
	}
}

func (h *handlers) CardInitiateAuthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse invoice_id")
		http.Error(w, "invalid invoice_id", http.StatusBadRequest)
		return
	}
	sessionID, err := uuid.Parse(chi.URLParam(r, "payment_session_id"))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse payment_session_id")
		http.Error(w, "invalid payment_session_id", http.StatusBadRequest)
		return
	}

	session, err := h.queries.GetPaymentSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Ctx(ctx).Error().Err(err).Msg("failed to get payment session by id")
			http.Error(w, "payment session doesn't exist", http.StatusNotFound)
			return
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get payment session by id")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if session.InvoiceID != invoiceID {
		log.Ctx(ctx).Error().Err(err).Msg("provided invoice id is not for provided session id")
		http.Error(w, "invalid session", http.StatusForbidden)
		return
	}

	invoice, err := h.queries.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id")
			http.Error(w, "invoice doesn't exist", http.StatusNotFound)
			return
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	gatewayTxID := uuid.New()
	dbTx, err := h.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		PaymentSessionID:     session.ID,
		InvoiceID:            invoice.ID,
		ProjectID:            invoice.ProjectID,
		TransactionType:      store.TransactionTransactionTypeInitiateAuthentication,
		GatewayTransactionID: gatewayTxID.String(),
		Amount:               invoice.Amount,
		Currency:             invoice.Currency,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create transaction in db")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp, err := h.mpgsCli.InitiateAuthentication(ctx, invoice.ProjectID.String(), gatewayTxID.String(), &mpgsclient.InitiateAuthenticationRequest{
		APIOperation: mpgsclient.OperationInitiateAuthentication,
		Authentication: mpgsclient.InitiateAuthenticationReqAuthentication{
			Channel: mpgsclient.ChannelPayerBrowser,
		},
		Order:   mpgsclient.InitiateAuthenticationOrder{Currency: invoice.Currency},
		Session: mpgsclient.InitiateAuthenticationSession{ID: session.GatewaySessionID},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("mpgs init auth failed")
		http.Error(w, "gateway error", http.StatusInternalServerError)
		return
	}
	status := store.TransactionStatusFailed
	if resp.Data.Result == mpgsclient.ResultSuccess {
		status = store.TransactionStatusSuccess
	}

	// TODO: make the mpgs client return the raw resp beside the parsed one :D
	rawResp, err := json.Marshal(resp)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to marshal raw resp")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.queries.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{
		ID: dbTx.ID, Status: status, RawResponse: rawResp,
	}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update transaction in db")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var nextStep string
	switch resp.Data.Response.GatewayRecommendation {
	case mpgsclient.GatewayRecommendationProceed:
		nextStep = "authenticate"
	case mpgsclient.GatewayRecommendationDoNotProceed:
		nextStep = "cant_continue"
	default:
		nextStep = "error"
	}

	if err := json.NewEncoder(w).Encode(map[string]any{"next_step": nextStep}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to encode the response")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *handlers) CardProcessAuthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse invoice_id")
		http.Error(w, "invalid invoice_id", http.StatusBadRequest)
		return
	}
	sessionID, err := uuid.Parse(chi.URLParam(r, "payment_session_id"))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse payment_session_id")
		http.Error(w, "invalid payment_session_id", http.StatusBadRequest)
		return
	}

	session, err := h.queries.GetPaymentSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Ctx(ctx).Error().Err(err).Msg("failed to get payment session by id")
			http.Error(w, "payment session doesn't exist", http.StatusNotFound)
			return
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get payment session by id")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if session.InvoiceID != invoiceID {
		log.Ctx(ctx).Error().Err(err).Msg("provided invoice id is not for provided session id")
		http.Error(w, "invalid session", http.StatusForbidden)
		return
	}

	invoice, err := h.queries.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id")
			http.Error(w, "invoice doesn't exist", http.StatusNotFound)
			return
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	lastTx, err := h.queries.GetLatestTransaction(ctx, session.ID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get latest transaction")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if lastTx.TransactionType != store.TransactionTransactionTypeInitiateAuthentication {
		log.Ctx(ctx).Warn().
			Str("expected", "initiate_authentication").
			Str("got", string(lastTx.TransactionType)).
			Msg("invalid transaction state")
		http.Error(w, "invalid state: expected initiate_authentication to be completed first", http.StatusConflict)
		return
	}

	if lastTx.Status != store.TransactionStatusSuccess {
		log.Ctx(ctx).Warn().
			Str("status", string(lastTx.Status)).
			Msg("previous transaction not successful")
		http.Error(w, "previous step failed", http.StatusPreconditionFailed)
		return
	}

	gatewayTxID := uuid.New()
	dbTx, err := h.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		PaymentSessionID:     session.ID,
		InvoiceID:            invoice.ID,
		ProjectID:            invoice.ProjectID,
		TransactionType:      store.TransactionTransactionTypeAuthenticatePayer,
		GatewayTransactionID: gatewayTxID.String(),
		Amount:               invoice.Amount,
		Currency:             invoice.Currency,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create transaction in db")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var browserDetails mpgsclient.AuthenticatePayerReqBrowserDetails
	if err := json.NewDecoder(r.Body).Decode(&browserDetails); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to decode body into browser details struct")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if r.Header.Get("Accept") != "" {
		browserDetails.AcceptHeaders = r.Header.Get("Accept")
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("remote_addr", r.RemoteAddr).Msg("failed to split host port of remote addr")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if ip == "::1" {
		ip = "0000:0000:0000:0000:0000:0000:0000:0001" // Full IPv6 loopback
	}

	// TODO: read the header of cloudflare, otherwise use remote addr if no cloudflare proxy in front of this.
	// if isCloudflareIP(ip) {
	// 	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
	// 		ip = cfIP
	// 	}
	// }

	if net.ParseIP(ip) == nil {
		log.Ctx(ctx).Error().Str("ip", ip).Msg("invalid ip address")
		http.Error(w, "invalid ip address", http.StatusBadRequest)
		return
	}

	resp, err := h.mpgsCli.AuthenticatePayer(ctx, invoice.ProjectID.String(), lastTx.GatewayTransactionID, &mpgsclient.AuthenticatePayerRequest{
		APIOperation: mpgsclient.OperationAuthenticatePayer,
		Authentication: mpgsclient.AuthenticatePayerReqAuthentication{
			// TODO: fetch this subdomain or domain or whatever from the organization :D
			RedirectResponseURL: fmt.Sprintf("%s/checkout/%s/pay/card/%s", "http://localhost:8080", invoiceID, session.ID),
		},
		Device: mpgsclient.AuthenticatePayerReqDevice{
			Browser:        r.Header.Get("User-Agent"),
			BrowserDetails: &browserDetails,
			IPAddress:      ip,
		},
		Order: mpgsclient.AuthenticatePayerReqOrder{
			Amount:   invoice.Amount.String(),
			Currency: invoice.Currency,
		},
		Session: mpgsclient.AuthenticatePayerReqSession{ID: session.GatewaySessionID},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("mpgs auth payer failed")
		http.Error(w, "gateway error", http.StatusInternalServerError)
		return
	}

	status := store.TransactionStatusFailed
	// PENDING = 3DS Challenge (HTML) returned
	// SUCCESS = Frictionless (No Challenge)
	if resp.Data.Result == mpgsclient.ResultPending || resp.Data.Result == mpgsclient.ResultSuccess {
		status = store.TransactionStatusSuccess
	}

	// TODO: make the mpgs client return the raw resp beside the parsed one :D
	rawResp, err := json.Marshal(resp)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to marshal raw resp")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.queries.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{
		ID: dbTx.ID, Status: status, RawResponse: rawResp,
	}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update transaction in db")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var nextStep, html string
	if resp.Data.Authentication.Redirect.HTML != "" {
		nextStep = "3ds_challenge"
		html = resp.Data.Authentication.Redirect.HTML
	} else {
		nextStep = "pay"
	}

	if err := json.NewEncoder(w).Encode(map[string]any{
		"next_step":     nextStep,
		"redirect_html": html,
	}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to encode the response")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *handlers) CardFinalizeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse invoice_id")
		http.Error(w, "invalid invoice_id", http.StatusBadRequest)
		return
	}
	sessionID, err := uuid.Parse(chi.URLParam(r, "payment_session_id"))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse payment_session_id")
		http.Error(w, "invalid payment_session_id", http.StatusBadRequest)
		return
	}

	session, err := h.queries.GetPaymentSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Ctx(ctx).Error().Err(err).Msg("failed to get payment session by id")
			http.Error(w, "payment session doesn't exist", http.StatusNotFound)
			return
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get payment session by id")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if session.InvoiceID != invoiceID {
		log.Ctx(ctx).Error().Err(err).Msg("provided invoice id is not for provided session id")
		http.Error(w, "invalid session", http.StatusForbidden)
		return
	}

	invoice, err := h.queries.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id")
			http.Error(w, "invoice doesn't exist", http.StatusNotFound)
			return
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	authTx, err := h.queries.GetSuccessfulAuthTransaction(ctx, session.ID)
	if err != nil {
		log.Ctx(ctx).Warn().Msg("no successful auth found for this session")
		http.Error(w, "authentication missing", http.StatusBadRequest)
		return
	}

	gatewayTxID := uuid.New()
	dbTx, err := h.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		PaymentSessionID:     session.ID,
		InvoiceID:            invoice.ID,
		ProjectID:            invoice.ProjectID,
		TransactionType:      store.TransactionTransactionTypePay,
		GatewayTransactionID: gatewayTxID.String(),
		Amount:               invoice.Amount,
		Currency:             invoice.Currency,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create transaction in db")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp, err := h.mpgsCli.ExecutePay(ctx, invoice.ProjectID.String(), gatewayTxID.String(), &mpgsclient.ExecutePayRequest{
		APIOperation: mpgsclient.OperationPay,
		Authentication: mpgsclient.ExecutePayReqAuthentication{
			TransactionID: authTx.GatewayTransactionID,
		},
		Order: mpgsclient.ExecutePayReqOrder{
			Amount:    invoice.Amount.String(),
			Currency:  invoice.Currency,
			Reference: invoice.ID.String(),
		},
		Session: mpgsclient.ExecutePayReqSession{
			ID: session.GatewaySessionID,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("mpgs execute pay failed")
		http.Error(w, "gateway error", http.StatusInternalServerError)
		return
	}

	// TODO: make the mpgs client return the raw resp beside the parsed one :D
	rawResp, err := json.Marshal(resp)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to marshal raw resp")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if resp.Data.Response.GatewayCode == mpgsclient.CodeApproved {
		if err := h.queries.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{
			ID: dbTx.ID, Status: store.TransactionStatusSuccess, RawResponse: rawResp,
		}); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to update transaction in db")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if err := h.queries.UpdateInvoiceStatus(ctx, store.UpdateInvoiceStatusParams{
			ID:     invoiceID,
			Status: store.InvoiceStatusPaid,
		}); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to update invoice in db")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	} else {
		h.queries.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{ID: dbTx.ID, Status: "declined"})
		json.NewEncoder(w).Encode(map[string]string{"status": "declined", "message": "Bank declined transaction"})
	}
}

// 6. Wallet Pay (Apple Pay - Simple Flow)
func (h *handlers) WalletPayHandler(w http.ResponseWriter, r *http.Request) {
	// Similar to CardFinalize, but no 'AuthTxID' needed usually
	// if token is pushed to session via JS.
	// Implementation depends on if you push token from JS or Backend.
}
