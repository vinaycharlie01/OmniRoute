package primary

import (
	"context"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type AuthUseCase interface {
	ValidateAPIKey(ctx context.Context, key string) (*domain.ApiKey, error)
	CreateAPIKey(ctx context.Context, name string, opts *APIKeyCreateOptions) (*domain.ApiKey, string, error)
	ListAPIKeys(ctx context.Context) ([]*domain.ApiKey, error)
	RevokeAPIKey(ctx context.Context, id string) error
	GetAPIKeyPolicy(ctx context.Context, keyID string) (*domain.ApiKeyPolicy, error)
	Login(ctx context.Context, password string) (token string, err error)
	ValidateJWT(ctx context.Context, token string) (claims map[string]any, err error)
	ChangePassword(ctx context.Context, oldPass, newPass string) error
}

type APIKeyCreateOptions struct {
	GroupID       string
	RestrictedTo  []string
	ExpiresInDays int
	RateLimit     *domain.RateLimit
}

type UsageUseCase interface {
	GetStats(ctx context.Context, filter *UsageFilter) ([]*domain.UsageStats, error)
	GetCallLogs(ctx context.Context, filter *CallLogFilter) ([]*domain.CallLog, error)
	RecordUsage(ctx context.Context, log *domain.CallLog) error
}

type UsageFilter struct {
	Provider  domain.ProviderType
	Model     string
	APIKeyID  string
	StartTime string
	EndTime   string
	Limit     int
}

type CallLogFilter struct {
	Provider  domain.ProviderType
	Model     string
	APIKeyID  string
	StartTime string
	EndTime   string
	Limit     int
	Offset    int
}
