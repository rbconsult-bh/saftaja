package auth

import (
	"context"
	"errors"

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
	// TODO: generate a random uuid, and save it as bytes

	err := s.queries.CreateAuthIntent(ctx, store.CreateAuthIntentParams{
		Email:     r.Email,
		TokenHash: []byte{},
	})
	if err != nil {
		return nil, errors.New("failed to create auth intent")
	}

	// TODO: send it via email :D

	return &InitiateAuthResponse{}, nil
}

func (s *service) CompleteAuth(ctx context.Context, r CompleteAuthRequest) (*CompleteAuthResponse, error) {
	return &CompleteAuthResponse{}, nil
}
