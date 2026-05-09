package workspacev1

import (
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/membership"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	workspacepbv1 "github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/workspace/v1"
)

func mapMembershipOrgWithProjsToProto(orgWithProjs membership.OrganizationWithProjects) *workspacepbv1.Organization {
	projects := make([]*workspacepbv1.Project, len(orgWithProjs.Projects))
	for idx, project := range orgWithProjs.Projects {
		projects[idx] = mapMembershipProjectToProto(project)
	}

	return &workspacepbv1.Organization{
		Id:       orgWithProjs.ID.String(),
		Name:     orgWithProjs.Name,
		Role:     mapStoreRoleToProto(orgWithProjs.Role),
		Projects: projects,
	}
}

func mapStoreRoleToProto(role store.OrganizationRole) workspacepbv1.Organization_Role {
	switch role {
	case store.OrganizationRoleOwner:
		return workspacepbv1.Organization_ROLE_OWNER
	case store.OrganizationRoleAdmin:
		return workspacepbv1.Organization_ROLE_ADMIN
	case store.OrganizationRoleMember:
		return workspacepbv1.Organization_ROLE_MEMBER
	default:
		return workspacepbv1.Organization_ROLE_UNSPECIFIED
	}
}

func mapMembershipProjectToProto(project membership.Project) *workspacepbv1.Project {
	return &workspacepbv1.Project{
		Id:          project.ID.String(),
		Name:        project.Name,
		Environment: mapProjectEnvironmentToProto(project.Environment),
	}
}

func mapProjectEnvironmentToProto(projEnv domain.ProjectEnvironment) workspacepbv1.Project_Environment {
	switch projEnv {
	case domain.ProjectEnvironmentSandbox:
		return workspacepbv1.Project_ENVIRONMENT_SANDBOX
	case domain.ProjectEnvironmentProduction:
		return workspacepbv1.Project_ENVIRONMENT_PRODUCTION
	default:
		return workspacepbv1.Project_ENVIRONMENT_UNSPECIFIED
	}
}
