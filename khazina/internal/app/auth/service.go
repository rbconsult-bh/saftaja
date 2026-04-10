package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rbconsult-bh/saftaja/khazina/internal/clients/email"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rs/zerolog/log"
)

var ErrTokenInvalid = errors.New("token is invalid")

type AuthService interface {
	InitiateAuth(ctx context.Context, r InitiateAuthRequest) (*InitiateAuthResponse, error)
	CompleteAuth(ctx context.Context, r CompleteAuthRequest) (*CompleteAuthResponse, error)
}

type service struct {
	db             *pgxpool.Pool
	queries        store.TransactionQuerier
	emailer        email.Emailer
	emailTemplates email.Templates
}

func New(dbPool *pgxpool.Pool, queries store.TransactionQuerier, emailer email.Emailer, emailTemplates email.Templates) AuthService {
	return &service{
		db:             dbPool,
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

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to begin tx")
		return nil, errors.New("failed to begin tx")
	}
	defer tx.Rollback(ctx)

	queriesWithTx := s.queries.WithTx(tx)

	userEmail, err := queriesWithTx.ConsumeAuthIntentByTokenHash(ctx, tokenHash[:])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Ctx(ctx).Info().Msg("token is invalid")
			return nil, ErrTokenInvalid
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to consume auth intent by token hash")
		return nil, errors.New("failed to consume auth intent by token hash")
	}

	log.Ctx(ctx).Info().Str("email", userEmail).Msg("we are good, the token is right :D")

	name := strings.Split(userEmail, "@")[0]
	createCustomerResp, err := queriesWithTx.CreateCustomerIfNotExists(ctx, store.CreateCustomerIfNotExistsParams{
		Name:  name,
		Email: userEmail,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create customer if not exists")
		return nil, errors.New("failed to create customer if not exists")
	}

	if createCustomerResp.IsNewCustomer {
		log.Ctx(ctx).Info().Msg("user is new, creating default organization")
		_, err = queriesWithTx.CreateDefaultOrganizationForCustomer(ctx, store.CreateDefaultOrganizationForCustomerParams{
			Name:       fmt.Sprintf("%s's Organization", name),
			CustomerID: createCustomerResp.ID,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to create create default organization for customer")
			return nil, errors.New("failed to create create default organization for customer")
		}
	}

	currentJti := uuid.New()
	currentJtiHash := sha256.Sum256([]byte(currentJti.String()))

	err = queriesWithTx.CreateCustomerSession(ctx, store.CreateCustomerSessionParams{
		CustomerID:     createCustomerResp.ID,
		CurrentJtiHash: currentJtiHash[:],
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create customer session")
		return nil, errors.New("failed to create customer session")
	}

	if err := tx.Commit(ctx); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to commit tx")
		return nil, errors.New("failed to commit tx")
	}

	// TODO: mint a pair of tokens for the user

	return &CompleteAuthResponse{
		AccessToken:  "fake good tokens",
		RefreshToken: "fake good tokens",
	}, nil
}
