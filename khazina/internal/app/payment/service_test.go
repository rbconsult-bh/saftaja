package payment

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/money"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	uuidProject                             = uuid.MustParse("00000000-0000-0000-0000-000000001001")
	uuidGateway                             = uuid.MustParse("00000000-0000-0000-0000-000000002001")
	uuidInvoice                             = uuid.MustParse("00000000-0000-0000-0000-000000003001")
	uuidInvoiceNoItems                      = uuid.MustParse("00000000-0000-0000-0000-000000003002")
	uuidInvoicePaid                         = uuid.MustParse("00000000-0000-0000-0000-000000003003")
	uuidInvoiceCancelled                    = uuid.MustParse("00000000-0000-0000-0000-000000003004")
	uuidPaymentIntent                       = uuid.MustParse("00000000-0000-0000-0000-000000005001")
	uuidPaymentExpired                      = uuid.MustParse("00000000-0000-0000-0000-000000005002")
	uuidPaymentReadyToAuthenticate          = uuid.MustParse("00000000-0000-0000-0000-000000005005")
	uuidPaymentNoSetup                      = uuid.MustParse("00000000-0000-0000-0000-000000005006")
	uuidPaymentAwaitingAuthenticationResult = uuid.MustParse("00000000-0000-0000-0000-000000005007")
	uuidPaymentReadyToCapture               = uuid.MustParse("00000000-0000-0000-0000-000000005008")
	uuidPaymentFailed                       = uuid.MustParse("00000000-0000-0000-0000-000000005009")
	uuidPaymentExpiredAwaiting              = uuid.MustParse("00000000-0000-0000-0000-000000005010")
	uuidNotExist                            = uuid.MustParse("00000000-0000-0000-0000-00000000ffff")
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

func validPrepareCardAuthenticationRequest() PrepareCardAuthenticationRequest {
	return PrepareCardAuthenticationRequest{
		PaymentIntentRef: PaymentIntentRef{
			ProjectID:       uuidProject,
			InvoiceID:       uuidInvoice,
			PaymentIntentID: uuidPaymentIntent,
		},
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

func validVerifyCardAuthenticationRequest() VerifyCardAuthenticationRequest {
	return VerifyCardAuthenticationRequest{
		PaymentIntentRef: PaymentIntentRef{
			ProjectID:       uuidProject,
			InvoiceID:       uuidInvoice,
			PaymentIntentID: uuidPaymentAwaitingAuthenticationResult,
		},
	}
}

func validCapturePaymentIntentRequest() CapturePaymentIntentRequest {
	return CapturePaymentIntentRequest{
		PaymentIntentRef: PaymentIntentRef{
			ProjectID:       uuidProject,
			InvoiceID:       uuidInvoice,
			PaymentIntentID: uuidPaymentReadyToCapture,
		},
	}
}

func countPaymentIntents(t *testing.T, env testEnv) int {
	t.Helper()

	var count int
	err := env.db.QueryRow(env.ctx, "SELECT COUNT(*) FROM payment_intents").Scan(&count)
	require.NoError(t, err)

	return count
}

func countGatewayOperations(t *testing.T, env testEnv, paymentIntentID uuid.UUID) int {
	t.Helper()

	var count int
	err := env.db.QueryRow(
		env.ctx,
		"SELECT COUNT(*) FROM gateway_operations WHERE payment_intent_id = $1",
		paymentIntentID,
	).Scan(&count)
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
	assert.Equal(t, money.CurrencyBHD, resp.Invoice.Currency)
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
	assert.Equal(t, money.MinorAmount(15000), intent.AmountMinor)
	assert.Equal(t, money.CurrencyBHD, intent.Currency)
	require.NotNil(t, intent.GatewaySetupReference)
	assert.Equal(t, "session-test-123", *intent.GatewaySetupReference)

	assert.Equal(t, 1, env.gatewayResolver.calls)
	assert.Equal(t, uuidGateway, env.gatewayResolver.account.ID)

	assert.Equal(t, 1, env.cardGateway.setupCardPaymentMethod.calls)
	assert.Equal(t, uuidInvoice, env.cardGateway.setupCardPaymentMethod.req.InvoiceID)
	assert.Equal(t, money.CurrencyBHD, env.cardGateway.setupCardPaymentMethod.req.Currency)
	assert.Equal(t, money.MinorAmount(15000), env.cardGateway.setupCardPaymentMethod.req.Amount)

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
		name   string
		change func(*PrepareCardAuthenticationRequest)
		err    error
	}{
		{
			name: "missing_project_ID",
			change: func(r *PrepareCardAuthenticationRequest) {
				r.ProjectID = uuid.Nil
			},
			err: ErrInvalidArgument,
		},
		{
			name: "missing_invoice_ID",
			change: func(r *PrepareCardAuthenticationRequest) {
				r.InvoiceID = uuid.Nil
			},
			err: ErrInvalidArgument,
		},
		{
			name: "missing_payment_intent_ID",
			change: func(r *PrepareCardAuthenticationRequest) {
				r.PaymentIntentID = uuid.Nil
			},
			err: ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validPrepareCardAuthenticationRequest()
			tt.change(&req)
			resp, err := env.svc.PrepareCardAuthentication(env.ctx, req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestPrepareCardAuthentication_ReturnsNotFound(t *testing.T) {
	env := setupTestEnv(t)

	req := validPrepareCardAuthenticationRequest()
	req.PaymentIntentID = uuidNotExist
	resp, err := env.svc.PrepareCardAuthentication(env.ctx, req)

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardAuthentication.calls)
}

func TestPrepareCardAuthentication_RejectsInvalidStateWithoutSideEffects(t *testing.T) {
	statuses := []store.PaymentIntentStatus{
		store.PaymentIntentStatusReadyToAuthenticate,
		store.PaymentIntentStatusAwaitingAuthenticationResult,
		store.PaymentIntentStatusReadyToCapture,
		store.PaymentIntentStatusCapturing,
		store.PaymentIntentStatusSucceeded,
		store.PaymentIntentStatusFailed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			env := setupTestEnv(t)
			require.NoError(t, env.queries.UpdatePaymentIntentStatus(env.ctx, store.UpdatePaymentIntentStatusParams{
				ID:     uuidPaymentIntent,
				Status: status,
			}))
			operationsBefore := countGatewayOperations(t, env, uuidPaymentIntent)

			resp, err := env.svc.PrepareCardAuthentication(env.ctx, validPrepareCardAuthenticationRequest())

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
			assert.Equal(t, 0, env.gatewayResolver.calls)
			assert.Equal(t, 0, env.cardGateway.prepareCardAuthentication.calls)
			assert.Equal(t, operationsBefore, countGatewayOperations(t, env, uuidPaymentIntent))
			intent, intentErr := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
			require.NoError(t, intentErr)
			assert.Equal(t, status, intent.Status)
		})
	}
}

func TestPrepareCardAuthentication_RejectsExpiredIntent(t *testing.T) {
	env := setupTestEnv(t)

	req := validPrepareCardAuthenticationRequest()
	req.PaymentIntentID = uuidPaymentExpired
	resp, err := env.svc.PrepareCardAuthentication(env.ctx, req)

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentExpired)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardAuthentication.calls)
}

