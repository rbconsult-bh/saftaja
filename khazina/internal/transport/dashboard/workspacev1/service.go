package workspacev1

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	workspacepbv1 "github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/workspace/v1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/workspace/v1/workspacepbv1connect"
)

var errInternal = errors.New("internal server error")

type service struct{}

func New() workspacepbv1connect.WorkspaceServiceClient {
	return &service{}
}

func (s *service) GetWorkspace(ctx context.Context, r *connect.Request[workspacepbv1.GetWorkspaceRequest]) (*connect.Response[workspacepbv1.GetWorkspaceResponse], error) {
	return &connect.Response[workspacepbv1.GetWorkspaceResponse]{}, nil
}
