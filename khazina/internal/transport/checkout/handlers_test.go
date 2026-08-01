package checkout_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/gateway"
	gwmocks "github.com/rbconsult-bh/saftaja/khazina/internal/app/gateway/mocks"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/payment"
	paymentmocks "github.com/rbconsult-bh/saftaja/khazina/internal/app/payment/mocks"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
	tenantmocks "github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant/mocks"
	saftajacontext "github.com/rbconsult-bh/saftaja/khazina/internal/pkg/context"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/money"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/ptr"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/checkout"
)

// --------------------------------------------------------------------------
// Test fixture
// --------------------------------------------------------------------------

type fixture struct {
	Payment *paymentmocks.MockService
	Tenant  *tenantmocks.MockService
	Gateway *gwmocks.MockService
	router  *chi.Mux
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	paymentSvc := paymentmocks.NewMockService(t)
	tenantSvc := tenantmocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	h := checkout.New(paymentSvc, tenantSvc, gwSvc, "test-secret")

	r := chi.NewRouter()
	r.Get("/checkout/{invoice_id}", h.CheckoutPageHandler)
	r.Post("/checkout/{invoice_id}/payment-intents", h.CreatePaymentIntentHandler)
	r.Route("/checkout/{invoice_id}/payment-intents/{payment_intent_id}", func(r chi.Router) {
		r.Route("/card-authentication", func(r chi.Router) {
			r.Post("/prepare", h.PrepareCardAuthenticationHandler)
			r.Post("/authenticate", h.AuthenticateCardholderHandler)
			r.Post("/return", h.CardAuthenticationReturnHandler)
			r.Post("/verify", h.VerifyCardAuthenticationHandler)
		})
		r.Post("/capture", h.CapturePaymentIntentHandler)
	})
	r.Get("/verify-domain", h.VerifyDomainHandler)

	return &fixture{
		Payment: paymentSvc,
		Tenant:  tenantSvc,
		Gateway: gwSvc,
		router:  r,
	}
}

