package membership

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

var ErrInvalidArgument = errors.New("invalid argument")

type MembershipService interface {
	GetForCustomer(ctx context.Context, r GetForCustomerRequest) (*GetForCustomerResponse, error)
}

type service struct {
	db      *pgxpool.Pool
	queries store.TransactionQuerier
}

func New(dbPool *pgxpool.Pool, queries store.TransactionQuerier) MembershipService {
	return &service{
		db:      dbPool,
		queries: queries,
	}
}

func (s *service) GetForCustomer(ctx context.Context, r GetForCustomerRequest) (*GetForCustomerResponse, error) {
	return &GetForCustomerResponse{
		Organizations: []OrganizationWithProjects{},
	}, nil
}
