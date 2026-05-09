package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type Service interface {
	ListActiveByProject(ctx context.Context, projectID uuid.UUID) ([]GatewayCredentials, error)
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
				_ = json.Unmarshal(acc.PaymentMethods, &methods)
			}
			if len(methods) == 0 {
				methods = []string{"card"}
			}

			result = append(result, GatewayCredentials{
				GatewayAccountID: acc.ID,
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
