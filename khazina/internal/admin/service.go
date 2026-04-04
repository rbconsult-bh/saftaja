package admin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/rbconsult-bh/saftaja/khazina/internal/crypto"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type Service interface {
	CreateOrganization(ctx context.Context, req *CreateOrganizationRequest) (*Organization, error)
	CreateProject(ctx context.Context, req *CreateProjectRequest) (*Project, error)
	CreateGatewayAccount(ctx context.Context, req *CreateGatewayAccountRequest) (*GatewayAccount, error)
	CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*Invoice, error)
}

type service struct {
	queries       *store.Queries
	encryptionKey []byte
}

func NewService(queries *store.Queries, encryptionKey []byte) Service {
	return &service{
		queries:       queries,
		encryptionKey: encryptionKey,
	}
}

type CreateOrganizationRequest struct {
	Name string `json:"name"`
}

type Organization struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (s *service) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest) (*Organization, error) {
	org, err := s.queries.CreateOrganization(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}
	return &Organization{ID: org.ID, Name: org.Name}, nil
}

type CreateProjectRequest struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Environment    string    `json:"environment"`
	CustomDomain   string    `json:"custom_domain"`
}

type Project struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Environment    string    `json:"environment"`
	CustomDomain   string    `json:"custom_domain,omitempty"`
}

func (s *service) CreateProject(ctx context.Context, req *CreateProjectRequest) (*Project, error) {
	var customDomain pgtype.Text
	if req.CustomDomain != "" {
		customDomain = pgtype.Text{String: req.CustomDomain, Valid: true}
	}

	proj, err := s.queries.CreateProject(ctx, store.CreateProjectParams{
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Environment:    domain.ProjectEnvironment(req.Environment),
		CustomDomain:   customDomain,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return &Project{
		ID:             proj.ID,
		OrganizationID: proj.OrganizationID,
		Name:           proj.Name,
		Environment:    string(proj.Environment),
		CustomDomain:   proj.CustomDomain.String,
	}, nil
}

type CreateGatewayAccountRequest struct {
	ProjectID      uuid.UUID              `json:"project_id"`
	ConnectorType  string                 `json:"connector_type"`
	AccountName    string                 `json:"account_name"`
	Credentials    map[string]interface{} `json:"credentials"`
	Settings       map[string]interface{} `json:"settings"`
	PaymentMethods []string               `json:"payment_methods"`
}

type GatewayAccount struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"project_id"`
	ConnectorType string    `json:"connector_type"`
	AccountName   string    `json:"account_name"`
}

func (s *service) CreateGatewayAccount(ctx context.Context, req *CreateGatewayAccountRequest) (*GatewayAccount, error) {
	credJSON, err := json.Marshal(req.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encryptedCreds, err := crypto.Encrypt(credJSON, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	settings := []byte("{}")
	if req.Settings != nil {
		settings, _ = json.Marshal(req.Settings)
	}

	paymentMethods := []byte(`["card"]`)
	if len(req.PaymentMethods) > 0 {
		paymentMethods, _ = json.Marshal(req.PaymentMethods)
	}

	acc, err := s.queries.CreateGatewayAccount(ctx, store.CreateGatewayAccountParams{
		ProjectID:      req.ProjectID,
		ConnectorType:  domain.ConnectorType(req.ConnectorType),
		AccountName:    req.AccountName,
		Credentials:    encryptedCreds,
		Settings:       settings,
		PaymentMethods: paymentMethods,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway account: %w", err)
	}

	return &GatewayAccount{
		ID:            acc.ID,
		ProjectID:     acc.ProjectID,
		ConnectorType: string(acc.ConnectorType),
		AccountName:   acc.AccountName,
	}, nil
}

type CreateInvoiceRequest struct {
	ProjectID     uuid.UUID `json:"project_id"`
	Amount        string    `json:"amount"`
	Currency      string    `json:"currency"`
	CustomerEmail string    `json:"customer_email"`
	CustomerName  string    `json:"customer_name"`
	Description   string    `json:"description"`
}

type Invoice struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Amount      string    `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	CheckoutURL string    `json:"checkout_url"`
}

func (s *service) CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*Invoice, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	inv, err := s.queries.CreateInvoice(ctx, store.CreateInvoiceParams{
		ProjectID:     req.ProjectID,
		Amount:        amount,
		Currency:      req.Currency,
		CustomerEmail: pgtype.Text{String: req.CustomerEmail, Valid: req.CustomerEmail != ""},
		CustomerName:  pgtype.Text{String: req.CustomerName, Valid: req.CustomerName != ""},
		Description:   pgtype.Text{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	return &Invoice{
		ID:          inv.ID,
		ProjectID:   inv.ProjectID,
		Amount:      inv.Amount.String(),
		Currency:    inv.Currency,
		Status:      string(inv.Status),
		CheckoutURL: fmt.Sprintf("/checkout/%s", inv.ID),
	}, nil
}
