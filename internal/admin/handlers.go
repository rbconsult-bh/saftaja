package admin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"
)

type Handlers struct {
	service Service
}

func NewHandlers(service Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Post("/organizations", h.CreateOrganization)
	r.Post("/projects", h.CreateProject)
	r.Post("/gateway-accounts", h.CreateGatewayAccount)
	r.Post("/invoices", h.CreateInvoice)
}

func (h *Handlers) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondError(w, "name is required", http.StatusBadRequest)
		return
	}

	org, err := h.service.CreateOrganization(r.Context(), &req)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to create organization")
		respondError(w, "failed to create organization", http.StatusInternalServerError)
		return
	}

	respondJSON(w, org, http.StatusCreated)
}

func (h *Handlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.OrganizationID.String() == "00000000-0000-0000-0000-000000000000" {
		respondError(w, "organization_id is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		respondError(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Environment == "" {
		req.Environment = "sandbox"
	}

	proj, err := h.service.CreateProject(r.Context(), &req)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to create project")
		respondError(w, "failed to create project", http.StatusInternalServerError)
		return
	}

	respondJSON(w, proj, http.StatusCreated)
}

func (h *Handlers) CreateGatewayAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateGatewayAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ProjectID.String() == "00000000-0000-0000-0000-000000000000" {
		respondError(w, "project_id is required", http.StatusBadRequest)
		return
	}
	if req.ConnectorType == "" {
		respondError(w, "connector_type is required", http.StatusBadRequest)
		return
	}
	if req.AccountName == "" {
		respondError(w, "account_name is required", http.StatusBadRequest)
		return
	}
	if req.Credentials == nil {
		respondError(w, "credentials is required", http.StatusBadRequest)
		return
	}

	acc, err := h.service.CreateGatewayAccount(r.Context(), &req)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to create gateway account")
		respondError(w, "failed to create gateway account", http.StatusInternalServerError)
		return
	}

	respondJSON(w, acc, http.StatusCreated)
}

func (h *Handlers) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	var req CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ProjectID.String() == "00000000-0000-0000-0000-000000000000" {
		respondError(w, "project_id is required", http.StatusBadRequest)
		return
	}
	if req.Amount == "" {
		respondError(w, "amount is required", http.StatusBadRequest)
		return
	}
	if req.Currency == "" {
		respondError(w, "currency is required", http.StatusBadRequest)
		return
	}

	inv, err := h.service.CreateInvoice(r.Context(), &req)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to create invoice")
		respondError(w, "failed to create invoice", http.StatusInternalServerError)
		return
	}

	respondJSON(w, inv, http.StatusCreated)
}

func respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
