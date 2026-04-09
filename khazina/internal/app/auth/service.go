package auth

import (
	"context"

	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type AuthService interface {
	InitiateAuth(ctx context.Context, r InitiateAuthRequest) (*InitiateAuthResponse, error)
	CompleteAuth(ctx context.Context, r CompleteAuthRequest) (*CompleteAuthResponse, error)
}

type service struct {
	queries store.Querier
}

func New(queries store.Querier) AuthService {
	return &service{
		queries: queries,
	}
}

func (s *service) InitiateAuth(ctx context.Context, r InitiateAuthRequest) (*InitiateAuthResponse, error) {
	return &InitiateAuthResponse{}, nil
}

func (s *service) CompleteAuth(ctx context.Context, r CompleteAuthRequest) (*CompleteAuthResponse, error) {
	return &CompleteAuthResponse{}, nil
}
