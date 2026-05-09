package checkout_test

import (
	"bytes"
	
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/gateway"
	gwmocks "github.com/rbconsult-bh/saftaja/khazina/internal/app/gateway/mocks"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/invoice"
	invomocks "github.com/rbconsult-bh/saftaja/khazina/internal/app/invoice/mocks"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/payment"
	paymocks "github.com/rbconsult-bh/saftaja/khazina/internal/app/payment/mocks"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
	saftajacontext "github.com/rbconsult-bh/saftaja/khazina/internal/pkg/context"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/checkout"
)

func testProject(id uuid.UUID) *tenant.Project {
	return &tenant.Project{
		ID:           id,
		CustomDomain: "pay.test.com",
	}
}

func withTenantContext(r *http.Request, project *tenant.Project) *http.Request {
	ctx := saftajacontext.WithProject(r.Context(), project)
	return r.WithContext(ctx)
}

func setupRouter(h checkout.Handlers) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/checkout/{invoice_id}", h.CheckoutPageHandler)
	r.Post("/checkout/{invoice_id}/initiate", h.InitiateSessionHandler)
	r.Post("/checkout/{invoice_id}/pay/card/{payment_session_id}/finalize", h.CardFinalizeHandler)
	r.Get("/verify-domain", h.VerifyDomainHandler)
	return r
}

func TestCheckoutPageHandler_Success(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)

	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	gatewayID := uuid.New()

	invSvc.EXPECT().GetByID(mock.Anything, invoiceID, projectID).Return(&invoice.InvoiceData{
		Invoice: invoice.InvoiceInfo{
			ID:            invoiceID,
			ProjectID:     projectID,
			Amount:        "100.000",
			Currency:      "BHD",
			Description:   "Test Invoice",
			CustomerEmail: "test@example.com",
			CustomerName:  "Test Customer",
		},
		Items: []invoice.ItemInfo{
			{Name: "Item 1", Quantity: 1, UnitPrice: "100.000", Amount: "100.000"},
		},
		IsPaid: false,
	}, nil)

	gwSvc.EXPECT().ListActiveByProject(mock.Anything, projectID).Return([]gateway.GatewayCredentials{
		{
			GatewayAccountID: gatewayID,
			ConnectorType:    domain.ConnectorTypeMPGS,
			BaseURL:          "https://test.gateway.mastercard.com",
			MerchantID:       "TESTMERCHANT",
			APIPassword:      "test-password",
			PaymentMethods:   []string{"card"},
		},
	}, nil)

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/checkout/"+invoiceID.String(), nil)
	req = withTenantContext(req, project)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected content type text/html; charset=utf-8, got %s", contentType)
	}
}

