package membership

import (
	"context"
	"errors"

	"github.com/google/uuid"
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
	rows, err := s.queries.ListOrganizationsWithProjectsForCustomer(ctx, r.CustomerID)
	if err != nil {
		return nil, errors.New("failed to call ListOrganizationsWithProjectsForCustomer")
	}

	orgMap := make(map[uuid.UUID]OrganizationWithProjects)
	var orderedOrgIDs []uuid.UUID
	for _, row := range rows {
		if _, exists := orgMap[row.OrgID]; !exists {
			orderedOrgIDs = append(orderedOrgIDs, row.OrgID)

			orgMap[row.OrgID] = OrganizationWithProjects{
				ID:       row.OrgID,
				Name:     row.OrgName,
				Role:     row.Role,
				Projects: []Project{},
			}
		}

		org := orgMap[row.OrgID]
		org.Projects = append(org.Projects, Project{
			ID:          row.ProjectID,
			Name:        row.ProjectName.String,
			Environment: row.Environment,
		})
		orgMap[row.OrgID] = org
	}

	result := make([]OrganizationWithProjects, 0, len(orderedOrgIDs))
	for _, id := range orderedOrgIDs {
		result = append(result, orgMap[id])
	}

	return &GetForCustomerResponse{
		Organizations: result,
	}, nil
}
