package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	mpgsmocks "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs/mocks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMPGSCardGateway_SetupCardPaymentMethod_Succeeds(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")

	mpgsClient := mpgsmocks.NewMockClient(t)
	mpgsClient.EXPECT().
		CreateSession(mock.Anything, mock.MatchedBy(func(req *mpgsclient.CreateSessionRequest) bool {
			return req.Session != nil &&
				req.Session.AuthenticationLimit != nil &&
				*req.Session.AuthenticationLimit == 25
		})).
		Return(&mpgsclient.Response[mpgsclient.CreateSessionResponse]{
			Data: mpgsclient.CreateSessionResponse{
				Session: &mpgsclient.SessionDetails{ID: "SESSION123"},
			},
		}, nil)

	mpgsClient.EXPECT().
		UpdateSession(mock.Anything, "SESSION123", mock.MatchedBy(func(req *mpgsclient.UpdateSessionRequest) bool {
			return req.Order.Amount == "15" &&
				req.Order.Currency == "BHD" &&
				req.Order.ID == invoiceID.String()
		})).
		Return(&mpgsclient.Response[mpgsclient.UpdateSessionResponse]{}, nil)

	gateway := &mpgsCardGateway{client: mpgsClient}
	resp, err := gateway.SetupCardPaymentMethod(ctx, SetupCardPaymentMethodGatewayRequest{
		InvoiceID: invoiceID,
		Amount:    decimal.RequireFromString("15.000"),
		Currency:  "BHD",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PaymentMethodReference("SESSION123"), resp.PaymentMethodReference)
}