func TestPrepareCardAuthentication_RejectsMissingPaymentMethodReference(t *testing.T) {
	env := setupTestEnv(t)

	req := validPrepareCardAuthenticationRequest()
	req.PaymentIntentID = uuidPaymentNoSetup
	resp, err := env.svc.PrepareCardAuthentication(env.ctx, req)

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Equal(t, 0, env.gatewayResolver.calls)
	assert.Equal(t, 0, env.cardGateway.prepareCardAuthentication.calls)
}

func TestPrepareCardAuthentication_ReturnsGatewayResolverError(t *testing.T) {
	env := setupTestEnv(t)
	errResolver := errors.New("resolver failed")
	env.gatewayResolver.err = errResolver

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, validPrepareCardAuthenticationRequest())

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

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, validPrepareCardAuthenticationRequest())

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
				Result:      PrepareCardAuthenticationGatewayResultAvailable,
				RawResponse: rawResp,
			}, nil
		},
	}

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, validPrepareCardAuthenticationRequest())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardAuthenticationNextStepAuthenticate, resp.NextStep)
	assert.Equal(t, 1, env.cardGateway.prepareCardAuthentication.calls)
	assert.Equal(t, uuidInvoice, env.cardGateway.prepareCardAuthentication.req.InvoiceID)
	assert.Equal(t, money.CurrencyBHD, env.cardGateway.prepareCardAuthentication.req.Currency)
	assert.Equal(t, PaymentMethodReference("session-existing-abc"), env.cardGateway.prepareCardAuthentication.req.PaymentMethodReference)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusReadyToAuthenticate, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusCompleted, op.Status)
	assert.JSONEq(t, string(rawResp), string(op.RawResponse))
}

