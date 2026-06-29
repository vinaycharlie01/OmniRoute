package primary

import (
	"context"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type ComboUseCase interface {
	ListCombos(ctx context.Context) ([]*domain.Combo, error)
	GetCombo(ctx context.Context, id string) (*domain.Combo, error)
	GetComboByModel(ctx context.Context, model string) (*domain.Combo, error)
	CreateCombo(ctx context.Context, combo *domain.Combo) (*domain.Combo, error)
	UpdateCombo(ctx context.Context, combo *domain.Combo) (*domain.Combo, error)
	DeleteCombo(ctx context.Context, id string) error
	ResolveTargets(ctx context.Context, combo *domain.Combo, req *domain.ChatRequest) ([]*domain.ComboTarget, error)
}