func TestMPGSCardGateway_SetupCardPaymentMethod_CreateSessionError(t *testing.T) {
	mpgsClient := mpgsmocks.NewMockClient(t)
	mpgsClient.EXPECT().
		CreateSession(mock.Anything, mock.Anything).
		Return(nil, errors.New("create failed"))

	gateway := &mpgsCardGateway{client: mpgsClient}
	resp, err := gateway.SetupCardPaymentMethod(context.Background(), validSetupCardPaymentMethodGatewayRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create mpgs session")
	assert.Nil(t, resp)
}

func TestMPGSCardGateway_SetupCardPaymentMethod_MissingSetupReference(t *testing.T) {
	tests := []struct {
		name string
		resp *mpgsclient.Response[mpgsclient.CreateSessionResponse]
	}{
		{
			name: "nil_session",
			resp: &mpgsclient.Response[mpgsclient.CreateSessionResponse]{
				Data: mpgsclient.CreateSessionResponse{},
			},
		},
		{
			name: "empty_session_id",
			resp: &mpgsclient.Response[mpgsclient.CreateSessionResponse]{
				Data: mpgsclient.CreateSessionResponse{
					Session: &mpgsclient.SessionDetails{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpgsClient := mpgsmocks.NewMockClient(t)
			mpgsClient.EXPECT().
				CreateSession(mock.Anything, mock.Anything).
				Return(tt.resp, nil)

			gateway := &mpgsCardGateway{client: mpgsClient}
			resp, err := gateway.SetupCardPaymentMethod(context.Background(), validSetupCardPaymentMethodGatewayRequest())

			require.Error(t, err)
			assert.Contains(t, err.Error(), "missing session id")
			assert.Nil(t, resp)
		})
	}
}

func TestMPGSCardGateway_SetupCardPaymentMethod_UpdateSessionError(t *testing.T) {
	mpgsClient := mpgsmocks.NewMockClient(t)
	mpgsClient.EXPECT().
		CreateSession(mock.Anything, mock.Anything).
		Return(&mpgsclient.Response[mpgsclient.CreateSessionResponse]{
			Data: mpgsclient.CreateSessionResponse{
				Session: &mpgsclient.SessionDetails{ID: "SESSION123"},
			},
		}, nil)

	mpgsClient.EXPECT().
		UpdateSession(mock.Anything, "SESSION123", mock.Anything).
		Return(nil, errors.New("update failed"))

	gateway := &mpgsCardGateway{client: mpgsClient}
	resp, err := gateway.SetupCardPaymentMethod(context.Background(), validSetupCardPaymentMethodGatewayRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update mpgs session")
	assert.Nil(t, resp)
}

func TestMPGSCardGateway_PrepareCardAuthentication_Authenticate(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")
	rawResp := []byte(`{"result":"SUCCESS"}`)

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.PrepareCardAuthentication(ctx, PrepareCardAuthenticationGatewayRequest{
		InvoiceID:              invoiceID,
		Currency:               "BHD",
		PaymentMethodReference: "SESSION123",
	})
	require.NoError(t, err)
	require.NotNil(t, prepared)
	assert.NotEmpty(t, prepared.AuthenticationReference)
	assert.JSONEq(t, `{
		"apiOperation": "INITIATE_AUTHENTICATION",
		"authentication": {"channel": "PAYER_BROWSER"},
		"order": {"currency": "BHD"},
		"session": {"id": "SESSION123"}
	}`, string(prepared.RawRequest))

	mpgsClient.EXPECT().
		InitiateAuthentication(mock.Anything, invoiceID.String(), string(prepared.AuthenticationReference), mock.MatchedBy(func(req *mpgsclient.InitiateAuthenticationRequest) bool {
			return req.APIOperation == mpgsclient.OperationInitiateAuthentication &&
				req.Authentication.Channel == mpgsclient.ChannelPayerBrowser &&
				req.Order.Currency == "BHD" &&
				req.Session.ID == "SESSION123"
		})).
		Return(&mpgsclient.Response[mpgsclient.InitiateAuthenticationResponse]{
			Data: mpgsclient.InitiateAuthenticationResponse{
				Result: mpgsclient.ResultSuccess,
				Response: mpgsclient.InitiateAuthenticationGatewayResponse{
					GatewayRecommendation: mpgsclient.GatewayRecommendationProceed,
				},
				Transaction: mpgsclient.InitiateAuthenticationTransaction{
					AuthenticationStatus: mpgsclient.AuthStatusAvailable,
				},
			},
			RawBody: rawResp,
		}, nil)

	resp, err := prepared.Send(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardAuthenticationGatewayResultAvailable, resp.Result)
	assert.Equal(t, rawResp, resp.RawResponse)
}

func TestMPGSCardGateway_PrepareCardAuthentication_CantContinue(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.PrepareCardAuthentication(ctx, PrepareCardAuthenticationGatewayRequest{
		InvoiceID:              invoiceID,
		Currency:               "BHD",
		PaymentMethodReference: "SESSION123",
	})
	require.NoError(t, err)

	mpgsClient.EXPECT().
		InitiateAuthentication(mock.Anything, invoiceID.String(), string(prepared.AuthenticationReference), mock.Anything).
		Return(&mpgsclient.Response[mpgsclient.InitiateAuthenticationResponse]{
			Data: mpgsclient.InitiateAuthenticationResponse{
				Result: mpgsclient.ResultFailure,
				Response: mpgsclient.InitiateAuthenticationGatewayResponse{
					GatewayRecommendation: mpgsclient.GatewayRecommendationDoNotProceed,
				},
			},
		}, nil)

	resp, err := prepared.Send(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardAuthenticationGatewayResultUnavailable, resp.Result)
}

func TestMPGSCardGateway_PrepareCardAuthentication_AuthenticationUnavailable(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.PrepareCardAuthentication(ctx, PrepareCardAuthenticationGatewayRequest{
		InvoiceID:              invoiceID,
		Currency:               "BHD",
		PaymentMethodReference: "SESSION123",
	})
	require.NoError(t, err)

	mpgsClient.EXPECT().
		InitiateAuthentication(mock.Anything, invoiceID.String(), string(prepared.AuthenticationReference), mock.Anything).
		Return(&mpgsclient.Response[mpgsclient.InitiateAuthenticationResponse]{
			Data: mpgsclient.InitiateAuthenticationResponse{
				Result: mpgsclient.ResultSuccess,
				Response: mpgsclient.InitiateAuthenticationGatewayResponse{
					GatewayRecommendation: mpgsclient.GatewayRecommendationProceed,
				},
				Transaction: mpgsclient.InitiateAuthenticationTransaction{
					AuthenticationStatus: mpgsclient.AuthStatusFailed,
				},
			},
		}, nil)

	resp, err := prepared.Send(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PrepareCardAuthenticationGatewayResultUnavailable, resp.Result)
}

func TestMPGSCardGateway_PrepareCardAuthentication_SendError(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.PrepareCardAuthentication(ctx, PrepareCardAuthenticationGatewayRequest{
		InvoiceID:              invoiceID,
		Currency:               "BHD",
		PaymentMethodReference: "SESSION123",
	})
	require.NoError(t, err)

	mpgsClient.EXPECT().
		InitiateAuthentication(mock.Anything, invoiceID.String(), string(prepared.AuthenticationReference), mock.Anything).
		Return(nil, errors.New("init auth failed"))

	resp, err := prepared.Send(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initiate authentication")
	assert.Nil(t, resp)
}

func TestMPGSCardGateway_AuthenticateCardholder_Challenge(t *testing.T) {
	ctx := context.Background()
	req := validAuthenticateCardholderGatewayRequest()
	rawResp := []byte(`{"result":"PENDING"}`)
	redirectHTML := `<div id="threedsChallengeRedirect"></div>`

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.AuthenticateCardholder(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, prepared)
	assert.JSONEq(t, `{
		"apiOperation": "AUTHENTICATE_PAYER",
		"authentication": {
			"redirectResponseUrl": "https://pay.example.com/checkout/3001/complete-challenge"
		},
		"device": {
			"browser": "Mozilla/5.0",
			"browserDetails": {
				"3DSecureChallengeWindowSize": "FULL_SCREEN",
				"acceptHeaders": "text/html,application/xhtml+xml",
				"colorDepth": 24,
				"javaEnabled": false,
				"language": "en-US",
				"screenHeight": 1080,
				"screenWidth": 1920,
				"timeZone": 180
			},
			"ipAddress": "192.0.2.1"
		},
		"order": {
			"amount": "15",
			"currency": "BHD"
		},
		"session": {
			"id": "SESSION123"
		}
	}`, string(prepared.RawRequest))

	mpgsClient.EXPECT().
		AuthenticatePayer(mock.Anything, req.InvoiceID.String(), string(req.AuthenticationReference), mock.MatchedBy(func(got *mpgsclient.AuthenticatePayerRequest) bool {
			return got.APIOperation == mpgsclient.OperationAuthenticatePayer &&
				got.Authentication.RedirectResponseURL == req.ChallengeReturnURL &&
				got.Device.Browser == req.Browser.UserAgent &&
				got.Device.BrowserDetails != nil &&
				got.Device.BrowserDetails.ThreeDSecureChallengeWindowSize == string(req.Browser.ChallengeWindowSize) &&
				got.Device.BrowserDetails.AcceptHeaders == req.Browser.AcceptHeader &&
				got.Device.BrowserDetails.ColorDepth == req.Browser.ColorDepth &&
				got.Device.BrowserDetails.JavaEnabled == req.Browser.JavaEnabled &&
				got.Device.BrowserDetails.Language == req.Browser.Language &&
				got.Device.BrowserDetails.ScreenHeight == req.Browser.ScreenHeight &&
				got.Device.BrowserDetails.ScreenWidth == req.Browser.ScreenWidth &&
				got.Device.BrowserDetails.TimeZone == req.Browser.TimeZone &&
				got.Device.IPAddress == req.Browser.IPAddress &&
				got.Order.Amount == req.Amount.String() &&
				got.Order.Currency == req.Currency &&
				got.Session.ID == string(req.PaymentMethodReference)
		})).
		Return(&mpgsclient.Response[mpgsclient.AuthenticatePayerResponse]{
			Data: mpgsclient.AuthenticatePayerResponse{
				Authentication: mpgsclient.AuthenticatePayerRespAuthentication{
					Redirect: mpgsclient.AuthenticatePayerRespRedirect{HTML: redirectHTML},
				},
				Response: mpgsclient.AuthenticatePayerGatewayResponse{
					GatewayCode:           mpgsclient.CodePending,
					GatewayRecommendation: mpgsclient.GatewayRecommendationProceed,
				},
				Result: mpgsclient.ResultPending,
				Transaction: mpgsclient.AuthenticatePayerRespTransaction{
					AuthenticationStatus: mpgsclient.AuthStatusPending,
				},
			},
			RawBody: rawResp,
		}, nil)

	resp, err := prepared.Send(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, AuthenticateCardholderGatewayResultChallengeRequired, resp.Result)
	assert.Equal(t, redirectHTML, resp.RedirectHTML)
	assert.Equal(t, rawResp, resp.RawResponse)
}

func TestMPGSCardGateway_AuthenticateCardholder_Capture(t *testing.T) {
	for _, authStatus := range []mpgsclient.AuthStatus{
		mpgsclient.AuthStatusSuccessful,
		mpgsclient.AuthStatusAttempted,
	} {
		t.Run(string(authStatus), func(t *testing.T) {
			ctx := context.Background()
			req := validAuthenticateCardholderGatewayRequest()
			rawResp := []byte(`{"result":"SUCCESS"}`)

			mpgsClient := mpgsmocks.NewMockClient(t)
			gateway := &mpgsCardGateway{client: mpgsClient}

			prepared, err := gateway.AuthenticateCardholder(ctx, req)
			require.NoError(t, err)

			mpgsClient.EXPECT().
				AuthenticatePayer(mock.Anything, req.InvoiceID.String(), string(req.AuthenticationReference), mock.Anything).
				Return(&mpgsclient.Response[mpgsclient.AuthenticatePayerResponse]{
					Data: mpgsclient.AuthenticatePayerResponse{
						Response: mpgsclient.AuthenticatePayerGatewayResponse{
							GatewayCode:           mpgsclient.CodeApproved,
							GatewayRecommendation: mpgsclient.GatewayRecommendationProceed,
						},
						Result: mpgsclient.ResultSuccess,
						Transaction: mpgsclient.AuthenticatePayerRespTransaction{
							AuthenticationStatus: authStatus,
						},
					},
					RawBody: rawResp,
				}, nil)

			resp, err := prepared.Send(ctx)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, AuthenticateCardholderGatewayResultSucceeded, resp.Result)
			assert.Empty(t, resp.RedirectHTML)
			assert.Equal(t, rawResp, resp.RawResponse)
		})
	}
}

func TestMPGSCardGateway_AuthenticateCardholder_CantContinue(t *testing.T) {
	tests := []struct {
		name           string
		result         string
		recommendation mpgsclient.GatewayRecommendation
		authStatus     mpgsclient.AuthStatus
		redirectHTML   string
	}{
		{
			name:           "alternative_payment_details",
			result:         mpgsclient.ResultFailure,
			recommendation: mpgsclient.GatewayRecommendationResubmitWithAltPay,
			authStatus:     mpgsclient.AuthStatusFailed,
		},
		{
			name:           "abandon_order",
			result:         mpgsclient.ResultFailure,
			recommendation: mpgsclient.GatewayRecommendationDoNotProceedAbandonOrder,
			authStatus:     mpgsclient.AuthStatusFailed,
		},
		{
			name:       "missing_recommendation",
			result:     mpgsclient.ResultSuccess,
			authStatus: mpgsclient.AuthStatusSuccessful,
		},
		{
			name:           "pending_without_redirect_html",
			result:         mpgsclient.ResultPending,
			recommendation: mpgsclient.GatewayRecommendationProceed,
			authStatus:     mpgsclient.AuthStatusPending,
		},
		{
			name:           "pending_without_pending_authentication_status",
			result:         mpgsclient.ResultPending,
			recommendation: mpgsclient.GatewayRecommendationProceed,
			redirectHTML:   `<div></div>`,
		},
		{
			name:           "success_without_authentication_status",
			result:         mpgsclient.ResultSuccess,
			recommendation: mpgsclient.GatewayRecommendationProceed,
		},
		{
			name:           "success_with_pending_authentication_status",
			result:         mpgsclient.ResultSuccess,
			recommendation: mpgsclient.GatewayRecommendationProceed,
			authStatus:     mpgsclient.AuthStatusPending,
			redirectHTML:   `<div></div>`,
		},
		{
			name:           "success_with_failed_authentication_status",
			result:         mpgsclient.ResultSuccess,
			recommendation: mpgsclient.GatewayRecommendationProceed,
			authStatus:     mpgsclient.AuthStatusFailed,
		},
		{
			name:           "unknown_result",
			result:         mpgsclient.ResultUnknown,
			recommendation: mpgsclient.GatewayRecommendationProceed,
			authStatus:     mpgsclient.AuthStatusSuccessful,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := validAuthenticateCardholderGatewayRequest()
			mpgsClient := mpgsmocks.NewMockClient(t)
			gateway := &mpgsCardGateway{client: mpgsClient}

			prepared, err := gateway.AuthenticateCardholder(ctx, req)
			require.NoError(t, err)

			mpgsClient.EXPECT().
				AuthenticatePayer(mock.Anything, req.InvoiceID.String(), string(req.AuthenticationReference), mock.Anything).
				Return(&mpgsclient.Response[mpgsclient.AuthenticatePayerResponse]{
					Data: mpgsclient.AuthenticatePayerResponse{
						Authentication: mpgsclient.AuthenticatePayerRespAuthentication{
							Redirect: mpgsclient.AuthenticatePayerRespRedirect{HTML: tt.redirectHTML},
						},
						Response: mpgsclient.AuthenticatePayerGatewayResponse{
							GatewayRecommendation: tt.recommendation,
						},
						Result: tt.result,
						Transaction: mpgsclient.AuthenticatePayerRespTransaction{
							AuthenticationStatus: tt.authStatus,
						},
					},
				}, nil)

			resp, err := prepared.Send(ctx)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, AuthenticateCardholderGatewayResultFailed, resp.Result)
		})
	}
}

func TestMPGSCardGateway_AuthenticateCardholder_SendError(t *testing.T) {
	ctx := context.Background()
	req := validAuthenticateCardholderGatewayRequest()
	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.AuthenticateCardholder(ctx, req)
	require.NoError(t, err)

	mpgsClient.EXPECT().
		AuthenticatePayer(mock.Anything, req.InvoiceID.String(), string(req.AuthenticationReference), mock.Anything).
		Return(nil, errors.New("authenticate failed"))

	resp, err := prepared.Send(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to authenticate payer")
	assert.Nil(t, resp)
}

func TestMPGSCardGateway_GetCardAuthenticationResult_MapsActionableResults(t *testing.T) {
	tests := []struct {
		name           string
		result         string
		recommendation mpgsclient.GatewayRecommendation
		authStatus     mpgsclient.AuthStatus
		want           CardAuthenticationResult
	}{
		{
			name:           "successful_authentication_can_proceed",
			result:         mpgsclient.ResultSuccess,
			recommendation: mpgsclient.GatewayRecommendationProceed,
			authStatus:     mpgsclient.AuthStatusSuccessful,
			want:           CardAuthenticationResultSucceeded,
		},
		{
			name:           "attempted_authentication_can_proceed",
			result:         mpgsclient.ResultSuccess,
			recommendation: mpgsclient.GatewayRecommendationProceed,
			authStatus:     mpgsclient.AuthStatusAttempted,
			want:           CardAuthenticationResultSucceeded,
		},
		{
			name:           "pending_authentication_with_proceed",
			result:         mpgsclient.ResultPending,
			recommendation: mpgsclient.GatewayRecommendationProceed,
			authStatus:     mpgsclient.AuthStatusPending,
			want:           CardAuthenticationResultPending,
		},
		{
			name:           "pending_authentication_requires_later_check",
			result:         mpgsclient.ResultPending,
			recommendation: mpgsclient.GatewayRecommendationCheckTransactionStatusLater,
			authStatus:     mpgsclient.AuthStatusPending,
			want:           CardAuthenticationResultPending,
		},
		{
			name:           "unknown_authentication_requires_later_check",
			result:         mpgsclient.ResultUnknown,
			recommendation: mpgsclient.GatewayRecommendationCheckTransactionStatusLater,
			authStatus:     mpgsclient.AuthStatusPending,
			want:           CardAuthenticationResultPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			req := validGetCardAuthenticationResultGatewayRequest()
			data := validRetrieveAuthenticationTransaction(req)
			data.Result = tt.result
			data.Response.GatewayRecommendation = tt.recommendation
			data.Order.AuthenticationStatus = tt.authStatus
			data.Transaction.AuthenticationStatus = tt.authStatus
			rawResponse := []byte(`{"result":"test"}`)

			mpgsClient := mpgsmocks.NewMockClient(t)
			mpgsClient.EXPECT().
				RetrieveTransaction(mock.Anything, req.InvoiceID.String(), string(req.AuthenticationReference)).
				Return(&mpgsclient.Response[mpgsclient.RetrieveTransactionResponse]{
					Data:    data,
					RawBody: rawResponse,
				}, nil)

			gateway := &mpgsCardGateway{client: mpgsClient}
			prepared, err := gateway.GetCardAuthenticationResult(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, prepared)
			assert.Nil(t, prepared.RawRequest)

			resp, err := prepared.Send(ctx)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.want, resp.Result)
			assert.Equal(t, rawResponse, resp.RawResponse)
		})
	}
}

func TestMPGSCardGateway_GetCardAuthenticationResult_RejectsUnsafeResults(t *testing.T) {
	tests := []struct {
		name   string
		change func(*mpgsclient.RetrieveTransactionResponse)
	}{
		{
			name: "failed_result",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Result = mpgsclient.ResultFailure
			},
		},
		{
			name: "do_not_proceed",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Response.GatewayRecommendation = mpgsclient.GatewayRecommendationDoNotProceed
			},
		},
		{
			name: "abandon_order",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Response.GatewayRecommendation = mpgsclient.GatewayRecommendationDoNotProceedAbandonOrder
			},
		},
		{
			name: "alternative_payment_details",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Response.GatewayRecommendation = mpgsclient.GatewayRecommendationResubmitWithAltPay
			},
		},
		{
			name: "missing_recommendation",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Response.GatewayRecommendation = ""
			},
		},
		{
			name: "missing_authentication_status",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Order.AuthenticationStatus = ""
				resp.Transaction.AuthenticationStatus = ""
			},
		},
		{
			name: "pending_result_with_completed_status",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Result = mpgsclient.ResultPending
			},
		},
		{
			name: "successful_result_with_pending_status",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Order.AuthenticationStatus = mpgsclient.AuthStatusPending
				resp.Transaction.AuthenticationStatus = mpgsclient.AuthStatusPending
			},
		},
		{
			name: "unknown_result_without_later_check",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Result = mpgsclient.ResultUnknown
			},
		},
		{
			name: "later_check_with_completed_status",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Result = mpgsclient.ResultUnknown
				resp.Response.GatewayRecommendation = mpgsclient.GatewayRecommendationCheckTransactionStatusLater
			},
		},
		{
			name:   "failed_authentication",
			change: setRetrieveAuthenticationStatus(mpgsclient.AuthStatusFailed),
		},
		{
			name:   "authentication_not_supported",
			change: setRetrieveAuthenticationStatus(mpgsclient.AuthStatusNotSupported),
		},
		{
			name:   "authentication_not_in_effect",
			change: setRetrieveAuthenticationStatus(mpgsclient.AuthStatusNotInEffect),
		},
		{
			name:   "authentication_rejected",
			change: setRetrieveAuthenticationStatus(mpgsclient.AuthStatusRejected),
		},
		{
			name:   "authentication_required",
			change: setRetrieveAuthenticationStatus(mpgsclient.AuthStatusRequired),
		},
		{
			name:   "authentication_unavailable",
			change: setRetrieveAuthenticationStatus(mpgsclient.AuthStatusUnavailable),
		},
		{
			name: "wrong_order_ID",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Order.ID = uuid.NewString()
			},
		},
		{
			name: "wrong_transaction_ID",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Transaction.ID = uuid.NewString()
			},
		},
		{
			name: "wrong_transaction_type",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Transaction.Type = mpgsclient.TypePayment
			},
		},
		{
			name: "wrong_order_amount",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Order.Amount = decimal.NewFromInt(99)
			},
		},
		{
			name: "wrong_transaction_amount",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Transaction.Amount = decimal.NewFromInt(99)
			},
		},
		{
			name: "wrong_order_currency",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Order.Currency = "USD"
			},
		},
		{
			name: "wrong_transaction_currency",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Transaction.Currency = "USD"
			},
		},
		{
			name: "contradictory_authentication_statuses",
			change: func(resp *mpgsclient.RetrieveTransactionResponse) {
				resp.Order.AuthenticationStatus = mpgsclient.AuthStatusPending
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			req := validGetCardAuthenticationResultGatewayRequest()
			data := validRetrieveAuthenticationTransaction(req)
			tt.change(&data)
			rawResponse := []byte(`{"result":"unsafe"}`)

			mpgsClient := mpgsmocks.NewMockClient(t)
			mpgsClient.EXPECT().
				RetrieveTransaction(mock.Anything, req.InvoiceID.String(), string(req.AuthenticationReference)).
				Return(&mpgsclient.Response[mpgsclient.RetrieveTransactionResponse]{
					Data:    data,
					RawBody: rawResponse,
				}, nil)

			gateway := &mpgsCardGateway{client: mpgsClient}
			prepared, err := gateway.GetCardAuthenticationResult(ctx, req)
			require.NoError(t, err)

			resp, err := prepared.Send(ctx)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, CardAuthenticationResultFailed, resp.Result)
			assert.Equal(t, rawResponse, resp.RawResponse)
		})
	}
}

