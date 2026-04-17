package jwt

import (
	"fmt"

	golangjwt "github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Claims struct {
	SessionID string    `json:"sid"`
	Type      TokenType `json:"type"`

	golangjwt.RegisteredClaims
}

func (c *Claims) MustBeAccess() error {
	if c.Type != TokenTypeAccess {
		return fmt.Errorf("expected access token, got %s", c.Type)
	}
	return nil
}

func (c *Claims) MustBeRefresh() error {
	if c.Type != TokenTypeRefresh {
		return fmt.Errorf("expected refresh token, got %s", c.Type)
	}
	return nil
}
