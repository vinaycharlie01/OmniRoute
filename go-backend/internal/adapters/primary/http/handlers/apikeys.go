package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"go.uber.org/zap"
)

type APIKeysHandler struct {
	authUseCase primary.AuthUseCase
	log         *zap.Logger
}

func NewAPIKeysHandler(authUseCase primary.AuthUseCase, log *zap.Logger) *APIKeysHandler {
	return &APIKeysHandler{authUseCase: authUseCase, log: log}
}

func (h *APIKeysHandler) List(w http.ResponseWriter, r *http.Request) {
	keys, err := h.authUseCase.ListAPIKeys(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to list API keys")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": keys})
}

func (h *APIKeysHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name          string   `json:"name"`
		RestrictedTo  []string `json:"restrictedTo"`
		ExpiresInDays int      `json:"expiresInDays"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON body")
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "name is required")
		return
	}

	key, rawKey, err := h.authUseCase.CreateAPIKey(r.Context(), body.Name, &primary.APIKeyCreateOptions{
		RestrictedTo:  body.RestrictedTo,
		ExpiresInDays: body.ExpiresInDays,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to create API key")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":   key.ID,
		"name": key.Name,
		"key":  rawKey,
	})
}

func (h *APIKeysHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.authUseCase.RevokeAPIKey(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to revoke API key")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIKeysHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON body")
		return
	}

	token, err := h.authUseCase.Login(r.Context(), body.Password)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token})
}
