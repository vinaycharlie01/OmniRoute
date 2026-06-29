package services

import (
	"context"
	"time"

	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/secondary"
	"go.uber.org/zap"
)

type ProviderService struct {
	repo           secondary.ProviderRepository
	circuitBreaker *CircuitBreakerService
	log            *zap.Logger
}

func NewProviderService(
	repo secondary.ProviderRepository,
	cb *CircuitBreakerService,
	log *zap.Logger,
) *ProviderService {
	return &ProviderService{repo: repo, circuitBreaker: cb, log: log}
}

func (s *ProviderService) ListProviders(ctx context.Context) ([]*domain.ProviderConnection, error) {
	return s.repo.FindAll(ctx)
}

func (s *ProviderService) GetProvider(ctx context.Context, id string) (*domain.ProviderConnection, error) {
	conn, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, domain.ErrNotFound
	}
	return conn, nil
}

func (s *ProviderService) CreateProvider(ctx context.Context, conn *domain.ProviderConnection) (*domain.ProviderConnection, error) {
	conn.CreatedAt = time.Now()
	conn.UpdatedAt = time.Now()
	if err := s.repo.Save(ctx, conn); err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *ProviderService) UpdateProvider(ctx context.Context, conn *domain.ProviderConnection) (*domain.ProviderConnection, error) {
	conn.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, conn); err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *ProviderService) DeleteProvider(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// GetAvailableCredentials returns credentials for available connections of the given provider,
// filtered by circuit breaker and cooldown state, ordered by priority.
func (s *ProviderService) GetAvailableCredentials(
	ctx context.Context,
	provider domain.ProviderType,
	model string,
) ([]*domain.ProviderCredentials, error) {
	if !s.circuitBreaker.CanExecute(ctx, provider) {
		return nil, domain.NewCircuitOpenError(string(provider))
	}

	connections, err := s.repo.FindAvailable(ctx, provider)
	if err != nil {
		return nil, err
	}

	creds := make([]*domain.ProviderCredentials, 0, len(connections))
	for _, conn := range connections {
		if !conn.IsAvailable() {
			continue
		}
		c := &domain.ProviderCredentials{
			ConnectionID:  conn.ID,
			Provider:      conn.Provider,
			BaseURL:       conn.BaseURL,
			MaxConcurrent: conn.MaxConcurrent,
			Email:         conn.Email,
		}
		switch conn.AuthType {
		case domain.AuthTypeAPIKey:
			c.APIKey = conn.APIKey
		case domain.AuthTypeOAuth:
			c.AccessToken = conn.OAuthToken
			c.RefreshToken = conn.OAuthRefreshToken
			c.ExpiresAt = conn.OAuthExpiresAt
		case domain.AuthTypeBearer:
			c.AccessToken = conn.OAuthToken
		}
		creds = append(creds, c)
	}

	if len(creds) == 0 {
		return nil, domain.ErrProviderUnavailable
	}
	return creds, nil
}

// MarkConnectionCooled temporarily cools down a connection with exponential backoff.
func (s *ProviderService) MarkConnectionCooled(
	ctx context.Context,
	connectionID string,
	baseCooldownMs int,
	backoffLevel int,
	errType, errCode, errMsg string,
) error {
	cooldown := time.Duration(baseCooldownMs) * time.Millisecond
	for i := 0; i < backoffLevel; i++ {
		cooldown *= 2
	}
	until := time.Now().Add(cooldown)
	return s.repo.UpdateCooldown(ctx, connectionID, until, backoffLevel+1, errType, errCode, errMsg)
}

func (s *ProviderService) ClearConnectionCooldown(ctx context.Context, connectionID string) error {
	return s.repo.ClearCooldown(ctx, connectionID)
}
