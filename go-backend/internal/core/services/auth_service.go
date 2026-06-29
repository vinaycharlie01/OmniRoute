package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"github.com/omniroute/go-backend/internal/core/ports/secondary"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	apiKeyRepo  secondary.ApiKeyRepository
	settingsRepo secondary.SettingsRepository
	jwtSecret   []byte
	log         *zap.Logger
}

func NewAuthService(
	apiKeyRepo secondary.ApiKeyRepository,
	settingsRepo secondary.SettingsRepository,
	jwtSecret string,
	log *zap.Logger,
) *AuthService {
	return &AuthService{
		apiKeyRepo:   apiKeyRepo,
		settingsRepo: settingsRepo,
		jwtSecret:    []byte(jwtSecret),
		log:          log,
	}
}

func (s *AuthService) ValidateAPIKey(ctx context.Context, key string) (*domain.ApiKey, error) {
	hash := hashAPIKey(key)
	apiKey, err := s.apiKeyRepo.FindByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, domain.ErrUnauthorized
	}
	if !apiKey.IsValid() {
		return nil, domain.ErrUnauthorized
	}
	_ = s.apiKeyRepo.UpdateLastUsed(ctx, apiKey.ID, time.Now())
	return apiKey, nil
}

func (s *AuthService) CreateAPIKey(ctx context.Context, name string, opts *primary.APIKeyCreateOptions) (*domain.ApiKey, string, error) {
	rawKey := "or-" + uuid.New().String()
	hash := hashAPIKey(rawKey)

	key := &domain.ApiKey{
		ID:        uuid.New().String(),
		KeyHash:   hash,
		Name:      name,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	if opts != nil {
		key.GroupID = opts.GroupID
		key.RestrictedTo = opts.RestrictedTo
		key.RateLimit = opts.RateLimit
		if opts.ExpiresInDays > 0 {
			t := time.Now().AddDate(0, 0, opts.ExpiresInDays)
			key.ExpiresAt = &t
		}
	}

	if err := s.apiKeyRepo.Save(ctx, key); err != nil {
		return nil, "", err
	}
	key.Key = rawKey
	return key, rawKey, nil
}

func (s *AuthService) ListAPIKeys(ctx context.Context) ([]*domain.ApiKey, error) {
	return s.apiKeyRepo.FindAll(ctx)
}

func (s *AuthService) RevokeAPIKey(ctx context.Context, id string) error {
	key, err := s.apiKeyRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if key == nil {
		return domain.ErrNotFound
	}
	key.IsActive = false
	return s.apiKeyRepo.Update(ctx, key)
}

func (s *AuthService) GetAPIKeyPolicy(ctx context.Context, keyID string) (*domain.ApiKeyPolicy, error) {
	key, err := s.apiKeyRepo.FindByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, domain.ErrNotFound
	}
	return &domain.ApiKeyPolicy{
		KeyID:         key.ID,
		AllowedModels: key.RestrictedTo,
		RateLimit:     key.RateLimit,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, password string) (string, error) {
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return "", err
	}
	if !settings.HasPassword {
		return "", errors.New("no password configured")
	}

	storedHash, err := s.settingsRepo.GetFeatureFlag(ctx, "password_hash")
	if err != nil || storedHash == "" {
		return "", domain.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		return "", domain.ErrUnauthorized
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "admin",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) ValidateJWT(_ context.Context, tokenStr string) (map[string]any, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, domain.ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	return map[string]any(claims), nil
}

func (s *AuthService) ChangePassword(ctx context.Context, _, newPass string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.settingsRepo.SetFeatureFlag(ctx, "password_hash", string(hash))
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
