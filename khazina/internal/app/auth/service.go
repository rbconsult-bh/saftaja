package auth

import (
	"context"
	"crypto/sha256"
	"errors"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/clients/email"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rs/zerolog/log"
)

type AuthService interface {
	InitiateAuth(ctx context.Context, r InitiateAuthRequest) (*InitiateAuthResponse, error)
	CompleteAuth(ctx context.Context, r CompleteAuthRequest) (*CompleteAuthResponse, error)
}

type service struct {
	queries        store.Querier
	emailer        email.Emailer
	emailTemplates email.Templates
}

func New(queries store.Querier, emailer email.Emailer, emailTemplates email.Templates) AuthService {
	return &service{
		queries:        queries,
		emailer:        emailer,
		emailTemplates: emailTemplates,
	}
}

func (s *service) InitiateAuth(ctx context.Context, r InitiateAuthRequest) (*InitiateAuthResponse, error) {
	token := uuid.New()
	tokenHash := sha256.Sum256([]byte(token.String()))

	err := s.queries.CreateAuthIntent(ctx, store.CreateAuthIntentParams{
		Email:     r.Email,
		TokenHash: tokenHash[:],
	})
	if err != nil {
		return nil, errors.New("failed to create auth intent")
	}

	magicLinkTemplate, err := s.emailTemplates.MagicLinkTemplate(token.String())
	if err != nil {
		return nil, errors.New("failed to create magic link template")
	}

	err = s.emailer.SendFromTemplate(ctx, email.FromEmail_NoReplyEmail, r.Email, *magicLinkTemplate)
	if err != nil {
		return nil, errors.New("failed to send email from template")
	}

	return &InitiateAuthResponse{}, nil
}

func (s *service) CompleteAuth(ctx context.Context, r CompleteAuthRequest) (*CompleteAuthResponse, error) {
	tokenHash := sha256.Sum256([]byte(r.Token))

	isTokenHashValid, err := s.queries.IsTokenHashValid(ctx, tokenHash[:])
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get auth intent by token hash")
		return nil, errors.New("failed to get auth intent by token hash")
	}
	if !isTokenHashValid {
		log.Ctx(ctx).Info().Msg("token is invalid")
		return nil, errors.New("token is invalid")
	}

	// TODO: mint a pair of tokens for the user

	return &CompleteAuthResponse{
		AccessToken:  "fake good tokens",
		RefreshToken: "fake good tokens",
	}, nil
}
