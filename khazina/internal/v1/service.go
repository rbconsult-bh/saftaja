package v1

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/rbconsult-bh/saftaja/internal/auth"
	"github.com/rbconsult-bh/saftaja/internal/crypto"
	"github.com/rbconsult-bh/saftaja/internal/domain"
	"github.com/rbconsult-bh/saftaja/internal/store"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrForbidden   = errors.New("forbidden")
	ErrConflict    = errors.New("conflict")
	ErrBadRequest  = errors.New("bad request")
)

var reservedSlugs = map[string]bool{
	"www": true, "api": true, "admin": true, "app": true,
	"pay": true, "checkout": true, "dashboard": true,
	"docs": true, "help": true, "mail": true, "smtp": true,
	"auth": true, "login": true, "signup": true, "billing": true,
	"status": true, "health": true, "cdn": true, "static": true,
}

// Service handles all v1 API business logic.
type Service interface {
	// Organizations
	CreateOrganization(ctx context.Context, userID uuid.UUID, name string) (*OrgResponse, error)
	ListOrganizations(ctx context.Context, userID uuid.UUID) ([]OrgResponse, error)
	ListOrganizationMembers(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) ([]MemberResponse, error)
	InviteMember(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, req *InviteRequest) (*InvitationResponse, error)
	RemoveMember(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, targetUserID uuid.UUID) error

	// Projects
	CreateProject(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, req *CreateProjectRequest) (*ProjectResponse, error)
	ListProjects(ctx context.Context, userID uuid.UUID) ([]ProjectResponse, error)
	GetProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) (*ProjectResponse, error)
	UpdateProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, req *UpdateProjectRequest) (*ProjectResponse, error)

	// Invoices
	CreateInvoice(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, req *CreateInvoiceRequest) (*InvoiceResponse, error)
	ListInvoices(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, limit, offset int32) ([]InvoiceResponse, error)
	GetInvoice(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, invoiceID uuid.UUID) (*InvoiceResponse, error)

	// API Keys
	CreateAPIKey(ctx context.Context, userID uuid.UUID, req *CreateAPIKeyRequest) (*APIKeyCreatedResponse, error)
	ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]APIKeyResponse, error)
	RevokeAPIKey(ctx context.Context, userID uuid.UUID, keyID uuid.UUID) error

	// Gateway Accounts
	CreateGatewayAccount(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, req *CreateGatewayAccountRequest) (*GatewayAccountResponse, error)
	ListGatewayAccounts(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) ([]GatewayAccountResponse, error)
}

type service struct {
	queries       *store.Queries
	encryptionKey []byte
	baseDomain    string
}

func NewService(queries *store.Queries, encryptionKey []byte, baseDomain string) Service {
	return &service{queries: queries, encryptionKey: encryptionKey, baseDomain: baseDomain}
}

func (s *service) requireOrgMember(ctx context.Context, userID, orgID uuid.UUID) (*store.OrganizationMember, error) {
	m, err := s.queries.GetOrganizationMember(ctx, store.GetOrganizationMemberParams{
		OrganizationID: orgID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrForbidden
		}
		return nil, err
	}
	return &m, nil
}

type OrgResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (s *service) CreateOrganization(ctx context.Context, userID uuid.UUID, name string) (*OrgResponse, error) {
	org, err := s.queries.CreateOrganizationV1(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("creating org: %w", err)
	}

	_, err = s.queries.CreateOrganizationMember(ctx, store.CreateOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           "owner",
	})
	if err != nil {
		return nil, fmt.Errorf("adding owner: %w", err)
	}

	return &OrgResponse{ID: org.ID, Name: org.Name}, nil
}

func (s *service) ListOrganizations(ctx context.Context, userID uuid.UUID) ([]OrgResponse, error) {
	orgs, err := s.queries.ListUserOrganizations(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := make([]OrgResponse, len(orgs))
	for i, o := range orgs {
		resp[i] = OrgResponse{ID: o.ID, Name: o.Name}
	}
	return resp, nil
}

type MemberResponse struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
}

type InviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type InvitationResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
	Token string    `json:"token"`
}

