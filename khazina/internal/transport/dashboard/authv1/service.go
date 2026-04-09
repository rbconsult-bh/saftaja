package authv1

import (
	"context"

	"connectrpc.com/connect"
	authpbv1 "github.com/rbconsult-bh/saftaja/khazina/internal/proto/saftaja/dashboard/auth/v1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/proto/saftaja/dashboard/auth/v1/authpbv1connect"
)

type service struct{}

func New() authpbv1connect.AuthServiceHandler {
	return &service{}
}

// Register implements authpbv1connect.AuthServiceHandler.
func (s *service) Register(context.Context, *connect.Request[authpbv1.RegisterRequest]) (*connect.Response[authpbv1.RegisterResponse], error) {
	panic("unimplemented")
}

// InitiateLogin implements authpbv1connect.AuthServiceHandler.
func (s *service) InitiateLogin(context.Context, *connect.Request[authpbv1.InitiateLoginRequest]) (*connect.Response[authpbv1.InitiateLoginResponse], error) {
	panic("unimplemented")
}

// CompleteLogin implements authpbv1connect.AuthServiceHandler.
func (s *service) CompleteLogin(context.Context, *connect.Request[authpbv1.CompleteLoginRequest]) (*connect.Response[authpbv1.CompleteLoginResponse], error) {
	panic("unimplemented")
}

// RefreshToken implements authpbv1connect.AuthServiceHandler.
func (s *service) RefreshToken(context.Context, *connect.Request[authpbv1.RefreshTokenRequest]) (*connect.Response[authpbv1.RefreshTokenResponse], error) {
	panic("unimplemented")
}

// Logout implements authpbv1connect.AuthServiceHandler.
func (s *service) Logout(context.Context, *connect.Request[authpbv1.LogoutRequest]) (*connect.Response[authpbv1.LogoutResponse], error) {
	panic("unimplemented")
}
