package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/crypto"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type Service interface {
	ListActiveByProject(ctx context.Context, projectID uuid.UUID) ([]GatewayCredentials, error)
	ListPaymentMethods(ctx context.Context, projectID uuid.UUID) ([]PaymentMethod, error)
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
		switch acc.ConnectorType {
		case domain.ConnectorTypeMPGS:
			cfg, err := mpgsclient.ParseConfig(acc.Config)
			if err != nil {
				return nil, fmt.Errorf("invalid gateway config for %s: %w", acc.ID, err)
			}
			result = append(result, GatewayCredentials{
				GatewayAccountID: acc.ID,
				AccountName:      acc.AccountName,
				ConnectorType:    acc.ConnectorType,
				BaseURL:          cfg.BaseURL,
				MerchantID:       cfg.MerchantID,
			})
		default:
			continue
		}
	}

	return result, nil
}

func (s *service) ListPaymentMethods(ctx context.Context, projectID uuid.UUID) ([]PaymentMethod, error) {
	accounts, err := s.queries.ListActiveGatewayAccounts(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list gateway accounts: %w", err)
	}

	var methods []PaymentMethod
	for _, acc := range accounts {
		switch acc.ConnectorType {
		case domain.ConnectorTypeMPGS:
			cfg, err := mpgsclient.ParseConfig(acc.Config)
			if err != nil {
				continue
			}
			// TODO: replace with MPGS Payment Options Inquiry API call
			for _, pm := range []PaymentMethodType{
				PaymentMethodTypeCard,
				// PaymentMethodTypeApplePay,
			} {
				methods = append(methods, PaymentMethod{
					Type:             pm,
					GatewayAccountID: acc.ID,
					MPGSBaseURL:      cfg.BaseURL,
					MPGSMerchantID:   cfg.MerchantID,
					MPGSApiVersion:   mpgsclient.APIVersion,
				})
			}
		default:
			continue
		}
	}

	return methods, nil
}

func (s *service) Create(ctx context.Context, req CreateGatewayRequest) (*GatewayAccount, error) {
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	secretJSON, err := json.Marshal(req.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret: %w", err)
	}

	encrypted, err := crypto.Encrypt(secretJSON, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	acc, err := s.queries.CreateGatewayAccount(ctx, store.CreateGatewayAccountParams{
		ProjectID:     req.ProjectID,
		ConnectorType: req.ConnectorType,
		AccountName:   req.AccountName,
		Secret:        encrypted,
		Config:        configJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway account: %w", err)
	}

	return &GatewayAccount{
		ID:            acc.ID,
		ProjectID:     acc.ProjectID,
		AccountName:   acc.AccountName,
		ConnectorType: acc.ConnectorType,
		IsActive:      acc.IsActive,
	}, nil
}
