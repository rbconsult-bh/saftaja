package projectv1

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/gateway"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/membership"
	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	saftajacontext "github.com/rbconsult-bh/saftaja/khazina/internal/pkg/context"
	projectpbv1 "github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/project/v1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/project/v1/projectpbv1connect"
)

var errInternal = connect.NewError(connect.CodeInternal, errors.New("internal server error"))

type service struct {
	gateway    gateway.Service
	membership membership.MembershipService
}

func New(gateway gateway.Service, membership membership.MembershipService) projectpbv1connect.ProjectServiceHandler {
	return &service{
		gateway:    gateway,
		membership: membership,
	}
}

func (s *service) ListGateways(ctx context.Context, r *connect.Request[projectpbv1.ListGatewaysRequest]) (*connect.Response[projectpbv1.ListGatewaysResponse], error) {
	customerID, err := saftajacontext.CustomerID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	projectID, err := uuid.Parse(r.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid project_id"))
	}

	if err := s.membership.VerifyProjectAccess(ctx, customerID, projectID); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, nil)
	}

	gateways, err := s.gateway.ListActiveByProject(ctx, projectID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to list gateways")
		return nil, errInternal
	}

	var pbGateways []*projectpbv1.Gateway
	for _, gw := range gateways {
		pbGateways = append(pbGateways, &projectpbv1.Gateway{
			Id:            gw.GatewayAccountID.String(),
			AccountName:   gw.AccountName,
			ConnectorType: mapConnectorType(gw.ConnectorType),
			IsActive:      true,
		})
	}

	return &connect.Response[projectpbv1.ListGatewaysResponse]{
		Msg: &projectpbv1.ListGatewaysResponse{Gateways: pbGateways},
	}, nil
}

func (s *service) CreateGateway(ctx context.Context, r *connect.Request[projectpbv1.CreateGatewayRequest]) (*connect.Response[projectpbv1.CreateGatewayResponse], error) {
	customerID, err := saftajacontext.CustomerID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	projectID, err := uuid.Parse(r.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid project_id"))
	}

	if err := s.membership.VerifyProjectAccess(ctx, customerID, projectID); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, nil)
	}

	gwReq := gateway.CreateGatewayRequest{
		ProjectID:     projectID,
		AccountName:   r.Msg.AccountName,
		ConnectorType: domain.ConnectorTypeMPGS,
	}

	switch creds := r.Msg.Credentials.(type) {
	case *projectpbv1.CreateGatewayRequest_Mpgs:
		gwReq.Config = &mpgsclient.Config{
			MerchantID: creds.Mpgs.MerchantId,
			BaseURL:    creds.Mpgs.BaseUrl,
		}
		gwReq.Secret = &mpgsclient.Secret{
			APIPassword: creds.Mpgs.ApiPassword,
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported connector type"))
	}

	_, err = s.gateway.Create(ctx, gwReq)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create gateway")
		return nil, errInternal
	}

	return &connect.Response[projectpbv1.CreateGatewayResponse]{
		Msg: &projectpbv1.CreateGatewayResponse{},
	}, nil
}
