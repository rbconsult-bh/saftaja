package gateway

import (
	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
)

type GatewayCredentials struct {
	GatewayAccountID uuid.UUID
	AccountName     string
	ConnectorType   domain.ConnectorType
	BaseURL         string
	MerchantID      string
	APIPassword     string
	PaymentMethods  []string
}

type CreateGatewayRequest struct {
	ProjectID      uuid.UUID
	AccountName    string
	ConnectorType  domain.ConnectorType
	Credentials    any
	PaymentMethods []string
}

type GatewayAccount struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	AccountName      string
	ConnectorType    domain.ConnectorType
	IsActive         bool
	PaymentMethods   []string
}