func (f *fixture) get(path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func (f *fixture) getWithProject(path string, project *tenant.Project) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	ctx := saftajacontext.WithProject(req.Context(), project)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func (f *fixture) post(path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func (f *fixture) postWithProject(path string, body any, project *tenant.Project) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	ctx := saftajacontext.WithProject(req.Context(), project)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func testProject(id uuid.UUID) *tenant.Project {
	return &tenant.Project{
		ID:           id,
		CustomDomain: "pay.test.com",
	}
}

// --------------------------------------------------------------------------
// CheckoutPageHandler
// --------------------------------------------------------------------------

func TestCheckoutPageHandler_Success(t *testing.T) {
	f := newFixture(t)

	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	gatewayID := uuid.New()

	f.Payment.EXPECT().GetInvoice(mock.Anything, payment.GetInvoiceRequest{
		InvoiceID: invoiceID,
		ProjectID: projectID,
	}).Return(&payment.GetInvoiceResponse{
		Invoice: payment.Invoice{
			ID:            invoiceID,
			ProjectID:     projectID,
			Amount:        100000,
			Currency:      money.CurrencyBHD,
			Description:   "Test Invoice",
			CustomerEmail: "test@example.com",
			CustomerName:  "Test Customer",
			Items: []payment.InvoiceItem{
				{Name: "Item 1", Quantity: 1, UnitPrice: 100000, Amount: 100000},
			},
		},
	}, nil)

	f.Gateway.EXPECT().ListPaymentMethods(mock.Anything, gateway.ListPaymentMethodsRequest{
		ProjectID: projectID,
	}).Return(&gateway.ListPaymentMethodsResponse{
		PaymentMethods: []gateway.PaymentMethod{
			{
				Type:             gateway.PaymentMethodTypeCard,
				GatewayAccountID: gatewayID,
				MPGSBaseURL:      "https://test.gateway.mastercard.com",
				MPGSMerchantID:   "TESTMERCHANT",
				MPGSApiVersion:   "100",
			},
		},
	}, nil)

	w := f.getWithProject("/checkout/"+invoiceID.String(), project)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("expected content type text/html; charset=utf-8, got %s", ct)
	}
}

func TestCheckoutPageHandler_NoTenantContext(t *testing.T) {
	f := newFixture(t)
	w := f.get("/checkout/" + uuid.New().String())
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCheckoutPageHandler_WrongProject(t *testing.T) {
	f := newFixture(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)

	f.Payment.EXPECT().GetInvoice(mock.Anything, payment.GetInvoiceRequest{
		InvoiceID: invoiceID,
		ProjectID: projectID,
	}).Return(nil, payment.ErrNotFound)

	w := f.getWithProject("/checkout/"+invoiceID.String(), project)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCheckoutPageHandler_InvalidInvoiceID(t *testing.T) {
	f := newFixture(t)
	project := testProject(uuid.New())
	w := f.getWithProject("/checkout/not-a-uuid", project)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCheckoutPageHandler_AlreadyPaid(t *testing.T) {
	f := newFixture(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	now := time.Now()

	f.Payment.EXPECT().GetInvoice(mock.Anything, payment.GetInvoiceRequest{
		InvoiceID: invoiceID,
		ProjectID: projectID,
	}).Return(&payment.GetInvoiceResponse{
		Invoice: payment.Invoice{
			ID:            invoiceID,
			ProjectID:     projectID,
			Amount:        100000,
			Currency:      money.CurrencyBHD,
			CustomerEmail: "test@example.com",
			Status:        payment.InvoiceStatusPaid,
			PaidAt:        &now,
		},
	}, nil)

	w := f.getWithProject("/checkout/"+invoiceID.String(), project)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// --------------------------------------------------------------------------
// CreatePaymentIntentHandler
// --------------------------------------------------------------------------

func TestCreatePaymentIntentHandler_Success(t *testing.T) {
	f := newFixture(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	gatewayAccountID := uuid.New()
	sessionID := uuid.New()

	f.Payment.EXPECT().CreatePaymentIntent(mock.Anything, payment.CreatePaymentIntentRequest{
		ProjectID:        projectID,
		InvoiceID:        invoiceID,
		GatewayAccountID: gatewayAccountID,
		PaymentMethod:    payment.PaymentMethodCard,
		PayerIP:          "192.0.2.1",
		PayerUserAgent:   "TestAgent",
		IdempotencyKey:   "test-key-123",
	}).Return(&payment.CreatePaymentIntentResponse{
		PaymentIntentID:        sessionID,
		PaymentMethodReference: ptr.To(payment.PaymentMethodReference("MPGS_SESSION_123")),
	}, nil)

	body := map[string]any{
		"gateway_account_id": gatewayAccountID.String(),
		"payment_method":     "card",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/checkout/"+invoiceID.String()+"/payment-intents", bytes.NewReader(b))
	ctx := saftajacontext.WithProject(req.Context(), project)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestAgent")
	req.Header.Set("Idempotency-Key", "test-key-123")
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["payment_intent_id"] != sessionID.String() {
		t.Errorf("expected payment_intent_id %s, got %v", sessionID.String(), resp["payment_intent_id"])
	}
	if resp["payment_method_reference"] != "MPGS_SESSION_123" {
		t.Errorf("expected payment_method_reference %s, got %v", "MPGS_SESSION_123", resp["payment_method_reference"])
	}
}

func TestCreatePaymentIntentHandler_NoTenantContext(t *testing.T) {
	f := newFixture(t)
	body := map[string]any{
		"gateway_account_id": uuid.New().String(),
		"payment_method":     "card",
	}
	w := f.post("/checkout/"+uuid.New().String()+"/payment-intents", body)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestCreatePaymentIntentHandler_InvalidRequest(t *testing.T) {
	f := newFixture(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	gatewayAccountID := uuid.New()

	f.Payment.EXPECT().CreatePaymentIntent(mock.Anything, payment.CreatePaymentIntentRequest{
		ProjectID:        projectID,
		InvoiceID:        invoiceID,
		GatewayAccountID: gatewayAccountID,
		PaymentMethod:    payment.PaymentMethodCard,
		PayerIP:          "192.0.2.1",
		PayerUserAgent:   "TestAgent",
	}).Return(nil, payment.ErrInvalidArgument)

	body := map[string]any{
		"gateway_account_id": gatewayAccountID.String(),
		"payment_method":     "card",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/checkout/"+invoiceID.String()+"/payment-intents", bytes.NewReader(b))
	ctx := saftajacontext.WithProject(req.Context(), project)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestAgent")
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestCreatePaymentIntentHandler_AlreadyPaid(t *testing.T) {
	f := newFixture(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	gatewayAccountID := uuid.New()

	f.Payment.EXPECT().CreatePaymentIntent(mock.Anything, payment.CreatePaymentIntentRequest{
		ProjectID:        projectID,
		InvoiceID:        invoiceID,
		GatewayAccountID: gatewayAccountID,
		PaymentMethod:    payment.PaymentMethodCard,
		PayerIP:          "192.0.2.1",
		PayerUserAgent:   "TestAgent",
		IdempotencyKey:   "test-key-123",
	}).Return(nil, payment.ErrInvoiceAlreadyPaid)

	body := map[string]any{
		"gateway_account_id": gatewayAccountID.String(),
		"payment_method":     "card",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/checkout/"+invoiceID.String()+"/payment-intents", bytes.NewReader(b))
	ctx := saftajacontext.WithProject(req.Context(), project)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestAgent")
	req.Header.Set("Idempotency-Key", "test-key-123")
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

// --------------------------------------------------------------------------
// Card payment intent flow
// --------------------------------------------------------------------------

func TestPrepareCardAuthenticationHandler(t *testing.T) {
	f := newFixture(t)
	invoiceID := uuid.New()
	paymentIntentID := uuid.New()
	project := testProject(uuid.New())

	f.Payment.EXPECT().PrepareCardAuthentication(mock.Anything, payment.PrepareCardAuthenticationRequest{
		PaymentIntentRef: payment.PaymentIntentRef{
			ProjectID:       project.ID,
			InvoiceID:       invoiceID,
			PaymentIntentID: paymentIntentID,
		},
	}).Return(&payment.PrepareCardAuthenticationResponse{
		NextStep: payment.PrepareCardAuthenticationNextStepAuthenticate,
	}, nil)

	path := "/checkout/" + invoiceID.String() + "/payment-intents/" + paymentIntentID.String() + "/card-authentication/prepare"
	w := f.postWithProject(path, nil, project)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["next_step"] != "authenticate" {
		t.Fatalf("expected authenticate, got %q", response["next_step"])
	}
}

func TestAuthenticateCardholderHandler(t *testing.T) {
	f := newFixture(t)
	invoiceID := uuid.New()
	paymentIntentID := uuid.New()
	project := testProject(uuid.New())

	expectedRequest := payment.AuthenticateCardholderRequest{
		PaymentIntentRef: payment.PaymentIntentRef{
			ProjectID:       project.ID,
			InvoiceID:       invoiceID,
			PaymentIntentID: paymentIntentID,
		},
		Browser: payment.ThreeDSBrowser{
			IPAddress:           "192.0.2.1",
			UserAgent:           "TestAgent",
			AcceptHeader:        "text/html,application/json",
			ChallengeWindowSize: payment.ThreeDSChallengeWindowSizeFullScreen,
			ColorDepth:          24,
			JavaEnabled:         false,
			Language:            "en-US",
			ScreenHeight:        1080,
			ScreenWidth:         1920,
			TimeZone:            180,
		},
		ChallengeReturnURL: "https://pay.test.com/checkout/" + invoiceID.String() +
			"/payment-intents/" + paymentIntentID.String() + "/card-authentication/return",
	}
	f.Payment.EXPECT().AuthenticateCardholder(mock.Anything, expectedRequest).Return(
		&payment.AuthenticateCardholderResponse{
			NextStep:     payment.AuthenticateCardholderNextStepChallenge,
			RedirectHTML: "<iframe id=\"challengeFrame\"></iframe>",
		},
		nil,
	)

	body := map[string]any{
		"challenge_window_size": "FULL_SCREEN",
		"color_depth":           24,
		"java_enabled":          false,
		"language":              "en-US",
		"screen_height":         1080,
		"screen_width":          1920,
		"time_zone":             180,
	}
	encodedBody, _ := json.Marshal(body)
	path := "/checkout/" + invoiceID.String() + "/payment-intents/" + paymentIntentID.String() + "/card-authentication/authenticate"
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encodedBody))
	req = req.WithContext(saftajacontext.WithProject(req.Context(), project))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/html,application/json")
	req.Header.Set("User-Agent", "TestAgent")
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["next_step"] != "challenge" {
		t.Fatalf("expected challenge, got %q", response["next_step"])
	}
	if response["redirect_html"] == "" {
		t.Fatal("expected redirect_html")
	}
}

func TestCardAuthenticationReturnHandler(t *testing.T) {
	f := newFixture(t)
	project := testProject(uuid.New())
	path := "/checkout/" + uuid.New().String() + "/payment-intents/" + uuid.New().String() + "/card-authentication/return"
	w := f.postWithProject(path, nil, project)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("CARD_AUTHENTICATION_RETURNED")) {
		t.Fatal("expected return page to notify parent")
	}
}

func TestVerifyCardAuthenticationHandler(t *testing.T) {
	f := newFixture(t)
	invoiceID := uuid.New()
	paymentIntentID := uuid.New()
	project := testProject(uuid.New())
	ref := payment.PaymentIntentRef{
		ProjectID:       project.ID,
		InvoiceID:       invoiceID,
		PaymentIntentID: paymentIntentID,
	}

	f.Payment.EXPECT().VerifyCardAuthentication(mock.Anything, payment.VerifyCardAuthenticationRequest{
		PaymentIntentRef: ref,
	}).Return(&payment.VerifyCardAuthenticationResponse{
		NextStep: payment.VerifyCardAuthenticationNextStepCapture,
	}, nil)

	path := "/checkout/" + invoiceID.String() + "/payment-intents/" + paymentIntentID.String() + "/card-authentication/verify"
	w := f.postWithProject(path, nil, project)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["next_step"] != "capture" {
		t.Fatalf("expected capture, got %q", response["next_step"])
	}
}

func TestCapturePaymentIntentHandler_ResultMatrix(t *testing.T) {
	tests := []struct {
		name     string
		nextStep payment.CapturePaymentIntentNextStep
	}{
		{name: "complete", nextStep: payment.CapturePaymentIntentNextStepComplete},
		{name: "processing", nextStep: payment.CapturePaymentIntentNextStepProcessing},
		{name: "cant_continue", nextStep: payment.CapturePaymentIntentNextStepCantContinue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			invoiceID := uuid.New()
			paymentIntentID := uuid.New()
			project := testProject(uuid.New())
			ref := payment.PaymentIntentRef{
				ProjectID:       project.ID,
				InvoiceID:       invoiceID,
				PaymentIntentID: paymentIntentID,
			}

			f.Payment.EXPECT().CapturePaymentIntent(mock.Anything, payment.CapturePaymentIntentRequest{
				PaymentIntentRef: ref,
			}).Return(&payment.CapturePaymentIntentResponse{NextStep: tt.nextStep}, nil)

			path := "/checkout/" + invoiceID.String() + "/payment-intents/" + paymentIntentID.String() + "/capture"
			w := f.postWithProject(path, nil, project)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
			}
			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response["next_step"] != string(tt.nextStep) {
				t.Fatalf("expected %q, got %q", tt.nextStep, response["next_step"])
			}
		})
	}
}

func TestCapturePaymentIntentHandler_Expired(t *testing.T) {
	f := newFixture(t)
	invoiceID := uuid.New()
	paymentIntentID := uuid.New()
	project := testProject(uuid.New())

	f.Payment.EXPECT().CapturePaymentIntent(mock.Anything, mock.Anything).Return(nil, payment.ErrPaymentIntentExpired)

	path := "/checkout/" + invoiceID.String() + "/payment-intents/" + paymentIntentID.String() + "/capture"
	w := f.postWithProject(path, nil, project)
	if w.Code != http.StatusGone {
		t.Fatalf("expected status %d, got %d: %s", http.StatusGone, w.Code, w.Body.String())
	}
}

func TestCapturePaymentIntentHandler_NoTenantContext(t *testing.T) {
	f := newFixture(t)
	path := "/checkout/" + uuid.New().String() + "/payment-intents/" + uuid.New().String() + "/capture"
	w := f.post(path, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

// --------------------------------------------------------------------------
// VerifyDomainHandler
// --------------------------------------------------------------------------

func TestVerifyDomainHandler_ValidDomain(t *testing.T) {
	f := newFixture(t)
	f.Tenant.EXPECT().IsDomainValid(mock.Anything, "pay.merchant.com").Return(true, nil)
	w := f.get("/verify-domain?secret=test-secret&domain=pay.merchant.com")
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestVerifyDomainHandler_InvalidDomain(t *testing.T) {
	f := newFixture(t)
	f.Tenant.EXPECT().IsDomainValid(mock.Anything, "unknown.domain.com").Return(false, nil)
	w := f.get("/verify-domain?secret=test-secret&domain=unknown.domain.com")
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestVerifyDomainHandler_MissingSecret(t *testing.T) {
	f := newFixture(t)
	w := f.get("/verify-domain?domain=pay.merchant.com")
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestVerifyDomainHandler_WrongSecret(t *testing.T) {
	f := newFixture(t)
	w := f.get("/verify-domain?secret=wrong-secret&domain=pay.merchant.com")
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestVerifyDomainHandler_MissingDomain(t *testing.T) {
	f := newFixture(t)
	w := f.get("/verify-domain?secret=test-secret")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
