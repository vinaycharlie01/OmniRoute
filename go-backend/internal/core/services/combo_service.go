package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/secondary"
	"go.uber.org/zap"
)

type ComboService struct {
	repo    secondary.ComboRepository
	routing *RoutingService
	log     *zap.Logger
}

func NewComboService(
	repo secondary.ComboRepository,
	routing *RoutingService,
	log *zap.Logger,
) *ComboService {
	return &ComboService{repo: repo, routing: routing, log: log}
}

func (s *ComboService) ListCombos(ctx context.Context) ([]*domain.Combo, error) {
	return s.repo.FindAll(ctx)
}

func (s *ComboService) GetCombo(ctx context.Context, id string) (*domain.Combo, error) {
	combo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if combo == nil {
		return nil, domain.ErrNotFound
	}
	return combo, nil
}

func (s *ComboService) GetComboByModel(ctx context.Context, model string) (*domain.Combo, error) {
	return s.repo.FindByModel(ctx, model)
}

func (s *ComboService) CreateCombo(ctx context.Context, combo *domain.Combo) (*domain.Combo, error) {
	combo.ID = uuid.New().String()
	combo.CreatedAt = time.Now()
	combo.UpdatedAt = time.Now()
	if err := s.repo.Save(ctx, combo); err != nil {
		return nil, err
	}
	return combo, nil
}

func (s *ComboService) UpdateCombo(ctx context.Context, combo *domain.Combo) (*domain.Combo, error) {
	combo.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, combo); err != nil {
		return nil, err
	}
	return combo, nil
}

func (s *ComboService) DeleteCombo(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ComboService) ResolveTargets(
	ctx context.Context,
	combo *domain.Combo,
	req *domain.ChatRequest,
) ([]*domain.ComboTarget, error) {
	return s.routing.ResolveTargets(ctx, combo, req)
}
