package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/omniroute/go-backend/internal/adapters/primary/http/middleware"
	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"go.uber.org/zap"
)

type ChatHandler struct {
	chatUseCase primary.ChatUseCase
	log         *zap.Logger
}

func NewChatHandler(chatUseCase primary.ChatUseCase, log *zap.Logger) *ChatHandler {
	return &ChatHandler{chatUseCase: chatUseCase, log: log}
}

// ChatCompletions handles POST /v1/chat/completions (OpenAI-compatible).
func (h *ChatHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON body: "+err.Error())
		return
	}

	if req.Model == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	if len(req.Messages) == 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "messages must not be empty")
		return
	}

	opts := &primary.ChatOptions{
		APIKeyID:   middleware.GetAPIKeyID(r.Context()),
		RequestID:  r.Header.Get("X-Request-ID"),
		UserAgent:  r.Header.Get("User-Agent"),
		ClientIP:   r.RemoteAddr,
	}
	if opts.RequestID == "" {
		opts.RequestID = uuid.New().String()
	}

	if req.Stream {
		h.handleStream(w, r, &req, opts)
		return
	}

	result, err := h.chatUseCase.Complete(r.Context(), &req, opts)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result.Response)
}

func (h *ChatHandler) handleStream(w http.ResponseWriter, r *http.Request, req *domain.ChatRequest, opts *primary.ChatOptions) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	events, err := h.chatUseCase.StreamComplete(r.Context(), req, opts)
	if err != nil {
		fmt.Fprintf(w, "data: {\"error\":{\"message\":%q}}\n\n", err.Error())
		flusher.Flush()
		return
	}

	for event := range events {
		if event.Error != nil {
			fmt.Fprintf(w, "data: {\"error\":{\"message\":%q}}\n\n", sanitizeError(event.Error))
			flusher.Flush()
			return
		}
		if event.Done {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}
		if len(event.Data) > 0 {
			w.Write(event.Data)
			flusher.Flush()
		}
	}
}

func (h *ChatHandler) handleError(w http.ResponseWriter, err error) {
	if domErr, ok := err.(*domain.DomainError); ok {
		status := domainErrToStatus(domErr.Code)
		writeJSONError(w, status, domErr.Type, domErr.Message)
		return
	}
	writeJSONError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred")
}

func domainErrToStatus(code string) int {
	switch code {
	case "validation_error":
		return http.StatusBadRequest
	case "unauthorized":
		return http.StatusUnauthorized
	case "rate_limit_exceeded", "model_cooldown":
		return http.StatusTooManyRequests
	case "provider_circuit_open", "provider_error":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func sanitizeError(err error) string {
	if domErr, ok := err.(*domain.DomainError); ok {
		return domErr.Message
	}
	return "An error occurred"
}

func writeJSONError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := map[string]any{
		"error": map[string]any{
			"type":    errType,
			"message": message,
		},
	}
	_ = json.NewEncoder(w).Encode(body)
}