func TestMPGSCardGateway_GetCardAuthenticationResult_SendError(t *testing.T) {
	ctx := t.Context()
	req := validGetCardAuthenticationResultGatewayRequest()
	mpgsClient := mpgsmocks.NewMockClient(t)
	mpgsClient.EXPECT().
		RetrieveTransaction(mock.Anything, req.InvoiceID.String(), string(req.AuthenticationReference)).
		Return(nil, errors.New("retrieve failed"))

	gateway := &mpgsCardGateway{client: mpgsClient}
	prepared, err := gateway.GetCardAuthenticationResult(ctx, req)
	require.NoError(t, err)

	resp, err := prepared.Send(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve authentication transaction")
	assert.Nil(t, resp)
}

func TestMPGSCardGateway_CaptureCardPayment_MapsResult(t *testing.T) {
	tests := []struct {
		name        string
		result      string
		gatewayCode mpgsclient.GatewayCode
		want        CaptureCardPaymentGatewayResult
	}{
		{
			name:        "approved",
			result:      mpgsclient.ResultSuccess,
			gatewayCode: mpgsclient.CodeApproved,
			want:        CaptureCardPaymentGatewayResultSucceeded,
		},
		{
			name:        "declined",
			result:      mpgsclient.ResultFailure,
			gatewayCode: mpgsclient.CodeDeclined,
			want:        CaptureCardPaymentGatewayResultDeclined,
		},
		{
			name:        "pending",
			result:      mpgsclient.ResultPending,
			gatewayCode: mpgsclient.CodePending,
			want:        CaptureCardPaymentGatewayResultPending,
		},
		{
			name:        "unknown",
			result:      mpgsclient.ResultUnknown,
			gatewayCode: mpgsclient.CodeUnknown,
			want:        CaptureCardPaymentGatewayResultUnknown,
		},
		{
			name:        "ambiguous_failure",
			result:      mpgsclient.ResultFailure,
			gatewayCode: mpgsclient.CodeTimedOut,
			want:        CaptureCardPaymentGatewayResultUnknown,
		},
		{
			name:        "contradictory_success",
			result:      mpgsclient.ResultSuccess,
			gatewayCode: mpgsclient.CodeApprovedAuto,
			want:        CaptureCardPaymentGatewayResultUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCaptureCardPaymentGatewayRequest()
			data := validExecutePayResponse(req)
			data.Result = tt.result
			data.Response.GatewayCode = tt.gatewayCode
			rawResponse := []byte(`{"result":"test"}`)

			mpgsClient := mpgsmocks.NewMockClient(t)
			mpgsClient.EXPECT().
				ExecutePay(mock.Anything, req.InvoiceID.String(), req.PaymentReference.String(), mock.MatchedBy(func(got *mpgsclient.ExecutePayRequest) bool {
					return got.APIOperation == mpgsclient.OperationPay &&
						got.Authentication.TransactionID == string(req.AuthenticationReference) &&
						got.Order.Amount == req.Amount.String() &&
						got.Order.Currency == req.Currency &&
						got.Session.ID == string(req.PaymentMethodReference)
				})).
				Return(&mpgsclient.Response[mpgsclient.ExecutePayResponse]{
					Data:    data,
					RawBody: rawResponse,
				}, nil)

			gateway := &mpgsCardGateway{client: mpgsClient}
			prepared, err := gateway.CaptureCardPayment(t.Context(), req)
			require.NoError(t, err)
			require.NotNil(t, prepared)
			assert.JSONEq(t, `{
				"apiOperation":"PAY",
				"authentication":{"transactionId":"AUTHENTICATION123"},
				"order":{"amount":"15","currency":"BHD"},
				"session":{"id":"SESSION123"}
			}`, string(prepared.RawRequest))

			resp, err := prepared.Send(t.Context())
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.want, resp.Result)
			assert.Equal(t, rawResponse, resp.RawResponse)
		})
	}
}

