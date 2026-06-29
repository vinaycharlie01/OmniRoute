package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type contextKey string

const (
	CtxAPIKey    contextKey = "api_key"
	CtxAPIKeyID  contextKey = "api_key_id"
	CtxRequestID contextKey = "request_id"
)

type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, key string) (*domain.ApiKey, error)
}

type JWTValidator interface {
	ValidateJWT(ctx context.Context, token string) (map[string]any, error)
}

// Auth middleware validates API keys or JWT tokens.
func Auth(validator APIKeyValidator, requireAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractBearerToken(r)
			if key == "" {
				key = r.Header.Get("X-API-Key")
			}

			if key == "" {
				if requireAuth {
					writeError(w, http.StatusUnauthorized, "unauthorized", "No API key provided")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			apiKey, err := validator.ValidateAPIKey(r.Context(), key)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid API key")
				return
			}

			ctx := context.WithValue(r.Context(), CtxAPIKeyID, apiKey.ID)
			ctx = context.WithValue(ctx, CtxAPIKey, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth validates an API key if present but doesn't require it.
func OptionalAuth(validator APIKeyValidator) func(http.Handler) http.Handler {
	return Auth(validator, false)
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func GetAPIKeyID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxAPIKeyID).(string); ok {
		return v
	}
	return ""
}

func GetAPIKey(ctx context.Context) *domain.ApiKey {
	if v, ok := ctx.Value(CtxAPIKey).(*domain.ApiKey); ok {
		return v
	}
	return nil
}
