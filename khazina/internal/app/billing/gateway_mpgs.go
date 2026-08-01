package billing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/ptr"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type mpgsCardGateway struct {
	client mpgsclient.Client
}

func newMPGSCardGateway(account store.GatewayAccount, encryptionKey []byte) (CardGateway, error) {
	cfg, err := mpgsclient.ParseConfig(account.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid mpgs config: %w", err)
	}

	secret, err := mpgsclient.DecryptSecret(account.Secret, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid mpgs secret: %w", err)
	}

	client := mpgsclient.New(
		cfg.BaseURL,
		cfg.MerchantID,
		secret.APIPassword,
	)

	return &mpgsCardGateway{
		client: client,
	}, nil
}

func (mcg *mpgsCardGateway) SetupCardPaymentMethod(ctx context.Context, r SetupCardPaymentMethodGatewayRequest) (*SetupCardPaymentMethodGatewayResponse, error) {
	createSessionResp, err := mcg.client.CreateSession(ctx, &mpgsclient.CreateSessionRequest{
		Session: &mpgsclient.CreateSessionRequestSession{
			AuthenticationLimit: ptr.To[int32](25),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mpgs session: %w", err)
	}
	if createSessionResp.Data.Session == nil || createSessionResp.Data.Session.ID == "" {
		return nil, fmt.Errorf("failed to create mpgs session: missing session id")
	}

	_, err = mcg.client.UpdateSession(ctx, createSessionResp.Data.Session.ID, &mpgsclient.UpdateSessionRequest{
		Order: mpgsclient.UpdateSessionOrder{
			Amount:   r.Amount.String(),
			Currency: r.Currency,
			ID:       r.InvoiceID.String(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update mpgs session: %w", err)
	}

	return &SetupCardPaymentMethodGatewayResponse{
		PaymentMethodReference: PaymentMethodReference(createSessionResp.Data.Session.ID),
	}, nil
}

func (mcg *mpgsCardGateway) PrepareCardAuthentication(ctx context.Context, r PrepareCardAuthenticationGatewayRequest) (*PreparedCardAuthenticationGatewayRequest, error) {
	gatewayReference := uuid.NewString()
	mpgsReq := &mpgsclient.InitiateAuthenticationRequest{
		APIOperation: mpgsclient.OperationInitiateAuthentication,
		Authentication: mpgsclient.InitiateAuthenticationReqAuthentication{
			Channel: mpgsclient.ChannelPayerBrowser,
		},
		Order: mpgsclient.InitiateAuthenticationOrder{
			Currency: r.Currency,
		},
		Session: mpgsclient.InitiateAuthenticationSession{
			ID: string(r.PaymentMethodReference),
		},
	}

	rawReq, err := json.Marshal(mpgsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initiate authentication request: %w", err)
	}

	return &PreparedCardAuthenticationGatewayRequest{
		AuthenticationReference: AuthenticationReference(gatewayReference),
		RawRequest:              rawReq,
		send: func(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error) {
			resp, err := mcg.client.InitiateAuthentication(ctx, r.InvoiceID.String(), gatewayReference, mpgsReq)
			if err != nil {
				return nil, fmt.Errorf("failed to initiate authentication: %w", err)
			}
			nextStep := PrepareCardAuthenticationGatewayNextStepCantContinue
			if resp.Data.Result == mpgsclient.ResultSuccess &&
				resp.Data.Response.GatewayRecommendation == mpgsclient.GatewayRecommendationProceed &&
				resp.Data.Transaction.AuthenticationStatus == mpgsclient.AuthStatusAvailable {
				nextStep = PrepareCardAuthenticationGatewayNextStepAuthenticate
			}

			return &PrepareCardAuthenticationGatewayResponse{
				NextStep:    nextStep,
				RawResponse: resp.RawBody,
			}, nil
		},
	}, nil
}

func (mcg *mpgsCardGateway) AuthenticateCardholder(ctx context.Context, r AuthenticateCardholderGatewayRequest) (*PreparedAuthenticateCardholderGatewayRequest, error) {
	mpgsReq := &mpgsclient.AuthenticatePayerRequest{
		APIOperation: mpgsclient.OperationAuthenticatePayer,
		Authentication: mpgsclient.AuthenticatePayerReqAuthentication{
			RedirectResponseURL: r.ChallengeReturnURL,
		},
		Device: mpgsclient.AuthenticatePayerReqDevice{
			Browser: r.Browser.UserAgent,
			BrowserDetails: &mpgsclient.AuthenticatePayerReqBrowserDetails{
				ThreeDSecureChallengeWindowSize: string(r.Browser.ChallengeWindowSize),
				AcceptHeaders:                   r.Browser.AcceptHeader,
				ColorDepth:                      r.Browser.ColorDepth,
				JavaEnabled:                     r.Browser.JavaEnabled,
				Language:                        r.Browser.Language,
				ScreenHeight:                    r.Browser.ScreenHeight,
				ScreenWidth:                     r.Browser.ScreenWidth,
				TimeZone:                        r.Browser.TimeZone,
			},
			IPAddress: r.Browser.IPAddress,
		},
		Order: mpgsclient.AuthenticatePayerReqOrder{
			Amount:   r.Amount.String(),
			Currency: r.Currency,
		},
		Session: mpgsclient.AuthenticatePayerReqSession{
			ID: string(r.PaymentMethodReference),
		},
	}

	rawReq, err := json.Marshal(mpgsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal authenticate payer request: %w", err)
	}

	return &PreparedAuthenticateCardholderGatewayRequest{
		RawRequest: rawReq,
		send: func(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error) {
			resp, err := mcg.client.AuthenticatePayer(ctx, r.InvoiceID.String(), string(r.AuthenticationReference), mpgsReq)
			if err != nil {
				return nil, fmt.Errorf("failed to authenticate payer: %w", err)
			}

			nextStep := AuthenticateCardholderGatewayNextStepCantContinue
			if resp.Data.Response.GatewayRecommendation == mpgsclient.GatewayRecommendationProceed {
				switch {
				case resp.Data.Result == mpgsclient.ResultPending &&
					resp.Data.Transaction.AuthenticationStatus == mpgsclient.AuthStatusPending &&
					resp.Data.Authentication.Redirect.HTML != "":
					nextStep = AuthenticateCardholderGatewayNextStepChallenge
				case resp.Data.Result == mpgsclient.ResultSuccess &&
					(resp.Data.Transaction.AuthenticationStatus == mpgsclient.AuthStatusSuccessful ||
						resp.Data.Transaction.AuthenticationStatus == mpgsclient.AuthStatusAttempted):
					nextStep = AuthenticateCardholderGatewayNextStepCapture
				}
			}

			return &AuthenticateCardholderGatewayResponse{
				NextStep:     nextStep,
				RedirectHTML: resp.Data.Authentication.Redirect.HTML,
				RawResponse:  resp.RawBody,
			}, nil
		},
	}, nil
}

func (mcg *mpgsCardGateway) GetCardAuthenticationResult(ctx context.Context, r GetCardAuthenticationResultGatewayRequest) (*PreparedGetCardAuthenticationResultGatewayRequest, error) {
	return &PreparedGetCardAuthenticationResultGatewayRequest{
		RawRequest: nil,
		send: func(ctx context.Context) (*GetCardAuthenticationResultGatewayResponse, error) {
			resp, err := mcg.client.RetrieveTransaction(
				ctx,
				r.InvoiceID.String(),
				string(r.AuthenticationReference),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve authentication transaction: %w", err)
			}

			result := mapMPGSCardAuthenticationResult(r, resp.Data)
			return &GetCardAuthenticationResultGatewayResponse{
				Result:      result,
				RawResponse: resp.RawBody,
			}, nil
		},
	}, nil
}

func mapMPGSCardAuthenticationResult(
	r GetCardAuthenticationResultGatewayRequest,
	resp mpgsclient.RetrieveTransactionResponse,
) CardAuthenticationResult {
	if resp.Order.ID != r.InvoiceID.String() ||
		resp.Transaction.ID != string(r.AuthenticationReference) ||
		resp.Transaction.Type != mpgsclient.TypeAuthentication ||
		!resp.Order.Amount.Equal(r.Amount) ||
		!resp.Transaction.Amount.Equal(r.Amount) ||
		resp.Order.Currency != r.Currency ||
		resp.Transaction.Currency != r.Currency ||
		(resp.Order.AuthenticationStatus != "" &&
			resp.Order.AuthenticationStatus != resp.Transaction.AuthenticationStatus) {
		return CardAuthenticationResultCantContinue
	}

	authenticationStatus := resp.Transaction.AuthenticationStatus
	switch {
	case resp.Result == mpgsclient.ResultSuccess &&
		resp.Response.GatewayRecommendation == mpgsclient.GatewayRecommendationProceed &&
		(authenticationStatus == mpgsclient.AuthStatusSuccessful ||
			authenticationStatus == mpgsclient.AuthStatusAttempted):
		return CardAuthenticationResultProceed

	case resp.Result == mpgsclient.ResultPending &&
		resp.Response.GatewayRecommendation == mpgsclient.GatewayRecommendationProceed &&
		authenticationStatus == mpgsclient.AuthStatusPending:
		return CardAuthenticationResultPending

	case (resp.Result == mpgsclient.ResultPending || resp.Result == mpgsclient.ResultUnknown) &&
		resp.Response.GatewayRecommendation == mpgsclient.GatewayRecommendationCheckTransactionStatusLater &&
		authenticationStatus == mpgsclient.AuthStatusPending:
		return CardAuthenticationResultPending

	default:
		return CardAuthenticationResultCantContinue
	}
}
