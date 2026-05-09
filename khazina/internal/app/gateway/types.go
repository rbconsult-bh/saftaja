package gateway

import (
	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
)

type GatewayCredentials struct {
	GatewayAccountID uuid.UUID
	ConnectorType    domain.ConnectorType
	BaseURL          string
	MerchantID       string
	APIPassword      string
	PaymentMethods   []string
}
