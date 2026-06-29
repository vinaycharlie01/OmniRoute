package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"go.uber.org/zap"
)

type CombosHandler struct {
	comboUseCase primary.ComboUseCase
	log          *zap.Logger
}

func NewCombosHandler(comboUseCase primary.ComboUseCase, log *zap.Logger) *CombosHandler {
	return &CombosHandler{comboUseCase: comboUseCase, log: log}
}

func (h *CombosHandler) List(w http.ResponseWriter, r *http.Request) {
	combos, err := h.comboUseCase.ListCombos(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to list combos")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": combos})
}

func (h *CombosHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	combo, err := h.comboUseCase.GetCombo(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", "Combo not found")
		return
	}
	writeJSON(w, http.StatusOK, combo)
}

func (h *CombosHandler) Create(w http.ResponseWriter, r *http.Request) {
	var combo domain.Combo
	if err := json.NewDecoder(r.Body).Decode(&combo); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON body")
		return
	}
	created, err := h.comboUseCase.CreateCombo(r.Context(), &combo)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to create combo")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *CombosHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var combo domain.Combo
	if err := json.NewDecoder(r.Body).Decode(&combo); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON body")
		return
	}
	combo.ID = id
	updated, err := h.comboUseCase.UpdateCombo(r.Context(), &combo)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to update combo")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *CombosHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.comboUseCase.DeleteCombo(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to delete combo")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
