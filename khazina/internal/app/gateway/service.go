package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/crypto"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type Service interface {
	ListActiveByProject(ctx context.Context, projectID uuid.UUID) ([]GatewayCredentials, error)
	Create(ctx context.Context, req CreateGatewayRequest) (*GatewayAccount, error)
}

type service struct {
	queries       store.TransactionQuerier
	encryptionKey []byte
}

func New(queries store.TransactionQuerier, encryptionKey []byte) Service {
	return &service{queries: queries, encryptionKey: encryptionKey}
}

func (s *service) ListActiveByProject(ctx context.Context, projectID uuid.UUID) ([]GatewayCredentials, error) {
	accounts, err := s.queries.ListActiveGatewayAccounts(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list gateway accounts: %w", err)
	}

	var result []GatewayCredentials
	for _, acc := range accounts {
		if acc.ConnectorType == domain.ConnectorTypeMPGS {
			creds, err := mpgsclient.ParseEncryptedCredentials(acc.Credentials, s.encryptionKey)
			if err != nil {
				return nil, fmt.Errorf("invalid gateway credentials: %w", err)
			}

			var methods []string
			if len(acc.PaymentMethods) > 0 {
				if err := json.Unmarshal(acc.PaymentMethods, &methods); err != nil {
					return nil, fmt.Errorf("invalid payment methods JSON: %w", err)
				}
			}

			result = append(result, GatewayCredentials{
				GatewayAccountID: acc.ID,
				AccountName:      acc.AccountName,
				ConnectorType:    acc.ConnectorType,
				BaseURL:          creds.BaseURL,
				MerchantID:       creds.MerchantID,
				APIPassword:      creds.APIPassword,
				PaymentMethods:   methods,
			})
		}
	}

	return result, nil
}

func (s *service) Create(ctx context.Context, req CreateGatewayRequest) (*GatewayAccount, error) {
	credsJSON, err := json.Marshal(req.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encrypted, err := crypto.Encrypt(credsJSON, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	methodsJSON, err := json.Marshal(req.PaymentMethods)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment methods: %w", err)
	}
	if req.PaymentMethods == nil {
		methodsJSON = []byte("[]")
	}

	acc, err := s.queries.CreateGatewayAccount(ctx, store.CreateGatewayAccountParams{
		ProjectID:      req.ProjectID,
		ConnectorType:  req.ConnectorType,
		AccountName:    req.AccountName,
		Credentials:    encrypted,
		Settings:       []byte("{}"),
		PaymentMethods: methodsJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway account: %w", err)
	}

	return &GatewayAccount{
		ID:             acc.ID,
		ProjectID:      acc.ProjectID,
		AccountName:    acc.AccountName,
		ConnectorType:  acc.ConnectorType,
		IsActive:       acc.IsActive,
	}, nil
}
