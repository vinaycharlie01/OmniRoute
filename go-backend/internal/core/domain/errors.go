package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrInvalidInput        = errors.New("invalid input")
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrAllProvidersFailed  = errors.New("all providers failed")
	ErrCircuitOpen         = errors.New("circuit breaker open")
	ErrModelLocked         = errors.New("model locked out")
	ErrRateLimited         = errors.New("rate limited")
	ErrQuotaExceeded       = errors.New("quota exceeded")
	ErrCredentialsExpired  = errors.New("credentials expired")
	ErrBanned              = errors.New("account banned")
	ErrContextTooLong      = errors.New("context too long")
	ErrStreamAborted       = errors.New("stream aborted")
	ErrUpstreamTimeout     = errors.New("upstream timeout")
	ErrUpstreamError       = errors.New("upstream error")
)

type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Cause   error  `json:"-"`
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *DomainError) Unwrap() error { return e.Cause }

func NewRateLimitError(model string, resetSeconds int) *DomainError {
	return &DomainError{
		Code:    "rate_limit_exceeded",
		Message: fmt.Sprintf("Rate limit exceeded for model %s, retry after %ds", model, resetSeconds),
		Type:    "rate_limit_error",
		Param:   model,
	}
}

func NewProviderError(provider string, upstream error) *DomainError {
	return &DomainError{
		Code:    "provider_error",
		Message: fmt.Sprintf("Provider %s returned an error", provider),
		Type:    "provider_error",
		Param:   provider,
		Cause:   upstream,
	}
}

func NewCircuitOpenError(provider string) *DomainError {
	return &DomainError{
		Code:    "provider_circuit_open",
		Message: fmt.Sprintf("Provider %s is temporarily unavailable (circuit open)", provider),
		Type:    "provider_error",
		Param:   provider,
	}
}

func NewModelCooldownError(model string, resetSeconds int) *DomainError {
	return &DomainError{
		Code:    "model_cooldown",
		Message: fmt.Sprintf("Model %s is temporarily unavailable, retry after %ds", model, resetSeconds),
		Type:    "rate_limit_error",
		Param:   model,
	}
}

func NewValidationError(field, message string) *DomainError {
	return &DomainError{
		Code:    "validation_error",
		Message: message,
		Type:    "invalid_request_error",
		Param:   field,
	}
}
