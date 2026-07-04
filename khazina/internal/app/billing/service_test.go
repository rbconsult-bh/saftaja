package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
	ctx             context.Context
	db              *pgxpool.Pool
	queries         store.TransactionQuerier
	cardGateway     *fakeCardGateway
	gatewayResolver *fakeGatewayResolver
	svc             Service
}

func setupTestEnv(t *testing.T) testEnv {
	t.Helper()

	db := testutil.SetupIsolatedDBWithFixtures(t, "./testdata/fixtures")
	queries := store.NewTransactionQuerier(db.Pool)
	cardGateway := &fakeCardGateway{
		sessionID: "session-test-123",
	}
	gatewayResolver := &fakeGatewayResolver{
		cardGateway: cardGateway,
	}

	return testEnv{
		ctx:             t.Context(),
		db:              db.Pool,
		queries:         queries,
		cardGateway:     cardGateway,
		gatewayResolver: gatewayResolver,
		svc:             New(queries, gatewayResolver),
	}
}

func validGetInvoiceRequest() GetInvoiceRequest {
	return GetInvoiceRequest{
		InvoiceID: uuidInvoice,
		ProjectID: uuidProject,
	}
}

func validStartPaymentRequest(idempotencyKey string) StartPaymentRequest {
	return StartPaymentRequest{
		InvoiceID:        uuidInvoice,
		ProjectID:        uuidProject,
		IdempotencyKey:   idempotencyKey,
		GatewayAccountID: uuidGateway,
		PaymentMethod:    PaymentMethodCard,
		PayerIP:          "192.168.1.1",
		PayerUserAgent:   "Mozilla/5.0",
	}
}

func countPaymentIntents(t *testing.T, env testEnv) int {
	t.Helper()

	var count int
	err := env.db.QueryRow(env.ctx, "SELECT COUNT(*) FROM payment_intents").Scan(&count)
	require.NoError(t, err)

	return count
}

func assertStartPaymentErrorWithoutNewIntent(t *testing.T, env testEnv, req StartPaymentRequest, wantErr error) {
	t.Helper()

	before := countPaymentIntents(t, env)

	resp, err := env.svc.StartPayment(env.ctx, req)

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, before, countPaymentIntents(t, env))
}

func TestGetInvoice_RejectsInvalidRequest(t *testing.T) {
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

	resp, err := env.svc.GetInvoice(env.ctx, validGetInvoiceRequest())

	require.NoError(t, err)
	assert.Equal(t, uuidInvoice, resp.Invoice.ID)
	assert.Equal(t, uuidProject, resp.Invoice.ProjectID)
	assert.Equal(t, InvoiceStatusPending, resp.Invoice.Status)
	assert.Equal(t, "BHD", resp.Invoice.Currency)
	assert.Equal(t, "testcustomer@saftaja.com", resp.Invoice.CustomerEmail)
	require.Len(t, resp.Invoice.Items, 1)
	assert.Equal(t, "test item", resp.Invoice.Items[0].Name)
}

func TestGetInvoice_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name string
		req  GetInvoiceRequest
	}{
		{
			name: "wrong_project",
			req: GetInvoiceRequest{
				InvoiceID: uuidInvoice,
				ProjectID: uuidNotExist,
			},
		},
		{
			name: "invoice_does_not_exist",
			req: GetInvoiceRequest{
				InvoiceID: uuidNotExist,
				ProjectID: uuidProject,
			},
		},
		{
			name: "invoice_has_no_items",
			req: GetInvoiceRequest{
				InvoiceID: uuidInvoiceNoItems,
				ProjectID: uuidProject,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := env.svc.GetInvoice(env.ctx, tt.req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, ErrNotFound)
		})
	}
}

func TestStartPayment_RejectsInvalidRequestBeforeSideEffects(t *testing.T) {
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
			req := validStartPaymentRequest("key-val")
			tt.modifier(&req)

			assertStartPaymentErrorWithoutNewIntent(t, env, req, tt.err)
		})
	}
}

func TestStartPayment_RejectsMissingInvoice(t *testing.T) {
	env := setupTestEnv(t)
	req := validStartPaymentRequest("key-missing-invoice")
	req.InvoiceID = uuidNotExist

	assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrInvoiceNotFound)
}

func TestStartPayment_RejectsPaidInvoice(t *testing.T) {
	env := setupTestEnv(t)
	req := validStartPaymentRequest("key-paid-invoice")
	req.InvoiceID = uuidInvoicePaid

	assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrInvoiceAlreadyPaid)
}

func TestStartPayment_RejectsCancelledInvoice(t *testing.T) {
	env := setupTestEnv(t)
	req := validStartPaymentRequest("key-cancelled-invoice")
	req.InvoiceID = uuidInvoiceCancelled

	assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrInvoiceCancelled)
}

func TestStartPayment_RejectsUnknownGatewayAccount(t *testing.T) {
	env := setupTestEnv(t)
	req := validStartPaymentRequest("key-no-gw")
	req.GatewayAccountID = uuidNotExist

	assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrGatewayAccountNotFound)
}

