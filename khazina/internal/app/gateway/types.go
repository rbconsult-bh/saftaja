package gateway

import (
	"github.com/google/uuid"

	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
)

type PaymentMethodType string

const (
	PaymentMethodTypeCard     PaymentMethodType = "card"
	PaymentMethodTypeApplePay PaymentMethodType = "apple_pay"
)

type PaymentMethod struct {
	Type             PaymentMethodType
	GatewayAccountID uuid.UUID
	MPGSBaseURL      string
	MPGSMerchantID   string
	MPGSApiVersion   string
}

type GatewayCredentials struct {
	GatewayAccountID uuid.UUID
	AccountName      string
	ConnectorType    domain.ConnectorType
	BaseURL          string
	MerchantID       string
}

type CreateGatewayRequest struct {
	ProjectID     uuid.UUID
	AccountName   string
	ConnectorType domain.ConnectorType
	Config        *mpgsclient.Config
	Secret        *mpgsclient.Secret
}

type GatewayAccount struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	AccountName   string
	ConnectorType domain.ConnectorType
	IsActive      bool
}
