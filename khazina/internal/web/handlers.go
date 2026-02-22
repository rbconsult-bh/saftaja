package web

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	mpgsclient "github.com/rbconsult-bh/saftaja/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/internal/domain"
	"github.com/rbconsult-bh/saftaja/internal/payment"
	"github.com/rbconsult-bh/saftaja/internal/web/middlewares"
	"github.com/rbconsult-bh/saftaja/internal/web/templfiles"
)

type handlers struct {
	paymentService     payment.Service
	verifyDomainSecret string
}

func New(paymentService payment.Service, verifyDomainSecret string) Handlers {
	return &handlers{
		paymentService:     paymentService,
		verifyDomainSecret: verifyDomainSecret,
	}
}

func (h *handlers) CheckoutPageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := middlewares.GetProjectFromContext(ctx)
	if project == nil {
		log.Ctx(ctx).Info().Str("host", r.Host).Msg("checkout accessed via unregistered domain")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	rawInvoiceID := chi.URLParam(r, "invoice_id")
	invoiceID, err := uuid.Parse(rawInvoiceID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse invoice_id")
		http.Error(w, "invalid invoice_id", http.StatusBadRequest)
		return
	}

	checkoutData, err := h.paymentService.GetCheckoutData(ctx, invoiceID, project.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Ctx(ctx).Info().Msg("invoice not found")
			http.Error(w, "invoice not found", http.StatusNotFound)
			return
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to get checkout data")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if checkoutData.IsPaid {
		data := templfiles.CheckoutPageData{
			Invoice: templfiles.CheckoutInvoice{
				ID:            checkoutData.Invoice.ID.String(),
				Amount:        checkoutData.Invoice.Amount,
				Currency:      checkoutData.Invoice.Currency,
				CustomerEmail: checkoutData.Invoice.CustomerEmail,
			},
			IsPaid: true,
			Lang:   domain.DetectLanguage(r),
		}
		if err := templfiles.CheckoutPage(data).Render(ctx, w); err != nil {
			log.Error().Err(err).Msg("failed to render checkout page")
		}
		return
	}

	// Map service data to template data
	checkoutItems := make([]templfiles.CheckoutItem, len(checkoutData.Items))
	for i, item := range checkoutData.Items {
		checkoutItems[i] = templfiles.CheckoutItem{
			Name:      item.Name,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Amount:    item.Amount,
		}
	}

	options := make([]templfiles.PaymentOption, len(checkoutData.PaymentOptions))
	for i, opt := range checkoutData.PaymentOptions {
		options[i] = templfiles.PaymentOption{
			ID:    opt.GatewayAccountID.String(),
			Label: opt.Label,
			Type:  string(opt.Method),
		}
	}

	var mpgsBaseURL, mpgsMerchantID string
	if checkoutData.MPGSConfig != nil {
		mpgsBaseURL = checkoutData.MPGSConfig.BaseURL
		mpgsMerchantID = checkoutData.MPGSConfig.MerchantID
	}

	data := templfiles.CheckoutPageData{
		MPGSBaseURL:    mpgsBaseURL,
		MPGSAPIVersion: mpgsclient.APIVersion,
		MPGSMerchantID: mpgsMerchantID,
		Invoice: templfiles.CheckoutInvoice{
			ID:            checkoutData.Invoice.ID.String(),
			Amount:        checkoutData.Invoice.Amount,
			Currency:      checkoutData.Invoice.Currency,
			Description:   checkoutData.Invoice.Description,
			CustomerEmail: checkoutData.Invoice.CustomerEmail,
			CustomerName:  checkoutData.Invoice.CustomerName,
		},
		Items:   checkoutItems,
		Options: options,
		IsPaid:  false,
		Lang:    domain.DetectLanguage(r),
	}

	if err := templfiles.CheckoutPage(data).Render(ctx, w); err != nil {
		log.Error().Err(err).Msg("failed to render checkout page")
	}
}

func (h *handlers) InitiateSessionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := middlewares.GetProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, domain.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	var req struct {
		GatewayAccountID uuid.UUID            `json:"gateway_account_id"`
		PaymentMethod    domain.PaymentMethod `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	payerIP, err := ExtractPayerIP(r)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("invalid ip address")
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.paymentService.InitiateSession(ctx, &payment.InitiateSessionRequest{
		ProjectID:        project.ID,
		InvoiceID:        invoiceID,
		GatewayAccountID: req.GatewayAccountID,
		PaymentMethod:    req.PaymentMethod,
		PayerIP:          payerIP,
		PayerUserAgent:   r.Header.Get("User-Agent"),
		IdempotencyKey:   r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		handlePaymentError(w, r, err)
		return
	}

	respondJSON(w, map[string]any{
		"action":             "render_embedded",
		"payment_session_id": result.PaymentSessionID.String(),
		"mpgs_session_id":    result.GatewaySessionID,
	})
}

func (h *handlers) CardInitiateAuthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := middlewares.GetProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, domain.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "payment_session_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.paymentService.InitiateAuth(ctx, &payment.InitiateAuthRequest{
		ProjectID:        project.ID,
		InvoiceID:        invoiceID,
		PaymentSessionID: sessionID,
	})
	if err != nil {
		handlePaymentError(w, r, err)
		return
	}

	respondJSON(w, map[string]any{
		"next_step": result.NextStep,
		"tx_id":     result.TransactionID,
	})
}

func (h *handlers) CardProcessAuthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := middlewares.GetProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, domain.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "payment_session_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	var browserDetails payment.BrowserDetails
	if err := json.NewDecoder(r.Body).Decode(&browserDetails); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to decode browser details")
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	payerIP, err := ExtractPayerIP(r)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("invalid ip address")
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.paymentService.ProcessAuth(ctx, &payment.ProcessAuthRequest{
		ProjectID:        project.ID,
		InvoiceID:        invoiceID,
		PaymentSessionID: sessionID,
		BrowserDetails:   browserDetails,
		PayerIP:          payerIP,
		UserAgent:        r.Header.Get("User-Agent"),
		AcceptHeaders:    r.Header.Get("Accept"),
	})
	if err != nil {
		handlePaymentError(w, r, err)
		return
	}

	respondJSON(w, map[string]any{
		"next_step":     result.NextStep,
		"redirect_html": result.RedirectHTML,
	})
}

func (h *handlers) CardFinalizeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := middlewares.GetProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, domain.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "payment_session_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, domain.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.paymentService.FinalizePayment(ctx, &payment.FinalizePaymentRequest{
		ProjectID:        project.ID,
		InvoiceID:        invoiceID,
		PaymentSessionID: sessionID,
	})
	if err != nil {
		handlePaymentError(w, r, err)
		return
	}

	status := "failed"
	message := PaymentResultMessages[result.ResultCode].ForRequest(r)
	if result.Success {
		status = "success"
	}

	lang := domain.DetectLanguage(r)
	data := templfiles.CompletionPageData{
		Status:      status,
		Message:     message,
		CheckoutURL: result.CheckoutURL,
		Lang:        lang,
	}

	if err := templfiles.CompletionPage(data).Render(ctx, w); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to render completion page")
	}
}

func (h *handlers) WalletPayHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Apple Pay implementation
}

func (h *handlers) VerifyDomainHandler(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(h.verifyDomainSecret)) == 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	valid, err := h.paymentService.VerifyDomain(r.Context(), domain)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Str("domain", domain).Msg("failed to verify domain")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !valid {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}
