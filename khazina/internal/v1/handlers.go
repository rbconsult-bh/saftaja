package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/internal/auth"
)

type Handlers struct {
	svc Service
}

func NewHandlers(svc Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/organizations", h.ListOrganizations)
	r.Post("/organizations", h.CreateOrganization)
	r.Get("/organizations/{org_id}/members", h.ListOrganizationMembers)
	r.Post("/organizations/{org_id}/invitations", h.InviteMember)
	r.Delete("/organizations/{org_id}/members/{user_id}", h.RemoveMember)

	r.Get("/projects", h.ListProjects)
	r.Post("/organizations/{org_id}/projects", h.CreateProject)
	r.Get("/projects/{project_id}", h.GetProject)
	r.Patch("/projects/{project_id}", h.UpdateProject)

	r.Get("/projects/{project_id}/invoices", h.ListInvoices)
	r.Post("/projects/{project_id}/invoices", h.CreateInvoice)
	r.Get("/projects/{project_id}/invoices/{invoice_id}", h.GetInvoice)

	r.Get("/api-keys", h.ListAPIKeys)
	r.Post("/api-keys", h.CreateAPIKey)
	r.Delete("/api-keys/{key_id}", h.RevokeAPIKey)

	r.Get("/projects/{project_id}/gateway-accounts", h.ListGatewayAccounts)
	r.Post("/projects/{project_id}/gateway-accounts", h.CreateGatewayAccount)
}

func (h *Handlers) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		respondError(w, "name is required", http.StatusBadRequest)
		return
	}

	org, err := h.svc.CreateOrganization(r.Context(), userID, body.Name)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("create org")
		respondError(w, "failed to create organization", http.StatusInternalServerError)
		return
	}
	respondJSON(w, org, http.StatusCreated)
}

func (h *Handlers) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}

	orgs, err := h.svc.ListOrganizations(r.Context(), userID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("list orgs")
		respondError(w, "failed to list organizations", http.StatusInternalServerError)
		return
	}
	respondJSON(w, orgs, http.StatusOK)
}

func (h *Handlers) ListOrganizationMembers(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	orgID, ok := parseUUID(w, r, "org_id")
	if !ok {
		return
	}

	members, err := h.svc.ListOrganizationMembers(r.Context(), userID, orgID)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, members, http.StatusOK)
}

func (h *Handlers) InviteMember(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	orgID, ok := parseUUID(w, r, "org_id")
	if !ok {
		return
	}

	var req InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		respondError(w, "email is required", http.StatusBadRequest)
		return
	}

	inv, err := h.svc.InviteMember(r.Context(), userID, orgID, &req)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, inv, http.StatusCreated)
}

func (h *Handlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	orgID, ok := parseUUID(w, r, "org_id")
	if !ok {
		return
	}
	targetID, ok := parseUUID(w, r, "user_id")
	if !ok {
		return
	}

	if err := h.svc.RemoveMember(r.Context(), userID, orgID, targetID); err != nil {
		handleServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	orgID, ok := parseUUID(w, r, "org_id")
	if !ok {
		return
	}

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		respondError(w, "name is required", http.StatusBadRequest)
		return
	}

	proj, err := h.svc.CreateProject(r.Context(), userID, orgID, &req)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, proj, http.StatusCreated)
}

func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}

	projects, err := h.svc.ListProjects(r.Context(), userID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("list projects")
		respondError(w, "failed to list projects", http.StatusInternalServerError)
		return
	}
	respondJSON(w, projects, http.StatusOK)
}

func (h *Handlers) GetProject(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	projectID, ok := parseUUID(w, r, "project_id")
	if !ok {
		return
	}

	proj, err := h.svc.GetProject(r.Context(), userID, projectID)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, proj, http.StatusOK)
}

func (h *Handlers) UpdateProject(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	projectID, ok := parseUUID(w, r, "project_id")
	if !ok {
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	proj, err := h.svc.UpdateProject(r.Context(), userID, projectID, &req)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, proj, http.StatusOK)
}

func (h *Handlers) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	projectID, ok := parseUUID(w, r, "project_id")
	if !ok {
		return
	}

	var req CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Amount == "" || req.Currency == "" {
		respondError(w, "amount and currency are required", http.StatusBadRequest)
		return
	}

	inv, err := h.svc.CreateInvoice(r.Context(), userID, projectID, &req)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, inv, http.StatusCreated)
}

func (h *Handlers) ListInvoices(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	projectID, ok := parseUUID(w, r, "project_id")
	if !ok {
		return
	}

	limit := int32(50)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = int32(n)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil {
			offset = int32(n)
		}
	}

	invs, err := h.svc.ListInvoices(r.Context(), userID, projectID, limit, offset)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, invs, http.StatusOK)
}

func (h *Handlers) GetInvoice(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	projectID, ok := parseUUID(w, r, "project_id")
	if !ok {
		return
	}
	invoiceID, ok := parseUUID(w, r, "invoice_id")
	if !ok {
		return
	}

	inv, err := h.svc.GetInvoice(r.Context(), userID, projectID, invoiceID)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, inv, http.StatusOK)
}

func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		respondError(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.OrganizationID == uuid.Nil {
		respondError(w, "organization_id is required", http.StatusBadRequest)
		return
	}

	key, err := h.svc.CreateAPIKey(r.Context(), userID, &req)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, key, http.StatusCreated)
}

func (h *Handlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}

	keys, err := h.svc.ListAPIKeys(r.Context(), userID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("list api keys")
		respondError(w, "failed to list api keys", http.StatusInternalServerError)
		return
	}
	respondJSON(w, keys, http.StatusOK)
}

func (h *Handlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	keyID, ok := parseUUID(w, r, "key_id")
	if !ok {
		return
	}

	if err := h.svc.RevokeAPIKey(r.Context(), userID, keyID); err != nil {
		handleServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) CreateGatewayAccount(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	projectID, ok := parseUUID(w, r, "project_id")
	if !ok {
		return
	}

	var req CreateGatewayAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ConnectorType == "" || req.AccountName == "" {
		respondError(w, "connector_type and account_name are required", http.StatusBadRequest)
		return
	}

	acc, err := h.svc.CreateGatewayAccount(r.Context(), userID, projectID, &req)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, acc, http.StatusCreated)
}

func (h *Handlers) ListGatewayAccounts(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(w, r)
	if userID == uuid.Nil {
		return
	}
	projectID, ok := parseUUID(w, r, "project_id")
	if !ok {
		return
	}

	accs, err := h.svc.ListGatewayAccounts(r.Context(), userID, projectID)
	if err != nil {
		handleServiceError(w, r, err)
		return
	}
	respondJSON(w, accs, http.StatusOK)
}

func mustUserID(w http.ResponseWriter, r *http.Request) uuid.UUID {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, "unauthorized", http.StatusUnauthorized)
		return uuid.Nil
	}
	return userID
}

func parseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		respondError(w, param+" is invalid", http.StatusBadRequest)
		return uuid.Nil, false
	}
	return id, true
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		respondError(w, "not found", http.StatusNotFound)
	case errors.Is(err, ErrForbidden):
		respondError(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, ErrConflict):
		respondError(w, "conflict", http.StatusConflict)
	case errors.Is(err, ErrBadRequest):
		respondError(w, err.Error(), http.StatusBadRequest)
	default:
		log.Ctx(r.Context()).Error().Err(err).Msg("v1 service error")
		respondError(w, "internal server error", http.StatusInternalServerError)
	}
}

func respondJSON(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
