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
	uuidProject                    = uuid.MustParse("00000000-0000-0000-0000-000000001001")
	uuidGateway                    = uuid.MustParse("00000000-0000-0000-0000-000000002001")
	uuidInvoice                    = uuid.MustParse("00000000-0000-0000-0000-000000003001")
	uuidInvoiceNoItems             = uuid.MustParse("00000000-0000-0000-0000-000000003002")
	uuidInvoicePaid                = uuid.MustParse("00000000-0000-0000-0000-000000003003")
	uuidInvoiceCancelled           = uuid.MustParse("00000000-0000-0000-0000-000000003004")
	uuidPaymentIntent              = uuid.MustParse("00000000-0000-0000-0000-000000005001")
	uuidPaymentExpired             = uuid.MustParse("00000000-0000-0000-0000-000000005002")
	uuidPaymentReadyToAuthenticate = uuid.MustParse("00000000-0000-0000-0000-000000005005")
	uuidPaymentNoSetup             = uuid.MustParse("00000000-0000-0000-0000-000000005006")
	uuidNotExist                   = uuid.MustParse("00000000-0000-0000-0000-00000000ffff")
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
		setupCardPaymentMethod: fakeSetupCardPaymentMethodCall{
			resp: &SetupCardPaymentMethodGatewayResponse{PaymentMethodReference: "session-test-123"},
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

func validCreatePaymentIntentRequest(idempotencyKey string) CreatePaymentIntentRequest {
	return CreatePaymentIntentRequest{
		InvoiceID:        uuidInvoice,
		ProjectID:        uuidProject,
		IdempotencyKey:   idempotencyKey,
		GatewayAccountID: uuidGateway,
		PaymentMethod:    PaymentMethodCard,
		PayerIP:          "192.168.1.1",
		PayerUserAgent:   "Mozilla/5.0",
	}
}

func validAuthenticateCardholderRequest() AuthenticateCardholderRequest {
	return AuthenticateCardholderRequest{
		PaymentIntentRef: PaymentIntentRef{
			ProjectID:       uuidProject,
			InvoiceID:       uuidInvoice,
			PaymentIntentID: uuidPaymentReadyToAuthenticate,
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

func assertCreatePaymentIntentErrorWithoutNewIntent(t *testing.T, env testEnv, req CreatePaymentIntentRequest, wantErr error) {
	t.Helper()

	before := countPaymentIntents(t, env)

	resp, err := env.svc.CreatePaymentIntent(env.ctx, req)

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

func TestCreatePaymentIntent_RejectsInvalidRequestBeforeSideEffects(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name     string
		modifier func(*CreatePaymentIntentRequest)
		err      error
	}{
		{
			name:     "missing_invoice_ID",
			modifier: func(r *CreatePaymentIntentRequest) { r.InvoiceID = uuid.Nil },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_project_ID",
			modifier: func(r *CreatePaymentIntentRequest) { r.ProjectID = uuid.Nil },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_idempotency_key",
			modifier: func(r *CreatePaymentIntentRequest) { r.IdempotencyKey = "" },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_gateway_account_ID",
			modifier: func(r *CreatePaymentIntentRequest) { r.GatewayAccountID = uuid.Nil },
			err:      ErrInvalidArgument,
		},
		{
			name:     "unsupported_payment_method",
			modifier: func(r *CreatePaymentIntentRequest) { r.PaymentMethod = "CASH_ON_DELIVERY" },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_payer_IP",
			modifier: func(r *CreatePaymentIntentRequest) { r.PayerIP = "" },
			err:      ErrInvalidArgument,
		},
		{
			name:     "missing_payer_user_agent",
			modifier: func(r *CreatePaymentIntentRequest) { r.PayerUserAgent = "" },
			err:      ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreatePaymentIntentRequest("key-val")
			tt.modifier(&req)

			assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, tt.err)
		})
	}
}

func TestCreatePaymentIntent_RejectsMissingInvoice(t *testing.T) {
	env := setupTestEnv(t)
	req := validCreatePaymentIntentRequest("key-missing-invoice")
	req.InvoiceID = uuidNotExist

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, ErrInvoiceNotFound)
}

func TestCreatePaymentIntent_RejectsPaidInvoice(t *testing.T) {
	env := setupTestEnv(t)
	req := validCreatePaymentIntentRequest("key-paid-invoice")
	req.InvoiceID = uuidInvoicePaid

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, ErrInvoiceAlreadyPaid)
}

