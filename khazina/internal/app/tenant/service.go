package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type Project struct {
	ID           uuid.UUID
	Name         string
	CustomDomain string
}

type Service interface {
	GetProjectByDomain(ctx context.Context, domain string) (*Project, error)
	IsDomainValid(ctx context.Context, domain string) (bool, error)
}

type service struct {
	queries store.TransactionQuerier
}

func NewService(queries store.TransactionQuerier) Service {
	return &service{queries: queries}
}

func (s *service) GetProjectByDomain(ctx context.Context, domain string) (*Project, error) {
	sp, err := s.queries.GetProjectByCustomDomain(ctx,
		pgtype.Text{String: domain, Valid: true})
	if err != nil {
		return nil, err
	}
	return &Project{
		ID:           sp.ID,
		Name:         sp.Name,
		CustomDomain: sp.CustomDomain.String,
	}, nil
}

func (s *service) IsDomainValid(ctx context.Context, domain string) (bool, error) {
	_, err := s.queries.GetProjectByCustomDomain(ctx,
		pgtype.Text{String: domain, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
