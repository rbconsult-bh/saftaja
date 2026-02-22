package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handlers struct {
	svc Service
}

func NewHandlers(svc Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		respondError(w, "email, password, and name are required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		respondError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	user, err := h.svc.Register(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			respondError(w, "email already registered", http.StatusConflict)
			return
		}
		respondError(w, "registration failed", http.StatusInternalServerError)
		return
	}

	respondJSON(w, user, http.StatusCreated)
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		respondError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.Login(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			respondError(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		respondError(w, "login failed", http.StatusInternalServerError)
		return
	}

	respondJSON(w, resp, http.StatusOK)
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		respondError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.svc.GetUser(r.Context(), userID)
	if err != nil {
		respondError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	respondJSON(w, user, http.StatusOK)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
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
