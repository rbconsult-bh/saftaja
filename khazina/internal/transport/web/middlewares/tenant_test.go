package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant/mocks"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/web/middlewares"
)

func TestTenantResolver_ValidDomain(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	projectID := uuid.New()
	project := &store.Project{
		ID:           projectID,
		CustomDomain: pgtype.Text{String: "pay.merchant.com", Valid: true},
	}

	mockSvc.EXPECT().GetProjectByDomain(mock.Anything, "pay.merchant.com").Return(project, nil)

	handler := middlewares.TenantResolver(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := middlewares.GetProjectFromContext(r.Context())
		require.NotNil(t, p)
		require.Equal(t, projectID, p.ID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/checkout/123", nil)
	req.Host = "pay.merchant.com"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestTenantResolver_ValidDomainWithPort(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	projectID := uuid.New()
	project := &store.Project{
		ID:           projectID,
		CustomDomain: pgtype.Text{String: "pay.merchant.com", Valid: true},
	}

	mockSvc.EXPECT().GetProjectByDomain(mock.Anything, "pay.merchant.com").Return(project, nil)

	handler := middlewares.TenantResolver(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := middlewares.GetProjectFromContext(r.Context())
		require.NotNil(t, p)
		require.Equal(t, projectID, p.ID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/checkout/123", nil)
	req.Host = "pay.merchant.com:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestTenantResolver_InvalidDomain(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	mockSvc.EXPECT().GetProjectByDomain(mock.Anything, "unknown.com").Return(nil, pgx.ErrNoRows)

	handler := middlewares.TenantResolver(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := middlewares.GetProjectFromContext(r.Context())
		require.Nil(t, p) // No project in context for invalid domain
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	req.Host = "unknown.com"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code) // Request still passes, handler can decide
}

func TestGetProjectFromContext_NilWhenNoProject(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	p := middlewares.GetProjectFromContext(req.Context())
	require.Nil(t, p)
}
