package checkout

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/billing"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/gateway"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/payment"
	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	saftajacontext "github.com/rbconsult-bh/saftaja/khazina/internal/pkg/context"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/checkout/templfiles"
)

type handlers struct {
	billing            billing.Service
	paymentService     payment.Service
	gatewayService     gateway.Service
	verifyDomainSecret string
}

func New(billing billing.Service, paymentService payment.Service, gatewayService gateway.Service, verifyDomainSecret string) Handlers {
	return &handlers{
		billing:            billing,
		paymentService:     paymentService,
		gatewayService:     gatewayService,
		verifyDomainSecret: verifyDomainSecret,
	}
}

func (h *handlers) CheckoutPageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := saftajacontext.ProjectFromContext(ctx)
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

	checkoutData, err := h.billing.GetInvoice(ctx, billing.GetInvoiceRequest{
		InvoiceID: invoiceID,
		ProjectID: project.ID,
	})
	if err != nil {
		switch {
		case errors.Is(err, billing.ErrInvalidArgument):
			log.Ctx(ctx).Error().Err(err).Msg("invalid argument")
			http.Error(w, "invalid argument", http.StatusBadRequest)
		case errors.Is(err, billing.ErrNotFound):
			log.Ctx(ctx).Info().Msg("invoice not found")
			http.Error(w, "invoice not found", http.StatusNotFound)
		default:
			log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice data")
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	if checkoutData.Invoice.Status == billing.InvoiceStatusPaid {
		data := templfiles.CheckoutPageData{
			Invoice: templfiles.CheckoutInvoice{
				ID:            checkoutData.Invoice.ID.String(),
				Amount:        checkoutData.Invoice.Amount.String(),
				Currency:      checkoutData.Invoice.Currency,
				CustomerEmail: checkoutData.Invoice.CustomerEmail,
			},
			IsPaid: true,
			Lang:   templfiles.DetectLanguage(r),
		}
		if err := templfiles.CheckoutPage(data).Render(ctx, w); err != nil {
			log.Error().Err(err).Msg("failed to render checkout page")
		}
		return
	}

	paymentMethodsResp, err := h.gatewayService.ListPaymentMethods(ctx, gateway.ListPaymentMethodsRequest{
		ProjectID: project.ID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to list payment methods")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	checkoutItems := make([]templfiles.CheckoutItem, len(checkoutData.Invoice.Items))
	for i, item := range checkoutData.Invoice.Items {
		checkoutItems[i] = templfiles.CheckoutItem{
			Name:      item.Name,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice.String(),
			Amount:    item.Amount.String(),
		}
	}

	var options []templfiles.PaymentOption
	var mpgsBaseURL, mpgsMerchantID string

	for _, pm := range paymentMethodsResp.PaymentMethods {
		mpgsBaseURL = pm.MPGSBaseURL
		mpgsMerchantID = pm.MPGSMerchantID

		options = append(options, templfiles.PaymentOption{
			ID:    pm.GatewayAccountID.String(),
			Label: paymentMethodLabel(string(pm.Type)),
			Type:  string(pm.Type),
		})
	}

	data := templfiles.CheckoutPageData{
		MPGSBaseURL:    mpgsBaseURL,
		MPGSAPIVersion: mpgsclient.APIVersion,
		MPGSMerchantID: mpgsMerchantID,
		Invoice: templfiles.CheckoutInvoice{
			ID:            checkoutData.Invoice.ID.String(),
			Amount:        checkoutData.Invoice.Amount.String(),
			Currency:      checkoutData.Invoice.Currency,
			Description:   checkoutData.Invoice.Description,
			CustomerEmail: checkoutData.Invoice.CustomerEmail,
			CustomerName:  checkoutData.Invoice.CustomerName,
		},
		Items:   checkoutItems,
		Options: options,
		IsPaid:  false,
		Lang:    templfiles.DetectLanguage(r),
	}

	if err := templfiles.CheckoutPage(data).Render(ctx, w); err != nil {
		log.Error().Err(err).Msg("failed to render checkout page")
	}
}

func paymentMethodLabel(method string) string {
	switch method {
	case "card":
		return "Credit / Debit Card"
	case "apple_pay":
		return "Apple Pay"
	default:
		return method
	}
}

func (h *handlers) CreatePaymentIntentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := saftajacontext.ProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, templfiles.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	var req struct {
		GatewayAccountID uuid.UUID `json:"gateway_account_id"`
		PaymentMethod    string    `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	payerIP, err := ExtractPayerIP(r)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("invalid ip address")
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.billing.CreatePaymentIntent(ctx, billing.CreatePaymentIntentRequest{
		InvoiceID:        invoiceID,
		ProjectID:        project.ID,
		IdempotencyKey:   r.Header.Get("Idempotency-Key"),
		GatewayAccountID: req.GatewayAccountID,
		PaymentMethod:    billing.PaymentMethod(req.PaymentMethod),
		PayerIP:          payerIP,
		PayerUserAgent:   r.Header.Get("User-Agent"),
	})
	if err != nil {
		handlePaymentError(w, r, err)
		return
	}

	respondJSON(w, map[string]any{
		"action":                   "render_embedded",
		"payment_intent_id":        result.PaymentIntentID.String(),
		"payment_method_reference": result.PaymentMethodReference,
	})
}

func (h *handlers) PrepareCardAuthenticationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := saftajacontext.ProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, templfiles.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	paymentIntentID, err := uuid.Parse(chi.URLParam(r, "payment_intent_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.billing.PrepareCardAuthentication(ctx, billing.PrepareCardAuthenticationRequest{
		ProjectID:       project.ID,
		InvoiceID:       invoiceID,
		PaymentIntentID: paymentIntentID,
	})
	if err != nil {
		handlePaymentError(w, r, err)
		return
	}

	respondJSON(w, map[string]any{
		"next_step": result.NextStep,
	})
}

func (h *handlers) AuthenticateCardholderHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := saftajacontext.ProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, templfiles.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	paymentIntentID, err := uuid.Parse(chi.URLParam(r, "payment_intent_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	var browserDetails struct {
		ChallengeWindowSize billing.ThreeDSChallengeWindowSize `json:"challenge_window_size"`
		ColorDepth          int                                `json:"color_depth"`
		JavaEnabled         bool                               `json:"java_enabled"`
		Language            string                             `json:"language"`
		ScreenHeight        int                                `json:"screen_height"`
		ScreenWidth         int                                `json:"screen_width"`
		TimeZone            int                                `json:"time_zone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&browserDetails); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to decode browser details")
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	payerIP, err := ExtractPayerIP(r)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("invalid ip address")
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	returnPath := fmt.Sprintf(
		"/checkout/%s/payment-intents/%s/card-authentication/return",
		invoiceID,
		paymentIntentID,
	)
	challengeReturnURL := (&url.URL{
		Scheme: "https",
		Host:   project.CustomDomain,
		Path:   returnPath,
	}).String()

	result, err := h.billing.AuthenticateCardholder(ctx, billing.AuthenticateCardholderRequest{
		PaymentIntentRef: billing.PaymentIntentRef{
			ProjectID:       project.ID,
			InvoiceID:       invoiceID,
			PaymentIntentID: paymentIntentID,
		},
		Browser: billing.ThreeDSBrowser{
			IPAddress:           payerIP,
			UserAgent:           r.Header.Get("User-Agent"),
			AcceptHeader:        r.Header.Get("Accept"),
			ChallengeWindowSize: browserDetails.ChallengeWindowSize,
			ColorDepth:          browserDetails.ColorDepth,
			JavaEnabled:         browserDetails.JavaEnabled,
			Language:            browserDetails.Language,
			ScreenHeight:        browserDetails.ScreenHeight,
			ScreenWidth:         browserDetails.ScreenWidth,
			TimeZone:            browserDetails.TimeZone,
		},
		ChallengeReturnURL: challengeReturnURL,
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

func (h *handlers) CardAuthenticationReturnHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := saftajacontext.ProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, templfiles.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	if _, err := uuid.Parse(chi.URLParam(r, "invoice_id")); err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(chi.URLParam(r, "payment_intent_id")); err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	if err := templfiles.CardAuthenticationReturnPage(templfiles.DetectLanguage(r)).Render(ctx, w); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to render card authentication return page")
	}
}

func (h *handlers) VerifyCardAuthenticationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := saftajacontext.ProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, templfiles.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	paymentIntentID, err := uuid.Parse(chi.URLParam(r, "payment_intent_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.billing.VerifyCardAuthentication(ctx, billing.VerifyCardAuthenticationRequest{
		PaymentIntentRef: billing.PaymentIntentRef{
			ProjectID:       project.ID,
			InvoiceID:       invoiceID,
			PaymentIntentID: paymentIntentID,
		},
	})
	if err != nil {
		handlePaymentError(w, r, err)
		return
	}

	respondJSON(w, map[string]any{
		"next_step": result.NextStep,
	})
}

func (h *handlers) CapturePaymentIntentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project := saftajacontext.ProjectFromContext(ctx)
	if project == nil {
		respondError(w, r, ErrCodeInvoiceNotFound, templfiles.MsgInvoiceNotFound, http.StatusNotFound)
		return
	}

	invoiceID, err := uuid.Parse(chi.URLParam(r, "invoice_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	paymentIntentID, err := uuid.Parse(chi.URLParam(r, "payment_intent_id"))
	if err != nil {
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.billing.CapturePaymentIntent(ctx, billing.CapturePaymentIntentRequest{
		PaymentIntentRef: billing.PaymentIntentRef{
			ProjectID:       project.ID,
			InvoiceID:       invoiceID,
			PaymentIntentID: paymentIntentID,
		},
	})
	if err != nil {
		handlePaymentError(w, r, err)
		return
	}

	respondJSON(w, map[string]any{
		"next_step": result.NextStep,
	})
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
