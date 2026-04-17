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
)

var ErrInternal = errors.New("internal server error")

var authlessEndpoints = []string{
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
			isAuthlessEndpoint := slices.Contains(authlessEndpoints, req.Spec().Procedure)
			if isAuthlessEndpoint {
				return next(ctx, req)
			}

			authHeader := req.Header().Get("authorization")
			token, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("ensure token has 'Bearer ' before its value in header"))
			}

			claims, err := tv.Verify(token)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}

			if err := claims.MustBeAccess(); err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, nil)
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, ErrInternal)
			}
			ctx = saftajacontext.WithCustomerID(ctx, userID)

			sessionID, err := uuid.Parse(claims.SessionID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, ErrInternal)
			}
			ctx = saftajacontext.WithSessionID(ctx, sessionID)

			return next(ctx, req)
		}
	}
}