func TestCreatePaymentIntent_RejectsCancelledInvoice(t *testing.T) {
	env := setupTestEnv(t)
	req := validCreatePaymentIntentRequest("key-cancelled-invoice")
	req.InvoiceID = uuidInvoiceCancelled

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, ErrInvoiceCancelled)
}

func TestCreatePaymentIntent_RejectsUnknownGatewayAccount(t *testing.T) {
	env := setupTestEnv(t)
	req := validCreatePaymentIntentRequest("key-no-gw")
	req.GatewayAccountID = uuidNotExist

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, ErrGatewayAccountNotFound)
}

func TestCreatePaymentIntent_ReplaysExistingIntent(t *testing.T) {
	env := setupTestEnv(t)
	before := countPaymentIntents(t, env)

	resp, err := env.svc.CreatePaymentIntent(env.ctx, validCreatePaymentIntentRequest("idem-created"))

	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000005001"), resp.PaymentIntentID)
	require.NotNil(t, resp.PaymentMethodReference)
	assert.Equal(t, PaymentMethodReference("session-existing-abc"), *resp.PaymentMethodReference)
	assert.Equal(t, before, countPaymentIntents(t, env))
	assert.Equal(t, 0, env.cardGateway.setupCardPaymentMethod.calls)
}

func TestCreatePaymentIntent_RejectsExistingIntentWithUnknownPaymentMethod(t *testing.T) {
	env := setupTestEnv(t)

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, validCreatePaymentIntentRequest("idem-unknown-method"), ErrPaymentIntentInvalidState)
}

func TestCreatePaymentIntent_RejectsIdempotencyMismatch(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name   string
		change func(*CreatePaymentIntentRequest)
	}{
		{
			name: "invoice",
			change: func(r *CreatePaymentIntentRequest) {
				r.IdempotencyKey = "idem-mismatch"
			},
		},
		{
			name: "gateway_account",
			change: func(r *CreatePaymentIntentRequest) {
				r.IdempotencyKey = "idem-created"
				r.GatewayAccountID = uuidNotExist
			},
		},
		{
			name: "payment_method",
			change: func(r *CreatePaymentIntentRequest) {
				r.IdempotencyKey = "idem-created"
				r.PaymentMethod = PaymentMethodApplePay
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreatePaymentIntentRequest("unused")
			tt.change(&req)

			assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, ErrIdempotencyMismatch)
		})
	}
}

func TestCreatePaymentIntent_RejectsExpiredIntent(t *testing.T) {
	env := setupTestEnv(t)
	req := validCreatePaymentIntentRequest("idem-expired")

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, ErrPaymentIntentExpired)
}

func TestCreatePaymentIntent_CreatesCardIntent(t *testing.T) {
	env := setupTestEnv(t)
	before := countPaymentIntents(t, env)

	resp, err := env.svc.CreatePaymentIntent(env.ctx, validCreatePaymentIntentRequest("key-card-success"))

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

	assert.Equal(t, 1, env.cardGateway.setupCardPaymentMethod.calls)
	assert.Equal(t, uuidInvoice, env.cardGateway.setupCardPaymentMethod.req.InvoiceID)
	assert.Equal(t, "BHD", env.cardGateway.setupCardPaymentMethod.req.Currency)
	assert.True(t, decimal.RequireFromString("15.000").Equal(env.cardGateway.setupCardPaymentMethod.req.Amount))

	require.NotNil(t, resp.PaymentMethodReference)
	assert.Equal(t, PaymentMethodReference("session-test-123"), *resp.PaymentMethodReference)
}

func TestCreatePaymentIntent_DoesNotCreateIntentWhenCardGatewayFails(t *testing.T) {
	env := setupTestEnv(t)
	errGateway := errors.New("gateway down")
	env.cardGateway.setupCardPaymentMethod.err = errGateway

	req := validCreatePaymentIntentRequest("key-gateway-fails")

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, errGateway)
	assert.Equal(t, 1, env.cardGateway.setupCardPaymentMethod.calls)
}

