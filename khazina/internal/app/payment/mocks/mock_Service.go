package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/payment"
)

type MockService struct {
	mock.Mock
}

func NewMockService(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockService {
	m := &MockService{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockService) GetInvoiceData(ctx context.Context, invoiceID, projectID uuid.UUID) (*payment.InvoiceData, error) {
	args := m.Called(ctx, invoiceID, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.InvoiceData), args.Error(1)
}

func (m *MockService) ListActiveGatewayCredentials(ctx context.Context, projectID uuid.UUID) ([]payment.GatewayCredentials, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]payment.GatewayCredentials), args.Error(1)
}

func (m *MockService) InitiateSession(ctx context.Context, req *payment.InitiateSessionRequest) (*payment.InitiateSessionResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.InitiateSessionResult), args.Error(1)
}

func (m *MockService) InitiateAuth(ctx context.Context, req *payment.InitiateAuthRequest) (*payment.InitiateAuthResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.InitiateAuthResult), args.Error(1)
}

func (m *MockService) ProcessAuth(ctx context.Context, req *payment.ProcessAuthRequest) (*payment.ProcessAuthResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.ProcessAuthResult), args.Error(1)
}

func (m *MockService) FinalizePayment(ctx context.Context, req *payment.FinalizePaymentRequest) (*payment.FinalizePaymentResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.FinalizePaymentResult), args.Error(1)
}

func (m *MockService) VerifyDomain(ctx context.Context, domain string) (bool, error) {
	args := m.Called(ctx, domain)
	return args.Bool(0), args.Error(1)
}
