package billing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/shopspring/decimal"
)

type GatewayResolver interface {
	CardGateway(account store.GatewayAccount) (CardGateway, error)
}

type CardGateway interface {
	SetupCardPayment(ctx context.Context, r SetupCardPaymentGatewayRequest) (*SetupCardPaymentGatewayResponse, error)
	PrepareVerifyCard(ctx context.Context, r VerifyCardGatewayRequest) (*PreparedVerifyCardGatewayRequest, error)
}

type (
	SetupCardPaymentGatewayRequest struct {
		InvoiceID uuid.UUID
		Amount    decimal.Decimal
		Currency  string
	}
	SetupCardPaymentGatewayResponse struct {
		GatewaySetupReference string
	}
)

type VerifyCardGatewayNextStep string

const (
	VerifyCardGatewayNextStepChallengeCard VerifyCardGatewayNextStep = "challenge_card"
	VerifyCardGatewayNextStepCantContinue  VerifyCardGatewayNextStep = "cant_continue"
)

type (
	VerifyCardGatewayRequest struct {
		InvoiceID             uuid.UUID
		Currency              string
		GatewaySetupReference string
	}
	PreparedVerifyCardGatewayRequest struct {
		GatewayReference string
		RawRequest       []byte
		send             func(ctx context.Context) (*VerifyCardGatewayResponse, error)
	}
	VerifyCardGatewayResponse struct {
		NextStep    VerifyCardGatewayNextStep
		RawResponse []byte
	}
)

func (p *PreparedVerifyCardGatewayRequest) Send(ctx context.Context) (*VerifyCardGatewayResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("prepared verify card request is nil")
	}
	if p.send == nil {
		return nil, fmt.Errorf("prepared verify card request is missing send function")
	}
	return p.send(ctx)
}

type gatewayResolver struct {
	encryptionKey []byte
}

func NewGatewayResolver(encryptionKey []byte) GatewayResolver {
	return &gatewayResolver{
		encryptionKey: encryptionKey,
	}
}

func (gr *gatewayResolver) CardGateway(account store.GatewayAccount) (CardGateway, error) {
	switch account.ConnectorType {
	case store.ConnectorTypeMPGS:
		return newMPGSCardGateway(account, gr.encryptionKey)
	default:
		return nil, fmt.Errorf("%w: connector_type: %s does not support card", ErrUnsupportedGateway, account.ConnectorType)
	}
}
