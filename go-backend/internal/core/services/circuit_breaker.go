package services

import (
	"context"
	"sync"
	"time"

	"github.com/omniroute/go-backend/internal/core/domain"
	"go.uber.org/zap"
)

type CBState string

const (
	CBClosed   CBState = "CLOSED"
	CBOpen     CBState = "OPEN"
	CBHalfOpen CBState = "HALF_OPEN"
)

type breaker struct {
	mu          sync.Mutex
	state       CBState
	failures    int
	threshold   int
	resetTimeout time.Duration
	lastFailed  time.Time
	resetAt     time.Time
}

func (b *breaker) getState() CBState {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == CBOpen && time.Now().After(b.resetAt) {
		b.state = CBHalfOpen
	}
	return b.state
}

func (b *breaker) canExecute() bool {
	s := b.getState()
	return s == CBClosed || s == CBHalfOpen
}

func (b *breaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.state = CBClosed
}

func (b *breaker) recordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	b.lastFailed = time.Now()
	if b.failures >= b.threshold {
		b.state = CBOpen
		b.resetAt = time.Now().Add(b.resetTimeout)
	}
}

func (b *breaker) retryAfterMs() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == CBOpen {
		remaining := time.Until(b.resetAt)
		if remaining > 0 {
			return remaining.Milliseconds()
		}
	}
	return 0
}

type CircuitBreakerService struct {
	mu       sync.RWMutex
	breakers map[domain.ProviderType]*breaker
	settings *domain.ResilienceSettings
	log      *zap.Logger
}

func NewCircuitBreakerService(settings *domain.ResilienceSettings, log *zap.Logger) *CircuitBreakerService {
	return &CircuitBreakerService{
		breakers: make(map[domain.ProviderType]*breaker),
		settings: settings,
		log:      log,
	}
}

func (s *CircuitBreakerService) getBreaker(provider domain.ProviderType) *breaker {
	s.mu.RLock()
	b, ok := s.breakers[provider]
	s.mu.RUnlock()
	if ok {
		return b
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok = s.breakers[provider]; ok {
		return b
	}

	threshold, resetTimeout := s.thresholdFor(provider)
	b = &breaker{
		state:        CBClosed,
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
	s.breakers[provider] = b
	return b
}

func (s *CircuitBreakerService) thresholdFor(provider domain.ProviderType) (int, time.Duration) {
	switch provider {
	case domain.ProviderTypeOllama:
		return s.settings.LocalThreshold, time.Duration(s.settings.LocalResetTimeoutSec) * time.Second
	default:
		return s.settings.APIKeyThreshold, time.Duration(s.settings.APIKeyResetTimeoutSec) * time.Second
	}
}

func (s *CircuitBreakerService) CanExecute(_ context.Context, provider domain.ProviderType) bool {
	return s.getBreaker(provider).canExecute()
}

func (s *CircuitBreakerService) RecordSuccess(_ context.Context, provider domain.ProviderType) {
	s.getBreaker(provider).recordSuccess()
}

func (s *CircuitBreakerService) RecordFailure(_ context.Context, provider domain.ProviderType, statusCode int) {
	// Only trip the breaker on provider-level failure codes.
	if isProviderLevelFailure(statusCode) {
		b := s.getBreaker(provider)
		b.recordFailure()
		s.log.Warn("circuit breaker failure recorded",
			zap.String("provider", string(provider)),
			zap.Int("statusCode", statusCode),
			zap.Int("failures", b.failures),
		)
	}
}

func (s *CircuitBreakerService) RetryAfterMs(_ context.Context, provider domain.ProviderType) int64 {
	return s.getBreaker(provider).retryAfterMs()
}

func (s *CircuitBreakerService) GetStatus(_ context.Context, provider domain.ProviderType) string {
	return string(s.getBreaker(provider).getState())
}

// isProviderLevelFailure returns true for status codes that indicate a provider-level
// outage rather than an account/key/model error.
func isProviderLevelFailure(code int) bool {
	switch code {
	case 408, 500, 502, 503, 504:
		return true
	}
	return false
}
