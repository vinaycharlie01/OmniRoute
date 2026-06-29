package primary

import (
	"context"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type ProviderUseCase interface {
	ListProviders(ctx context.Context) ([]*domain.ProviderConnection, error)
	GetProvider(ctx context.Context, id string) (*domain.ProviderConnection, error)
	CreateProvider(ctx context.Context, conn *domain.ProviderConnection) (*domain.ProviderConnection, error)
	UpdateProvider(ctx context.Context, conn *domain.ProviderConnection) (*domain.ProviderConnection, error)
	DeleteProvider(ctx context.Context, id string) error
	TestProvider(ctx context.Context, id string) (*domain.ProviderHealthStatus, error)
	GetProviderHealth(ctx context.Context) ([]*domain.ProviderHealthStatus, error)
	GetAvailableCredentials(ctx context.Context, provider domain.ProviderType, model string) ([]*domain.ProviderCredentials, error)
}

type ModelUseCase interface {
	ListModels(ctx context.Context) ([]*domain.Model, error)
	GetModel(ctx context.Context, id string) (*domain.Model, error)
	SearchModels(ctx context.Context, query string, provider domain.ProviderType) ([]*domain.Model, error)
	ResolveModelInfo(ctx context.Context, modelName string) (*domain.ModelInfo, error)
}
