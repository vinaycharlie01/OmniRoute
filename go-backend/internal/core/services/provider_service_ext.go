package services

import (
	"context"
	"net/http"
	"time"

	"github.com/omniroute/go-backend/internal/core/domain"
)

// GetProviderHealth returns health status for all providers (primary port).
func (s *ProviderService) GetProviderHealth(ctx context.Context) ([]*domain.ProviderHealthStatus, error) {
	connections, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make([]*domain.ProviderHealthStatus, 0, len(connections))
	for _, conn := range connections {
		cbState := s.circuitBreaker.GetStatus(ctx, conn.Provider)
		statuses = append(statuses, &domain.ProviderHealthStatus{
			Provider:       conn.Provider,
			ConnectionID:   conn.ID,
			IsHealthy:      conn.IsAvailable() && cbState == "CLOSED",
			LastCheckedAt:  time.Now(),
			CircuitBreaker: cbState,
		})
	}
	return statuses, nil
}

// TestProvider pings the provider with a lightweight request (primary port).
func (s *ProviderService) TestProvider(ctx context.Context, id string) (*domain.ProviderHealthStatus, error) {
	conn, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, domain.ErrNotFound
	}

	start := time.Now()
	isHealthy := false

	url := conn.BaseURL + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err == nil {
		if conn.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+conn.APIKey)
		}
		c := &http.Client{Timeout: 5 * time.Second}
		resp, err := c.Do(req)
		if err == nil {
			defer resp.Body.Close()
			isHealthy = resp.StatusCode < 500
		}
	}

	latency := float64(time.Since(start).Milliseconds())
	cbState := s.circuitBreaker.GetStatus(ctx, conn.Provider)

	return &domain.ProviderHealthStatus{
		Provider:      conn.Provider,
		ConnectionID:  conn.ID,
		IsHealthy:     isHealthy,
		LastCheckedAt: time.Now(),
		AvgResponseMs: latency,
		CircuitBreaker: cbState,
	}, nil
}
