package billing

import (
	"context"
	"errors"
	"strings"
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
	uuidProject                      = uuid.MustParse("00000000-0000-0000-0000-000000001001")
	uuidGateway                      = uuid.MustParse("00000000-0000-0000-0000-000000002001")
	uuidInvoice                      = uuid.MustParse("00000000-0000-0000-0000-000000003001")
	uuidInvoiceNoItems               = uuid.MustParse("00000000-0000-0000-0000-000000003002")
	uuidInvoicePaid                  = uuid.MustParse("00000000-0000-0000-0000-000000003003")
	uuidInvoiceCancelled             = uuid.MustParse("00000000-0000-0000-0000-000000003004")
	uuidPaymentIntent                = uuid.MustParse("00000000-0000-0000-0000-000000005001")
	uuidPaymentExpired               = uuid.MustParse("00000000-0000-0000-0000-000000005002")
	uuidPaymentReadyToStartChallenge = uuid.MustParse("00000000-0000-0000-0000-000000005005")
	uuidPaymentNoSetup               = uuid.MustParse("00000000-0000-0000-0000-000000005006")
	uuidNotExist                     = uuid.MustParse("00000000-0000-0000-0000-00000000ffff")
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
			resp: &SetupCardPaymentGatewayResponse{PaymentMethodReference: "session-test-123"},
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

func validStartCardChallengeRequest() StartCardChallengeRequest {
	return StartCardChallengeRequest{
		PaymentIntentRef: PaymentIntentRef{
			ProjectID:       uuidProject,
			InvoiceID:       uuidInvoice,
			PaymentIntentID: uuidPaymentReadyToStartChallenge,
		},
		Browser: ThreeDSBrowser{
			IPAddress:           "192.0.2.1",
			UserAgent:           "Mozilla/5.0",
			AcceptHeader:        "text/html,application/xhtml+xml",
			ChallengeWindowSize: ThreeDSChallengeWindowSizeFullScreen,
			ColorDepth:          24,
			JavaEnabled:         false,
			Language:            "en-US",
			ScreenHeight:        1080,
			ScreenWidth:         1920,
			TimeZone:            -180,
		},
		ChallengeReturnURL: "https://pay.example.com/checkout/complete-challenge",
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
	require.NotNil(t, resp.PaymentMethodReference)
	assert.Equal(t, PaymentMethodReference("session-existing-abc"), *resp.PaymentMethodReference)
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
	require.NotNil(t, resp.PaymentMethodReference)
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

	require.NotNil(t, resp.PaymentMethodReference)
	assert.Equal(t, PaymentMethodReference("session-test-123"), *resp.PaymentMethodReference)
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

func TestPrepareCardChallenge_RejectsInvalidRequest(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name string
		req  PrepareCardChallengeRequest
		err  error
	}{
		{
			name: "missing_project_ID",
			req:  PrepareCardChallengeRequest{InvoiceID: uuidInvoice},
			err:  ErrInvalidArgument,
		},
		{
			name: "missing_invoice_ID",
			req:  PrepareCardChallengeRequest{ProjectID: uuidProject},
			err:  ErrInvalidArgument,
		},
		{
			name: "missing_payment_intent_ID",
			req:  PrepareCardChallengeRequest{InvoiceID: uuidInvoice, ProjectID: uuidProject},
			err:  ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := env.svc.PrepareCardChallenge(env.ctx, tt.req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestPrepareCardChallenge_ReturnsNotFound(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.PrepareCardChallenge(env.ctx, PrepareCardChallengeRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidNotExist,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardChallenge.calls)
}

func TestPrepareCardChallenge_RejectsInvalidStateTransition(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.PrepareCardChallenge(env.ctx, PrepareCardChallengeRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentReadyToStartChallenge,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidTransition)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardChallenge.calls)
}

func TestPrepareCardChallenge_RejectsExpiredIntent(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.PrepareCardChallenge(env.ctx, PrepareCardChallengeRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentExpired,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentExpired)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardChallenge.calls)
}

func TestPrepareCardChallenge_RejectsMissingPaymentMethodReference(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.PrepareCardChallenge(env.ctx, PrepareCardChallengeRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentNoSetup,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardChallenge.calls)
}

func TestPrepareCardChallenge_ReturnsGatewayResolverError(t *testing.T) {
	env := setupTestEnv(t)
	errResolver := errors.New("resolver failed")
	env.gatewayResolver.err = errResolver

	resp, err := env.svc.PrepareCardChallenge(env.ctx, PrepareCardChallengeRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errResolver)
	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardChallenge.calls)

	_, err = env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	assert.Error(t, err)
}

func TestPrepareCardChallenge_ReturnsPrepareCardChallengeErrorBeforeCreatingOperation(t *testing.T) {
	env := setupTestEnv(t)
	errPrepare := errors.New("prepare failed")
	env.cardGateway.prepareCardChallenge.err = errPrepare

	resp, err := env.svc.PrepareCardChallenge(env.ctx, PrepareCardChallengeRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errPrepare)
	assert.Equal(t, 1, env.cardGateway.prepareCardChallenge.calls)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

	_, err = env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	assert.Error(t, err)
}

func TestPrepareCardChallenge_RecordsPendingGatewayOperationBeforeSending(t *testing.T) {
	env := setupTestEnv(t)
	rawReq := []byte(`{"prepared":true}`)
	rawResp := []byte(`{"result":"SUCCESS"}`)

	env.cardGateway.prepareCardChallenge.resp = &PreparedCardChallengeGatewayRequest{
		AuthenticationReference: "gw-init-auth-123",
		RawRequest:              rawReq,
		send: func(ctx context.Context) (*PrepareCardChallengeGatewayResponse, error) {
			intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
			require.NoError(t, err)
			assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

			op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
			require.NoError(t, err)
			assert.Equal(t, store.GatewayOperationStatusPending, op.Status)
			assert.JSONEq(t, string(rawReq), string(op.RawRequest))
			assert.Equal(t, "gw-init-auth-123", op.GatewayReference)

			return &PrepareCardChallengeGatewayResponse{
				NextStep:    PrepareCardChallengeGatewayNextStepStartChallenge,
				RawResponse: rawResp,
			}, nil
		},
	}

	resp, err := env.svc.PrepareCardChallenge(env.ctx, PrepareCardChallengeRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardChallengeNextStepStartChallenge, resp.NextStep)
	assert.Equal(t, 1, env.cardGateway.prepareCardChallenge.calls)
	assert.Equal(t, uuidInvoice, env.cardGateway.prepareCardChallenge.req.InvoiceID)
	assert.Equal(t, "BHD", env.cardGateway.prepareCardChallenge.req.Currency)
	assert.Equal(t, PaymentMethodReference("session-existing-abc"), env.cardGateway.prepareCardChallenge.req.PaymentMethodReference)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusReadyToStartChallenge, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.JSONEq(t, string(rawResp), string(op.RawResponse))
}

func TestPrepareCardChallenge_RecordsCantContinueGatewayResponseAndAllowsRetry(t *testing.T) {
	env := setupTestEnv(t)
	rawResp := []byte(`{"result":"FAILURE"}`)
	req := PrepareCardChallengeRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	}

	env.cardGateway.prepareCardChallenge.resp = &PreparedCardChallengeGatewayRequest{
		AuthenticationReference: "gw-init-auth-failed",
		RawRequest:              []byte(`{"prepared":true}`),
		send: func(ctx context.Context) (*PrepareCardChallengeGatewayResponse, error) {
			return &PrepareCardChallengeGatewayResponse{
				NextStep:    PrepareCardChallengeGatewayNextStepCantContinue,
				RawResponse: rawResp,
			}, nil
		},
	}

	resp, err := env.svc.PrepareCardChallenge(env.ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardChallengeNextStepCantContinue, resp.NextStep)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.JSONEq(t, string(rawResp), string(op.RawResponse))

	retryRawResp := []byte(`{"result":"SUCCESS"}`)
	env.cardGateway.prepareCardChallenge.resp = &PreparedCardChallengeGatewayRequest{
		AuthenticationReference: "gw-init-auth-retry",
		RawRequest:              []byte(`{"prepared":"retry"}`),
		send: func(ctx context.Context) (*PrepareCardChallengeGatewayResponse, error) {
			return &PrepareCardChallengeGatewayResponse{
				NextStep:    PrepareCardChallengeGatewayNextStepStartChallenge,
				RawResponse: retryRawResp,
			}, nil
		},
	}

	resp, err = env.svc.PrepareCardChallenge(env.ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardChallengeNextStepStartChallenge, resp.NextStep)
	assert.Equal(t, 2, env.cardGateway.prepareCardChallenge.calls)

	intent, err = env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusReadyToStartChallenge, intent.Status)

	op, err = env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.Equal(t, "gw-init-auth-retry", op.GatewayReference)
	assert.JSONEq(t, string(retryRawResp), string(op.RawResponse))
}

func TestPrepareCardChallenge_RecordsFailedOperationWhenGatewaySendFails(t *testing.T) {
	env := setupTestEnv(t)
	errGateway := errors.New("gateway unavailable")
	env.cardGateway.prepareCardChallenge.resp = &PreparedCardChallengeGatewayRequest{
		AuthenticationReference: "gw-init-auth-error",
		RawRequest:              []byte(`{"prepared":true}`),
		send: func(ctx context.Context) (*PrepareCardChallengeGatewayResponse, error) {
			return nil, errGateway
		},
	}

	resp, err := env.svc.PrepareCardChallenge(env.ctx, PrepareCardChallengeRequest{
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

func TestStartCardChallenge_RejectsInvalidRequest(t *testing.T) {
	svc := New(nil, nil, nil)

	tests := []struct {
		name      string
		change    func(*StartCardChallengeRequest)
		errorText string
	}{
		{"missing_project_ID", func(r *StartCardChallengeRequest) { r.ProjectID = uuid.Nil }, "ProjectID is required"},
		{"missing_invoice_ID", func(r *StartCardChallengeRequest) { r.InvoiceID = uuid.Nil }, "InvoiceID is required"},
		{"missing_payment_intent_ID", func(r *StartCardChallengeRequest) { r.PaymentIntentID = uuid.Nil }, "PaymentIntentID is required"},
		{"missing_IP_address", func(r *StartCardChallengeRequest) { r.Browser.IPAddress = "" }, "IPAddress is required"},
		{"invalid_IP_address", func(r *StartCardChallengeRequest) { r.Browser.IPAddress = "invalid" }, "IPAddress is invalid"},
		{"missing_user_agent", func(r *StartCardChallengeRequest) { r.Browser.UserAgent = "" }, "UserAgent is required"},
		{"user_agent_too_long", func(r *StartCardChallengeRequest) { r.Browser.UserAgent = strings.Repeat("a", 2049) }, "UserAgent must not exceed 2048 characters"},
		{"missing_accept_header", func(r *StartCardChallengeRequest) { r.Browser.AcceptHeader = "" }, "AcceptHeader is required"},
		{"accept_header_too_long", func(r *StartCardChallengeRequest) { r.Browser.AcceptHeader = strings.Repeat("a", 2049) }, "AcceptHeader must not exceed 2048 characters"},
		{"unsupported_challenge_size", func(r *StartCardChallengeRequest) { r.Browser.ChallengeWindowSize = "100_X_100" }, "unsupported ChallengeWindowSize"},
		{"color_depth_too_small", func(r *StartCardChallengeRequest) { r.Browser.ColorDepth = 0 }, "ColorDepth must be between 1 and 48"},
		{"color_depth_too_large", func(r *StartCardChallengeRequest) { r.Browser.ColorDepth = 49 }, "ColorDepth must be between 1 and 48"},
		{"missing_language", func(r *StartCardChallengeRequest) { r.Browser.Language = "" }, "Language is required"},
		{"language_too_long", func(r *StartCardChallengeRequest) { r.Browser.Language = "zh-Hant-HK" }, "Language must not exceed 8 characters"},
		{"screen_height_too_small", func(r *StartCardChallengeRequest) { r.Browser.ScreenHeight = 0 }, "ScreenHeight must be between 1 and 999999"},
		{"screen_height_too_large", func(r *StartCardChallengeRequest) { r.Browser.ScreenHeight = 1000000 }, "ScreenHeight must be between 1 and 999999"},
		{"screen_width_too_small", func(r *StartCardChallengeRequest) { r.Browser.ScreenWidth = 0 }, "ScreenWidth must be between 1 and 999999"},
		{"screen_width_too_large", func(r *StartCardChallengeRequest) { r.Browser.ScreenWidth = 1000000 }, "ScreenWidth must be between 1 and 999999"},
		{"time_zone_too_small", func(r *StartCardChallengeRequest) { r.Browser.TimeZone = -841 }, "TimeZone must be between -840 and 840"},
		{"time_zone_too_large", func(r *StartCardChallengeRequest) { r.Browser.TimeZone = 841 }, "TimeZone must be between -840 and 840"},
		{"missing_challenge_return_URL", func(r *StartCardChallengeRequest) { r.ChallengeReturnURL = "" }, "ChallengeReturnURL must be an absolute HTTPS URL"},
		{"invalid_challenge_return_URL", func(r *StartCardChallengeRequest) { r.ChallengeReturnURL = "://invalid" }, "ChallengeReturnURL must be an absolute HTTPS URL"},
		{"non_HTTPS_challenge_return_URL", func(r *StartCardChallengeRequest) { r.ChallengeReturnURL = "http://pay.example.com/complete" }, "ChallengeReturnURL must be an absolute HTTPS URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validStartCardChallengeRequest()
			tt.change(&req)

			resp, err := svc.StartCardChallenge(t.Context(), req)

			assert.Nil(t, resp)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidArgument)
			assert.Contains(t, err.Error(), tt.errorText)
		})
	}
}

func TestStartCardChallenge_CompletesChallenge(t *testing.T) {
	env := setupTestEnv(t)
	initiateAuthOperation := createSuccessfulInitiateAuthGatewayOperation(t, env)
	rawRequest := []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`)
	rawResponse := []byte(`{"result":"PENDING"}`)
	redirectHTML := `<div id="threedsChallengeRedirect"></div>`
	env.cardGateway.startCardChallenge.resp = &PreparedStartCardChallengeGatewayRequest{
		RawRequest: rawRequest,
		send: func(ctx context.Context) (*StartCardChallengeGatewayResponse, error) {
			op, err := env.queries.GetLatestGatewayOperation(ctx, uuidPaymentReadyToStartChallenge)
			require.NoError(t, err)
			assert.Equal(t, store.GatewayOperationTypeAuthenticatePayer, op.OperationType)
			assert.Equal(t, store.GatewayOperationStatusPending, op.Status)

			return &StartCardChallengeGatewayResponse{
				NextStep:     StartCardChallengeGatewayNextStepCompleteChallenge,
				RedirectHTML: redirectHTML,
				RawResponse:  rawResponse,
			}, nil
		},
	}
	req := validStartCardChallengeRequest()

	resp, err := env.svc.StartCardChallenge(env.ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, StartCardChallengeNextStepCompleteChallenge, resp.NextStep)
	assert.Equal(t, redirectHTML, resp.RedirectHTML)
	assert.Equal(t, 1, env.cardGateway.startCardChallenge.calls)
	assert.Equal(t, StartCardChallengeGatewayRequest{
		InvoiceID:               uuidInvoice,
		Amount:                  decimal.RequireFromString("15.000"),
		Currency:                "BHD",
		PaymentMethodReference:  "session-ready-to-start-challenge",
		AuthenticationReference: AuthenticationReference(initiateAuthOperation.GatewayReference),
		ChallengeReturnURL:      req.ChallengeReturnURL,
		Browser:                 req.Browser,
	}, env.cardGateway.startCardChallenge.req)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToStartChallenge)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusAwaitingChallengeCompletion, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentReadyToStartChallenge)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationTypeAuthenticatePayer, op.OperationType)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.Equal(t, initiateAuthOperation.GatewayReference, op.GatewayReference)
	assert.JSONEq(t, string(rawRequest), string(op.RawRequest))
	assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
}

func TestStartCardChallenge_FrictionlessIsReadyToCapture(t *testing.T) {
	env := setupTestEnv(t)
	createSuccessfulInitiateAuthGatewayOperation(t, env)
	env.cardGateway.startCardChallenge.resp = &PreparedStartCardChallengeGatewayRequest{
		RawRequest: []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`),
		send: func(ctx context.Context) (*StartCardChallengeGatewayResponse, error) {
			return &StartCardChallengeGatewayResponse{
				NextStep:    StartCardChallengeGatewayNextStepCapture,
				RawResponse: []byte(`{"result":"SUCCESS"}`),
			}, nil
		},
	}

	resp, err := env.svc.StartCardChallenge(env.ctx, validStartCardChallengeRequest())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, StartCardChallengeNextStepCapture, resp.NextStep)
	assert.Empty(t, resp.RedirectHTML)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToStartChallenge)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusReadyToCapture, intent.Status)
}