func TestMPGSCardGateway_CaptureCardPayment_MapsDefinitiveDeclines(t *testing.T) {
	codes := []mpgsclient.GatewayCode{
		mpgsclient.CodeAborted,
		mpgsclient.CodeAuthenticationFailed,
		mpgsclient.CodeBlocked,
		mpgsclient.CodeCancelled,
		mpgsclient.CodeDeclined,
		mpgsclient.CodeDeclinedAVS,
		mpgsclient.CodeDeclinedAVSCSC,
		mpgsclient.CodeDeclinedCSC,
		mpgsclient.CodeDeclinedDoNotContact,
		mpgsclient.CodeDeclinedInvalidPIN,
		mpgsclient.CodeDeclinedPaymentPlan,
		mpgsclient.CodeDeclinedPINRequired,
		mpgsclient.CodeExceededRetryLimit,
		mpgsclient.CodeExpiredCard,
		mpgsclient.CodeInsufficientFunds,
		mpgsclient.CodeInvalidCSC,
		mpgsclient.CodeNotEnrolled3DS,
		mpgsclient.CodeNotSupported,
		mpgsclient.CodePartiallyApproved,
		mpgsclient.CodeReferred,
		mpgsclient.CodeUnspecifiedFailure,
	}

	for _, code := range codes {
		t.Run(string(code), func(t *testing.T) {
			req := validCaptureCardPaymentGatewayRequest()
			data := validExecutePayResponse(req)
			data.Result = mpgsclient.ResultFailure
			data.Response.GatewayCode = code

			mpgsClient := mpgsmocks.NewMockClient(t)
			mpgsClient.EXPECT().
				ExecutePay(mock.Anything, req.InvoiceID.String(), req.PaymentReference.String(), mock.Anything).
				Return(&mpgsclient.Response[mpgsclient.ExecutePayResponse]{Data: data}, nil)

			gateway := &mpgsCardGateway{client: mpgsClient}
			prepared, err := gateway.CaptureCardPayment(t.Context(), req)
			require.NoError(t, err)
			resp, err := prepared.Send(t.Context())
			require.NoError(t, err)
			assert.Equal(t, CaptureCardPaymentGatewayResultDeclined, resp.Result)
		})
	}
}

