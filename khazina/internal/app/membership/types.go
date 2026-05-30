package membership

import (
	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type ProjectEnvironment string

const (
	EnvironmentSandbox    ProjectEnvironment = "sandbox"
	EnvironmentProduction ProjectEnvironment = "production"
	EnvironmentUnknown    ProjectEnvironment = "unknown"
)

type (
	GetForCustomerRequest struct {
		CustomerID uuid.UUID
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
	Environment ProjectEnvironment
}
