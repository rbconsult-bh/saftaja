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
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/jwt"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rs/zerolog/log"
)

var (
	ErrAuthIntentTokenInvalid = errors.New("auth intent token invalid")
	ErrSessionExpired         = errors.New("session expired")
	ErrSessionRevoked         = errors.New("session revoked")
	ErrInvalidArgument        = errors.New("invalid argument")
)

type AuthService interface {
	InitiateAuth(ctx context.Context, r InitiateAuthRequest) (*InitiateAuthResponse, error)
	CompleteAuth(ctx context.Context, r CompleteAuthRequest) (*CompleteAuthResponse, error)
	RefreshToken(ctx context.Context, r RefreshTokenRequest) (*RefreshTokenResponse, error)
	Logout(ctx context.Context, r LogoutRequest) (*LogoutResponse, error)
}

type service struct {
	db             *pgxpool.Pool
	queries        store.TransactionQuerier
	emailer        email.Emailer
	emailTemplates email.Templates
	jwtIssuer      *jwt.Issuer
	jwtVerifier    *jwt.Verifier
}

func New(dbPool *pgxpool.Pool, queries store.TransactionQuerier, emailer email.Emailer, emailTemplates email.Templates, jwtIssuer *jwt.Issuer, jwtVerifier *jwt.Verifier) AuthService {
	return &service{
		db:             dbPool,
		queries:        queries,
		emailer:        emailer,
		emailTemplates: emailTemplates,
		jwtIssuer:      jwtIssuer,
		jwtVerifier:    jwtVerifier,
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
			return nil, ErrAuthIntentTokenInvalid
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

	if createCustomerResp.NeedsDefaultOrg {
		log.Ctx(ctx).Info().Msg("user is new, creating default organization and default project")
		orgID, err := queriesWithTx.CreateOrganizationForCustomer(ctx, store.CreateOrganizationForCustomerParams{
			Name:       fmt.Sprintf("%s's Organization", name),
			CustomerID: createCustomerResp.ID,
			Role:       store.OrganizationRoleOwner,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to create create default organization for customer")
			return nil, errors.New("failed to create create default organization for customer")
		}

		_, err = queriesWithTx.CreateProjectForOrganization(ctx, store.CreateProjectForOrganizationParams{
			OrganizationID: orgID,
			Name:           "Default Project",
			Environment:    store.ProjectEnvironmentSandbox,
			// TODO: we need to pass slug, this way it uses our domain as default and custom_domain is extra user can set later.
			CustomDomain: nil,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to create create default project for customer")
			return nil, errors.New("failed to create create default project for customer")
		}
	}

	currentJti := uuid.New()
	currentJtiHash := sha256.Sum256([]byte(currentJti.String()))

	customerSession, err := queriesWithTx.CreateCustomerSession(ctx, store.CreateCustomerSessionParams{
		CustomerID:     createCustomerResp.ID,
		CurrentJtiHash: currentJtiHash[:],
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create customer session")
		return nil, errors.New("failed to create customer session")
	}

	tokenPair, err := s.jwtIssuer.IssueTokenPair(customerSession.CustomerID, customerSession.ID, currentJti)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to issue token pair")
		return nil, errors.New("failed to issue token pair")
	}

	if err := tx.Commit(ctx); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to commit tx")
		return nil, errors.New("failed to commit tx")
	}

	return &CompleteAuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

func (s *service) RefreshToken(ctx context.Context, r RefreshTokenRequest) (*RefreshTokenResponse, error) {
	claims, err := s.jwtVerifier.Verify(r.Token)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			log.Ctx(ctx).Error().Err(err).Msg("session expired")
			return nil, ErrSessionExpired
		}
		if errors.Is(err, jwt.ErrTokenMalformed) {
			log.Ctx(ctx).Error().Err(err).Msg("token malformed")
			return nil, ErrInvalidArgument
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to verify refresh token")
		return nil, errors.New("failed to verify refresh token")
	}

	if err := claims.MustBeRefresh(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("token sent is not refresh token")
		return nil, errors.New("token sent is not refresh token")
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse session id")
		return nil, errors.New("failed to parse session id")
	}

	customerID, err := uuid.Parse(claims.RegisteredClaims.Subject)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse customer id")
		return nil, errors.New("failed to parse customer id")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to begin tx")
		return nil, errors.New("failed to begin tx")
	}
	defer tx.Rollback(ctx)

	queriesWithTx := s.queries.WithTx(tx)

	newJti := uuid.New()
	newJtiHash := sha256.Sum256([]byte(newJti.String()))
	currentJtiHash := sha256.Sum256([]byte(claims.RegisteredClaims.ID))

	resRow, err := queriesWithTx.RotateOrRevokeCustomerSession(ctx, store.RotateOrRevokeCustomerSessionArgs{
		SessionID:      sessionID,
		CustomerID:     customerID,
		NewJtiHash:     newJtiHash[:],
		CurrentJtiHash: currentJtiHash[:],
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update customer session jti hash by id and customer id")
		return nil, errors.New("failed to update customer session jti hash by id and customer id")
	}

	if resRow.WasRotated {
		tokenPair, err := s.jwtIssuer.IssueTokenPair(customerID, sessionID, newJti)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to issue token pair")
			return nil, errors.New("failed to issue token pair")
		}

		if err := tx.Commit(ctx); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to commit tx")
			return nil, errors.New("failed to commit tx")
		}

		return &RefreshTokenResponse{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
		}, nil
	}

	if resRow.WasRevoked {
		log.Ctx(ctx).Error().Msg("session revoked: old refresh token used (potential replay attack)")
		return nil, ErrSessionRevoked
	}

	log.Ctx(ctx).Error().Msg("session expired or not found")
	return nil, ErrSessionExpired
}

func (s *service) Logout(ctx context.Context, r LogoutRequest) (*LogoutResponse, error) {
	err := s.queries.DeleteCustomerSessionByIDAndCustomerID(ctx, store.DeleteCustomerSessionByIDAndCustomerIDParams{
		ID:         r.SessionID,
		CustomerID: r.CustomerID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to delete customer session by id and customer id")
		return nil, errors.New("failed to delete customer sessoin by and and customer id")
	}

	return &LogoutResponse{}, nil
}
