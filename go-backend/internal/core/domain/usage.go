package domain

import "time"

type CallLog struct {
	ID           string       `json:"id"`
	RequestID    string       `json:"requestId"`
	APIKeyID     string       `json:"apiKeyId,omitempty"`
	Provider     ProviderType `json:"provider"`
	Model        string       `json:"model"`
	ConnectionID string       `json:"connectionId"`
	InputTokens  int          `json:"inputTokens"`
	OutputTokens int          `json:"outputTokens"`
	TotalTokens  int          `json:"totalTokens"`
	CostUSD      float64      `json:"costUSD"`
	StatusCode   int          `json:"statusCode"`
	ErrorType    string       `json:"errorType,omitempty"`
	ErrorMessage string       `json:"errorMessage,omitempty"`
	DurationMs   int64        `json:"durationMs"`
	IsStreaming   bool         `json:"isStreaming"`
	CacheHit     bool         `json:"cacheHit"`
	ComboID      string       `json:"comboId,omitempty"`
	Timestamp    time.Time    `json:"timestamp"`
}

type UsageEntry struct {
	ID           string       `json:"id"`
	APIKeyID     string       `json:"apiKeyId,omitempty"`
	Provider     ProviderType `json:"provider"`
	Model        string       `json:"model"`
	Timestamp    time.Time    `json:"timestamp"`
	InputTokens  int          `json:"inputTokens"`
	OutputTokens int          `json:"outputTokens"`
	TotalTokens  int          `json:"totalTokens"`
	CostUSD      float64      `json:"costUSD"`
	Reason       string       `json:"reason,omitempty"`
}

type UsageStats struct {
	Provider     ProviderType `json:"provider"`
	Model        string       `json:"model,omitempty"`
	TotalTokens  int          `json:"totalTokens"`
	InputTokens  int          `json:"inputTokens"`
	OutputTokens int          `json:"outputTokens"`
	TotalCost    float64      `json:"totalCost"`
	CallCount    int          `json:"callCount"`
	AvgLatencyMs float64      `json:"avgLatencyMs"`
	ErrorRate    float64      `json:"errorRate"`
	Period       string       `json:"period"`
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
