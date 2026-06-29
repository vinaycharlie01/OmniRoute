package secondary

import (
	"context"
	"time"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type ProviderRepository interface {
	FindAll(ctx context.Context) ([]*domain.ProviderConnection, error)
	FindByID(ctx context.Context, id string) (*domain.ProviderConnection, error)
	FindByProvider(ctx context.Context, provider domain.ProviderType) ([]*domain.ProviderConnection, error)
	FindAvailable(ctx context.Context, provider domain.ProviderType) ([]*domain.ProviderConnection, error)
	Save(ctx context.Context, conn *domain.ProviderConnection) error
	Update(ctx context.Context, conn *domain.ProviderConnection) error
	Delete(ctx context.Context, id string) error
	UpdateCooldown(ctx context.Context, id string, until time.Time, level int, errType, errCode, errMsg string) error
	ClearCooldown(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status domain.ConnectionStatus) error
}

type ComboRepository interface {
	FindAll(ctx context.Context) ([]*domain.Combo, error)
	FindByID(ctx context.Context, id string) (*domain.Combo, error)
	FindByModel(ctx context.Context, model string) (*domain.Combo, error)
	Save(ctx context.Context, combo *domain.Combo) error
	Update(ctx context.Context, combo *domain.Combo) error
	Delete(ctx context.Context, id string) error
}

type ApiKeyRepository interface {
	FindAll(ctx context.Context) ([]*domain.ApiKey, error)
	FindByID(ctx context.Context, id string) (*domain.ApiKey, error)
	FindByHash(ctx context.Context, hash string) (*domain.ApiKey, error)
	Save(ctx context.Context, key *domain.ApiKey) error
	Update(ctx context.Context, key *domain.ApiKey) error
	Delete(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string, at time.Time) error
}

type ModelRepository interface {
	FindAll(ctx context.Context) ([]*domain.Model, error)
	FindByID(ctx context.Context, id string) (*domain.Model, error)
	FindByProvider(ctx context.Context, provider domain.ProviderType) ([]*domain.Model, error)
	Search(ctx context.Context, query string) ([]*domain.Model, error)
	Save(ctx context.Context, model *domain.Model) error
	Update(ctx context.Context, model *domain.Model) error
}

type UsageRepository interface {
	SaveCallLog(ctx context.Context, log *domain.CallLog) error
	FindCallLogs(ctx context.Context, filter CallLogFilter) ([]*domain.CallLog, error)
	GetUsageStats(ctx context.Context, filter UsageStatsFilter) ([]*domain.UsageStats, error)
	GetTotalCost(ctx context.Context, apiKeyID string, since time.Time) (float64, error)
	GetTokenCount(ctx context.Context, apiKeyID string, windowStart time.Time) (int, error)
}

type SettingsRepository interface {
	Get(ctx context.Context) (*domain.Settings, error)
	Update(ctx context.Context, settings *domain.Settings) error
	GetFeatureFlag(ctx context.Context, key string) (string, error)
	SetFeatureFlag(ctx context.Context, key, value string) error
}

type CircuitBreakerRepository interface {
	GetState(ctx context.Context, provider domain.ProviderType) (*CircuitBreakerState, error)
	SaveState(ctx context.Context, state *CircuitBreakerState) error
}

type CircuitBreakerState struct {
	Provider        domain.ProviderType
	State           string
	Failures        int
	LastFailureAt   *time.Time
	ResetAt         *time.Time
}

type CallLogFilter struct {
	Provider  domain.ProviderType
	Model     string
	APIKeyID  string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

type UsageStatsFilter struct {
	Provider  domain.ProviderType
	Model     string
	APIKeyID  string
	StartTime *time.Time
	EndTime   *time.Time
	GroupBy   string
}
