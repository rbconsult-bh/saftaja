package billing

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/testutil"
	"github.com/stretchr/testify/assert"
)

type billingTestEnv struct {
	ctx     context.Context
	queries store.TransactionQuerier
	svc     Service
}

func setupTestEnv(t *testing.T) billingTestEnv {
	db := testutil.SetupIsolatedDBWithFixtures(
		t,
		"./testdata/fixtures",
	)

	queries := store.NewTransactionQuerier(db.Pool)
	svc := New(queries)

	return billingTestEnv{
		ctx:     t.Context(),
		queries: queries,
		svc:     svc,
	}
}

// --- Test: GetInvoice

func TestGetInvoice_Validation(t *testing.T) {
	testEnv := setupTestEnv(t)
	validUUID := uuid.MustParse("00000000-0000-0000-0000-000000001000")

	tests := []struct {
		name string
		req  GetInvoiceRequest
		err  error
	}{
		{
			name: "missing invoice ID",
			req:  GetInvoiceRequest{ProjectID: validUUID},
			err:  ErrInvalidArgument,
		},
		{
			name: "missing project ID",
			req:  GetInvoiceRequest{InvoiceID: validUUID},
			err:  ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := testEnv.svc.GetInvoice(testEnv.ctx, tt.req)
			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestGetInvoice_Success(t *testing.T) {
	testEnv := setupTestEnv(t)

	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000001000"),
		ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
	})
	assert.NoError(t, err)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000001000"), resp.Invoice.ID)
	assert.Len(t, resp.Invoice.Items, 1)
}

func TestGetInvoice_NotFound_WhenProjectIDWrong(t *testing.T) {
	testEnv := setupTestEnv(t)

	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000001000"),
		ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000200"),
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetInvoice_NotFound_WhenInvoiceIDNotExists(t *testing.T) {
	testEnv := setupTestEnv(t)

	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000002222"),
		ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetInvoice_NotFound_WhenNoItems(t *testing.T) {
	testEnv := setupTestEnv(t)

	// Invoice 001001 exists but has zero invoice_items — JOIN drops it.
	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000001001"),
		ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- Test: StartPayment

func TestStartPayment_Validation(t *testing.T) {
	testEnv := setupTestEnv(t)
	validUUID := uuid.MustParse("00000000-0000-0000-0000-000000001000")

	tests := []struct {
		name     string
		modifier func(*StartPaymentRequest)
		err      error
	}{
		{
			name:     "missing invoice ID",
			modifier: func(r *StartPaymentRequest) { r.InvoiceID = uuid.Nil },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing project ID",
			modifier: func(r *StartPaymentRequest) { r.ProjectID = uuid.Nil },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing idempotency key",
			modifier: func(r *StartPaymentRequest) { r.IdempotencyKey = "" },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing gateway account ID",
			modifier: func(r *StartPaymentRequest) { r.GatewayAccountID = uuid.Nil },
			err:      ErrInvalidArgument,
		},
		{
			name:     "unsupported payment method",
			modifier: func(r *StartPaymentRequest) { r.PaymentMethod = "CASH_ON_DELIVERY" },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing payer IP",
			modifier: func(r *StartPaymentRequest) { r.PayerIP = "" },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing payer user agent",
			modifier: func(r *StartPaymentRequest) { r.PayerUserAgent = "" },
			err:      ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := StartPaymentRequest{
				InvoiceID:        validUUID,
				ProjectID:        validUUID,
				IdempotencyKey:   "idemp-key-12345",
				GatewayAccountID: validUUID,
				PaymentMethod:    PaymentMethodCard,
				PayerIP:          "192.168.1.1",
				PayerUserAgent:   "Mozilla/5.0",
			}

			tt.modifier(&req)

			resp, err := testEnv.svc.StartPayment(testEnv.ctx, req)
			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}
