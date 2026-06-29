package primary

import (
	"context"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type ChatUseCase interface {
	// Complete executes a chat completion (streaming or non-streaming).
	Complete(ctx context.Context, req *domain.ChatRequest, opts *ChatOptions) (*ChatResult, error)

	// StreamComplete executes a streaming chat completion, writing SSE chunks to the channel.
	StreamComplete(ctx context.Context, req *domain.ChatRequest, opts *ChatOptions) (<-chan StreamEvent, error)

	// Embed creates text embeddings.
	Embed(ctx context.Context, req *domain.EmbeddingRequest, opts *ChatOptions) (*domain.EmbeddingResponse, error)
}

type ChatOptions struct {
	APIKeyID     string
	APIKeyName   string
	RequestID    string
	UserAgent    string
	ClientIP     string
	ForwardedFor string
}

type ChatResult struct {
	Response    *domain.ChatResponse
	CallLog     *domain.CallLog
	CacheHit    bool
	Provider    domain.ProviderType
	Model       string
	ConnectionID string
}

type StreamEvent struct {
	Data  []byte
	Error error
	Done  bool
}
