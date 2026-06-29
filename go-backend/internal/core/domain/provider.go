package domain

import "time"

type ProviderType string

const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAnthropic ProviderType = "anthropic"
	ProviderTypeGemini    ProviderType = "gemini"
	ProviderTypeAzure     ProviderType = "azure-openai"
	ProviderTypeBedrock   ProviderType = "bedrock"
	ProviderTypeOllama    ProviderType = "ollama"
	ProviderTypeCustom    ProviderType = "custom"
)

type AuthType string

const (
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeOAuth  AuthType = "oauth"
	AuthTypeBearer AuthType = "bearer"
	AuthTypeNone   AuthType = "none"
)

type ConnectionStatus string

const (
	ConnectionStatusActive      ConnectionStatus = "active"
	ConnectionStatusInactive    ConnectionStatus = "inactive"
	ConnectionStatusBanned      ConnectionStatus = "banned"
	ConnectionStatusExpired     ConnectionStatus = "expired"
	ConnectionStatusUnavailable ConnectionStatus = "unavailable"
	ConnectionStatusExhausted   ConnectionStatus = "credits_exhausted"
)

type ProviderConnection struct {
	ID                  string           `json:"id"`
	Provider            ProviderType     `json:"provider"`
	Label               string           `json:"label"`
	BaseURL             string           `json:"baseUrl"`
	AuthType            AuthType         `json:"authType"`
	APIKey              string           `json:"-"`
	OAuthToken          string           `json:"-"`
	OAuthRefreshToken   string           `json:"-"`
	OAuthExpiresAt      *time.Time       `json:"oauthExpiresAt,omitempty"`
	Status              ConnectionStatus `json:"status"`
	IsActive            bool             `json:"isActive"`
	Priority            int              `json:"priority"`
	Weight              int              `json:"weight"`
	RateLimitedUntil    *time.Time       `json:"rateLimitedUntil,omitempty"`
	LastError           string           `json:"lastError,omitempty"`
	LastErrorType       string           `json:"lastErrorType,omitempty"`
	ErrorCode           string           `json:"errorCode,omitempty"`
	BackoffLevel        int              `json:"backoffLevel"`
	MaxConcurrent       int              `json:"maxConcurrent"`
	Email               string           `json:"email,omitempty"`
	ProviderSpecific    map[string]any   `json:"providerSpecific,omitempty"`
	CreatedAt           time.Time        `json:"createdAt"`
	UpdatedAt           time.Time        `json:"updatedAt"`
}

func (c *ProviderConnection) IsAvailable() bool {
	if !c.IsActive {
		return false
	}
	if c.Status == ConnectionStatusBanned ||
		c.Status == ConnectionStatusExpired ||
		c.Status == ConnectionStatusExhausted {
		return false
	}
	if c.RateLimitedUntil != nil && c.RateLimitedUntil.After(time.Now()) {
		return false
	}
	return true
}

func (c *ProviderConnection) IsCooledDown() bool {
	return c.RateLimitedUntil != nil && c.RateLimitedUntil.After(time.Now())
}

type ProviderNode struct {
	ID           string       `json:"id"`
	ConnectionID string       `json:"connectionId"`
	Provider     ProviderType `json:"provider"`
	Model        string       `json:"model"`
	BaseURL      string       `json:"baseUrl"`
	IsActive     bool         `json:"isActive"`
	Priority     int          `json:"priority"`
	Weight       int          `json:"weight"`
}

type ProviderCredentials struct {
	AccessToken          string         `json:"-"`
	RefreshToken         string         `json:"-"`
	APIKey               string         `json:"-"`
	ConnectionID         string         `json:"connectionId"`
	Provider             ProviderType   `json:"provider"`
	BaseURL              string         `json:"baseUrl"`
	MaxConcurrent        int            `json:"maxConcurrent"`
	Email                string         `json:"email,omitempty"`
	ExpiresAt            *time.Time     `json:"expiresAt,omitempty"`
	ProviderSpecificData map[string]any `json:"providerSpecificData,omitempty"`
}

type ProviderHealthStatus struct {
	Provider        ProviderType `json:"provider"`
	ConnectionID    string       `json:"connectionId"`
	IsHealthy       bool         `json:"isHealthy"`
	LastCheckedAt   time.Time    `json:"lastCheckedAt"`
	AvgResponseMs   float64      `json:"avgResponseMs"`
	SuccessRate     float64      `json:"successRate"`
	CircuitBreaker  string       `json:"circuitBreaker"`
}
