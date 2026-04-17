package saftajacontext

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type (
	userIDKey    struct{}
	sessionIDKey struct{}
)

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey{}, id)
}

func UserID(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("user id not found in context")
	}
	return id, nil
}

func WithSessionID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, id)
}

func SessionID(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(sessionIDKey{}).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("session id not found in context")
	}
	return id, nil
}
