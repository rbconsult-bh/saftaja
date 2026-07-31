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
	PrepareCardChallenge(ctx context.Context, r PrepareCardChallengeGatewayRequest) (*PreparedCardChallengeGatewayRequest, error)
	StartCardChallenge(ctx context.Context, r StartCardChallengeGatewayRequest) (*PreparedStartCardChallengeGatewayRequest, error)
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

type PrepareCardChallengeGatewayNextStep string

const (
	PrepareCardChallengeGatewayNextStepStartChallenge PrepareCardChallengeGatewayNextStep = "start_challenge"
	PrepareCardChallengeGatewayNextStepCantContinue   PrepareCardChallengeGatewayNextStep = "cant_continue"
)

type (
	PrepareCardChallengeGatewayRequest struct {
		InvoiceID             uuid.UUID
		Currency              string
		GatewaySetupReference string
	}
	PreparedCardChallengeGatewayRequest struct {
		GatewayReference string
		RawRequest       []byte
		send             func(ctx context.Context) (*PrepareCardChallengeGatewayResponse, error)
	}
	PrepareCardChallengeGatewayResponse struct {
		NextStep    PrepareCardChallengeGatewayNextStep
		RawResponse []byte
	}
)

func (p *PreparedCardChallengeGatewayRequest) Send(ctx context.Context) (*PrepareCardChallengeGatewayResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("prepared card challenge request is nil")
	}
	if p.send == nil {
		return nil, fmt.Errorf("prepared card challenge request is missing send function")
	}
	return p.send(ctx)
}

type StartCardChallengeGatewayNextStep string

const (
	StartCardChallengeGatewayNextStepCapture           StartCardChallengeGatewayNextStep = "capture"
	StartCardChallengeGatewayNextStepCompleteChallenge StartCardChallengeGatewayNextStep = "complete_challenge"
	StartCardChallengeGatewayNextStepCantContinue      StartCardChallengeGatewayNextStep = "cant_continue"
)

type (
	StartCardChallengeGatewayRequest struct {
		InvoiceID             uuid.UUID
		Amount                decimal.Decimal
		Currency              string
		GatewaySetupReference string
		GatewayReference      string
		ChallengeReturnURL    string
		Browser               ThreeDSBrowser
	}
	PreparedStartCardChallengeGatewayRequest struct {
		RawRequest []byte
		send       func(ctx context.Context) (*StartCardChallengeGatewayResponse, error)
	}
	StartCardChallengeGatewayResponse struct {
		NextStep     StartCardChallengeGatewayNextStep
		RedirectHTML string
		RawResponse  []byte
	}
)

func (p *PreparedStartCardChallengeGatewayRequest) Send(ctx context.Context) (*StartCardChallengeGatewayResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("prepared start card challenge request is nil")
	}
	if p.send == nil {
		return nil, fmt.Errorf("prepared start card challenge request is missing send function")
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
