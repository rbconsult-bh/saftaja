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

func (mcg *mpgsCardGateway) SetupCardPayment(ctx context.Context, r SetupCardPaymentGatewayRequest) (*SetupCardPaymentGatewayResponse, error) {
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

	return &SetupCardPaymentGatewayResponse{
		GatewaySetupReference: createSessionResp.Data.Session.ID,
	}, nil
}

func (mcg *mpgsCardGateway) PrepareCardChallenge(ctx context.Context, r PrepareCardChallengeGatewayRequest) (*PreparedCardChallengeGatewayRequest, error) {
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
			ID: r.GatewaySetupReference,
		},
	}

	rawReq, err := json.Marshal(mpgsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initiate authentication request: %w", err)
	}

	return &PreparedCardChallengeGatewayRequest{
		GatewayReference: gatewayReference,
		RawRequest:       rawReq,
		send: func(ctx context.Context) (*PrepareCardChallengeGatewayResponse, error) {
			resp, err := mcg.client.InitiateAuthentication(ctx, r.InvoiceID.String(), gatewayReference, mpgsReq)
			if err != nil {
				return nil, fmt.Errorf("failed to initiate authentication: %w", err)
			}
			nextStep := PrepareCardChallengeGatewayNextStepCantContinue
			if resp.Data.Result == mpgsclient.ResultSuccess &&
				resp.Data.Response.GatewayRecommendation == mpgsclient.GatewayRecommendationProceed {
				nextStep = PrepareCardChallengeGatewayNextStepStartChallenge
			}

			return &PrepareCardChallengeGatewayResponse{
				NextStep:    nextStep,
				RawResponse: resp.RawBody,
			}, nil
		},
	}, nil
}