func TestPrepareCardAuthentication_RecordsCantContinueGatewayResponseAndAllowsRetry(t *testing.T) {
	env := setupTestEnv(t)
	rawResp := []byte(`{"result":"FAILURE"}`)
	req := validPrepareCardAuthenticationRequest()

	env.cardGateway.prepareCardAuthentication.resp = &PreparedCardAuthenticationGatewayRequest{
		AuthenticationReference: "gw-init-auth-failed",
		RawRequest:              []byte(`{"prepared":true}`),
		send: func(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error) {
			return &PrepareCardAuthenticationGatewayResponse{
				Result:      PrepareCardAuthenticationGatewayResultUnavailable,
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
	assert.Equal(t, store.GatewayOperationStatusCompleted, op.Status)
	assert.JSONEq(t, string(rawResp), string(op.RawResponse))

	retryRawResp := []byte(`{"result":"SUCCESS"}`)
	env.cardGateway.prepareCardAuthentication.resp = &PreparedCardAuthenticationGatewayRequest{
		AuthenticationReference: "gw-init-auth-retry",
		RawRequest:              []byte(`{"prepared":"retry"}`),
		send: func(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error) {
			return &PrepareCardAuthenticationGatewayResponse{
				Result:      PrepareCardAuthenticationGatewayResultAvailable,
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
	assert.Equal(t, store.GatewayOperationStatusCompleted, op.Status)
	assert.Equal(t, "gw-init-auth-retry", op.GatewayReference)
	assert.JSONEq(t, string(retryRawResp), string(op.RawResponse))
}

func TestPrepareCardAuthentication_RecordsErroredOperationWhenGatewaySendFails(t *testing.T) {
	env := setupTestEnv(t)
	errGateway := errors.New("gateway unavailable")
	env.cardGateway.prepareCardAuthentication.resp = &PreparedCardAuthenticationGatewayRequest{
		AuthenticationReference: "gw-init-auth-error",
		RawRequest:              []byte(`{"prepared":true}`),
		send: func(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error) {
			return nil, errGateway
		},
	}

	resp, err := env.svc.PrepareCardAuthentication(env.ctx, validPrepareCardAuthenticationRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errGateway)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusCreated, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentIntent)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusErrored, op.Status)
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

func TestAuthenticateCardholder_RejectsInvalidStateWithoutSideEffects(t *testing.T) {
	statuses := []store.PaymentIntentStatus{
		store.PaymentIntentStatusCreated,
		store.PaymentIntentStatusAwaitingAuthenticationResult,
		store.PaymentIntentStatusReadyToCapture,
		store.PaymentIntentStatusCapturing,
		store.PaymentIntentStatusSucceeded,
		store.PaymentIntentStatusFailed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			env := setupTestEnv(t)
			require.NoError(t, env.queries.UpdatePaymentIntentStatus(env.ctx, store.UpdatePaymentIntentStatusParams{
				ID:     uuidPaymentIntent,
				Status: status,
			}))
			operationsBefore := countGatewayOperations(t, env, uuidPaymentIntent)
			req := validAuthenticateCardholderRequest()
			req.PaymentIntentID = uuidPaymentIntent

			resp, err := env.svc.AuthenticateCardholder(env.ctx, req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
			assert.Equal(t, 0, env.gatewayResolver.calls)
			assert.Equal(t, 0, env.cardGateway.authenticateCardholder.calls)
			assert.Equal(t, operationsBefore, countGatewayOperations(t, env, uuidPaymentIntent))
			intent, intentErr := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
			require.NoError(t, intentErr)
			assert.Equal(t, status, intent.Status)
		})
	}
}

func TestAuthenticateCardholder_CompletesChallenge(t *testing.T) {
	env := setupTestEnv(t)
	prepareAuthenticationOperation := createCompletedPrepareCardAuthenticationGatewayOperation(t, env)
	rawRequest := []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`)
	rawResponse := []byte(`{"result":"PENDING"}`)
	redirectHTML := `<div id="threedsChallengeRedirect"></div>`
	env.cardGateway.authenticateCardholder.resp = &PreparedAuthenticateCardholderGatewayRequest{
		RawRequest: rawRequest,
		send: func(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error) {
			op, err := env.queries.GetLatestGatewayOperation(ctx, uuidPaymentReadyToAuthenticate)
			require.NoError(t, err)
			assert.Equal(t, store.GatewayOperationTypeAuthenticateCardholder, op.OperationType)
			assert.Equal(t, store.GatewayOperationStatusPending, op.Status)

			return &AuthenticateCardholderGatewayResponse{
				Result:       AuthenticateCardholderGatewayResultChallengeRequired,
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
		Amount:                  15000,
		Currency:                money.CurrencyBHD,
		PaymentMethodReference:  "session-ready-to-start-challenge",
		AuthenticationReference: AuthenticationReference(prepareAuthenticationOperation.GatewayReference),
		ChallengeReturnURL:      req.ChallengeReturnURL,
		Browser:                 req.Browser,
	}, env.cardGateway.authenticateCardholder.req)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToAuthenticate)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusAwaitingAuthenticationResult, intent.Status)

	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentReadyToAuthenticate)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationTypeAuthenticateCardholder, op.OperationType)
	assert.Equal(t, store.GatewayOperationStatusCompleted, op.Status)
	assert.Equal(t, prepareAuthenticationOperation.GatewayReference, op.GatewayReference)
	assert.JSONEq(t, string(rawRequest), string(op.RawRequest))
	assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
}

func TestAuthenticateCardholder_FrictionlessIsReadyToCapture(t *testing.T) {
	env := setupTestEnv(t)
	createCompletedPrepareCardAuthenticationGatewayOperation(t, env)
	env.cardGateway.authenticateCardholder.resp = &PreparedAuthenticateCardholderGatewayRequest{
		RawRequest: []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`),
		send: func(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error) {
			return &AuthenticateCardholderGatewayResponse{
				Result:      AuthenticateCardholderGatewayResultSucceeded,
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
	createCompletedPrepareCardAuthenticationGatewayOperation(t, env)
	rawResponse := []byte(`{"result":"FAILURE"}`)
	env.cardGateway.authenticateCardholder.resp = &PreparedAuthenticateCardholderGatewayRequest{
		RawRequest: []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`),
		send: func(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error) {
			return &AuthenticateCardholderGatewayResponse{
				Result:      AuthenticateCardholderGatewayResultFailed,
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
	assert.Equal(t, store.GatewayOperationStatusCompleted, op.Status)
	assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
}

func TestAuthenticateCardholder_GatewayErrorKeepsPaymentIntentReadyForRetry(t *testing.T) {
	env := setupTestEnv(t)
	createCompletedPrepareCardAuthenticationGatewayOperation(t, env)
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
	assert.Equal(t, store.GatewayOperationStatusErrored, op.Status)
}

func TestAuthenticateCardholder_RequiresCompletedPrepareCardAuthentication(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.AuthenticateCardholder(env.ctx, validAuthenticateCardholderRequest())

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Contains(t, err.Error(), "completed prepare card authentication gateway operation is missing")
	assert.Equal(t, 0, env.cardGateway.authenticateCardholder.calls)
}

func createCompletedPrepareCardAuthenticationGatewayOperation(t *testing.T, env testEnv) store.GatewayOperation {
	t.Helper()

	op, err := env.queries.CreateGatewayOperation(env.ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  uuidPaymentReadyToAuthenticate,
		InvoiceID:        uuidInvoice,
		ProjectID:        uuidProject,
		GatewayAccountID: uuidGateway,
		OperationType:    store.GatewayOperationTypePrepareCardAuthentication,
		GatewayReference: "gw-init-auth-ready",
		RawRequest:       []byte(`{"apiOperation":"INITIATE_AUTHENTICATION"}`),
	})
	require.NoError(t, err)
	require.NoError(t, env.queries.UpdateGatewayOperationStatus(env.ctx, store.UpdateGatewayOperationStatusParams{
		ID:          op.ID,
		Status:      store.GatewayOperationStatusCompleted,
		RawResponse: []byte(`{"result":"SUCCESS"}`),
	}))

	return op
}

func TestVerifyCardAuthentication_RejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		req  VerifyCardAuthenticationRequest
	}{
		{
			name: "missing_project_ID",
			req: VerifyCardAuthenticationRequest{PaymentIntentRef: PaymentIntentRef{
				InvoiceID:       uuidInvoice,
				PaymentIntentID: uuidPaymentAwaitingAuthenticationResult,
			}},
		},
		{
			name: "missing_invoice_ID",
			req: VerifyCardAuthenticationRequest{PaymentIntentRef: PaymentIntentRef{
				ProjectID:       uuidProject,
				PaymentIntentID: uuidPaymentAwaitingAuthenticationResult,
			}},
		},
		{
			name: "missing_payment_intent_ID",
			req: VerifyCardAuthenticationRequest{PaymentIntentRef: PaymentIntentRef{
				ProjectID: uuidProject,
				InvoiceID: uuidInvoice,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &service{}

			resp, err := svc.VerifyCardAuthentication(t.Context(), tt.req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, ErrInvalidArgument)
		})
	}
}

func TestVerifyCardAuthentication_ReturnsNotFound(t *testing.T) {
	env := setupTestEnv(t)
	req := validVerifyCardAuthenticationRequest()
	req.PaymentIntentID = uuidNotExist

	resp, err := env.svc.VerifyCardAuthentication(env.ctx, req)

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, 0, env.cardGateway.getCardAuthenticationResult.calls)
}

func TestVerifyCardAuthentication_RejectsInvalidState(t *testing.T) {
	statuses := []store.PaymentIntentStatus{
		store.PaymentIntentStatusCreated,
		store.PaymentIntentStatusReadyToAuthenticate,
		store.PaymentIntentStatusCapturing,
		store.PaymentIntentStatusSucceeded,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			env := setupTestEnv(t)
			require.NoError(t, env.queries.UpdatePaymentIntentStatus(env.ctx, store.UpdatePaymentIntentStatusParams{
				ID:     uuidPaymentIntent,
				Status: status,
			}))
			operationsBefore := countGatewayOperations(t, env, uuidPaymentIntent)
			req := validVerifyCardAuthenticationRequest()
			req.PaymentIntentID = uuidPaymentIntent

			resp, err := env.svc.VerifyCardAuthentication(env.ctx, req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
			assert.Equal(t, 0, env.gatewayResolver.calls)
			assert.Equal(t, 0, env.cardGateway.getCardAuthenticationResult.calls)
			assert.Equal(t, operationsBefore, countGatewayOperations(t, env, uuidPaymentIntent))
			intent, intentErr := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
			require.NoError(t, intentErr)
			assert.Equal(t, status, intent.Status)
		})
	}
}

func TestVerifyCardAuthentication_RejectsExpiredAwaitingIntent(t *testing.T) {
	env := setupTestEnv(t)
	req := validVerifyCardAuthenticationRequest()
	req.PaymentIntentID = uuidPaymentExpiredAwaiting

	resp, err := env.svc.VerifyCardAuthentication(env.ctx, req)

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentExpired)
	assert.Equal(t, 0, env.cardGateway.getCardAuthenticationResult.calls)
}

func TestVerifyCardAuthentication_ReplaysTerminalVerificationDecision(t *testing.T) {
	tests := []struct {
		name            string
		paymentIntentID uuid.UUID
		want            VerifyCardAuthenticationNextStep
	}{
		{
			name:            "ready_to_capture",
			paymentIntentID: uuidPaymentReadyToCapture,
			want:            VerifyCardAuthenticationNextStepCapture,
		},
		{
			name:            "failed",
			paymentIntentID: uuidPaymentFailed,
			want:            VerifyCardAuthenticationNextStepCantContinue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			req := validVerifyCardAuthenticationRequest()
			req.PaymentIntentID = tt.paymentIntentID

			resp, err := env.svc.VerifyCardAuthentication(env.ctx, req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.want, resp.NextStep)
			assert.Equal(t, 0, env.cardGateway.getCardAuthenticationResult.calls)
		})
	}
}

func TestVerifyCardAuthentication_RequiresCompletedAuthenticateCardholderOperation(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.VerifyCardAuthentication(env.ctx, validVerifyCardAuthenticationRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Contains(t, err.Error(), "completed authenticate cardholder gateway operation is missing")
	assert.Equal(t, 0, env.cardGateway.getCardAuthenticationResult.calls)
}

func TestVerifyCardAuthentication_RequiresAuthenticationReference(t *testing.T) {
	env := setupTestEnv(t)
	createCompletedAuthenticateCardholderGatewayOperation(t, env, uuidPaymentAwaitingAuthenticationResult, "")

	resp, err := env.svc.VerifyCardAuthentication(env.ctx, validVerifyCardAuthenticationRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Contains(t, err.Error(), "authenticate cardholder gateway operation is missing gateway reference")
	assert.Equal(t, 0, env.cardGateway.getCardAuthenticationResult.calls)
}

func TestVerifyCardAuthentication_ReturnsGatewayResolverError(t *testing.T) {
	env := setupTestEnv(t)
	createCompletedAuthenticateCardholderGatewayOperation(t, env, uuidPaymentAwaitingAuthenticationResult, "auth-reference")
	env.gatewayResolver.err = errors.New("resolver failed")

	resp, err := env.svc.VerifyCardAuthentication(env.ctx, validVerifyCardAuthenticationRequest())

	assert.Nil(t, resp)
	assert.ErrorContains(t, err, "resolver failed")
	assert.Equal(t, 0, env.cardGateway.getCardAuthenticationResult.calls)
}

func TestVerifyCardAuthentication_ReturnsGatewayPreparationError(t *testing.T) {
	env := setupTestEnv(t)
	createCompletedAuthenticateCardholderGatewayOperation(t, env, uuidPaymentAwaitingAuthenticationResult, "auth-reference")
	env.cardGateway.getCardAuthenticationResult.err = errors.New("prepare failed")

	resp, err := env.svc.VerifyCardAuthentication(env.ctx, validVerifyCardAuthenticationRequest())

	assert.Nil(t, resp)
	assert.ErrorContains(t, err, "prepare failed")
	op, opErr := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentAwaitingAuthenticationResult)
	require.NoError(t, opErr)
	assert.Equal(t, store.GatewayOperationTypeAuthenticateCardholder, op.OperationType)
}

func TestVerifyCardAuthentication_PendingResultRemainsAwaiting(t *testing.T) {
	env := setupTestEnv(t)
	authenticationOperation := createCompletedAuthenticateCardholderGatewayOperation(
		t,
		env,
		uuidPaymentAwaitingAuthenticationResult,
		"auth-reference",
	)
	rawResponse := []byte(`{"result":"PENDING"}`)
	env.cardGateway.getCardAuthenticationResult.resp = &PreparedGetCardAuthenticationResultGatewayRequest{
		RawRequest: nil,
		send: func(ctx context.Context) (*GetCardAuthenticationResultGatewayResponse, error) {
			op, err := env.queries.GetLatestGatewayOperation(ctx, uuidPaymentAwaitingAuthenticationResult)
			require.NoError(t, err)
			assert.Equal(t, store.GatewayOperationTypeGetCardAuthenticationResult, op.OperationType)
			assert.Equal(t, store.GatewayOperationStatusPending, op.Status)
			assert.Equal(t, authenticationOperation.GatewayReference, op.GatewayReference)
			assert.Nil(t, op.RawRequest)

			return &GetCardAuthenticationResultGatewayResponse{
				Result:      CardAuthenticationResultPending,
				RawResponse: rawResponse,
			}, nil
		},
	}

	resp, err := env.svc.VerifyCardAuthentication(env.ctx, validVerifyCardAuthenticationRequest())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, VerifyCardAuthenticationNextStepPending, resp.NextStep)
	assert.Equal(t, 1, env.cardGateway.getCardAuthenticationResult.calls)
	assert.Equal(t, GetCardAuthenticationResultGatewayRequest{
		InvoiceID:               uuidInvoice,
		Amount:                  15000,
		Currency:                money.CurrencyBHD,
		AuthenticationReference: AuthenticationReference(authenticationOperation.GatewayReference),
	}, env.cardGateway.getCardAuthenticationResult.req)

	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentAwaitingAuthenticationResult)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusAwaitingAuthenticationResult, intent.Status)
	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentAwaitingAuthenticationResult)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusCompleted, op.Status)
	assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
}

func TestVerifyCardAuthentication_AppliesFinalResult(t *testing.T) {
	tests := []struct {
		name       string
		result     CardAuthenticationResult
		wantStep   VerifyCardAuthenticationNextStep
		wantStatus store.PaymentIntentStatus
	}{
		{
			name:       "authenticated",
			result:     CardAuthenticationResultSucceeded,
			wantStep:   VerifyCardAuthenticationNextStepCapture,
			wantStatus: store.PaymentIntentStatusReadyToCapture,
		},
		{
			name:       "cant_continue",
			result:     CardAuthenticationResultFailed,
			wantStep:   VerifyCardAuthenticationNextStepCantContinue,
			wantStatus: store.PaymentIntentStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			createCompletedAuthenticateCardholderGatewayOperation(
				t,
				env,
				uuidPaymentAwaitingAuthenticationResult,
				"auth-reference",
			)
			rawResponse := []byte(`{"result":"final"}`)
			env.cardGateway.getCardAuthenticationResult.resp = &PreparedGetCardAuthenticationResultGatewayRequest{
				send: func(ctx context.Context) (*GetCardAuthenticationResultGatewayResponse, error) {
					return &GetCardAuthenticationResultGatewayResponse{
						Result:      tt.result,
						RawResponse: rawResponse,
					}, nil
				},
			}

			resp, err := env.svc.VerifyCardAuthentication(env.ctx, validVerifyCardAuthenticationRequest())

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantStep, resp.NextStep)
			intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentAwaitingAuthenticationResult)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, intent.Status)
			op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentAwaitingAuthenticationResult)
			require.NoError(t, err)
			assert.Equal(t, store.GatewayOperationTypeGetCardAuthenticationResult, op.OperationType)
			assert.Equal(t, store.GatewayOperationStatusCompleted, op.Status)
			assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
		})
	}
}

func TestVerifyCardAuthentication_GatewayErrorKeepsIntentAwaiting(t *testing.T) {
	env := setupTestEnv(t)
	createCompletedAuthenticateCardholderGatewayOperation(t, env, uuidPaymentAwaitingAuthenticationResult, "auth-reference")
	errGateway := errors.New("gateway unavailable")
	env.cardGateway.getCardAuthenticationResult.resp = &PreparedGetCardAuthenticationResultGatewayRequest{
		send: func(ctx context.Context) (*GetCardAuthenticationResultGatewayResponse, error) {
			return nil, errGateway
		},
	}

	resp, err := env.svc.VerifyCardAuthentication(env.ctx, validVerifyCardAuthenticationRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, errGateway)
	intent, intentErr := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentAwaitingAuthenticationResult)
	require.NoError(t, intentErr)
	assert.Equal(t, store.PaymentIntentStatusAwaitingAuthenticationResult, intent.Status)
	op, opErr := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentAwaitingAuthenticationResult)
	require.NoError(t, opErr)
	assert.Equal(t, store.GatewayOperationStatusErrored, op.Status)
	assert.Nil(t, op.RawResponse)
}

func TestVerifyCardAuthentication_UnsupportedGatewayResultKeepsIntentAwaiting(t *testing.T) {
	env := setupTestEnv(t)
	createCompletedAuthenticateCardholderGatewayOperation(t, env, uuidPaymentAwaitingAuthenticationResult, "auth-reference")
	rawResponse := []byte(`{"result":"unexpected"}`)
	env.cardGateway.getCardAuthenticationResult.resp = &PreparedGetCardAuthenticationResultGatewayRequest{
		send: func(ctx context.Context) (*GetCardAuthenticationResultGatewayResponse, error) {
			return &GetCardAuthenticationResultGatewayResponse{
				Result:      "unexpected",
				RawResponse: rawResponse,
			}, nil
		},
	}

	resp, err := env.svc.VerifyCardAuthentication(env.ctx, validVerifyCardAuthenticationRequest())

	assert.Nil(t, resp)
	assert.ErrorContains(t, err, "unsupported card authentication result")
	intent, intentErr := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentAwaitingAuthenticationResult)
	require.NoError(t, intentErr)
	assert.Equal(t, store.PaymentIntentStatusAwaitingAuthenticationResult, intent.Status)
	op, opErr := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentAwaitingAuthenticationResult)
	require.NoError(t, opErr)
	assert.Equal(t, store.GatewayOperationStatusErrored, op.Status)
	assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
}

func TestCapturePaymentIntent_RejectsInvalidRequest(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.CapturePaymentIntent(env.ctx, CapturePaymentIntentRequest{})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrInvalidArgument)
	assert.Equal(t, 0, env.cardGateway.captureCardPayment.calls)
}

func TestCapturePaymentIntent_RejectsInvalidStateWithoutSideEffects(t *testing.T) {
	statuses := []store.PaymentIntentStatus{
		store.PaymentIntentStatusCreated,
		store.PaymentIntentStatusReadyToAuthenticate,
		store.PaymentIntentStatusAwaitingAuthenticationResult,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			env := setupTestEnv(t)
			require.NoError(t, env.queries.UpdatePaymentIntentStatus(env.ctx, store.UpdatePaymentIntentStatusParams{
				ID:     uuidPaymentIntent,
				Status: status,
			}))
			operationsBefore := countGatewayOperations(t, env, uuidPaymentIntent)
			req := validCapturePaymentIntentRequest()
			req.PaymentIntentID = uuidPaymentIntent

			resp, err := env.svc.CapturePaymentIntent(env.ctx, req)

			assert.Nil(t, resp)
			assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
			assert.Equal(t, 0, env.gatewayResolver.calls)
			assert.Equal(t, 0, env.cardGateway.captureCardPayment.calls)
			assert.Equal(t, operationsBefore, countGatewayOperations(t, env, uuidPaymentIntent))
			intent, intentErr := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentIntent)
			require.NoError(t, intentErr)
			assert.Equal(t, status, intent.Status)
		})
	}
}

func TestCapturePaymentIntent_ReplaysKnownStateWithoutGatewayCall(t *testing.T) {
	tests := []struct {
		name     string
		status   store.PaymentIntentStatus
		nextStep CapturePaymentIntentNextStep
	}{
		{name: "succeeded", status: store.PaymentIntentStatusSucceeded, nextStep: CapturePaymentIntentNextStepComplete},
		{name: "failed", status: store.PaymentIntentStatusFailed, nextStep: CapturePaymentIntentNextStepCantContinue},
		{name: "capturing", status: store.PaymentIntentStatusCapturing, nextStep: CapturePaymentIntentNextStepProcessing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			require.NoError(t, env.queries.UpdatePaymentIntentStatus(env.ctx, store.UpdatePaymentIntentStatusParams{
				ID:     uuidPaymentReadyToCapture,
				Status: tt.status,
			}))

			resp, err := env.svc.CapturePaymentIntent(env.ctx, validCapturePaymentIntentRequest())

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.nextStep, resp.NextStep)
			assert.Equal(t, 0, env.cardGateway.captureCardPayment.calls)
		})
	}
}

func TestCapturePaymentIntent_RejectsExpiredIntent(t *testing.T) {
	env := setupTestEnv(t)
	_, err := env.db.Exec(env.ctx, "UPDATE payment_intents SET expires_at = '2025-01-01' WHERE id = $1", uuidPaymentReadyToCapture)
	require.NoError(t, err)

	resp, err := env.svc.CapturePaymentIntent(env.ctx, validCapturePaymentIntentRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentExpired)
	assert.Equal(t, 0, env.cardGateway.captureCardPayment.calls)
}

func TestCapturePaymentIntent_RequiresPaymentMethodReference(t *testing.T) {
	env := setupTestEnv(t)
	_, err := env.db.Exec(env.ctx, "UPDATE payment_intents SET gateway_setup_reference = NULL WHERE id = $1", uuidPaymentReadyToCapture)
	require.NoError(t, err)

	resp, err := env.svc.CapturePaymentIntent(env.ctx, validCapturePaymentIntentRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Equal(t, 0, env.cardGateway.captureCardPayment.calls)
}

func TestCapturePaymentIntent_RequiresCompletedAuthentication(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := env.svc.CapturePaymentIntent(env.ctx, validCapturePaymentIntentRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrPaymentIntentInvalidState)
	assert.Equal(t, 0, env.cardGateway.captureCardPayment.calls)
}

func TestCapturePaymentIntent_BlocksAnotherCapturingIntentForInvoice(t *testing.T) {
	env := setupTestEnv(t)
	require.NoError(t, env.queries.UpdatePaymentIntentStatus(env.ctx, store.UpdatePaymentIntentStatusParams{
		ID:     uuidPaymentAwaitingAuthenticationResult,
		Status: store.PaymentIntentStatusCapturing,
	}))

	resp, err := env.svc.CapturePaymentIntent(env.ctx, validCapturePaymentIntentRequest())

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrInvoicePaymentInProgress)
	assert.Equal(t, 0, env.cardGateway.captureCardPayment.calls)
}

func TestCapturePaymentIntent_ReturnsGatewayPreparationError(t *testing.T) {
	env := setupTestEnv(t)
	createCompletedAuthenticateCardholderGatewayOperation(t, env, uuidPaymentReadyToCapture, "auth-reference")
	env.cardGateway.captureCardPayment.err = errors.New("prepare failed")

	resp, err := env.svc.CapturePaymentIntent(env.ctx, validCapturePaymentIntentRequest())

	assert.Nil(t, resp)
	assert.ErrorContains(t, err, "failed to prepare capture card payment")
	intent, intentErr := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToCapture)
	require.NoError(t, intentErr)
	assert.Equal(t, store.PaymentIntentStatusReadyToCapture, intent.Status)
}

func TestCapturePaymentIntent_AppliesGatewayResult(t *testing.T) {
	tests := []struct {
		name          string
		gatewayResult CaptureCardPaymentGatewayResult
		nextStep      CapturePaymentIntentNextStep
		intentStatus  store.PaymentIntentStatus
		invoiceStatus store.InvoiceStatus
	}{
		{
			name:          "succeeded",
			gatewayResult: CaptureCardPaymentGatewayResultSucceeded,
			nextStep:      CapturePaymentIntentNextStepComplete,
			intentStatus:  store.PaymentIntentStatusSucceeded,
			invoiceStatus: store.InvoiceStatusPaid,
		},
		{
			name:          "declined",
			gatewayResult: CaptureCardPaymentGatewayResultDeclined,
			nextStep:      CapturePaymentIntentNextStepCantContinue,
			intentStatus:  store.PaymentIntentStatusFailed,
			invoiceStatus: store.InvoiceStatusPending,
		},
		{
			name:          "pending",
			gatewayResult: CaptureCardPaymentGatewayResultPending,
			nextStep:      CapturePaymentIntentNextStepProcessing,
			intentStatus:  store.PaymentIntentStatusCapturing,
			invoiceStatus: store.InvoiceStatusPending,
		},
		{
			name:          "unknown",
			gatewayResult: CaptureCardPaymentGatewayResultUnknown,
			nextStep:      CapturePaymentIntentNextStepProcessing,
			intentStatus:  store.PaymentIntentStatusCapturing,
			invoiceStatus: store.InvoiceStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			authenticationOperation := createCompletedAuthenticateCardholderGatewayOperation(
				t,
				env,
				uuidPaymentReadyToCapture,
				"auth-reference",
			)
			rawRequest := []byte(`{"apiOperation":"PAY"}`)
			rawResponse := []byte(`{"result":"test"}`)
			env.cardGateway.captureCardPayment.resp = &PreparedCaptureCardPaymentGatewayRequest{
				RawRequest: rawRequest,
				send: func(ctx context.Context) (*CaptureCardPaymentGatewayResponse, error) {
					intent, err := env.queries.GetPaymentIntentByID(ctx, uuidPaymentReadyToCapture)
					require.NoError(t, err)
					assert.Equal(t, store.PaymentIntentStatusCapturing, intent.Status)

					op, err := env.queries.GetLatestGatewayOperation(ctx, uuidPaymentReadyToCapture)
					require.NoError(t, err)
					assert.Equal(t, store.GatewayOperationTypeCapturePayment, op.OperationType)
					assert.Equal(t, store.GatewayOperationStatusPending, op.Status)
					assert.JSONEq(t, string(rawRequest), string(op.RawRequest))

					return &CaptureCardPaymentGatewayResponse{
						Result:      tt.gatewayResult,
						RawResponse: rawResponse,
					}, nil
				},
			}

			resp, err := env.svc.CapturePaymentIntent(env.ctx, validCapturePaymentIntentRequest())

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.nextStep, resp.NextStep)
			assert.Equal(t, 1, env.cardGateway.captureCardPayment.calls)
			assert.Equal(t, CaptureCardPaymentGatewayRequest{
				InvoiceID:               uuidInvoice,
				PaymentReference:        uuidPaymentReadyToCapture,
				Amount:                  15000,
				Currency:                money.CurrencyBHD,
				PaymentMethodReference:  "session-ready-to-capture",
				AuthenticationReference: AuthenticationReference(authenticationOperation.GatewayReference),
			}, env.cardGateway.captureCardPayment.req)

			intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToCapture)
			require.NoError(t, err)
			assert.Equal(t, tt.intentStatus, intent.Status)
			invoice, err := env.queries.GetInvoiceByID(env.ctx, uuidInvoice)
			require.NoError(t, err)
			assert.Equal(t, tt.invoiceStatus, invoice.Status)
			if tt.invoiceStatus == store.InvoiceStatusPaid {
				assert.NotNil(t, invoice.PaidAt)
			} else {
				assert.Nil(t, invoice.PaidAt)
			}
			op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentReadyToCapture)
			require.NoError(t, err)
			assert.Equal(t, store.GatewayOperationStatusCompleted, op.Status)
			assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
		})
	}
}

func TestCapturePaymentIntent_GatewayErrorRemainsProcessing(t *testing.T) {
	env := setupTestEnv(t)
	createCompletedAuthenticateCardholderGatewayOperation(t, env, uuidPaymentReadyToCapture, "auth-reference")
	rawResponse := []byte(`{"result":"SUCCESS"}`)
	env.cardGateway.captureCardPayment.resp = &PreparedCaptureCardPaymentGatewayRequest{
		RawRequest: []byte(`{"apiOperation":"PAY"}`),
		send: func(ctx context.Context) (*CaptureCardPaymentGatewayResponse, error) {
			return nil, &GatewayResponseError{
				Err:         ErrGatewayResponseMismatch,
				RawResponse: rawResponse,
			}
		},
	}

	resp, err := env.svc.CapturePaymentIntent(env.ctx, validCapturePaymentIntentRequest())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, CapturePaymentIntentNextStepProcessing, resp.NextStep)
	intent, err := env.queries.GetPaymentIntentByID(env.ctx, uuidPaymentReadyToCapture)
	require.NoError(t, err)
	assert.Equal(t, store.PaymentIntentStatusCapturing, intent.Status)
	invoice, err := env.queries.GetInvoiceByID(env.ctx, uuidInvoice)
	require.NoError(t, err)
	assert.Equal(t, store.InvoiceStatusPending, invoice.Status)
	op, err := env.queries.GetLatestGatewayOperation(env.ctx, uuidPaymentReadyToCapture)
	require.NoError(t, err)
	assert.Equal(t, store.GatewayOperationStatusErrored, op.Status)
	assert.JSONEq(t, string(rawResponse), string(op.RawResponse))
}

func createCompletedAuthenticateCardholderGatewayOperation(
	t *testing.T,
	env testEnv,
	paymentIntentID uuid.UUID,
	gatewayReference string,
) store.GatewayOperation {
	t.Helper()

	op, err := env.queries.CreateGatewayOperation(env.ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  paymentIntentID,
		InvoiceID:        uuidInvoice,
		ProjectID:        uuidProject,
		GatewayAccountID: uuidGateway,
		OperationType:    store.GatewayOperationTypeAuthenticateCardholder,
		GatewayReference: gatewayReference,
		RawRequest:       []byte(`{"apiOperation":"AUTHENTICATE_PAYER"}`),
	})
	require.NoError(t, err)
	require.NoError(t, env.queries.UpdateGatewayOperationStatus(env.ctx, store.UpdateGatewayOperationStatusParams{
		ID:          op.ID,
		Status:      store.GatewayOperationStatusCompleted,
		RawResponse: []byte(`{"result":"PENDING"}`),
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
	captureCardPayment          fakeCaptureCardPaymentCall
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

type fakeCaptureCardPaymentCall struct {
	calls int
	req   CaptureCardPaymentGatewayRequest
	resp  *PreparedCaptureCardPaymentGatewayRequest
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

func (f *fakeCardGateway) CaptureCardPayment(ctx context.Context, r CaptureCardPaymentGatewayRequest) (*PreparedCaptureCardPaymentGatewayRequest, error) {
	f.captureCardPayment.calls++
	f.captureCardPayment.req = r
	if f.captureCardPayment.err != nil {
		return nil, f.captureCardPayment.err
	}
	return f.captureCardPayment.resp, nil
}