func TestMPGSCardGateway_CaptureCardPayment_RejectsUnsafeSuccess(t *testing.T) {
	tests := []struct {
		name   string
		change func(*mpgsclient.ExecutePayResponse)
	}{
		{name: "wrong_order_id", change: func(resp *mpgsclient.ExecutePayResponse) { resp.Order.ID = uuid.NewString() }},
		{name: "wrong_transaction_id", change: func(resp *mpgsclient.ExecutePayResponse) { resp.Transaction.ID = uuid.NewString() }},
		{name: "wrong_transaction_type", change: func(resp *mpgsclient.ExecutePayResponse) { resp.Transaction.Type = mpgsclient.TypeAuthorization }},
		{name: "wrong_order_amount", change: func(resp *mpgsclient.ExecutePayResponse) { resp.Order.Amount = decimal.NewFromInt(99) }},
		{name: "wrong_transaction_amount", change: func(resp *mpgsclient.ExecutePayResponse) { resp.Transaction.Amount = decimal.NewFromInt(99) }},
		{name: "wrong_order_currency", change: func(resp *mpgsclient.ExecutePayResponse) { resp.Order.Currency = "USD" }},
		{name: "wrong_transaction_currency", change: func(resp *mpgsclient.ExecutePayResponse) { resp.Transaction.Currency = "USD" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCaptureCardPaymentGatewayRequest()
			data := validExecutePayResponse(req)
			tt.change(&data)
			rawResponse := []byte(`{"result":"SUCCESS"}`)

			mpgsClient := mpgsmocks.NewMockClient(t)
			mpgsClient.EXPECT().
				ExecutePay(mock.Anything, req.InvoiceID.String(), req.PaymentReference.String(), mock.Anything).
				Return(&mpgsclient.Response[mpgsclient.ExecutePayResponse]{
					Data:    data,
					RawBody: rawResponse,
				}, nil)

			gateway := &mpgsCardGateway{client: mpgsClient}
			prepared, err := gateway.CaptureCardPayment(t.Context(), req)
			require.NoError(t, err)
			resp, err := prepared.Send(t.Context())
			require.ErrorIs(t, err, ErrGatewayResponseMismatch)
			assert.Equal(t, rawResponse, rawGatewayResponse(err))
			assert.Nil(t, resp)
		})
	}
}

