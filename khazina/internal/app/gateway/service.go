package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/crypto"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type service struct {
	queries       store.TransactionQuerier
	encryptionKey []byte
}

func New(queries store.TransactionQuerier, encryptionKey []byte) Service {
	return &service{queries: queries, encryptionKey: encryptionKey}
}

func (s *service) ListActiveByProject(ctx context.Context, r ListActiveByProjectRequest) (*ListActiveByProjectResponse, error) {
	accounts, err := s.queries.ListActiveGatewayAccounts(ctx, r.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list gateway accounts: %w", err)
	}

	var gateways []GatewayCredentials
	for _, acc := range accounts {
		switch acc.ConnectorType {
		case store.ConnectorTypeMPGS:
			cfg, err := mpgsclient.ParseConfig(acc.Config)
			if err != nil {
				return nil, fmt.Errorf("invalid gateway config for %s: %w", acc.ID, err)
			}
			gateways = append(gateways, GatewayCredentials{
				GatewayAccountID: acc.ID,
				AccountName:      acc.AccountName,
				ConnectorType:    mapConnectorTypeFromStore(acc.ConnectorType),
				BaseURL:          cfg.BaseURL,
				MerchantID:       cfg.MerchantID,
			})
		default:
			continue
		}
	}

	return &ListActiveByProjectResponse{Gateways: gateways}, nil
}

func (s *service) ListPaymentMethods(ctx context.Context, r ListPaymentMethodsRequest) (*ListPaymentMethodsResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	accounts, err := s.queries.ListActiveGatewayAccounts(ctx, r.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list gateway accounts: %w", err)
	}

	var methods []PaymentMethod
	for _, acc := range accounts {
		switch acc.ConnectorType {
		case store.ConnectorTypeMPGS:
			cfg, err := mpgsclient.ParseConfig(acc.Config)
			if err != nil {
				continue
			}
			for _, pm := range []PaymentMethodType{
				PaymentMethodTypeCard,
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

	return &ListPaymentMethodsResponse{PaymentMethods: methods}, nil
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
		ConnectorType: mapConnectorTypeToStore(req.ConnectorType),
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
		ConnectorType: mapConnectorTypeFromStore(acc.ConnectorType),
		IsActive:      acc.IsActive,
	}, nil
}
