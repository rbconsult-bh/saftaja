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

	accessToken, err := i.issue(Claims{
		SessionID: sid.String(),
		Type:      TokenTypeAccess,
		RegisteredClaims: golangjwt.RegisteredClaims{
			Subject:   sub.String(),
			IssuedAt:  golangjwt.NewNumericDate(now),
			ExpiresAt: golangjwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
	})
	if err != nil {
		return nil, err
	}

	refreshToken, err := i.issue(Claims{
		SessionID: sid.String(),
		Type:      TokenTypeRefresh,
		RegisteredClaims: golangjwt.RegisteredClaims{
			ID:        refreshJti.String(),
			Subject:   sub.String(),
			IssuedAt:  golangjwt.NewNumericDate(now),
			ExpiresAt: golangjwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
		},
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (i *Issuer) issue(claims Claims) (string, error) {
	token := golangjwt.NewWithClaims(
		golangjwt.SigningMethodRS256,
		claims,
	)

	return token.SignedString(i.key)
}
