package jwt

import (
	"crypto/rsa"
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Issuer struct {
	key *rsa.PrivateKey
}

func NewIssuer(key *rsa.PrivateKey) *Issuer {
	return &Issuer{
		key: key,
	}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (i *Issuer) IssueTokenPair(sub, sid, refreshJti uuid.UUID) (*TokenPair, error) {
	now := time.Now()

	accessToken, err := i.issue(golangjwt.MapClaims{
		"type": "access",
		"sub":  sub.String(),
		"sid":  sid.String(),
		"iat":  now.Unix(),
		"exp":  now.Add(5 * time.Minute).Unix(),
	})
	if err != nil {
		return nil, err
	}

	refreshToken, err := i.issue(golangjwt.MapClaims{
		"type": "refresh",
		"sub":  sub.String(),
		"sid":  sid.String(),
		"jti":  refreshJti.String(),
		"iat":  now.Unix(),
		"exp":  now.Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (i *Issuer) issue(mapClaims golangjwt.MapClaims) (string, error) {
	token := golangjwt.NewWithClaims(
		golangjwt.SigningMethodRS256,
		mapClaims,
	)

	return token.SignedString(i.key)
}