func TestMPGSCardGateway_CaptureCardPayment_SendError(t *testing.T) {
	req := validCaptureCardPaymentGatewayRequest()
	mpgsClient := mpgsmocks.NewMockClient(t)
	mpgsClient.EXPECT().
		ExecutePay(mock.Anything, req.InvoiceID.String(), req.PaymentReference.String(), mock.Anything).
		Return(nil, errors.New("pay failed"))

	gateway := &mpgsCardGateway{client: mpgsClient}
	prepared, err := gateway.CaptureCardPayment(t.Context(), req)
	require.NoError(t, err)
	resp, err := prepared.Send(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to capture card payment")
	assert.Nil(t, resp)
}

func setRetrieveAuthenticationStatus(status mpgsclient.AuthStatus) func(*mpgsclient.RetrieveTransactionResponse) {
	return func(resp *mpgsclient.RetrieveTransactionResponse) {
		resp.Order.AuthenticationStatus = status
		resp.Transaction.AuthenticationStatus = status
	}
}

func validRetrieveAuthenticationTransaction(r GetCardAuthenticationResultGatewayRequest) mpgsclient.RetrieveTransactionResponse {
	return mpgsclient.RetrieveTransactionResponse{
		Order: mpgsclient.RetrieveTransactionOrder{
			Amount:               r.Amount,
			AuthenticationStatus: mpgsclient.AuthStatusSuccessful,
			Currency:             r.Currency,
			ID:                   r.InvoiceID.String(),
		},
		Response: mpgsclient.RetrieveTransactionGatewayResp{
			GatewayRecommendation: mpgsclient.GatewayRecommendationProceed,
		},
		Result: mpgsclient.ResultSuccess,
		Transaction: mpgsclient.RetrieveTransactionTx{
			Amount:               r.Amount,
			AuthenticationStatus: mpgsclient.AuthStatusSuccessful,
			Currency:             r.Currency,
			ID:                   string(r.AuthenticationReference),
			Type:                 mpgsclient.TypeAuthentication,
		},
	}
}

func validSetupCardPaymentMethodGatewayRequest() SetupCardPaymentMethodGatewayRequest {
	return SetupCardPaymentMethodGatewayRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000003001"),
		Amount:    decimal.RequireFromString("15.000"),
		Currency:  "BHD",
	}
}

