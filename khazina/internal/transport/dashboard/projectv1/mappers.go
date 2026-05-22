package projectv1

import (
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	projectpbv1 "github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/project/v1"
)

func mapConnectorType(ct domain.ConnectorType) projectpbv1.Gateway_ConnectorType {
	switch ct {
	case domain.ConnectorTypeMPGS:
		return projectpbv1.Gateway_CONNECTOR_TYPE_MPGS
	default:
		return projectpbv1.Gateway_CONNECTOR_TYPE_UNSPECIFIED
	}
}
