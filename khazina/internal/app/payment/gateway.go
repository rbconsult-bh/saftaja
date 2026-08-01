package payment

import (
	"context"
	"errors"
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

var ErrGatewayResponseMismatch = errors.New("gateway response does not match request")

type GatewayResponseError struct {
	Err         error
	RawResponse []byte
}

func (e *GatewayResponseError) Error() string {
	return e.Err.Error()
}

func (e *GatewayResponseError) Unwrap() error {
	return e.Err
}

func rawGatewayResponse(err error) []byte {
	var responseErr *GatewayResponseError
	if errors.As(err, &responseErr) {
		return responseErr.RawResponse
	}
	return nil
}

type CardGateway interface {
	SetupCardPaymentMethod(ctx context.Context, r SetupCardPaymentMethodGatewayRequest) (*SetupCardPaymentMethodGatewayResponse, error)
	PrepareCardAuthentication(ctx context.Context, r PrepareCardAuthenticationGatewayRequest) (*PreparedCardAuthenticationGatewayRequest, error)
	AuthenticateCardholder(ctx context.Context, r AuthenticateCardholderGatewayRequest) (*PreparedAuthenticateCardholderGatewayRequest, error)
	GetCardAuthenticationResult(ctx context.Context, r GetCardAuthenticationResultGatewayRequest) (*PreparedGetCardAuthenticationResultGatewayRequest, error)
	CaptureCardPayment(ctx context.Context, r CaptureCardPaymentGatewayRequest) (*PreparedCaptureCardPaymentGatewayRequest, error)
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

type PrepareCardAuthenticationGatewayResult string

const (
	PrepareCardAuthenticationGatewayResultAvailable   PrepareCardAuthenticationGatewayResult = "available"
	PrepareCardAuthenticationGatewayResultUnavailable PrepareCardAuthenticationGatewayResult = "unavailable"
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
		Result      PrepareCardAuthenticationGatewayResult
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

type AuthenticateCardholderGatewayResult string

const (
	AuthenticateCardholderGatewayResultSucceeded         AuthenticateCardholderGatewayResult = "succeeded"
	AuthenticateCardholderGatewayResultChallengeRequired AuthenticateCardholderGatewayResult = "challenge_required"
	AuthenticateCardholderGatewayResultFailed            AuthenticateCardholderGatewayResult = "failed"
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
		Result       AuthenticateCardholderGatewayResult
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
	CardAuthenticationResultSucceeded CardAuthenticationResult = "succeeded"
	CardAuthenticationResultPending   CardAuthenticationResult = "pending"
	CardAuthenticationResultFailed    CardAuthenticationResult = "failed"
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

type CaptureCardPaymentGatewayResult string

const (
	CaptureCardPaymentGatewayResultSucceeded CaptureCardPaymentGatewayResult = "succeeded"
	CaptureCardPaymentGatewayResultPending   CaptureCardPaymentGatewayResult = "pending"
	CaptureCardPaymentGatewayResultUnknown   CaptureCardPaymentGatewayResult = "unknown"
	CaptureCardPaymentGatewayResultDeclined  CaptureCardPaymentGatewayResult = "declined"
)

type (
	CaptureCardPaymentGatewayRequest struct {
		InvoiceID               uuid.UUID
		PaymentReference        uuid.UUID
		Amount                  decimal.Decimal
		Currency                string
		PaymentMethodReference  PaymentMethodReference
		AuthenticationReference AuthenticationReference
	}
	PreparedCaptureCardPaymentGatewayRequest struct {
		RawRequest []byte
		send       func(ctx context.Context) (*CaptureCardPaymentGatewayResponse, error)
	}
	CaptureCardPaymentGatewayResponse struct {
		Result      CaptureCardPaymentGatewayResult
		RawResponse []byte
	}
)

func (p *PreparedCaptureCardPaymentGatewayRequest) Send(ctx context.Context) (*CaptureCardPaymentGatewayResponse, error) {
	if p == nil {
		return nil, fmt.Errorf("prepared capture card payment request is nil")
	}
	if p.send == nil {
		return nil, fmt.Errorf("prepared capture card payment request is missing send function")
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
