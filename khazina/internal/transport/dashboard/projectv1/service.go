package projectv1

import (
	"context"

	"connectrpc.com/connect"
	projectpbv1 "github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/project/v1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/project/v1/projectpbv1connect"
)

type service struct{}

func New() projectpbv1connect.ProjectServiceHandler {
	return &service{}
}

func (s *service) CreateGateway(ctx context.Context, r *connect.Request[projectpbv1.CreateGatewayRequest]) (*connect.Response[projectpbv1.CreateGatewayResponse], error) {
	return &connect.Response[projectpbv1.CreateGatewayResponse]{}, nil
}

func (s *service) ListGateways(ctx context.Context, r *connect.Request[projectpbv1.ListGatewaysRequest]) (*connect.Response[projectpbv1.ListGatewaysResponse], error) {
	return &connect.Response[projectpbv1.ListGatewaysResponse]{}, nil
}
