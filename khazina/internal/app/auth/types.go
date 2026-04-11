package auth

import "github.com/google/uuid"

type (
	InitiateAuthRequest struct {
		Email string
	}
	InitiateAuthResponse struct{}
)

type (
	CompleteAuthRequest struct {
		Token string
	}
	CompleteAuthResponse struct {
		AccessToken  string
		RefreshToken string
	}
)

type (
	RefreshTokenRequest struct {
		Token string
	}
	RefreshTokenResponse struct {
		AccessToken  string
		RefreshToken string
	}
)

type (
	LogoutRequest struct {
		CustomerSessionID uuid.UUID
		CustomerID        uuid.UUID
	}
	LogoutResponse struct{}
)
