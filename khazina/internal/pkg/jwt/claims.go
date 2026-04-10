package jwt

import "github.com/google/uuid"

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Claims struct {
	Sub       uuid.UUID
	SessionID uuid.UUID
	Jti       uuid.UUID
	Type      TokenType
}
