package membership

import (
	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type (
	GetForCustomerRequest struct {
		CustomerID string
	}
	GetForCustomerResponse struct {
		Organizations []OrganizationWithProjects
	}
)

type OrganizationWithProjects struct {
	ID       uuid.UUID
	Name     string
	Role     store.OrganizationRole
	Projects []Project
}

type Project struct {
	ID          uuid.UUID
	Name        string
	Environment domain.ProjectEnvironment
}
