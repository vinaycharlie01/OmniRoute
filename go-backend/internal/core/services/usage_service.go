package services

import (
	"context"

	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"github.com/omniroute/go-backend/internal/core/ports/secondary"
	"go.uber.org/zap"
)

type UsageService struct {
	repo secondary.UsageRepository
	log  *zap.Logger
}

func NewUsageService(repo secondary.UsageRepository, log *zap.Logger) *UsageService {
	return &UsageService{repo: repo, log: log}
}

func (s *UsageService) GetStats(ctx context.Context, filter *primary.UsageFilter) ([]*domain.UsageStats, error) {
	f := secondary.UsageStatsFilter{
		Provider: filter.Provider,
		Model:    filter.Model,
		APIKeyID: filter.APIKeyID,
	}
	return s.repo.GetUsageStats(ctx, f)
}

func (s *UsageService) GetCallLogs(ctx context.Context, filter *primary.CallLogFilter) ([]*domain.CallLog, error) {
	f := secondary.CallLogFilter{
		Provider: filter.Provider,
		Model:    filter.Model,
		APIKeyID: filter.APIKeyID,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	}
	return s.repo.FindCallLogs(ctx, f)
}

func (s *UsageService) RecordUsage(ctx context.Context, log *domain.CallLog) error {
	return s.repo.SaveCallLog(ctx, log)
}
