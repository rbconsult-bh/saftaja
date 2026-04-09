package authv1

import (
	"context"

	"connectrpc.com/connect"
	authpbv1 "github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1/authpbv1connect"
)

type service struct{}

func New() authpbv1connect.AuthServiceHandler {
	return &service{}
}

func (s *service) InitiateAuth(context.Context, *connect.Request[authpbv1.InitiateAuthRequest]) (*connect.Response[authpbv1.InitiateAuthResponse], error) {
	panic("unimplemented")
}

func (s *service) CompleteAuth(context.Context, *connect.Request[authpbv1.CompleteAuthRequest]) (*connect.Response[authpbv1.CompleteAuthResponse], error) {
	panic("unimplemented")
}

func (s *service) RefreshToken(context.Context, *connect.Request[authpbv1.RefreshTokenRequest]) (*connect.Response[authpbv1.RefreshTokenResponse], error) {
	panic("unimplemented")
}

func (s *service) Logout(context.Context, *connect.Request[authpbv1.LogoutRequest]) (*connect.Response[authpbv1.LogoutResponse], error) {
	panic("unimplemented")
}
