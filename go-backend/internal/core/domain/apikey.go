package domain

import "time"

type ApiKey struct {
	ID           string     `json:"id"`
	Key          string     `json:"-"`
	KeyHash      string     `json:"-"`
	Name         string     `json:"name"`
	GroupID      string     `json:"groupId,omitempty"`
	IsActive     bool       `json:"isActive"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	RestrictedTo []string   `json:"restrictedTo,omitempty"`
	RateLimit    *RateLimit `json:"rateLimit,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

func (k *ApiKey) IsExpired() bool {
	return k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now())
}

func (k *ApiKey) IsValid() bool {
	return k.IsActive && !k.IsExpired()
}

type RateLimit struct {
	RPM        int `json:"rpm,omitempty"`
	TPM        int `json:"tpm,omitempty"`
	DailyLimit int `json:"dailyLimit,omitempty"`
}

type ApiKeyGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"isActive"`
	RateLimit   *RateLimit `json:"rateLimit,omitempty"`
	AllowedModels []string `json:"allowedModels,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ApiKeyPolicy struct {
	KeyID         string
	AllowedModels []string
	RateLimit     *RateLimit
	QuotaPoolID   string
}
