package jwt

import (
	"crypto/rsa"
	"fmt"

	golangjwt "github.com/golang-jwt/jwt/v5"
)

type Verifier struct {
	key *rsa.PublicKey
}

func NewVerifier(key *rsa.PublicKey) *Verifier {
	return &Verifier{
		key: key,
	}
}

func (v *Verifier) Verify(token string) (*Claims, error) {
	var claims Claims
	parsedToken, err := golangjwt.ParseWithClaims(
		token,
		&claims,
		func(t *golangjwt.Token) (any, error) {
			return v.key, nil
		},
		golangjwt.WithValidMethods(
			[]string{golangjwt.SigningMethodRS256.Name},
		),
		golangjwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return &claims, nil
}
