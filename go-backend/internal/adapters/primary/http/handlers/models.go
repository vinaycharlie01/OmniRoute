package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"go.uber.org/zap"
)

type ModelsHandler struct {
	modelUseCase primary.ModelUseCase
	log          *zap.Logger
}

func NewModelsHandler(modelUseCase primary.ModelUseCase, log *zap.Logger) *ModelsHandler {
	return &ModelsHandler{modelUseCase: modelUseCase, log: log}
}

// ListModels handles GET /v1/models.
func (h *ModelsHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.modelUseCase.ListModels(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to list models")
		return
	}

	data := make([]map[string]any, len(models))
	for i, m := range models {
		data[i] = map[string]any{
			"id":       m.ID,
			"object":   "model",
			"created":  0,
			"owned_by": string(m.Provider),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   data,
	})
}

// GetModel handles GET /v1/models/{model}.
func (h *ModelsHandler) GetModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "model")
	model, err := h.modelUseCase.GetModel(r.Context(), modelID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", "Model not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":       model.ID,
		"object":   "model",
		"created":  0,
		"owned_by": string(model.Provider),
	})
}
