package saftajacontext

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type (
	customerIDKey struct{}
	sessionIDKey  struct{}
)

func WithCustomerID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, customerIDKey{}, id)
}

func CustomerID(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(customerIDKey{}).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("customer id not found in context")
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
