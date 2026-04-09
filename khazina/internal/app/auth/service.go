package auth

import "context"

type AuthService interface {
	InitiateAuth(ctx context.Context, r InitiateAuthRequest) (*InitiateAuthResponse, error)
	CompleteAuth(ctx context.Context, r CompleteAuthRequest) (*CompleteAuthResponse, error)
}

type service struct{}

func New() AuthService {
	return &service{}
}

func (s *service) InitiateAuth(ctx context.Context, r InitiateAuthRequest) (*InitiateAuthResponse, error) {
	return &InitiateAuthResponse{}, nil
}

func (s *service) CompleteAuth(ctx context.Context, r CompleteAuthRequest) (*CompleteAuthResponse, error) {
	return &CompleteAuthResponse{}, nil
}
