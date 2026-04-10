package jwt

import (
	"crypto/rsa"

	"github.com/google/uuid"
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
	return &Claims{
		Sub:       uuid.UUID{},
		SessionID: uuid.UUID{},
		Jti:       uuid.UUID{},
		Type:      "",
	}, nil
}
