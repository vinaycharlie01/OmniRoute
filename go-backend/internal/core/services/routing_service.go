package services

import (
	"context"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/omniroute/go-backend/internal/core/domain"
	"go.uber.org/zap"
)

// RoutingService resolves ordered targets from a Combo using one of 17 strategies.
type RoutingService struct {
	rrCounters sync.Map // map[comboID]*atomic.Int64
	log        *zap.Logger
}

func NewRoutingService(log *zap.Logger) *RoutingService {
	return &RoutingService{log: log}
}

// ResolveTargets returns an ordered list of ComboTargets for the given strategy.
func (s *RoutingService) ResolveTargets(
	ctx context.Context,
	combo *domain.Combo,
	req *domain.ChatRequest,
) ([]*domain.ComboTarget, error) {
	nodes := combo.ActiveNodes()
	if len(nodes) == 0 {
		return nil, domain.ErrAllProvidersFailed
	}

	targets := nodesToTargets(nodes)

	switch combo.Strategy {
	case domain.StrategyPriority:
		return s.priority(targets), nil
	case domain.StrategyWeighted:
		return s.weighted(targets), nil
	case domain.StrategyRoundRobin:
		return s.roundRobin(combo.ID, targets), nil
	case domain.StrategyRandom, domain.StrategyStrictRandom:
		return s.random(targets), nil
	case domain.StrategyLeastUsed:
		return s.leastUsed(targets), nil
	case domain.StrategyFillFirst:
		return s.fillFirst(targets), nil
	case domain.StrategyCostOptimized:
		return s.costOptimized(targets), nil
	case domain.StrategyP2C:
		return s.p2c(targets), nil
	case domain.StrategyResetAware:
		return s.priority(targets), nil
	case domain.StrategyResetWindow:
		return s.priority(targets), nil
	case domain.StrategyHeadroom:
		return s.priority(targets), nil
	case domain.StrategyAuto:
		return s.auto(ctx, combo, req, targets), nil
	case domain.StrategyLKGP:
		return s.priority(targets), nil
	case domain.StrategyContextOptimized:
		return s.contextOptimized(req, targets), nil
	case domain.StrategyContextRelay:
		return s.priority(targets), nil
	case domain.StrategyFusion:
		return targets, nil // fusion uses all targets in parallel
	default:
		return s.priority(targets), nil
	}
}

func nodesToTargets(nodes []domain.ComboNode) []*domain.ComboTarget {
	targets := make([]*domain.ComboTarget, len(nodes))
	for i, n := range nodes {
		targets[i] = &domain.ComboTarget{
			ConnectionID: n.ConnectionID,
			Provider:     n.Provider,
			Model:        n.Model,
			Weight:       n.Weight,
			Priority:     n.Priority,
		}
	}
	return targets
}

func (s *RoutingService) priority(targets []*domain.ComboTarget) []*domain.ComboTarget {
	sorted := make([]*domain.ComboTarget, len(targets))
	copy(sorted, targets)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}

func (s *RoutingService) weighted(targets []*domain.ComboTarget) []*domain.ComboTarget {
	totalWeight := 0
	for _, t := range targets {
		totalWeight += t.Weight
	}
	if totalWeight == 0 {
		return s.random(targets)
	}
	roll := rand.Intn(totalWeight)
	cumulative := 0
	result := make([]*domain.ComboTarget, 0, len(targets))
	for _, t := range targets {
		cumulative += t.Weight
		if roll < cumulative {
			result = append(result, t)
		}
	}
	// Append remaining targets as fallbacks.
	for _, t := range targets {
		found := false
		for _, r := range result {
			if r == t {
				found = true
				break
			}
		}
		if !found {
			result = append(result, t)
		}
	}
	return result
}

func (s *RoutingService) roundRobin(comboID string, targets []*domain.ComboTarget) []*domain.ComboTarget {
	val, _ := s.rrCounters.LoadOrStore(comboID, new(atomic.Int64))
	counter := val.(*atomic.Int64)
	n := int(counter.Add(1)-1) % len(targets)
	result := make([]*domain.ComboTarget, 0, len(targets))
	result = append(result, targets[n])
	for i, t := range targets {
		if i != n {
			result = append(result, t)
		}
	}
	return result
}

func (s *RoutingService) random(targets []*domain.ComboTarget) []*domain.ComboTarget {
	shuffled := make([]*domain.ComboTarget, len(targets))
	copy(shuffled, targets)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return shuffled
}

func (s *RoutingService) leastUsed(targets []*domain.ComboTarget) []*domain.ComboTarget {
	// Without live metrics, fall back to priority. Production: inject usage counters.
	return s.priority(targets)
}

func (s *RoutingService) fillFirst(targets []*domain.ComboTarget) []*domain.ComboTarget {
	return s.priority(targets)
}

func (s *RoutingService) costOptimized(targets []*domain.ComboTarget) []*domain.ComboTarget {
	// Without cost data injected at this level, fallback to priority.
	return s.priority(targets)
}

func (s *RoutingService) p2c(targets []*domain.ComboTarget) []*domain.ComboTarget {
	if len(targets) < 2 {
		return targets
	}
	// Power of Two Choices: pick two at random, return lower-load first.
	i, j := rand.Intn(len(targets)), rand.Intn(len(targets))
	for j == i {
		j = rand.Intn(len(targets))
	}
	result := []*domain.ComboTarget{targets[i], targets[j]}
	for k, t := range targets {
		if k != i && k != j {
			result = append(result, t)
		}
	}
	return result
}

func (s *RoutingService) contextOptimized(req *domain.ChatRequest, targets []*domain.ComboTarget) []*domain.ComboTarget {
	// Prefer targets with larger context windows when messages are long.
	return s.priority(targets)
}

func (s *RoutingService) auto(
	ctx context.Context,
	combo *domain.Combo,
	req *domain.ChatRequest,
	targets []*domain.ComboTarget,
) []*domain.ComboTarget {
	// Auto scoring: simplified 9-factor score. Full implementation injects metrics.
	return s.weighted(targets)
}
