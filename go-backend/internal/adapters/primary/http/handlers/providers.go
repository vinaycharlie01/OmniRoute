package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"go.uber.org/zap"
)

type ProvidersHandler struct {
	providerUseCase primary.ProviderUseCase
	log             *zap.Logger
}

func NewProvidersHandler(providerUseCase primary.ProviderUseCase, log *zap.Logger) *ProvidersHandler {
	return &ProvidersHandler{providerUseCase: providerUseCase, log: log}
}

func (h *ProvidersHandler) List(w http.ResponseWriter, r *http.Request) {
	providers, err := h.providerUseCase.ListProviders(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to list providers")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": providers})
}

func (h *ProvidersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	provider, err := h.providerUseCase.GetProvider(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", "Provider not found")
		return
	}
	writeJSON(w, http.StatusOK, provider)
}

func (h *ProvidersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var conn domain.ProviderConnection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON body")
		return
	}
	created, err := h.providerUseCase.CreateProvider(r.Context(), &conn)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to create provider")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *ProvidersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var conn domain.ProviderConnection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON body")
		return
	}
	conn.ID = id
	updated, err := h.providerUseCase.UpdateProvider(r.Context(), &conn)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to update provider")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *ProvidersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.providerUseCase.DeleteProvider(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to delete provider")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProvidersHandler) Health(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.providerUseCase.GetProviderHealth(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to get provider health")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": statuses})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