func TestCheckoutPageHandler_NoTenantContext(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	invoiceID := uuid.New()

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/checkout/"+invoiceID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCheckoutPageHandler_WrongProject(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	invoiceID := uuid.New()
	requestProjectID := uuid.New()
	project := testProject(requestProjectID)

	invSvc.EXPECT().GetByID(mock.Anything, invoiceID, requestProjectID).Return(nil, sql.ErrNoRows)

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/checkout/"+invoiceID.String(), nil)
	req = withTenantContext(req, project)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCheckoutPageHandler_InvalidInvoiceID(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	projectID := uuid.New()
	project := testProject(projectID)

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/checkout/not-a-uuid", nil)
	req = withTenantContext(req, project)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCheckoutPageHandler_AlreadyPaid(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)

	invSvc.EXPECT().GetByID(mock.Anything, invoiceID, projectID).Return(&invoice.InvoiceData{
		Invoice: invoice.InvoiceInfo{
			ID:        invoiceID,
			ProjectID: projectID,
			Amount:    "100.000",
			Currency:  "BHD",
		},
		IsPaid: true,
	}, nil)

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/checkout/"+invoiceID.String(), nil)
	req = withTenantContext(req, project)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestInitiateSessionHandler_Success(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	gatewayAccountID := uuid.New()
	sessionID := uuid.New()

	paySvc.EXPECT().InitiateSession(mock.Anything, &payment.InitiateSessionRequest{
		ProjectID:        projectID,
		InvoiceID:        invoiceID,
		GatewayAccountID: gatewayAccountID,
		PaymentMethod:    domain.PaymentMethodCard,
		PayerIP:          "192.0.2.1",
		PayerUserAgent:   "TestAgent",
		IdempotencyKey:   "test-key-123",
	}).Return(&payment.InitiateSessionResult{
		PaymentSessionID: sessionID,
		GatewaySessionID: "MPGS_SESSION_123",
	}, nil)

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	body, _ := json.Marshal(map[string]any{
		"gateway_account_id": gatewayAccountID.String(),
		"payment_method":     "card",
	})

	req := httptest.NewRequest(http.MethodPost, "/checkout/"+invoiceID.String()+"/initiate", bytes.NewReader(body))
	req = withTenantContext(req, project)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestAgent")
	req.Header.Set("Idempotency-Key", "test-key-123")
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["payment_session_id"] != sessionID.String() {
		t.Errorf("expected payment_session_id %s, got %v", sessionID.String(), resp["payment_session_id"])
	}
}

func TestInitiateSessionHandler_NoTenantContext(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	invoiceID := uuid.New()
	gatewayAccountID := uuid.New()

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	body, _ := json.Marshal(map[string]any{
		"gateway_account_id": gatewayAccountID.String(),
		"payment_method":     "card",
	})

	req := httptest.NewRequest(http.MethodPost, "/checkout/"+invoiceID.String()+"/initiate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestInitiateSessionHandler_AlreadyPaid(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	gatewayAccountID := uuid.New()

	paySvc.EXPECT().InitiateSession(mock.Anything, &payment.InitiateSessionRequest{
		ProjectID:        projectID,
		InvoiceID:        invoiceID,
		GatewayAccountID: gatewayAccountID,
		PaymentMethod:    domain.PaymentMethodCard,
		PayerIP:          "192.0.2.1",
		PayerUserAgent:   "",
		IdempotencyKey:   "",
	}).Return(nil, &payment.InvoiceAlreadyPaidError{InvoiceID: invoiceID.String()})

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	body, _ := json.Marshal(map[string]any{
		"gateway_account_id": gatewayAccountID.String(),
		"payment_method":     "card",
	})

	req := httptest.NewRequest(http.MethodPost, "/checkout/"+invoiceID.String()+"/initiate", bytes.NewReader(body))
	req = withTenantContext(req, project)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestCardFinalizeHandler_Success(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	sessionID := uuid.New()

	paySvc.EXPECT().FinalizePayment(mock.Anything, &payment.FinalizePaymentRequest{
		ProjectID:        projectID,
		InvoiceID:        invoiceID,
		PaymentSessionID: sessionID,
	}).Return(&payment.FinalizePaymentResult{
		Success:     true,
		ResultCode:  payment.ResultSuccess,
		CheckoutURL: "/checkout/" + invoiceID.String(),
	}, nil)

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/checkout/"+invoiceID.String()+"/pay/card/"+sessionID.String()+"/finalize", nil)
	req = withTenantContext(req, project)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestCardFinalizeHandler_NoTenantContext(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	invoiceID := uuid.New()
	sessionID := uuid.New()

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/checkout/"+invoiceID.String()+"/pay/card/"+sessionID.String()+"/finalize", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestCardFinalizeHandler_SessionExpired(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	invoiceID := uuid.New()
	projectID := uuid.New()
	project := testProject(projectID)
	sessionID := uuid.New()

	paySvc.EXPECT().FinalizePayment(mock.Anything, &payment.FinalizePaymentRequest{
		ProjectID:        projectID,
		InvoiceID:        invoiceID,
		PaymentSessionID: sessionID,
	}).Return(nil, &payment.SessionExpiredError{SessionID: sessionID.String()})

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/checkout/"+invoiceID.String()+"/pay/card/"+sessionID.String()+"/finalize", nil)
	req = withTenantContext(req, project)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusGone {
		t.Errorf("expected status %d, got %d: %s", http.StatusGone, w.Code, w.Body.String())
	}
}

func TestVerifyDomainHandler_ValidDomain(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)

	paySvc.EXPECT().VerifyDomain(mock.Anything, "pay.merchant.com").Return(true, nil)

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/verify-domain?secret=test-secret&domain=pay.merchant.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestVerifyDomainHandler_InvalidDomain(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)

	paySvc.EXPECT().VerifyDomain(mock.Anything, "unknown.domain.com").Return(false, nil)

	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/verify-domain?secret=test-secret&domain=unknown.domain.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestVerifyDomainHandler_MissingSecret(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/verify-domain?domain=pay.merchant.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestVerifyDomainHandler_WrongSecret(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/verify-domain?secret=wrong-secret&domain=pay.merchant.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestVerifyDomainHandler_MissingDomain(t *testing.T) {
	paySvc := paymocks.NewMockService(t)
	invSvc := invomocks.NewMockService(t)
	gwSvc := gwmocks.NewMockService(t)
	h := checkout.New(paySvc, invSvc, gwSvc, "test-secret")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/verify-domain?secret=test-secret", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
