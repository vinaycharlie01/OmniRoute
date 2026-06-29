package services

import (
	"github.com/omniroute/go-backend/internal/core/ports/primary"
)

// Compile-time interface assertions.
var (
	_ primary.ChatUseCase     = (*ChatService)(nil)
	_ primary.ProviderUseCase = (*ProviderService)(nil)
	_ primary.ComboUseCase    = (*ComboService)(nil)
	_ primary.AuthUseCase     = (*AuthService)(nil)
	_ primary.UsageUseCase    = (*UsageService)(nil)
)
