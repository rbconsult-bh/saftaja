package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/testutil"
	"github.com/shopspring/decimal"
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
	uuidPaymentIntent    = uuid.MustParse("00000000-0000-0000-0000-000000005001")
	uuidPaymentExpired   = uuid.MustParse("00000000-0000-0000-0000-000000005002")
	uuidPaymentVerifying = uuid.MustParse("00000000-0000-0000-0000-000000005005")
	uuidPaymentNoSetup   = uuid.MustParse("00000000-0000-0000-0000-000000005006")
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
		setupCardPayment: fakeSetupCardPaymentCall{
			resp: &SetupCardPaymentGatewayResponse{GatewaySetupReference: "session-test-123"},
		},
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
		svc:             New(db.Pool, queries, gatewayResolver),
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
	require.NotNil(t, resp.GatewaySetupReference)
	assert.Equal(t, "session-existing-abc", *resp.GatewaySetupReference)
	assert.Equal(t, before, countPaymentIntents(t, env))
	assert.Equal(t, 0, env.cardGateway.setupCardPayment.calls)
}

func TestStartPayment_RejectsExistingIntentWithUnknownPaymentMethod(t *testing.T) {
	env := setupTestEnv(t)

	assertStartPaymentErrorWithoutNewIntent(t, env, validStartPaymentRequest("idem-unknown-method"), ErrPaymentIntentInvalidState)
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
	require.NotNil(t, resp.GatewaySetupReference)
	assert.Equal(t, before+1, countPaymentIntents(t, env))

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, resp.PaymentIntentID)
	require.NoError(t, err)
	assert.Equal(t, uuidInvoice, intent.InvoiceID)
	assert.Equal(t, uuidProject, intent.ProjectID)
	assert.Equal(t, uuidGateway, intent.GatewayAccountID)
	assert.Equal(t, store.PaymentMethodCard, intent.PaymentMethod)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)
	assert.Equal(t, "key-card-success", intent.IdempotencyKey)
	require.NotNil(t, intent.GatewaySetupReference)
	assert.Equal(t, "session-test-123", *intent.GatewaySetupReference)

	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, uuidGateway, env.gatewayResolver.account.ID)

	assert.Equal(t, 1, env.cardGateway.setupCardPayment.calls)
	assert.Equal(t, uuidInvoice, env.cardGateway.setupCardPayment.req.InvoiceID)
	assert.Equal(t, "BHD", env.cardGateway.setupCardPayment.req.Currency)
	assert.True(t, decimal.RequireFromString("15.000").Equal(env.cardGateway.setupCardPayment.req.Amount))

	require.NotNil(t, resp.GatewaySetupReference)
	assert.Equal(t, "session-test-123", *resp.GatewaySetupReference)
}

func TestStartPayment_DoesNotCreateIntentWhenCardGatewayFails(t *testing.T) {
	env := setupTestEnv(t)
	errGateway := errors.New("gateway down")
	env.cardGateway.setupCardPayment.err = errGateway

	req := validStartPaymentRequest("key-gateway-fails")

	assertStartPaymentErrorWithoutNewIntent(t, env, req, errGateway)
	assert.Equal(t, 1, env.cardGateway.setupCardPayment.calls)
}

func TestStartPayment_RejectsUnsupportedCardGateway(t *testing.T) {
	env := setupTestEnv(t)
	env.gatewayResolver.err = ErrUnsupportedGateway

	req := validStartPaymentRequest("key-unsupported-gateway")

	assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrUnsupportedPaymentMethod)
	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.setupCardPayment.calls)
}

func TestStartPayment_ReturnsGatewayResolverError(t *testing.T) {
	env := setupTestEnv(t)
	errResolver := errors.New("resolver failed")
	env.gatewayResolver.err = errResolver

	req := validStartPaymentRequest("key-resolver-error")

	assertStartPaymentErrorWithoutNewIntent(t, env, req, errResolver)
	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.setupCardPayment.calls)
}

