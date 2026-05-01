package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant/mocks"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/web/middlewares"
)

func TestDynamicCORS_ValidOrigin(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	mockSvc.EXPECT().IsDomainValid(mock.Anything, "pay.merchant.com").Return(true, nil)

	handler := middlewares.DynamicCORS(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/checkout/123", nil)
	req.Header.Set("Origin", "https://pay.merchant.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, "https://pay.merchant.com", w.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestDynamicCORS_ValidOriginHTTP(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	mockSvc.EXPECT().IsDomainValid(mock.Anything, "pay.merchant.com").Return(true, nil)

	handler := middlewares.DynamicCORS(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/checkout/123", nil)
	req.Header.Set("Origin", "http://pay.merchant.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, "http://pay.merchant.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestDynamicCORS_InvalidOrigin(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	mockSvc.EXPECT().IsDomainValid(mock.Anything, "evil.com").Return(false, nil)

	handler := middlewares.DynamicCORS(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/checkout/123", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestDynamicCORS_NoOriginHeader(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	// No mock expectation needed - IsDomainValid won't be called

	handler := middlewares.DynamicCORS(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/checkout/123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestDynamicCORS_PreflightRequest(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	mockSvc.EXPECT().IsDomainValid(mock.Anything, "pay.merchant.com").Return(true, nil)

	handler := middlewares.DynamicCORS(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	}))

	req := httptest.NewRequest("OPTIONS", "/checkout/123/initiate", nil)
	req.Header.Set("Origin", "https://pay.merchant.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "https://pay.merchant.com", w.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "POST, GET, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
}

func TestDynamicCORS_PreflightWithInvalidOrigin(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	mockSvc.EXPECT().IsDomainValid(mock.Anything, "evil.com").Return(false, nil)

	handler := middlewares.DynamicCORS(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	}))

	req := httptest.NewRequest("OPTIONS", "/checkout/123/initiate", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}
