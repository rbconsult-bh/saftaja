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

type PaymentMethodReference string
type AuthenticationReference string

type CardGateway interface {
	SetupCardPaymentMethod(ctx context.Context, r SetupCardPaymentMethodGatewayRequest) (*SetupCardPaymentMethodGatewayResponse, error)
	PrepareCardAuthentication(ctx context.Context, r PrepareCardAuthenticationGatewayRequest) (*PreparedCardAuthenticationGatewayRequest, error)
	AuthenticateCardholder(ctx context.Context, r AuthenticateCardholderGatewayRequest) (*PreparedAuthenticateCardholderGatewayRequest, error)
	GetCardAuthenticationResult(ctx context.Context, r GetCardAuthenticationResultGatewayRequest) (*PreparedGetCardAuthenticationResultGatewayRequest, error)
}

type (
	SetupCardPaymentMethodGatewayRequest struct {
		InvoiceID uuid.UUID
		Amount    decimal.Decimal
		Currency  string
	}
	SetupCardPaymentMethodGatewayResponse struct {
		PaymentMethodReference PaymentMethodReference
	}
)

type PrepareCardAuthenticationGatewayNextStep string

const (
	PrepareCardAuthenticationGatewayNextStepAuthenticate PrepareCardAuthenticationGatewayNextStep = "authenticate"
	PrepareCardAuthenticationGatewayNextStepCantContinue PrepareCardAuthenticationGatewayNextStep = "cant_continue"
)

type (
	PrepareCardAuthenticationGatewayRequest struct {
		InvoiceID              uuid.UUID
		Currency               string
		PaymentMethodReference PaymentMethodReference
	}
	PreparedCardAuthenticationGatewayRequest struct {
		AuthenticationReference AuthenticationReference
		RawRequest              []byte
		send                    func(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error)
	}
	PrepareCardAuthenticationGatewayResponse struct {
		NextStep    PrepareCardAuthenticationGatewayNextStep
		RawResponse []byte
	}
)

func (p *PreparedCardAuthenticationGatewayRequest) Send(ctx context.Context) (*PrepareCardAuthenticationGatewayResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("prepared card authentication request is nil")
	}
	if p.send == nil {
		return nil, fmt.Errorf("prepared card authentication request is missing send function")
	}
	return p.send(ctx)
}

type AuthenticateCardholderGatewayNextStep string

const (
	AuthenticateCardholderGatewayNextStepCapture      AuthenticateCardholderGatewayNextStep = "capture"
	AuthenticateCardholderGatewayNextStepChallenge    AuthenticateCardholderGatewayNextStep = "challenge"
	AuthenticateCardholderGatewayNextStepCantContinue AuthenticateCardholderGatewayNextStep = "cant_continue"
)

type (
	AuthenticateCardholderGatewayRequest struct {
		InvoiceID               uuid.UUID
		Amount                  decimal.Decimal
		Currency                string
		PaymentMethodReference  PaymentMethodReference
		AuthenticationReference AuthenticationReference
		ChallengeReturnURL      string
		Browser                 ThreeDSBrowser
	}
	PreparedAuthenticateCardholderGatewayRequest struct {
		RawRequest []byte
		send       func(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error)
	}
	AuthenticateCardholderGatewayResponse struct {
		NextStep     AuthenticateCardholderGatewayNextStep
		RedirectHTML string
		RawResponse  []byte
	}
)

func (p *PreparedAuthenticateCardholderGatewayRequest) Send(ctx context.Context) (*AuthenticateCardholderGatewayResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("prepared authenticate cardholder request is nil")
	}
	if p.send == nil {
		return nil, fmt.Errorf("prepared authenticate cardholder request is missing send function")
	}
	return p.send(ctx)
}

type CardAuthenticationResult string

const (
	CardAuthenticationResultProceed      CardAuthenticationResult = "proceed"
	CardAuthenticationResultPending      CardAuthenticationResult = "pending"
	CardAuthenticationResultCantContinue CardAuthenticationResult = "cant_continue"
)

type (
	GetCardAuthenticationResultGatewayRequest struct {
		InvoiceID               uuid.UUID
		Amount                  decimal.Decimal
		Currency                string
		AuthenticationReference AuthenticationReference
	}
	PreparedGetCardAuthenticationResultGatewayRequest struct {
		RawRequest []byte
		send       func(ctx context.Context) (*GetCardAuthenticationResultGatewayResponse, error)
	}
	GetCardAuthenticationResultGatewayResponse struct {
		Result      CardAuthenticationResult
		RawResponse []byte
	}
)

func (p *PreparedGetCardAuthenticationResultGatewayRequest) Send(ctx context.Context) (*GetCardAuthenticationResultGatewayResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("prepared get card authentication result request is nil")
	}
	if p.send == nil {
		return nil, fmt.Errorf("prepared get card authentication result request is missing send function")
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