func TestStartPayment_RejectsApplePayUntilSupported(t *testing.T) {
	env := setupTestEnv(t)
	req := validStartPaymentRequest("key-apple-pay")
	req.PaymentMethod = PaymentMethodApplePay

	assertStartPaymentErrorWithoutNewIntent(t, env, req, ErrUnsupportedPaymentMethod)
}

func TestVerifyCard_RejectsInvalidRequest(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name string
		req  VerifyCardRequest
		err  error
	}{
		{
			name: "missing_project_ID",
			req:  VerifyCardRequest{InvoiceID: uuidInvoice},
			err:  ErrInvalidArgument,
		},
		{
			name: "missing_invoice_ID",
			req:  VerifyCardRequest{ProjectID: uuidProject},
			err:  ErrInvalidArgument,
		},
		{
			name: "missing_payment_intent_ID",
			req:  VerifyCardRequest{InvoiceID: uuidInvoice, ProjectID: uuidProject},
			err:  ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := env.svc.VerifyCard(env.ctx, tt.req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestVerifyCard_ReturnsNotFound(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.VerifyCard(env.ctx, VerifyCardRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidNotExist,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareVerifyCard.calls)
}

func TestVerifyCard_RejectsInvalidStateTransition(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.VerifyCard(env.ctx, VerifyCardRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentVerifying,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidTransition)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareVerifyCard.calls)
}

func TestVerifyCard_RejectsExpiredIntent(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.VerifyCard(env.ctx, VerifyCardRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentExpired,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentExpired)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareVerifyCard.calls)
}

func TestVerifyCard_RejectsMissingGatewaySetupReference(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.VerifyCard(env.ctx, VerifyCardRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentNoSetup,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareVerifyCard.calls)
}

func TestVerifyCard_ReturnsGatewayResolverError(t *testing.T) {
	env := setupTestEnv(t)
	errResolver := errors.New("resolver failed")
	env.gatewayResolver.err = errResolver

	resp, err := env.svc.VerifyCard(env.ctx, VerifyCardRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errResolver)
	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareVerifyCard.calls)

	_, err = env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	assert.Error(t, err)
}

func TestVerifyCard_ReturnsPrepareVerifyCardErrorBeforeCreatingOperation(t *testing.T) {
	env := setupTestEnv(t)
	errPrepare := errors.New("prepare failed")
	env.cardGateway.prepareVerifyCard.err = errPrepare

	resp, err := env.svc.VerifyCard(env.ctx, VerifyCardRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errPrepare)
	assert.Equal(t, 1, env.cardGateway.prepareVerifyCard.calls)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

	_, err = env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	assert.Error(t, err)
}

func TestVerifyCard_RecordsPendingGatewayOperationBeforeSending(t *testing.T) {
	env := setupTestEnv(t)
	rawReq := []byte(`{"prepared":true}`)
	rawResp := []byte(`{"result":"SUCCESS"}`)

	env.cardGateway.prepareVerifyCard.resp = &PreparedVerifyCardGatewayRequest{
		GatewayReference: "gw-init-auth-123",
		RawRequest:       rawReq,
		send: func(ctx context.Context) (*VerifyCardGatewayResponse, error) {
			intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
			require.NoError(t, err)
			assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

			op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
			require.NoError(t, err)
			assert.Equal(t, store.GatewayOperationStatusPending, op.Status)
			assert.JSONEq(t, string(rawReq), string(op.RawRequest))
			assert.Equal(t, "gw-init-auth-123", op.GatewayReference)

			return &VerifyCardGatewayResponse{
				NextStep:    VerifyCardGatewayNextStepChallengeCard,
				RawResponse: rawResp,
			}, nil
		},
	}

	resp, err := env.svc.VerifyCard(env.ctx, VerifyCardRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, VerifyCardNextStepChallengeCard, resp.NextStep)
	assert.Equal(t, 1, env.cardGateway.prepareVerifyCard.calls)
	assert.Equal(t, uuidInvoice, env.cardGateway.prepareVerifyCard.req.InvoiceID)
	assert.Equal(t, "BHD", env.cardGateway.prepareVerifyCard.req.Currency)
	assert.Equal(t, "session-existing-abc", env.cardGateway.prepareVerifyCard.req.GatewaySetupReference)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusVerifyingCard, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.JSONEq(t, string(rawResp), string(op.RawResponse))
}

