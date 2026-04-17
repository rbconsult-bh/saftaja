package authv1

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/auth"
	saftajacontext "github.com/rbconsult-bh/saftaja/khazina/internal/pkg/context"
	authpbv1 "github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1/authpbv1connect"
	"github.com/rs/zerolog/log"
)

var errInternal = errors.New("internal server error")

type service struct {
	auth auth.AuthService
}

func New(auth auth.AuthService) authpbv1connect.AuthServiceHandler {
	return &service{
		auth: auth,
	}
}

func (s *service) InitiateAuth(ctx context.Context, r *connect.Request[authpbv1.InitiateAuthRequest]) (*connect.Response[authpbv1.InitiateAuthResponse], error) {
	_, err := s.auth.InitiateAuth(ctx, auth.InitiateAuthRequest{
		Email: r.Msg.Email,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to call InitiateAuth")
		return nil, connect.NewError(connect.CodeInternal, errInternal)
	}

	return &connect.Response[authpbv1.InitiateAuthResponse]{
		Msg: &authpbv1.InitiateAuthResponse{},
	}, nil
}

func (s *service) CompleteAuth(ctx context.Context, r *connect.Request[authpbv1.CompleteAuthRequest]) (*connect.Response[authpbv1.CompleteAuthResponse], error) {
	resp, err := s.auth.CompleteAuth(ctx, auth.CompleteAuthRequest{
		Token: r.Msg.Token,
	})
	if err != nil {
		if errors.Is(err, auth.ErrTokenInvalid) {
			log.Ctx(ctx).Info().Msg("invalid auth token attempt")
			return nil, connect.NewError(connect.CodeUnauthenticated, nil)
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to call CompleteAuth")
		return nil, connect.NewError(connect.CodeInternal, errInternal)
	}

	return &connect.Response[authpbv1.CompleteAuthResponse]{
		Msg: &authpbv1.CompleteAuthResponse{
			AccessToken:  resp.AccessToken,
			RefreshToken: resp.RefreshToken,
		},
	}, nil
}

func (s *service) RefreshToken(ctx context.Context, r *connect.Request[authpbv1.RefreshTokenRequest]) (*connect.Response[authpbv1.RefreshTokenResponse], error) {
	resp, err := s.auth.RefreshToken(ctx, auth.RefreshTokenRequest{
		Token: r.Msg.RefreshToken,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to call RefreshToken")
		return nil, connect.NewError(connect.CodeInternal, errInternal)
	}

	return &connect.Response[authpbv1.RefreshTokenResponse]{
		Msg: &authpbv1.RefreshTokenResponse{
			AccessToken:  resp.AccessToken,
			RefreshToken: resp.RefreshToken,
		},
	}, nil
}

func (s *service) Logout(ctx context.Context, r *connect.Request[authpbv1.LogoutRequest]) (*connect.Response[authpbv1.LogoutResponse], error) {
	sessionID, err := saftajacontext.SessionID(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get session id from context")
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	customerID, err := saftajacontext.CustomerID(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get customer id from context")
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	_, err = s.auth.Logout(ctx, auth.LogoutRequest{
		SessionID:  sessionID,
		CustomerID: customerID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to call Logout")
		return nil, connect.NewError(connect.CodeInternal, errInternal)
	}

	return &connect.Response[authpbv1.LogoutResponse]{
		Msg: &authpbv1.LogoutResponse{},
	}, nil
}
