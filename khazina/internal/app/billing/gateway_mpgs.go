package billing

import (
	"context"
	"fmt"

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

func (mcg *mpgsCardGateway) CreateSession(ctx context.Context, r CreateSessionRequest) (*CreateSessionResponse, error) {
	createSessionResp, err := mcg.client.CreateSession(ctx, &mpgsclient.CreateSessionRequest{
		Session: &mpgsclient.CreateSessionRequestSession{
			AuthenticationLimit: ptr.To[int32](25),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mpgs session: %w", err)
	}
	if createSessionResp == nil || createSessionResp.Data.Session == nil || createSessionResp.Data.Session.ID == "" {
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

	return &CreateSessionResponse{
		GatewaySessionID: createSessionResp.Data.Session.ID,
	}, nil
}