func TestCreatePaymentIntent_RejectsUnsupportedCardGateway(t *testing.T) {
	env := setupTestEnv(t)
	env.gatewayResolver.err = ErrUnsupportedGateway

	req := validCreatePaymentIntentRequest("key-unsupported-gateway")

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, ErrUnsupportedPaymentMethod)
	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.setupCardPaymentMethod.calls)
}

func TestCreatePaymentIntent_ReturnsGatewayResolverError(t *testing.T) {
	env := setupTestEnv(t)
	errResolver := errors.New("resolver failed")
	env.gatewayResolver.err = errResolver

	req := validCreatePaymentIntentRequest("key-resolver-error")

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, errResolver)
	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.setupCardPaymentMethod.calls)
}

func TestCreatePaymentIntent_RejectsApplePayUntilSupported(t *testing.T) {
	env := setupTestEnv(t)
	req := validCreatePaymentIntentRequest("key-apple-pay")
	req.PaymentMethod = PaymentMethodApplePay

	assertCreatePaymentIntentErrorWithoutNewIntent(t, env, req, ErrUnsupportedPaymentMethod)
}

func TestPrepareCardAuthentication_RejectsInvalidRequest(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name string
		req  PrepareCardAuthenticationRequest
		err  error
	}{
		{
			name: "missing_project_ID",
			req:  PrepareCardAuthenticationRequest{InvoiceID: uuidInvoice},
			err:  ErrInvalidArgument,
		},
		{
			name: "missing_invoice_ID",
			req:  PrepareCardAuthenticationRequest{ProjectID: uuidProject},
			err:  ErrInvalidArgument,
		},
		{
			name: "missing_payment_intent_ID",
			req:  PrepareCardAuthenticationRequest{InvoiceID: uuidInvoice, ProjectID: uuidProject},
			err:  ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := env.svc.PrepareCardAuthentication(env.ctx, tt.req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestPrepareCardAuthentication_ReturnsNotFound(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, PrepareCardAuthenticationRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidNotExist,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardAuthentication.calls)
}

func TestPrepareCardAuthentication_RejectsInvalidStateTransition(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, PrepareCardAuthenticationRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentReadyToAuthenticate,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidTransition)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardAuthentication.calls)
}

func TestPrepareCardAuthentication_RejectsExpiredIntent(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, PrepareCardAuthenticationRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentExpired,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentExpired)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardAuthentication.calls)
}

func TestPrepareCardAuthentication_RejectsMissingPaymentMethodReference(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, PrepareCardAuthenticationRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentNoSetup,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardAuthentication.calls)
}

func TestPrepareCardAuthentication_ReturnsGatewayResolverError(t *testing.T) {
	env := setupTestEnv(t)
	errResolver := errors.New("resolver failed")
	env.gatewayResolver.err = errResolver

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, PrepareCardAuthenticationRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errResolver)
	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardAuthentication.calls)

	_, err = env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	assert.Error(t, err)
}

func TestPrepareCardAuthentication_ReturnsPrepareCardAuthenticationErrorBeforeCreatingOperation(t *testing.T) {
	env := setupTestEnv(t)
	errPrepare := errors.New("prepare failed")
	env.cardGateway.prepareCardAuthentication.err = errPrepare

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, PrepareCardAuthenticationRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errPrepare)
	assert.Equal(t, 1, env.cardGateway.prepareCardAuthentication.calls)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

	_, err = env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	assert.Error(t, err)
}

