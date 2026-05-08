package workspacev1

import (
	"context"
	"errors"
	"strconv"

	"connectrpc.com/connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/membership"
	saftajacontext "github.com/rbconsult-bh/saftaja/khazina/internal/pkg/context"
	workspacepbv1 "github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/workspace/v1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/workspace/v1/workspacepbv1connect"
	"github.com/rs/zerolog/log"
)

var errInternal = errors.New("internal server error")

type service struct {
	membership membership.MembershipService
}

func New(membership membership.MembershipService) workspacepbv1connect.WorkspaceServiceHandler {
	return &service{
		membership: membership,
	}
}

func (s *service) GetWorkspace(ctx context.Context, r *connect.Request[workspacepbv1.GetWorkspaceRequest]) (*connect.Response[workspacepbv1.GetWorkspaceResponse], error) {
	customerID, err := saftajacontext.CustomerID(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get customer id from context")
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	resp, err := s.membership.GetForCustomer(ctx, membership.GetForCustomerRequest{
		CustomerID: customerID.String(),
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to call membership.GetForCustomer")
		return nil, connect.NewError(connect.CodeInternal, errInternal)
	}

	return &connect.Response[workspacepbv1.GetWorkspaceResponse]{
		Msg: &workspacepbv1.GetWorkspaceResponse{
			Organizations: []*workspacepbv1.Organization{
				{
					Id:       customerID.String(),
					Name:     strconv.Itoa(len(resp.Organizations)),
					Role:     0,
					Projects: []*workspacepbv1.Project{},
				},
			},
		},
	}, nil
}
