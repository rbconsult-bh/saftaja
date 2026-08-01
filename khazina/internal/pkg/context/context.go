package saftajacontext

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
)

type (
	customerIDKey struct{}
	sessionIDKey  struct{}
	projectKey    struct{}
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

func WithProject(ctx context.Context, project *tenant.Project) context.Context {
	return context.WithValue(ctx, projectKey{}, project)
}

func ProjectFromContext(ctx context.Context) *tenant.Project {
	if p, ok := ctx.Value(projectKey{}).(*tenant.Project); ok {
		return p
	}
	return nil
}
