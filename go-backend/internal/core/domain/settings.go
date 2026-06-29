package domain

type Settings struct {
	RequireLogin             bool              `json:"requireLogin"`
	HasPassword              bool              `json:"hasPassword"`
	FallbackStrategy         string            `json:"fallbackStrategy"`
	StickyRoundRobinLimit    int               `json:"stickyRoundRobinLimit"`
	RequestRetry             int               `json:"requestRetry"`
	MaxRetryIntervalSec      int               `json:"maxRetryIntervalSec"`
	MCPEnabled               bool              `json:"mcpEnabled"`
	MCPTransport             string            `json:"mcpTransport"`
	A2AEnabled               bool              `json:"a2aEnabled"`
	ResilienceSettings       *ResilienceSettings `json:"resilienceSettings,omitempty"`
	LocalOnlyManageScope     bool              `json:"localOnlyManageScopeBypassEnabled"`
	DefaultComboStrategy     ComboStrategy     `json:"defaultComboStrategy"`
	DefaultMaxRetries        int               `json:"defaultMaxRetries"`
	DefaultRetryDelayMs      int               `json:"defaultRetryDelayMs"`
	DefaultTimeoutMs         int               `json:"defaultTimeoutMs"`
	SSEHeartbeatIntervalMs   int               `json:"sseHeartbeatIntervalMs"`
	CompressionEnabled       bool              `json:"compressionEnabled"`
	SemanticCacheEnabled     bool              `json:"semanticCacheEnabled"`
	PIIRedactionEnabled      bool              `json:"piiRedactionEnabled"`
}

func DefaultSettings() *Settings {
	return &Settings{
		RequireLogin:           false,
		FallbackStrategy:       "priority",
		StickyRoundRobinLimit:  5,
		RequestRetry:           3,
		MaxRetryIntervalSec:    30,
		MCPEnabled:             false,
		MCPTransport:           "stdio",
		A2AEnabled:             false,
		DefaultComboStrategy:   StrategyPriority,
		DefaultMaxRetries:      3,
		DefaultRetryDelayMs:    1000,
		DefaultTimeoutMs:       60000,
		SSEHeartbeatIntervalMs: 5000,
		CompressionEnabled:     false,
		SemanticCacheEnabled:   false,
		PIIRedactionEnabled:    false,
	}
}

type ResilienceSettings struct {
	CircuitBreakerEnabled     bool    `json:"circuitBreakerEnabled"`
	OAuthThreshold            int     `json:"oauthThreshold"`
	OAuthResetTimeoutSec      int     `json:"oauthResetTimeoutSec"`
	APIKeyThreshold           int     `json:"apiKeyThreshold"`
	APIKeyResetTimeoutSec     int     `json:"apiKeyResetTimeoutSec"`
	LocalThreshold            int     `json:"localThreshold"`
	LocalResetTimeoutSec      int     `json:"localResetTimeoutSec"`
	OAuthBaseCooldownMs       int     `json:"oauthBaseCooldownMs"`
	APIKeyBaseCooldownMs      int     `json:"apiKeyBaseCooldownMs"`
	ModelLockoutEnabled       bool    `json:"modelLockoutEnabled"`
	AntiThunderingHerd        bool    `json:"antiThunderingHerd"`
}

func DefaultResilienceSettings() *ResilienceSettings {
	return &ResilienceSettings{
		CircuitBreakerEnabled:  true,
		OAuthThreshold:         3,
		OAuthResetTimeoutSec:   60,
		APIKeyThreshold:        5,
		APIKeyResetTimeoutSec:  30,
		LocalThreshold:         2,
		LocalResetTimeoutSec:   15,
		OAuthBaseCooldownMs:    5000,
		APIKeyBaseCooldownMs:   3000,
		ModelLockoutEnabled:    true,
		AntiThunderingHerd:     true,
	}
}

type FeatureFlag struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	DefaultValue string `json:"defaultValue"`
	Description  string `json:"description"`
}
