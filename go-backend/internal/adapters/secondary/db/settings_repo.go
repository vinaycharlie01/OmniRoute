package db

import (
	"context"
	"strconv"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type SettingsRepository struct {
	db *DB
}

func NewSettingsRepository(db *DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) Get(ctx context.Context) (*domain.Settings, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kv := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		kv[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s := domain.DefaultSettings()
	s.RequireLogin = kv["require_login"] == "true"
	s.HasPassword = kv["has_password"] == "true"
	s.FallbackStrategy = getString(kv, "fallback_strategy", "priority")
	s.StickyRoundRobinLimit = getInt(kv, "sticky_round_robin_limit", 5)
	s.RequestRetry = getInt(kv, "request_retry", 3)
	s.MaxRetryIntervalSec = getInt(kv, "max_retry_interval_sec", 30)
	s.MCPEnabled = kv["mcp_enabled"] == "true"
	s.MCPTransport = getString(kv, "mcp_transport", "stdio")
	s.A2AEnabled = kv["a2a_enabled"] == "true"
	s.DefaultComboStrategy = domain.ComboStrategy(getString(kv, "default_combo_strategy", "priority"))
	s.DefaultMaxRetries = getInt(kv, "default_max_retries", 3)
	s.DefaultRetryDelayMs = getInt(kv, "default_retry_delay_ms", 1000)
	s.DefaultTimeoutMs = getInt(kv, "default_timeout_ms", 60000)
	s.SSEHeartbeatIntervalMs = getInt(kv, "sse_heartbeat_interval_ms", 5000)
	s.CompressionEnabled = kv["compression_enabled"] == "true"
	s.SemanticCacheEnabled = kv["semantic_cache_enabled"] == "true"
	s.PIIRedactionEnabled = kv["pii_redaction_enabled"] == "true"
	s.ResilienceSettings = domain.DefaultResilienceSettings()
	return s, nil
}

func (r *SettingsRepository) Update(ctx context.Context, settings *domain.Settings) error {
	pairs := map[string]string{
		"require_login":              boolStr(settings.RequireLogin),
		"has_password":               boolStr(settings.HasPassword),
		"fallback_strategy":          settings.FallbackStrategy,
		"sticky_round_robin_limit":   strconv.Itoa(settings.StickyRoundRobinLimit),
		"request_retry":              strconv.Itoa(settings.RequestRetry),
		"max_retry_interval_sec":     strconv.Itoa(settings.MaxRetryIntervalSec),
		"mcp_enabled":                boolStr(settings.MCPEnabled),
		"mcp_transport":              settings.MCPTransport,
		"a2a_enabled":                boolStr(settings.A2AEnabled),
		"default_combo_strategy":     string(settings.DefaultComboStrategy),
		"default_max_retries":        strconv.Itoa(settings.DefaultMaxRetries),
		"default_retry_delay_ms":     strconv.Itoa(settings.DefaultRetryDelayMs),
		"default_timeout_ms":         strconv.Itoa(settings.DefaultTimeoutMs),
		"sse_heartbeat_interval_ms":  strconv.Itoa(settings.SSEHeartbeatIntervalMs),
		"compression_enabled":        boolStr(settings.CompressionEnabled),
		"semantic_cache_enabled":     boolStr(settings.SemanticCacheEnabled),
		"pii_redaction_enabled":      boolStr(settings.PIIRedactionEnabled),
	}

	for k, v := range pairs {
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
			k, v); err != nil {
			return err
		}
	}
	return nil
}

func (r *SettingsRepository) GetFeatureFlag(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	return value, err
}

func (r *SettingsRepository) SetFeatureFlag(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		key, value)
	return err
}

func getString(kv map[string]string, key, def string) string {
	if v, ok := kv[key]; ok && v != "" {
		return v
	}
	return def
}

func getInt(kv map[string]string, key string, def int) int {
	if v, ok := kv[key]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
