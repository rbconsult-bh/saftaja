package dashboard

import (
	"context"
	"errors"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	saftajacontext "github.com/rbconsult-bh/saftaja/khazina/internal/pkg/context"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/jwt"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1/authpbv1connect"
	"github.com/rs/zerolog/log"
)

var ErrInternal = errors.New("internal server error")

var authlessProcedures = []string{
	authpbv1connect.AuthServiceInitiateAuthProcedure,
	authpbv1connect.AuthServiceCompleteAuthProcedure,
	authpbv1connect.AuthServiceRefreshTokenProcedure,
}

func NewAuthTokenInterceptor(tv *jwt.Verifier) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			isAuthlessProcedure := slices.Contains(authlessProcedures, req.Spec().Procedure)
			if isAuthlessProcedure {
				log.Ctx(ctx).Info().Msg("endpoint is authless")
				return next(ctx, req)
			}

			authHeader := req.Header().Get("authorization")
			token, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found {
				log.Ctx(ctx).Error().Msg("'Bearer ' prefix not found in authorization header")
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("ensure token has 'Bearer ' before its value in header"))
			}

			claims, err := tv.Verify(token)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to verify token")
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}

			if err := claims.MustBeAccess(); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("token sent is not access token")
				return nil, connect.NewError(connect.CodePermissionDenied, nil)
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to parse subject claim as uuid")
				return nil, connect.NewError(connect.CodeInternal, ErrInternal)
			}
			ctx = saftajacontext.WithCustomerID(ctx, userID)

			sessionID, err := uuid.Parse(claims.SessionID)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to parse session id claim as uuid")
				return nil, connect.NewError(connect.CodeInternal, ErrInternal)
			}
			ctx = saftajacontext.WithSessionID(ctx, sessionID)

			log.Ctx(ctx).Info().Msg("endpoint is authful, all is good")
			return next(ctx, req)
		}
	}
}
