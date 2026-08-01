package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant/mocks"
	saftajacontext "github.com/rbconsult-bh/saftaja/khazina/internal/pkg/context"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/middleware"
)

func TestTenantResolver_ValidDomain(t *testing.T) {
	mockSvc := mocks.NewMockService(t)
	projectID := uuid.New()
	project := &tenant.Project{
		ID:           projectID,
		CustomDomain: "pay.merchant.com",
	}

	mockSvc.EXPECT().GetProjectByDomain(mock.Anything, "pay.merchant.com").Return(project, nil)

	handler := middleware.TenantResolver(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := saftajacontext.ProjectFromContext(r.Context())
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
	project := &tenant.Project{
		ID:           projectID,
		CustomDomain: "pay.merchant.com",
	}

	mockSvc.EXPECT().GetProjectByDomain(mock.Anything, "pay.merchant.com").Return(project, nil)

	handler := middleware.TenantResolver(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := saftajacontext.ProjectFromContext(r.Context())
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

	handler := middleware.TenantResolver(mockSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := saftajacontext.ProjectFromContext(r.Context())
		require.Nil(t, p)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	req.Host = "unknown.com"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetProjectFromContext_NilWhenNoProject(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	p := saftajacontext.ProjectFromContext(req.Context())
	require.Nil(t, p)
}