func TestStartCardChallenge_CantContinueFailsPaymentIntent(t *testing.T) {
	env := setupTestEnv(t)
	createSuccessfulInitiateAuthGatewayOperation(t, env)
	rawResponse := []byte(`{"result":"FAILURE"}`)
	env.cardGateway.startCardChallenge.resp = &PreparedStartCardChallengeGatewayRequest{
		RawRequest: []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`),
		send: func(ctx context.Context) (*StartCardChallengeGatewayResponse, error) {
			return &StartCardChallengeGatewayResponse{
				NextStep:    StartCardChallengeGatewayNextStepCantContinue,
				RawResponse: rawResponse,
			}, nil
		},
	}

	resp, err := env.svc.StartCardChallenge(env.ctx, validStartCardChallengeRequest())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, StartCardChallengeNextStepCantContinue, resp.NextStep)
	assert.Empty(t, resp.RedirectHTML)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToStartChallenge)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusFailed, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentReadyToStartChallenge)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
}

func TestStartCardChallenge_GatewayErrorKeepsPaymentIntentReadyForRetry(t *testing.T) {
	env := setupTestEnv(t)
	createSuccessfulInitiateAuthGatewayOperation(t, env)
	errGateway := errors.New("gateway unavailable")
	env.cardGateway.startCardChallenge.resp = &PreparedStartCardChallengeGatewayRequest{
		RawRequest: []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`),
		send: func(ctx context.Context) (*StartCardChallengeGatewayResponse, error) {
			return nil, errGateway
		},
	}

	resp, err := env.svc.StartCardChallenge(env.ctx, validStartCardChallengeRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errGateway)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToStartChallenge)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusReadyToStartChallenge, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentReadyToStartChallenge)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusFailed, op.Status)
}