func TestPrepareCardAuthentication_RecordsPendingGatewayOperationBeforeSending(t *testing.T) {
	env := setupTestEnv(t)
	rawReq := []byte(`{"prepared":true}`)
	rawResp := []byte(`{"result":"SUCCESS"}`)

	env.cardGateway.prepareCardAuthentication.resp = &PreparedCardAuthenticationGatewayRequest{
		AuthenticationReference: "gw-init-auth-123",
		RawRequest:              rawReq,
		send: func(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error) {
			intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
			require.NoError(t, err)
			assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

			op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
			require.NoError(t, err)
			assert.Equal(t, store.GatewayOperationStatusPending, op.Status)
			assert.JSONEq(t, string(rawReq), string(op.RawRequest))
			assert.Equal(t, "gw-init-auth-123", op.GatewayReference)

			return &PrepareCardAuthenticationGatewayResponse{
				NextStep:    PrepareCardAuthenticationGatewayNextStepAuthenticate,
				RawResponse: rawResp,
			}, nil
		},
	}

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, PrepareCardAuthenticationRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardAuthenticationNextStepAuthenticate, resp.NextStep)
	assert.Equal(t, 1, env.cardGateway.prepareCardAuthentication.calls)
	assert.Equal(t, uuidInvoice, env.cardGateway.prepareCardAuthentication.req.InvoiceID)
	assert.Equal(t, "BHD", env.cardGateway.prepareCardAuthentication.req.Currency)
	assert.Equal(t, PaymentMethodReference("session-existing-abc"), env.cardGateway.prepareCardAuthentication.req.PaymentMethodReference)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusReadyToAuthenticate, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.JSONEq(t, string(rawResp), string(op.RawResponse))
}

func TestPrepareCardAuthentication_RecordsCantContinueGatewayResponseAndAllowsRetry(t *testing.T) {
	env := setupTestEnv(t)
	rawResp := []byte(`{"result":"FAILURE"}`)
	req := PrepareCardAuthenticationRequest{
		ProjectID:       uuidProject,
		InvoiceID:       uuidInvoice,
		PaymentIntentID: uuidPaymentIntent,
	}

	env.cardGateway.prepareCardAuthentication.resp = &PreparedCardAuthenticationGatewayRequest{
		AuthenticationReference: "gw-init-auth-failed",
		RawRequest:              []byte(`{"prepared":true}`),
		send: func(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error) {
			return &PrepareCardAuthenticationGatewayResponse{
				NextStep:    PrepareCardAuthenticationGatewayNextStepCantContinue,
				RawResponse: rawResp,
			}, nil
		},
	}

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardAuthenticationNextStepCantContinue, resp.NextStep)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.JSONEq(t, string(rawResp), string(op.RawResponse))

	retryRawResp := []byte(`{"result":"SUCCESS"}`)
	env.cardGateway.prepareCardAuthentication.resp = &PreparedCardAuthenticationGatewayRequest{
		AuthenticationReference: "gw-init-auth-retry",
		RawRequest:              []byte(`{"prepared":"retry"}`),
		send: func(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error) {
			return &PrepareCardAuthenticationGatewayResponse{
				NextStep:    PrepareCardAuthenticationGatewayNextStepAuthenticate,
				RawResponse: retryRawResp,
			}, nil
		},
	}

	resp, err = env.svc.PrepareCardAuthentication(env.ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardAuthenticationNextStepAuthenticate, resp.NextStep)
	assert.Equal(t, 2, env.cardGateway.prepareCardAuthentication.calls)

	intent, err = env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusReadyToAuthenticate, intent.Status)

	op, err = env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.Equal(t, "gw-init-auth-retry", op.GatewayReference)
	assert.JSONEq(t, string(retryRawResp), string(op.RawResponse))
}

