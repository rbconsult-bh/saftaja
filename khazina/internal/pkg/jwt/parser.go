package jwt

import (
	"encoding/base64"
	"fmt"

	golangjwt "github.com/golang-jwt/jwt/v5"
)

func ParseJWTPrivateKey(privKeyB64 string) (*Issuer, *Verifier, error) {
	pemBytes, err := base64.StdEncoding.DecodeString(privKeyB64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode pem base64 private key: %w", err)
	}

	key, err := golangjwt.ParseRSAPrivateKeyFromPEM(pemBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse rsa private key from pem: %w", err)
	}

	return NewIssuer(key), NewVerifier(&key.PublicKey), nil
}