func TestStartCardChallenge_RequiresSuccessfulInitiateAuthentication(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.StartCardChallenge(env.ctx, validStartCardChallengeRequest())

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Contains(t, err.Error(), "successful initiate authentication gateway operation is missing")
	assert.Equal(t, 0, env.cardGateway.startCardChallenge.calls)
}

func createSuccessfulInitiateAuthGatewayOperation(t *testing.T, env testEnv) store.GatewayOperation {
	t.Helper()

	op, err := env.queries.CreateGatewayOperation(env.ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  uuidPaymentReadyToStartChallenge,
		InvoiceID:        uuidInvoice,
		ProjectID:        uuidProject,
		GatewayAccountID: uuidGateway,
		OperationType:    store.GatewayOperationTypeInitiateAuth,
		GatewayReference: "gw-init-auth-ready",
		Amount:           decimal.RequireFromString("15.000"),
		Currency:         "BHD",
		RawRequest:       []byte(`{"apiOperation":"INITIATE_AUTHENTICATION"}`),
	})
	require.NoError(t, err)
	require.NoError(t, env.queries.UpdateGatewayOperationStatus(env.ctx, store.UpdateGatewayOperationStatusParams{
		ID:          op.ID,
		Status:      store.GatewayOperationStatusSuccess,
		RawResponse: []byte(`{"result":"SUCCESS"}`),
	}))

	return op
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
	setupCardPayment     fakeSetupCardPaymentCall
	prepareCardChallenge fakePrepareCardChallengeCall
	startCardChallenge   fakeStartCardChallengeCall
}