func TestPrepareCardAuthentication_RecordsFailedOperationWhenGatewaySendFails(t *testing.T) {
	env := setupTestEnv(t)
	errGateway := errors.New("gateway unavailable")
	env.cardGateway.prepareCardAuthentication.resp = &PreparedCardAuthenticationGatewayRequest{
		AuthenticationReference: "gw-init-auth-error",
		RawRequest:              []byte(`{"prepared":true}`),
		send: func(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error) {
			return nil, errGateway
		},
	}

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, PrepareCardAuthenticationRequest{
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

func TestAuthenticateCardholder_RejectsInvalidRequest(t *testing.T) {
	svc := New(nil, nil, nil)

	tests := []struct {
		name      string
		change    func(*AuthenticateCardholderRequest)
		errorText string
	}{
		{"missing_project_ID", func(r *AuthenticateCardholderRequest) { r.ProjectID = uuid.Nil }, "ProjectID is required"},
		{"missing_invoice_ID", func(r *AuthenticateCardholderRequest) { r.InvoiceID = uuid.Nil }, "InvoiceID is required"},
		{"missing_payment_intent_ID", func(r *AuthenticateCardholderRequest) { r.PaymentIntentID = uuid.Nil }, "PaymentIntentID is required"},
		{"missing_IP_address", func(r *AuthenticateCardholderRequest) { r.Browser.IPAddress = "" }, "IPAddress is required"},
		{"invalid_IP_address", func(r *AuthenticateCardholderRequest) { r.Browser.IPAddress = "invalid" }, "IPAddress is invalid"},
		{"missing_user_agent", func(r *AuthenticateCardholderRequest) { r.Browser.UserAgent = "" }, "UserAgent is required"},
		{"user_agent_too_long", func(r *AuthenticateCardholderRequest) { r.Browser.UserAgent = strings.Repeat("a", 2049) }, "UserAgent must not exceed 2048 characters"},
		{"missing_accept_header", func(r *AuthenticateCardholderRequest) { r.Browser.AcceptHeader = "" }, "AcceptHeader is required"},
		{"accept_header_too_long", func(r *AuthenticateCardholderRequest) { r.Browser.AcceptHeader = strings.Repeat("a", 2049) }, "AcceptHeader must not exceed 2048 characters"},
		{"unsupported_challenge_size", func(r *AuthenticateCardholderRequest) { r.Browser.ChallengeWindowSize = "100_X_100" }, "unsupported ChallengeWindowSize"},
		{"color_depth_too_small", func(r *AuthenticateCardholderRequest) { r.Browser.ColorDepth = 0 }, "ColorDepth must be between 1 and 48"},
		{"color_depth_too_large", func(r *AuthenticateCardholderRequest) { r.Browser.ColorDepth = 49 }, "ColorDepth must be between 1 and 48"},
		{"missing_language", func(r *AuthenticateCardholderRequest) { r.Browser.Language = "" }, "Language is required"},
		{"language_too_long", func(r *AuthenticateCardholderRequest) { r.Browser.Language = "zh-Hant-HK" }, "Language must not exceed 8 characters"},
		{"screen_height_too_small", func(r *AuthenticateCardholderRequest) { r.Browser.ScreenHeight = 0 }, "ScreenHeight must be between 1 and 999999"},
		{"screen_height_too_large", func(r *AuthenticateCardholderRequest) { r.Browser.ScreenHeight = 1000000 }, "ScreenHeight must be between 1 and 999999"},
		{"screen_width_too_small", func(r *AuthenticateCardholderRequest) { r.Browser.ScreenWidth = 0 }, "ScreenWidth must be between 1 and 999999"},
		{"screen_width_too_large", func(r *AuthenticateCardholderRequest) { r.Browser.ScreenWidth = 1000000 }, "ScreenWidth must be between 1 and 999999"},
		{"time_zone_too_small", func(r *AuthenticateCardholderRequest) { r.Browser.TimeZone = -841 }, "TimeZone must be between -840 and 840"},
		{"time_zone_too_large", func(r *AuthenticateCardholderRequest) { r.Browser.TimeZone = 841 }, "TimeZone must be between -840 and 840"},
		{"missing_challenge_return_URL", func(r *AuthenticateCardholderRequest) { r.ChallengeReturnURL = "" }, "ChallengeReturnURL must be an absolute HTTPS URL"},
		{"invalid_challenge_return_URL", func(r *AuthenticateCardholderRequest) { r.ChallengeReturnURL = "://invalid" }, "ChallengeReturnURL must be an absolute HTTPS URL"},
		{"non_HTTPS_challenge_return_URL", func(r *AuthenticateCardholderRequest) { r.ChallengeReturnURL = "http://pay.example.com/complete" }, "ChallengeReturnURL must be an absolute HTTPS URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validAuthenticateCardholderRequest()
			tt.change(&req)

			resp, err := svc.AuthenticateCardholder(t.Context(), req)

			assert.Nil(t, resp)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidArgument)
			assert.Contains(t, err.Error(), tt.errorText)
		})
	}
}

