package billing

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	uuidProject          = uuid.MustParse("00000000-0000-0000-0000-000000001001")
	uuidGateway          = uuid.MustParse("00000000-0000-0000-0000-000000002001")
	uuidInvoice          = uuid.MustParse("00000000-0000-0000-0000-000000003001")
	uuidInvoiceNoItems   = uuid.MustParse("00000000-0000-0000-0000-000000003002")
	uuidInvoicePaid      = uuid.MustParse("00000000-0000-0000-0000-000000003003")
	uuidInvoiceCancelled = uuid.MustParse("00000000-0000-0000-0000-000000003004")
	uuidNotExist         = uuid.MustParse("00000000-0000-0000-0000-00000000ffff")
)

var testEncryptionKey = []byte("12345678901234567890123456789012")

type testEnv struct {
	ctx     context.Context
	queries store.TransactionQuerier
	svc     Service
}

func setupTestEnv(t *testing.T) testEnv {
	t.Helper()

	db := testutil.SetupIsolatedDBWithFixtures(t, "./testdata/fixtures")
	queries := store.NewTransactionQuerier(db.Pool)

	return testEnv{
		ctx:     t.Context(),
		queries: queries,
		svc:     New(queries, testEncryptionKey),
	}
}

func TestGetInvoice_Validation(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name string
		req  GetInvoiceRequest
		err  error
	}{
		{
			name: "missing_invoice_ID",
			req:  GetInvoiceRequest{ProjectID: uuidProject},
			err:  ErrInvalidArgument,
		},
		{
			name: "missing_project_ID",
			req:  GetInvoiceRequest{InvoiceID: uuidInvoice},
			err:  ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := env.svc.GetInvoice(env.ctx, tt.req)
			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestGetInvoice_Success(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.GetInvoice(env.ctx, GetInvoiceRequest{
		InvoiceID: uuidInvoice,
		ProjectID: uuidProject,
	})
	require.NoError(t, err)
	assert.Equal(t, uuidInvoice, resp.Invoice.ID)
	assert.Len(t, resp.Invoice.Items, 1)
	assert.Equal(t, InvoiceStatusPending, resp.Invoice.Status)
}

func TestGetInvoice_NotFound_WrongProject(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.GetInvoice(env.ctx, GetInvoiceRequest{
		InvoiceID: uuidInvoice,
		ProjectID: uuidNotExist,
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetInvoice_NotFound_InvoiceNotExist(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.GetInvoice(env.ctx, GetInvoiceRequest{
		InvoiceID: uuidNotExist,
		ProjectID: uuidProject,
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetInvoice_NotFound_NoItems(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.GetInvoice(env.ctx, GetInvoiceRequest{
		InvoiceID: uuidInvoiceNoItems,
		ProjectID: uuidProject,
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStartPayment_Validation(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name     string
		modifier func(*StartPaymentRequest)
		err      error
	}{
		{
			name:     "missing_invoice_ID",
			modifier: func(r *StartPaymentRequest) { r.InvoiceID = uuid.Nil },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_project_ID",
			modifier: func(r *StartPaymentRequest) { r.ProjectID = uuid.Nil },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_idempotency_key",
			modifier: func(r *StartPaymentRequest) { r.IdempotencyKey = "" },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_gateway_account_ID",
			modifier: func(r *StartPaymentRequest) { r.GatewayAccountID = uuid.Nil },
			err:      ErrInvalidArgument,
		},
		{
			name:     "unsupported_payment_method",
			modifier: func(r *StartPaymentRequest) { r.PaymentMethod = "CASH_ON_DELIVERY" },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_payer_IP",
			modifier: func(r *StartPaymentRequest) { r.PayerIP = "" },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_payer_user_agent",
			modifier: func(r *StartPaymentRequest) { r.PayerUserAgent = "" },
			err:      ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := StartPaymentRequest{
				InvoiceID:        uuidInvoice,
				ProjectID:        uuidProject,
				IdempotencyKey:   "key-val",
				GatewayAccountID: uuidGateway,
				PaymentMethod:    PaymentMethodCard,
				PayerIP:          "192.168.1.1",
				PayerUserAgent:   "Mozilla/5.0",
			}
			tt.modifier(&req)

			resp, err := env.svc.StartPayment(env.ctx, req)
			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestStartPayment_InvoiceErrors(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name      string
		invoiceID uuid.UUID
		err       error
	}{
		{"not_found", uuidNotExist, ErrInvoiceNotFound},
		{"already_paid", uuidInvoicePaid, ErrInvoiceAlreadyPaid},
		{"cancelled", uuidInvoiceCancelled, ErrInvoiceCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := env.svc.StartPayment(env.ctx, StartPaymentRequest{
				InvoiceID:        tt.invoiceID,
				ProjectID:        uuidProject,
				IdempotencyKey:   "key-" + tt.name,
				GatewayAccountID: uuidGateway,
				PaymentMethod:    PaymentMethodCard,
				PayerIP:          "192.168.1.1",
				PayerUserAgent:   "Mozilla/5.0",
			})
			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestStartPayment_GatewayAccountNotFound(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.StartPayment(env.ctx, StartPaymentRequest{
		InvoiceID:        uuidInvoice,
		ProjectID:        uuidProject,
		IdempotencyKey:   "key-no-gw",
		GatewayAccountID: uuidNotExist,
		PaymentMethod:    PaymentMethodCard,
		PayerIP:          "192.168.1.1",
		PayerUserAgent:   "Mozilla/5.0",
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrGatewayAccountNotFound)
}

func TestStartPayment_Idempotency(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name        string
		idempotKey  string
		invoiceID   uuid.UUID
		gatewayID   uuid.UUID
		wantErr     error
		wantIntent  uuid.UUID
		wantSession string
	}{
		{
			name:        "replay",
			idempotKey:  "idem-created",
			invoiceID:   uuidInvoice,
			gatewayID:   uuidGateway,
			wantIntent:  uuid.MustParse("00000000-0000-0000-0000-000000005001"),
			wantSession: "session-existing-abc",
		},
		{
			name:       "mismatched_invoice",
			idempotKey: "idem-mismatch",
			invoiceID:  uuidInvoice, // fixture key is on uuidInvoiceNoItems
			gatewayID:  uuidGateway,
			wantErr:    ErrIdempotencyMismatch,
		},
		{
			name:       "expired",
			idempotKey: "idem-expired",
			invoiceID:  uuidInvoice,
			gatewayID:  uuidGateway,
			wantErr:    ErrPaymentIntentExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := env.svc.StartPayment(env.ctx, StartPaymentRequest{
				InvoiceID:        tt.invoiceID,
				ProjectID:        uuidProject,
				IdempotencyKey:   tt.idempotKey,
				GatewayAccountID: tt.gatewayID,
				PaymentMethod:    PaymentMethodCard,
				PayerIP:          "192.168.1.1",
				PayerUserAgent:   "Mozilla/5.0",
			})

			if tt.wantErr != nil {
				assert.Nil(t, resp)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantIntent, resp.PaymentIntentID)
			require.NotNil(t, resp.GatewaySessionID)
			assert.Equal(t, tt.wantSession, *resp.GatewaySessionID)
		})
	}
}

func TestStartPayment_CardSuccess(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.StartPayment(env.ctx, StartPaymentRequest{
		InvoiceID:        uuidInvoice,
		ProjectID:        uuidProject,
		IdempotencyKey:   "key-card-success",
		GatewayAccountID: uuidGateway,
		PaymentMethod:    PaymentMethodCard,
		PayerIP:          "192.168.1.1",
		PayerUserAgent:   "Mozilla/5.0",
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.PaymentIntentID)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, resp.PaymentIntentID)
	require.NoError(t, err)
	assert.Equal(t, uuidInvoice, intent.InvoiceID)
	assert.Equal(t, uuidProject, intent.ProjectID)
	assert.Equal(t, uuidGateway, intent.GatewayAccountID)
	assert.Equal(t, store.PaymentMethodCard, intent.PaymentMethod)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)
	assert.Equal(t, "key-card-success", intent.IdempotencyKey)
}

func TestStartPayment_ApplePay(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.StartPayment(env.ctx, StartPaymentRequest{
		InvoiceID:        uuidInvoice,
		ProjectID:        uuidProject,
		IdempotencyKey:   "key-apple-pay",
		GatewayAccountID: uuidGateway,
		PaymentMethod:    PaymentMethodApplePay,
		PayerIP:          "192.168.1.1",
		PayerUserAgent:   "Mozilla/5.0",
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrUnsupportedPaymentMethod)
}
