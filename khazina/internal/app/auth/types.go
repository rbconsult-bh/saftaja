package auth

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