func validAuthenticateCardholderGatewayRequest() AuthenticateCardholderGatewayRequest {
	return AuthenticateCardholderGatewayRequest{
		InvoiceID:               uuid.MustParse("00000000-0000-0000-0000-000000003001"),
		Amount:                  decimal.RequireFromString("15.000"),
		Currency:                "BHD",
		PaymentMethodReference:  "SESSION123",
		AuthenticationReference: "AUTHENTICATION123",
		ChallengeReturnURL:      "https://pay.example.com/checkout/3001/complete-challenge",
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
			TimeZone:            180,
		},
	}
}

func validGetCardAuthenticationResultGatewayRequest() GetCardAuthenticationResultGatewayRequest {
	return GetCardAuthenticationResultGatewayRequest{
		InvoiceID:               uuid.MustParse("00000000-0000-0000-0000-000000003001"),
		Amount:                  decimal.RequireFromString("15.000"),
		Currency:                "BHD",
		AuthenticationReference: "AUTHENTICATION123",
	}
}

func validCaptureCardPaymentGatewayRequest() CaptureCardPaymentGatewayRequest {
	return CaptureCardPaymentGatewayRequest{
		InvoiceID:               uuid.MustParse("00000000-0000-0000-0000-000000003001"),
		PaymentReference:        uuid.MustParse("00000000-0000-0000-0000-000000005008"),
		Amount:                  decimal.RequireFromString("15.000"),
		Currency:                "BHD",
		PaymentMethodReference:  "SESSION123",
		AuthenticationReference: "AUTHENTICATION123",
	}
}

func validExecutePayResponse(r CaptureCardPaymentGatewayRequest) mpgsclient.ExecutePayResponse {
	return mpgsclient.ExecutePayResponse{
		Order: mpgsclient.ExecutePayOrder{
			Amount:   r.Amount,
			Currency: r.Currency,
			ID:       r.InvoiceID.String(),
		},
		Response: mpgsclient.ExecutePayGatewayResponse{
			GatewayCode: mpgsclient.CodeApproved,
		},
		Result: mpgsclient.ResultSuccess,
		Transaction: mpgsclient.ExecutePayTransaction{
			Amount:   r.Amount,
			Currency: r.Currency,
			ID:       r.PaymentReference.String(),
			Type:     mpgsclient.TypePayment,
		},
	}
}