type fakeSetupCardPaymentCall struct {
	calls int
	req   SetupCardPaymentGatewayRequest
	resp  *SetupCardPaymentGatewayResponse
	err   error
}

type fakePrepareCardChallengeCall struct {
	calls int
	req   PrepareCardChallengeGatewayRequest
	resp  *PreparedCardChallengeGatewayRequest
	err   error
}

type fakeStartCardChallengeCall struct {
	calls int
	req   StartCardChallengeGatewayRequest
	resp  *PreparedStartCardChallengeGatewayRequest
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

func (f *fakeCardGateway) PrepareCardChallenge(ctx context.Context, r PrepareCardChallengeGatewayRequest) (*PreparedCardChallengeGatewayRequest, error) {
	f.prepareCardChallenge.calls++
	f.prepareCardChallenge.req = r
	if f.prepareCardChallenge.err != nil {
		return nil, f.prepareCardChallenge.err
	}
	return f.prepareCardChallenge.resp, nil
}

func (f *fakeCardGateway) StartCardChallenge(ctx context.Context, r StartCardChallengeGatewayRequest) (*PreparedStartCardChallengeGatewayRequest, error) {
	f.startCardChallenge.calls++
	f.startCardChallenge.req = r
	if f.startCardChallenge.err != nil {
		return nil, f.startCardChallenge.err
	}
	return f.startCardChallenge.resp, nil
}
