package tenant

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

// Service handles tenant/project resolution from custom domains.
type Service interface {
	// GetProjectByDomain returns the project for a custom domain.
	// Used by tenant resolution middleware.
	GetProjectByDomain(ctx context.Context, domain string) (*store.Project, error)

	// IsDomainValid checks if a domain is registered as a project's custom_domain.
	// Used by CORS middleware and verify-domain endpoint.
	IsDomainValid(ctx context.Context, domain string) (bool, error)
}

type service struct {
	queries store.TransactionQuerier
}

// NewService creates a new tenant service.
func NewService(queries store.TransactionQuerier) Service {
	return &service{queries: queries}
}

func (s *service) GetProjectByDomain(ctx context.Context, domain string) (*store.Project, error) {
	project, err := s.queries.GetProjectByCustomDomain(ctx,
		pgtype.Text{String: domain, Valid: true})
	if err != nil {
		return nil, err
	}
	return &project, nil
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