func (s *service) ListOrganizationMembers(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) ([]MemberResponse, error) {
	if _, err := s.requireOrgMember(ctx, userID, orgID); err != nil {
		return nil, err
	}

	rows, err := s.queries.ListOrganizationMembers(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp := make([]MemberResponse, len(rows))
	for i, r := range rows {
		resp[i] = MemberResponse{
			ID:     r.ID,
			UserID: r.UserID,
			Email:  r.Email,
			Name:   r.UserName,
			Role:   r.Role,
		}
	}
	return resp, nil
}

func (s *service) InviteMember(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, req *InviteRequest) (*InvitationResponse, error) {
	member, err := s.requireOrgMember(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}
	if member.Role != "owner" && member.Role != "admin" {
		return nil, ErrForbidden
	}

	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	role := req.Role
	if role == "" {
		role = "member"
	}

	inv, err := s.queries.CreateInvitation(ctx, store.CreateInvitationParams{
		OrganizationID: orgID,
		Email:          req.Email,
		Role:           role,
		InvitedBy:      userID,
		Token:          token,
	})
	if err != nil {
		return nil, fmt.Errorf("creating invitation: %w", err)
	}

	return &InvitationResponse{
		ID:    inv.ID,
		Email: inv.Email,
		Role:  inv.Role,
		Token: inv.Token,
	}, nil
}

func (s *service) RemoveMember(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, targetUserID uuid.UUID) error {
	member, err := s.requireOrgMember(ctx, userID, orgID)
	if err != nil {
		return err
	}
	if member.Role != "owner" && member.Role != "admin" {
		return ErrForbidden
	}
	return s.queries.DeleteOrganizationMember(ctx, store.DeleteOrganizationMemberParams{
		OrganizationID: orgID,
		UserID:         targetUserID,
	})
}

type CreateProjectRequest struct {
	Name         string `json:"name"`
	Environment  string `json:"environment"`
	CustomDomain string `json:"custom_domain"`
	Slug         string `json:"slug"`
}

type UpdateProjectRequest struct {
	Name         string `json:"name"`
	CustomDomain string `json:"custom_domain"`
	Slug         string `json:"slug"`
}

type ProjectResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Environment    string    `json:"environment"`
	CustomDomain   string    `json:"custom_domain,omitempty"`
	Slug           string    `json:"slug,omitempty"`
	CheckoutBase   string    `json:"checkout_base,omitempty"`
}

func (s *service) projectResponse(p store.Project) ProjectResponse {
	r := ProjectResponse{
		ID:             p.ID,
		OrganizationID: p.OrganizationID,
		Name:           p.Name,
		Environment:    string(p.Environment),
		CustomDomain:   p.CustomDomain.String,
		Slug:           p.Slug.String,
	}
	if p.Slug.Valid && p.Slug.String != "" && s.baseDomain != "" {
		r.CheckoutBase = fmt.Sprintf("https://%s.%s", p.Slug.String, s.baseDomain)
	} else if p.CustomDomain.Valid && p.CustomDomain.String != "" {
		r.CheckoutBase = fmt.Sprintf("https://%s", p.CustomDomain.String)
	}
	return r
}

func (s *service) requireProjectAccess(ctx context.Context, userID, projectID uuid.UUID) (*store.Project, error) {
	orgs, err := s.queries.ListUserOrganizations(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, org := range orgs {
		proj, err := s.queries.GetProjectByIDAndOrg(ctx, store.GetProjectByIDAndOrgParams{
			ID:             projectID,
			OrganizationID: org.ID,
		})
		if err == nil {
			return &proj, nil
		}
	}
	return nil, ErrNotFound
}

func (s *service) CreateProject(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, req *CreateProjectRequest) (*ProjectResponse, error) {
	if _, err := s.requireOrgMember(ctx, userID, orgID); err != nil {
		return nil, err
	}

	if req.Slug != "" && reservedSlugs[req.Slug] {
		return nil, fmt.Errorf("%w: slug is reserved", ErrBadRequest)
	}

	env := domain.ProjectEnvironment(req.Environment)
	if env == "" {
		env = domain.ProjectEnvironmentSandbox
	}

	var customDomain pgtype.Text
	if req.CustomDomain != "" {
		customDomain = pgtype.Text{String: req.CustomDomain, Valid: true}
	}

	var slug pgtype.Text
	if req.Slug != "" {
		slug = pgtype.Text{String: req.Slug, Valid: true}
	}

	proj, err := s.queries.CreateProjectV1(ctx, store.CreateProjectV1Params{
		OrganizationID: orgID,
		Name:           req.Name,
		Environment:    env,
		CustomDomain:   customDomain,
		Slug:           slug,
	})
	if err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	r := s.projectResponse(proj)
	return &r, nil
}

func (s *service) ListProjects(ctx context.Context, userID uuid.UUID) ([]ProjectResponse, error) {
	orgs, err := s.queries.ListUserOrganizations(ctx, userID)
	if err != nil {
		return nil, err
	}

	var projects []ProjectResponse
	for _, org := range orgs {
		projs, err := s.queries.ListProjectsByOrganization(ctx, org.ID)
		if err != nil {
			return nil, err
		}
		for _, p := range projs {
			projects = append(projects, s.projectResponse(p))
		}
	}
	return projects, nil
}

func (s *service) GetProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) (*ProjectResponse, error) {
	proj, err := s.requireProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	r := s.projectResponse(*proj)
	return &r, nil
}