func TestAuthenticateCardholder_CompletesChallenge(t *testing.T) {
	env := setupTestEnv(t)
	initiateAuthOperation := createSuccessfulInitiateAuthGatewayOperation(t, env)
	rawRequest := []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`)
	rawResponse := []byte(`{"result":"PENDING"}`)
	redirectHTML := `<div id="threedsChallengeRedirect"></div>`
	env.cardGateway.authenticateCardholder.resp = &PreparedAuthenticateCardholderGatewayRequest{
		RawRequest: rawRequest,
		send: func(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error) {
			op, err := env.queries.GetLatestGatewayOperation(ctx, uuidPaymentReadyToAuthenticate)
			require.NoError(t, err)
			assert.Equal(t, store.GatewayOperationTypeAuthenticatePayer, op.OperationType)
			assert.Equal(t, store.GatewayOperationStatusPending, op.Status)

			return &AuthenticateCardholderGatewayResponse{
				NextStep:     AuthenticateCardholderGatewayNextStepChallenge,
				RedirectHTML: redirectHTML,
				RawResponse:  rawResponse,
			}, nil
		},
	}
	req := validAuthenticateCardholderRequest()

	resp, err := env.svc.AuthenticateCardholder(env.ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, AuthenticateCardholderNextStepChallenge, resp.NextStep)
	assert.Equal(t, redirectHTML, resp.RedirectHTML)
	assert.Equal(t, 1, env.cardGateway.authenticateCardholder.calls)
	assert.Equal(t, AuthenticateCardholderGatewayRequest{
		InvoiceID:               uuidInvoice,
		Amount:                  decimal.RequireFromString("15.000"),
		Currency:                "BHD",
		PaymentMethodReference:  "session-ready-to-start-challenge",
		AuthenticationReference: AuthenticationReference(initiateAuthOperation.GatewayReference),
		ChallengeReturnURL:      req.ChallengeReturnURL,
		Browser:                 req.Browser,
	}, env.cardGateway.authenticateCardholder.req)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToAuthenticate)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusAwaitingAuthenticationResult, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentReadyToAuthenticate)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationTypeAuthenticatePayer, op.OperationType)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.Equal(t, initiateAuthOperation.GatewayReference, op.GatewayReference)
	assert.JSONEq(t, string(rawRequest), string(op.RawRequest))
	assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
}

func TestAuthenticateCardholder_FrictionlessIsReadyToCapture(t *testing.T) {
	env := setupTestEnv(t)
	createSuccessfulInitiateAuthGatewayOperation(t, env)
	env.cardGateway.authenticateCardholder.resp = &PreparedAuthenticateCardholderGatewayRequest{
		RawRequest: []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`),
		send: func(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error) {
			return &AuthenticateCardholderGatewayResponse{
				NextStep:    AuthenticateCardholderGatewayNextStepCapture,
				RawResponse: []byte(`{"result":"SUCCESS"}`),
			}, nil
		},
	}

	resp, err := env.svc.AuthenticateCardholder(env.ctx, validAuthenticateCardholderRequest())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, AuthenticateCardholderNextStepCapture, resp.NextStep)
	assert.Empty(t, resp.RedirectHTML)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToAuthenticate)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusReadyToCapture, intent.Status)
}

func TestAuthenticateCardholder_CantContinueFailsPaymentIntent(t *testing.T) {
	env := setupTestEnv(t)
	createSuccessfulInitiateAuthGatewayOperation(t, env)
	rawResponse := []byte(`{"result":"FAILURE"}`)
	env.cardGateway.authenticateCardholder.resp = &PreparedAuthenticateCardholderGatewayRequest{
		RawRequest: []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`),
		send: func(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error) {
			return &AuthenticateCardholderGatewayResponse{
				NextStep:    AuthenticateCardholderGatewayNextStepCantContinue,
				RawResponse: rawResponse,
			}, nil
		},
	}

	resp, err := env.svc.AuthenticateCardholder(env.ctx, validAuthenticateCardholderRequest())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, AuthenticateCardholderNextStepCantContinue, resp.NextStep)
	assert.Empty(t, resp.RedirectHTML)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToAuthenticate)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusFailed, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentReadyToAuthenticate)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusSuccess, op.Status)
	assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
}

func TestAuthenticateCardholder_GatewayErrorKeepsPaymentIntentReadyForRetry(t *testing.T) {
	env := setupTestEnv(t)
	createSuccessfulInitiateAuthGatewayOperation(t, env)
	errGateway := errors.New("gateway unavailable")
	env.cardGateway.authenticateCardholder.resp = &PreparedAuthenticateCardholderGatewayRequest{
		RawRequest: []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`),
		send: func(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error) {
			return nil, errGateway
		},
	}

	resp, err := env.svc.AuthenticateCardholder(env.ctx, validAuthenticateCardholderRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errGateway)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToAuthenticate)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusReadyToAuthenticate, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentReadyToAuthenticate)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusFailed, op.Status)
}