func TestVerifyCard_RecordsCantContinueGatewayResponseAndAllowsRetry(t *testing.T) {
	env := setupTestEnv(t)
	rawResp := []byte(`{"result":"FAILURE"}`)
	req := VerifyCardRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	}

	env.cardGateway.prepareVerifyCard.resp = &PreparedVerifyCardGatewayRequest{
		GatewayReference: "gw-init-auth-failed",
		RawRequest:       []byte(`{"prepared":true}`),
		send: func(ctx context.Context) (*VerifyCardGatewayResponse, error) {
			return &VerifyCardGatewayResponse{
				NextStep:    VerifyCardGatewayNextStepCantContinue,
				RawResponse: rawResp,
			}, nil
		},
	}

	resp, err := env.svc.VerifyCard(env.ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, VerifyCardNextStepCantContinue, resp.NextStep)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusFailed, op.Status)
	assert.JSONEq(t, string(rawResp), string(op.RawResponse))

	retryRawResp := []byte(`{"result":"SUCCESS"}`)
	env.cardGateway.prepareVerifyCard.resp = &PreparedVerifyCardGatewayRequest{
		GatewayReference: "gw-init-auth-retry",
		RawRequest:       []byte(`{"prepared":"retry"}`),
		send: func(ctx context.Context) (*VerifyCardGatewayResponse, error) {
			return &VerifyCardGatewayResponse{
				NextStep:    VerifyCardGatewayNextStepChallengeCard,
				RawResponse: retryRawResp,
			}, nil
		},
	}

	resp, err = env.svc.VerifyCard(env.ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, VerifyCardNextStepChallengeCard, resp.NextStep)
	assert.Equal(t, 2, env.cardGateway.prepareVerifyCard.calls)

	intent, err = env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusVerifyingCard, intent.Status)

	op, err = env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.Equal(t, "gw-init-auth-retry", op.GatewayReference)
	assert.JSONEq(t, string(retryRawResp), string(op.RawResponse))
}

func TestVerifyCard_RecordsFailedOperationWhenGatewaySendFails(t *testing.T) {
	env := setupTestEnv(t)
	errGateway := errors.New("gateway unavailable")
	env.cardGateway.prepareVerifyCard.resp = &PreparedVerifyCardGatewayRequest{
		GatewayReference: "gw-init-auth-error",
		RawRequest:       []byte(`{"prepared":true}`),
		send: func(ctx context.Context) (*VerifyCardGatewayResponse, error) {
			return nil, errGateway
		},
	}

	resp, err := env.svc.VerifyCard(env.ctx, VerifyCardRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errGateway)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusFailed, op.Status)
	assert.JSONEq(t, `{"prepared":true}`, string(op.RawRequest))
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
	setupCardPayment  fakeSetupCardPaymentCall
	prepareVerifyCard fakePrepareVerifyCardCall
}

type fakeSetupCardPaymentCall struct {
	calls int
	req   SetupCardPaymentGatewayRequest
	resp  *SetupCardPaymentGatewayResponse
	err   error
}

type fakePrepareVerifyCardCall struct {
	calls int
	req   VerifyCardGatewayRequest
	resp  *PreparedVerifyCardGatewayRequest
	err   error
}

func (f *fakeCardGateway) SetupCardPayment(ctx context.Context, r SetupCardPaymentGatewayRequest) (*SetupCardPaymentGatewayResponse, error) {
	f.setupCardPayment.calls++
	f.setupCardPayment.req = r
	if f.setupCardPayment.err != nil {
		return nil, f.setupCardPayment.err
	}
	return f.setupCardPayment.resp, nil
}

func (f *fakeCardGateway) PrepareVerifyCard(ctx context.Context, r VerifyCardGatewayRequest) (*PreparedVerifyCardGatewayRequest, error) {
	f.prepareVerifyCard.calls++
	f.prepareVerifyCard.req = r
	if f.prepareVerifyCard.err != nil {
		return nil, f.prepareVerifyCard.err
	}
	return f.prepareVerifyCard.resp, nil
}
