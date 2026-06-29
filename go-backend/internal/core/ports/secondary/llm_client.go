package secondary

import (
	"context"
	"io"

	"github.com/omniroute/go-backend/internal/core/domain"
)

// LLMClient is the secondary port for communicating with upstream LLM providers.
type LLMClient interface {
	// ChatComplete performs a non-streaming chat completion.
	ChatComplete(ctx context.Context, req *LLMRequest) (*LLMResponse, error)

	// ChatStream initiates a streaming chat completion and returns an SSE reader.
	ChatStream(ctx context.Context, req *LLMRequest) (io.ReadCloser, error)

	// Embed creates embeddings for the given input.
	Embed(ctx context.Context, req *LLMEmbedRequest) (*domain.EmbeddingResponse, error)

	// Provider returns the provider type this client handles.
	Provider() domain.ProviderType

	// Supports returns true if this client can handle the given provider+model combo.
	Supports(provider domain.ProviderType, model string) bool
}

type LLMRequest struct {
	Credentials  *domain.ProviderCredentials
	Body         *domain.ChatRequest
	TargetModel  string
	RequestID    string
	UserAgent    string
	ExtraHeaders map[string]string
	TimeoutMs    int
}

type LLMResponse struct {
	Response     *domain.ChatResponse
	StatusCode   int
	Headers      map[string]string
	DurationMs   int64
}

type LLMEmbedRequest struct {
	Credentials  *domain.ProviderCredentials
	Body         *domain.EmbeddingRequest
	TargetModel  string
	RequestID    string
}

// LLMClientFactory creates LLMClient instances for a given provider.
type LLMClientFactory interface {
	Create(provider domain.ProviderType) (LLMClient, error)
	Supports(provider domain.ProviderType) bool
}

// Cache is the secondary port for caching responses.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttlSec int) error
	Delete(ctx context.Context, key string) error
}

// Translator is the secondary port for format translation.
type Translator interface {
	// TranslateRequest normalizes a ChatRequest to the target provider's format.
	TranslateRequest(req *domain.ChatRequest, targetFormat domain.ModelFormat) (map[string]any, error)

	// TranslateResponse normalizes a provider response back to OpenAI format.
	TranslateResponse(body []byte, sourceFormat domain.ModelFormat) (*domain.ChatResponse, error)

	// TranslateStreamChunk translates a single SSE chunk from provider format.
	TranslateStreamChunk(data []byte, sourceFormat domain.ModelFormat) (*domain.ChatStreamChunk, error)
}