func TestAuthenticateCardholder_RequiresSuccessfulInitiateAuthentication(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.AuthenticateCardholder(env.ctx, validAuthenticateCardholderRequest())

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Contains(t, err.Error(), "successful initiate authentication gateway operation is missing")
	assert.Equal(t, 0, env.cardGateway.authenticateCardholder.calls)
}

func createSuccessfulInitiateAuthGatewayOperation(t *testing.T, env testEnv) store.GatewayOperation {
	t.Helper()

	op, err := env.queries.CreateGatewayOperation(env.ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  uuidPaymentReadyToAuthenticate,
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
	setupCardPaymentMethod      fakeSetupCardPaymentMethodCall
	prepareCardAuthentication   fakePrepareCardAuthenticationCall
	authenticateCardholder      fakeAuthenticateCardholderCall
	getCardAuthenticationResult fakeGetCardAuthenticationResultCall
}

type fakeSetupCardPaymentMethodCall struct {
	calls int
	req   SetupCardPaymentMethodGatewayRequest
	resp  *SetupCardPaymentMethodGatewayResponse
	err   error
}

type fakePrepareCardAuthenticationCall struct {
	calls int
	req   PrepareCardAuthenticationGatewayRequest
	resp  *PreparedCardAuthenticationGatewayRequest
	err   error
}

type fakeAuthenticateCardholderCall struct {
	calls int
	req   AuthenticateCardholderGatewayRequest
	resp  *PreparedAuthenticateCardholderGatewayRequest
	err   error
}

type fakeGetCardAuthenticationResultCall struct {
	calls int
	req   GetCardAuthenticationResultGatewayRequest
	resp  *PreparedGetCardAuthenticationResultGatewayRequest
	err   error
}

func (f *fakeCardGateway) SetupCardPaymentMethod(ctx context.Context, r SetupCardPaymentMethodGatewayRequest) (*SetupCardPaymentMethodGatewayResponse, error) {
	f.setupCardPaymentMethod.calls++
	f.setupCardPaymentMethod.req = r
	if f.setupCardPaymentMethod.err != nil {
		return nil, f.setupCardPaymentMethod.err
	}
	return f.setupCardPaymentMethod.resp, nil
}

func (f *fakeCardGateway) PrepareCardAuthentication(ctx context.Context, r PrepareCardAuthenticationGatewayRequest) (*PreparedCardAuthenticationGatewayRequest, error) {
	f.prepareCardAuthentication.calls++
	f.prepareCardAuthentication.req = r
	if f.prepareCardAuthentication.err != nil {
		return nil, f.prepareCardAuthentication.err
	}
	return f.prepareCardAuthentication.resp, nil
}

func (f *fakeCardGateway) AuthenticateCardholder(ctx context.Context, r AuthenticateCardholderGatewayRequest) (*PreparedAuthenticateCardholderGatewayRequest, error) {
	f.authenticateCardholder.calls++
	f.authenticateCardholder.req = r
	if f.authenticateCardholder.err != nil {
		return nil, f.authenticateCardholder.err
	}
	return f.authenticateCardholder.resp, nil
}

func (f *fakeCardGateway) GetCardAuthenticationResult(ctx context.Context, r GetCardAuthenticationResultGatewayRequest) (*PreparedGetCardAuthenticationResultGatewayRequest, error) {
	f.getCardAuthenticationResult.calls++
	f.getCardAuthenticationResult.req = r
	if f.getCardAuthenticationResult.err != nil {
		return nil, f.getCardAuthenticationResult.err
	}
	return f.getCardAuthenticationResult.resp, nil
}
