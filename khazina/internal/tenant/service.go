package tenant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rbconsult-bh/saftaja/internal/store"
)

// Service handles tenant/project resolution from custom domains and slugs.
type Service interface {
	// GetProjectByDomain returns the project for a custom domain or slug-based subdomain.
	// Used by tenant resolution middleware.
	GetProjectByDomain(ctx context.Context, domain string) (*store.Project, error)

	// IsDomainValid checks if a domain is registered as a project's custom_domain or slug.
	// Used by CORS middleware and verify-domain endpoint.
	IsDomainValid(ctx context.Context, domain string) (bool, error)
}

type service struct {
	queries    *store.Queries
	baseDomain string // e.g. "saftaja.xyz"
}

// NewService creates a new tenant service.
// baseDomain is used to resolve slug-based subdomains (e.g. "saftaja.xyz" → "shop.saftaja.xyz").
func NewService(queries *store.Queries, baseDomain string) Service {
	return &service{queries: queries, baseDomain: baseDomain}
}

func (s *service) GetProjectByDomain(ctx context.Context, domain string) (*store.Project, error) {
	project, err := s.queries.GetProjectByCustomDomain(ctx,
		pgtype.Text{String: domain, Valid: true})
	if err == nil {
		return &project, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if s.baseDomain != "" {
		suffix := fmt.Sprintf(".%s", s.baseDomain)
		if strings.HasSuffix(domain, suffix) {
			slug := strings.TrimSuffix(domain, suffix)
			if slug != "" && !strings.Contains(slug, ".") {
				proj, err := s.queries.GetProjectBySlug(ctx,
					pgtype.Text{String: slug, Valid: true})
				if err == nil {
					return &proj, nil
				}
			}
		}
	}

	return nil, pgx.ErrNoRows
}

func (s *service) IsDomainValid(ctx context.Context, domain string) (bool, error) {
	_, err := s.GetProjectByDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