func (s *service) UpdateProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, req *UpdateProjectRequest) (*ProjectResponse, error) {
	existing, err := s.requireProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	if req.Slug != "" && reservedSlugs[req.Slug] {
		return nil, fmt.Errorf("%w: slug is reserved", ErrBadRequest)
	}

	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}

	customDomain := existing.CustomDomain
	if req.CustomDomain != "" {
		customDomain = pgtype.Text{String: req.CustomDomain, Valid: true}
	}

	slug := existing.Slug
	if req.Slug != "" {
		slug = pgtype.Text{String: req.Slug, Valid: true}
	}

	proj, err := s.queries.UpdateProject(ctx, store.UpdateProjectParams{
		ID:             projectID,
		OrganizationID: existing.OrganizationID,
		Name:           name,
		CustomDomain:   customDomain,
		Slug:           slug,
	})
	if err != nil {
		return nil, fmt.Errorf("updating project: %w", err)
	}

	r := s.projectResponse(proj)
	return &r, nil
}

type CreateInvoiceRequest struct {
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
	Description   string `json:"description"`
	ExternalID    string `json:"external_id"`
}

type InvoiceResponse struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"project_id"`
	Amount        string    `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CustomerEmail string    `json:"customer_email,omitempty"`
	CustomerName  string    `json:"customer_name,omitempty"`
	Description   string    `json:"description,omitempty"`
	CheckoutURL   string    `json:"checkout_url"`
}

func (s *service) invoiceResponse(inv store.Invoice, proj *store.Project) InvoiceResponse {
	checkoutURL := fmt.Sprintf("/checkout/%s", inv.ID)
	if proj != nil {
		base := ""
		if proj.Slug.Valid && proj.Slug.String != "" && s.baseDomain != "" {
			base = fmt.Sprintf("https://%s.%s", proj.Slug.String, s.baseDomain)
		} else if proj.CustomDomain.Valid && proj.CustomDomain.String != "" {
			base = fmt.Sprintf("https://%s", proj.CustomDomain.String)
		}
		if base != "" {
			checkoutURL = fmt.Sprintf("%s/checkout/%s", base, inv.ID)
		}
	}
	return InvoiceResponse{
		ID:            inv.ID,
		ProjectID:     inv.ProjectID,
		Amount:        inv.Amount.String(),
		Currency:      inv.Currency,
		Status:        string(inv.Status),
		CustomerEmail: inv.CustomerEmail.String,
		CustomerName:  inv.CustomerName.String,
		Description:   inv.Description.String,
		CheckoutURL:   checkoutURL,
	}
}

