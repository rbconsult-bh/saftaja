package payment

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type Service interface {
	VerifyDomain(ctx context.Context, domain string) (bool, error)
}

type service struct {
	pool          *pgxpool.Pool
	queries       store.TransactionQuerier
	encryptionKey []byte
}

func NewService(pool *pgxpool.Pool, queries store.TransactionQuerier, encryptionKey []byte) Service {
	return &service{
		pool:          pool,
		queries:       queries,
		encryptionKey: encryptionKey,
	}
}

func (s *service) VerifyDomain(ctx context.Context, domain string) (bool, error) {
	_, err := s.queries.GetProjectByCustomDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
