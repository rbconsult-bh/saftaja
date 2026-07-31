package billing

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

func TestMPGSCardGateway_SetupCardPayment_Succeeds(t *testing.T) {
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
	resp, err := gateway.SetupCardPayment(ctx, SetupCardPaymentGatewayRequest{
		InvoiceID: invoiceID,
		Amount:    decimal.RequireFromString("15.000"),
		Currency:  "BHD",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, PaymentMethodReference("SESSION123"), resp.PaymentMethodReference)
}

func TestMPGSCardGateway_SetupCardPayment_CreateSessionError(t *testing.T) {
	mpgsClient := mpgsmocks.NewMockClient(t)
	mpgsClient.EXPECT().
		CreateSession(mock.Anything, mock.Anything).
		Return(nil, errors.New("create failed"))

	gateway := &mpgsCardGateway{client: mpgsClient}
	resp, err := gateway.SetupCardPayment(context.Background(), validSetupCardPaymentGatewayRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create mpgs session")
	assert.Nil(t, resp)
}

func TestMPGSCardGateway_SetupCardPayment_MissingSetupReference(t *testing.T) {
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
			resp, err := gateway.SetupCardPayment(context.Background(), validSetupCardPaymentGatewayRequest())

			require.Error(t, err)
			assert.Contains(t, err.Error(), "missing session id")
			assert.Nil(t, resp)
		})
	}
}

func TestMPGSCardGateway_SetupCardPayment_UpdateSessionError(t *testing.T) {
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
	resp, err := gateway.SetupCardPayment(context.Background(), validSetupCardPaymentGatewayRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update mpgs session")
	assert.Nil(t, resp)
}

func TestMPGSCardGateway_PrepareCardChallenge_StartChallenge(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")
	rawResp := []byte(`{"result":"SUCCESS"}`)

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.PrepareCardChallenge(ctx, PrepareCardChallengeGatewayRequest{
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
	assert.Equal(t, PrepareCardChallengeGatewayNextStepStartChallenge, resp.NextStep)
	assert.Equal(t, rawResp, resp.RawResponse)
}

func TestMPGSCardGateway_PrepareCardChallenge_CantContinue(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.PrepareCardChallenge(ctx, PrepareCardChallengeGatewayRequest{
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
	assert.Equal(t, PrepareCardChallengeGatewayNextStepCantContinue, resp.NextStep)
}

func TestMPGSCardGateway_PrepareCardChallenge_AuthenticationUnavailable(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.PrepareCardChallenge(ctx, PrepareCardChallengeGatewayRequest{
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
	assert.Equal(t, PrepareCardChallengeGatewayNextStepCantContinue, resp.NextStep)
}

func TestMPGSCardGateway_PrepareCardChallenge_SendError(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.PrepareCardChallenge(ctx, PrepareCardChallengeGatewayRequest{
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

func TestMPGSCardGateway_StartCardChallenge_CompleteChallenge(t *testing.T) {
	ctx := context.Background()
	req := validStartCardChallengeGatewayRequest()
	rawResp := []byte(`{"result":"PENDING"}`)
	redirectHTML := `<div id="threedsChallengeRedirect"></div>`

	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.StartCardChallenge(ctx, req)
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
	assert.Equal(t, StartCardChallengeGatewayNextStepCompleteChallenge, resp.NextStep)
	assert.Equal(t, redirectHTML, resp.RedirectHTML)
	assert.Equal(t, rawResp, resp.RawResponse)
}

func TestMPGSCardGateway_StartCardChallenge_Capture(t *testing.T) {
	for _, authStatus := range []mpgsclient.AuthStatus{
		mpgsclient.AuthStatusSuccessful,
		mpgsclient.AuthStatusAttempted,
	} {
		t.Run(string(authStatus), func(t *testing.T) {
			ctx := context.Background()
			req := validStartCardChallengeGatewayRequest()
			rawResp := []byte(`{"result":"SUCCESS"}`)

			mpgsClient := mpgsmocks.NewMockClient(t)
			gateway := &mpgsCardGateway{client: mpgsClient}

			prepared, err := gateway.StartCardChallenge(ctx, req)
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
			assert.Equal(t, StartCardChallengeGatewayNextStepCapture, resp.NextStep)
			assert.Empty(t, resp.RedirectHTML)
			assert.Equal(t, rawResp, resp.RawResponse)
		})
	}
}

func TestMPGSCardGateway_StartCardChallenge_CantContinue(t *testing.T) {
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
			req := validStartCardChallengeGatewayRequest()
			mpgsClient := mpgsmocks.NewMockClient(t)
			gateway := &mpgsCardGateway{client: mpgsClient}

			prepared, err := gateway.StartCardChallenge(ctx, req)
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
			assert.Equal(t, StartCardChallengeGatewayNextStepCantContinue, resp.NextStep)
		})
	}
}

func TestMPGSCardGateway_StartCardChallenge_SendError(t *testing.T) {
	ctx := context.Background()
	req := validStartCardChallengeGatewayRequest()
	mpgsClient := mpgsmocks.NewMockClient(t)
	gateway := &mpgsCardGateway{client: mpgsClient}

	prepared, err := gateway.StartCardChallenge(ctx, req)
	require.NoError(t, err)

	mpgsClient.EXPECT().
		AuthenticatePayer(mock.Anything, req.InvoiceID.String(), string(req.AuthenticationReference), mock.Anything).
		Return(nil, errors.New("authenticate failed"))

	resp, err := prepared.Send(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to authenticate payer")
	assert.Nil(t, resp)
}

func validSetupCardPaymentGatewayRequest() SetupCardPaymentGatewayRequest {
	return SetupCardPaymentGatewayRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000003001"),
		Amount:    decimal.RequireFromString("15.000"),
		Currency:  "BHD",
	}
}

func validStartCardChallengeGatewayRequest() StartCardChallengeGatewayRequest {
	return StartCardChallengeGatewayRequest{
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