func (s *service) CreateInvoice(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, req *CreateInvoiceRequest) (*InvoiceResponse, error) {
	proj, err := s.requireProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid amount", ErrBadRequest)
	}

	inv, err := s.queries.CreateInvoice(ctx, store.CreateInvoiceParams{
		ProjectID:     projectID,
		Amount:        amount,
		Currency:      req.Currency,
		CustomerEmail: pgtype.Text{String: req.CustomerEmail, Valid: req.CustomerEmail != ""},
		CustomerName:  pgtype.Text{String: req.CustomerName, Valid: req.CustomerName != ""},
		Description:   pgtype.Text{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("creating invoice: %w", err)
	}

	r := s.invoiceResponse(inv, proj)
	return &r, nil
}

func (s *service) ListInvoices(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, limit, offset int32) ([]InvoiceResponse, error) {
	proj, err := s.requireProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	invs, err := s.queries.ListInvoicesByProject(ctx, store.ListInvoicesByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	resp := make([]InvoiceResponse, len(invs))
	for i, inv := range invs {
		resp[i] = s.invoiceResponse(inv, proj)
	}
	return resp, nil
}

func (s *service) GetInvoice(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, invoiceID uuid.UUID) (*InvoiceResponse, error) {
	proj, err := s.requireProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	inv, err := s.queries.GetInvoiceByIDAndOrgProject(ctx, store.GetInvoiceByIDAndOrgProjectParams{
		ID:        invoiceID,
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	r := s.invoiceResponse(inv, proj)
	return &r, nil
}

type CreateAPIKeyRequest struct {
	OrganizationID uuid.UUID  `json:"organization_id"`
	ProjectID      *uuid.UUID `json:"project_id"`
	Name           string     `json:"name"`
	Environment    string     `json:"environment"`
}

type APIKeyCreatedResponse struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Key    string    `json:"key"`
	Prefix string    `json:"prefix"`
}

type APIKeyResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ProjectID      string    `json:"project_id,omitempty"`
	Name           string    `json:"name"`
	Prefix         string    `json:"prefix"`
}

func (s *service) CreateAPIKey(ctx context.Context, userID uuid.UUID, req *CreateAPIKeyRequest) (*APIKeyCreatedResponse, error) {
	if _, err := s.requireOrgMember(ctx, userID, req.OrganizationID); err != nil {
		return nil, err
	}

	env := req.Environment
	if env == "" {
		env = "live"
	}

	fullKey, prefix, keyHash, err := auth.GenerateAPIKey(env)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	var projectID uuid.UUID
	if req.ProjectID != nil {
		projectID = *req.ProjectID
	}

	k, err := s.queries.CreateAPIKey(ctx, store.CreateAPIKeyParams{
		OrganizationID: req.OrganizationID,
		ProjectID:      projectID,
		UserID:         userID,
		Name:           req.Name,
		KeyHash:        keyHash,
		KeyPrefix:      prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("creating api key: %w", err)
	}

	return &APIKeyCreatedResponse{
		ID:     k.ID,
		Name:   k.Name,
		Key:    fullKey,
		Prefix: k.KeyPrefix,
	}, nil
}

func (s *service) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]APIKeyResponse, error) {
	keys, err := s.queries.ListAPIKeysByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := make([]APIKeyResponse, len(keys))
	for i, k := range keys {
		r := APIKeyResponse{
			ID:             k.ID,
			OrganizationID: k.OrganizationID,
			Name:           k.Name,
			Prefix:         k.KeyPrefix,
		}
		if k.ProjectID != (uuid.UUID{}) {
			r.ProjectID = k.ProjectID.String()
		}
		resp[i] = r
	}
	return resp, nil
}

func (s *service) RevokeAPIKey(ctx context.Context, userID uuid.UUID, keyID uuid.UUID) error {
	return s.queries.RevokeAPIKey(ctx, store.RevokeAPIKeyParams{
		ID:     keyID,
		UserID: userID,
	})
}

type CreateGatewayAccountRequest struct {
	ConnectorType  string                 `json:"connector_type"`
	AccountName    string                 `json:"account_name"`
	Credentials    map[string]any         `json:"credentials"`
	Settings       map[string]any         `json:"settings"`
	PaymentMethods []string               `json:"payment_methods"`
}

type GatewayAccountResponse struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"project_id"`
	ConnectorType string    `json:"connector_type"`
	AccountName   string    `json:"account_name"`
	IsActive      bool      `json:"is_active"`
}

func (s *service) CreateGatewayAccount(ctx context.Context, userID uuid.UUID, projectID uuid.UUID, req *CreateGatewayAccountRequest) (*GatewayAccountResponse, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}

	credJSON, err := json.Marshal(req.Credentials)
	if err != nil {
		return nil, fmt.Errorf("marshaling credentials: %w", err)
	}

	encryptedCreds, err := crypto.Encrypt(credJSON, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting credentials: %w", err)
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
		ProjectID:      projectID,
		ConnectorType:  domain.ConnectorType(req.ConnectorType),
		AccountName:    req.AccountName,
		Credentials:    encryptedCreds,
		Settings:       settings,
		PaymentMethods: paymentMethods,
	})
	if err != nil {
		return nil, fmt.Errorf("creating gateway account: %w", err)
	}

	return &GatewayAccountResponse{
		ID:            acc.ID,
		ProjectID:     acc.ProjectID,
		ConnectorType: string(acc.ConnectorType),
		AccountName:   acc.AccountName,
		IsActive:      acc.IsActive,
	}, nil
}

func (s *service) ListGatewayAccounts(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) ([]GatewayAccountResponse, error) {
	if _, err := s.requireProjectAccess(ctx, userID, projectID); err != nil {
		return nil, err
	}

	accs, err := s.queries.ListGatewayAccountsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	resp := make([]GatewayAccountResponse, len(accs))
	for i, a := range accs {
		resp[i] = GatewayAccountResponse{
			ID:            a.ID,
			ProjectID:     a.ProjectID,
			ConnectorType: string(a.ConnectorType),
			AccountName:   a.AccountName,
			IsActive:      a.IsActive,
		}
	}
	return resp, nil
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
