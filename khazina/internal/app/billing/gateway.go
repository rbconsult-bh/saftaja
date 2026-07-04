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
	CreateSession(ctx context.Context, r CreateSessionRequest) (*CreateSessionResponse, error)
}

type (
	CreateSessionRequest struct {
		InvoiceID uuid.UUID
		Amount    decimal.Decimal
		Currency  string
	}
	CreateSessionResponse struct {
		GatewaySessionID string
	}
)

type gatewayResolver struct{}

func NewGatewayResolver() GatewayResolver {
	return &gatewayResolver{}
}

func (gr *gatewayResolver) CardGateway(account store.GatewayAccount) (CardGateway, error) {
	switch account.ConnectorType {
	case store.ConnectorTypeMPGS:
		return NewMPGSCardGateway(), nil
	default:
		return nil, fmt.Errorf("%w: connector_type: %s does not support card", ErrUnsupportedGateway, account.ConnectorType)
	}
}
