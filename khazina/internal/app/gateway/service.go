package gateway

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type GatewayService interface {
	CreateGateway(ctx context.Context, r CreateGatewayRequest) (*CreateGatewayResponse, error)
	ListGateways(ctx context.Context, r ListGatewaysRequest) (*ListGatewaysResponse, error)
}

type service struct {
	db      *pgxpool.Pool
	queries store.TransactionQuerier
}

func New(db *pgxpool.Pool, queries store.TransactionQuerier) GatewayService {
	return &service{
		db:      db,
		queries: queries,
	}
}

func (s *service) CreateGateway(ctx context.Context, r CreateGatewayRequest) (*CreateGatewayResponse, error) {
	return &CreateGatewayResponse{}, nil
}

func (s *service) ListGateways(ctx context.Context, r ListGatewaysRequest) (*ListGatewaysResponse, error) {
	return &ListGatewaysResponse{}, nil
}
