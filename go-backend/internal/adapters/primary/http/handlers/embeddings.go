package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/omniroute/go-backend/internal/adapters/primary/http/middleware"
	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"go.uber.org/zap"
)

type EmbeddingsHandler struct {
	chatUseCase primary.ChatUseCase
	log         *zap.Logger
}

func NewEmbeddingsHandler(chatUseCase primary.ChatUseCase, log *zap.Logger) *EmbeddingsHandler {
	return &EmbeddingsHandler{chatUseCase: chatUseCase, log: log}
}

// Embeddings handles POST /v1/embeddings.
func (h *EmbeddingsHandler) Embeddings(w http.ResponseWriter, r *http.Request) {
	var req domain.EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON body")
		return
	}
	if req.Model == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	if req.Input == nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "input is required")
		return
	}

	opts := &primary.ChatOptions{
		APIKeyID:  middleware.GetAPIKeyID(r.Context()),
		RequestID: r.Header.Get("X-Request-ID"),
		UserAgent: r.Header.Get("User-Agent"),
	}

	resp, err := h.chatUseCase.Embed(r.Context(), &req, opts)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Embedding failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