func TestStartPayment_ReplaysExistingIntent(t *testing.T) {
	env := setupTestEnv(t)
	before := countPaymentIntents(t, env)

	resp, err := env.svc.StartPayment(env.ctx, validStartPaymentRequest("idem-created"))

	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000005001"), resp.PaymentIntentID)
	require.NotNil(t, resp.GatewaySessionID)
	assert.Equal(t, "session-existing-abc", *resp.GatewaySessionID)
	assert.Equal(t, before, countPaymentIntents(t, env))
}

func TestStartPayment_RejectsIdempotencyMismatch(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name   string
		change func(*StartPaymentRequest)
	}{
		{
			name: "invoice",
			change: func(r *StartPaymentRequest) {
				r.IdempotencyKey = "idem-mismatch"
			},
		},
		{
			name: "gateway_account",
			change: func(r *StartPaymentRequest) {
				r.IdempotencyKey = "idem-created"
				r.GatewayAccountID = uuidNotExist
			},
		},
		{
			name: "payment_method",
			change: func(r *StartPaymentRequest) {
				r.IdempotencyKey = "idem-created"
				r.PaymentMethod = PaymentMethodApplePay
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validStartPaymentRequest("unused")
			tt.change(&req)

			assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrIdempotencyMismatch)
		})
	}
}

func TestStartPayment_RejectsExpiredIntent(t *testing.T) {
	env := setupTestEnv(t)
	req := validStartPaymentRequest("idem-expired")

	assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrPaymentIntentExpired)
}

func TestStartPayment_CreatesCardIntent(t *testing.T) {
	env := setupTestEnv(t)
	before := countPaymentIntents(t, env)

	resp, err := env.svc.StartPayment(env.ctx, validStartPaymentRequest("key-card-success"))

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.PaymentIntentID)
	require.NotNil(t, resp.GatewaySessionID)
	assert.Equal(t, before+1, countPaymentIntents(t, env))

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, resp.PaymentIntentID)
	require.NoError(t, err)
	assert.Equal(t, uuidInvoice, intent.InvoiceID)
	assert.Equal(t, uuidProject, intent.ProjectID)
	assert.Equal(t, uuidGateway, intent.GatewayAccountID)
	assert.Equal(t, store.PaymentMethodCard, intent.PaymentMethod)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)
	assert.Equal(t, "key-card-success", intent.IdempotencyKey)
	require.NotNil(t, intent.GatewaySessionID)
	assert.Equal(t, "session-test-123", *intent.GatewaySessionID)

	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, uuidGateway, env.gatewayResolver.account.ID)

	assert.Equal(t, 1, env.cardGateway.calls)
	assert.Equal(t, uuidInvoice, env.cardGateway.req.InvoiceID)
	assert.Equal(t, "BHD", env.cardGateway.req.Currency)

	require.NotNil(t, resp.GatewaySessionID)
	assert.Equal(t, "session-test-123", *resp.GatewaySessionID)
}

func TestStartPayment_DoesNotCreateIntentWhenCardGatewayFails(t *testing.T) {
	env := setupTestEnv(t)
	env.cardGateway.err = errors.New("gateway down")

	req := validStartPaymentRequest("key-gateway-fails")

	assertStartPaymentErrorWithoutNewIntent(t, env, req, env.cardGateway.err)
	assert.Equal(t, 1, env.cardGateway.calls)
}

func TestStartPayment_RejectsUnsupportedCardGateway(t *testing.T) {
	env := setupTestEnv(t)
	env.gatewayResolver.err = ErrUnsupportedGateway

	req := validStartPaymentRequest("key-unsupported-gateway")

	assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrUnsupportedPaymentMethod)
	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.calls)
}

func TestStartPayment_ReturnsGatewayResolverError(t *testing.T) {
	env := setupTestEnv(t)
	errResolver := errors.New("resolver failed")
	env.gatewayResolver.err = errResolver

	req := validStartPaymentRequest("key-resolver-error")

	assertStartPaymentErrorWithoutNewIntent(t, env, req, errResolver)
	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.calls)
}

func TestStartPayment_RejectsApplePayUntilSupported(t *testing.T) {
	env := setupTestEnv(t)
	req := validStartPaymentRequest("key-apple-pay")
	req.PaymentMethod = PaymentMethodApplePay

	assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrUnsupportedPaymentMethod)
}

type fakeGatewayResolver struct {
	cardGateway CardGateway
	err         error
	account     store.GatewayAccount
	calls       int
}

func (f *fakeGatewayResolver) CardGateway(account store.GatewayAccount) (CardGateway, error) {
	f.calls++
	f.account = account
	return f.cardGateway, f.err
}

type fakeCardGateway struct {
	sessionID string
	req       CreateSessionRequest
	calls     int
	err       error
}

func (f *fakeCardGateway) CreateSession(ctx context.Context, r CreateSessionRequest) (*CreateSessionResponse, error) {
	f.calls++
	f.req = r
	if f.err != nil {
		return nil, f.err
	}
	return &CreateSessionResponse{GatewaySessionID: f.sessionID}, nil
}
