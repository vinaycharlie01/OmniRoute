package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

// Open opens a SQLite database at the given path with WAL mode.
func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?cache=shared&_journal_mode=WAL&_foreign_keys=on&_synchronous=NORMAL", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	sqlDB.SetMaxOpenConns(1) // SQLite WAL allows one writer.
	sqlDB.SetMaxIdleConns(1)

	if err := sqlDB.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return &DB{sqlDB}, nil
}

// Migrate runs all embedded SQL migration files in order.
func (db *DB) Migrate() error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	for _, m := range migrations {
		var exists bool
		row := db.QueryRow(`SELECT 1 FROM schema_migrations WHERE version = ?`, m.version)
		_ = row.Scan(&exists)
		if exists {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin transaction for %s: %w", m.version, err)
		}

		if _, err := tx.Exec(m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", m.version, err)
		}

		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.version, err)
		}
	}
	return nil
}

type migration struct {
	version string
	sql     string
}

var migrations = []migration{
	{"001_initial_schema", sqlInitialSchema},
	{"002_circuit_breakers", sqlCircuitBreakers},
	{"003_settings", sqlSettings},
}

const sqlInitialSchema = `
CREATE TABLE IF NOT EXISTS provider_connections (
	id TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	label TEXT NOT NULL,
	base_url TEXT NOT NULL,
	auth_type TEXT NOT NULL DEFAULT 'api_key',
	api_key_encrypted TEXT,
	oauth_token_encrypted TEXT,
	oauth_refresh_token_encrypted TEXT,
	oauth_expires_at DATETIME,
	status TEXT NOT NULL DEFAULT 'active',
	is_active INTEGER NOT NULL DEFAULT 1,
	priority INTEGER NOT NULL DEFAULT 0,
	weight INTEGER NOT NULL DEFAULT 1,
	rate_limited_until DATETIME,
	last_error TEXT,
	last_error_type TEXT,
	error_code TEXT,
	backoff_level INTEGER NOT NULL DEFAULT 0,
	max_concurrent INTEGER NOT NULL DEFAULT 0,
	email TEXT,
	provider_specific TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS combos (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	model TEXT NOT NULL UNIQUE,
	strategy TEXT NOT NULL DEFAULT 'priority',
	is_active INTEGER NOT NULL DEFAULT 1,
	max_retries INTEGER NOT NULL DEFAULT 3,
	retry_delay_ms INTEGER NOT NULL DEFAULT 1000,
	fallback_delay_ms INTEGER NOT NULL DEFAULT 0,
	timeout_ms INTEGER NOT NULL DEFAULT 60000,
	health_check_enabled INTEGER NOT NULL DEFAULT 0,
	max_combo_depth INTEGER NOT NULL DEFAULT 3,
	track_metrics INTEGER NOT NULL DEFAULT 1,
	reasoning_token_buffer INTEGER NOT NULL DEFAULT 0,
	fusion_judge_model TEXT,
	fusion_max_candidates INTEGER NOT NULL DEFAULT 3,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS combo_nodes (
	id TEXT PRIMARY KEY,
	combo_id TEXT NOT NULL REFERENCES combos(id) ON DELETE CASCADE,
	connection_id TEXT NOT NULL REFERENCES provider_connections(id) ON DELETE CASCADE,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	weight INTEGER NOT NULL DEFAULT 1,
	priority INTEGER NOT NULL DEFAULT 0,
	is_active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY,
	key_hash TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	group_id TEXT,
	is_active INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	last_used_at DATETIME,
	expires_at DATETIME,
	restricted_to TEXT,
	rate_limit_rpm INTEGER,
	rate_limit_tpm INTEGER,
	rate_limit_daily INTEGER
);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_provider_connections_provider ON provider_connections(provider);
CREATE INDEX IF NOT EXISTS idx_combo_nodes_combo ON combo_nodes(combo_id);

CREATE TABLE IF NOT EXISTS models (
	id TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	name TEXT NOT NULL,
	display_name TEXT,
	format TEXT NOT NULL DEFAULT 'openai',
	capabilities TEXT,
	context_window INTEGER NOT NULL DEFAULT 4096,
	max_output_tokens INTEGER NOT NULL DEFAULT 4096,
	input_cost_per_mtok REAL NOT NULL DEFAULT 0,
	output_cost_per_mtok REAL NOT NULL DEFAULT 0,
	is_deprecated INTEGER NOT NULL DEFAULT 0,
	deprecated_by TEXT,
	is_active INTEGER NOT NULL DEFAULT 1,
	tags TEXT
);

CREATE TABLE IF NOT EXISTS call_logs (
	id TEXT PRIMARY KEY,
	request_id TEXT NOT NULL,
	api_key_id TEXT,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	connection_id TEXT,
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	cost_usd REAL NOT NULL DEFAULT 0,
	status_code INTEGER NOT NULL DEFAULT 200,
	error_type TEXT,
	error_message TEXT,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	is_streaming INTEGER NOT NULL DEFAULT 0,
	cache_hit INTEGER NOT NULL DEFAULT 0,
	combo_id TEXT,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_call_logs_timestamp ON call_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_call_logs_provider ON call_logs(provider);
CREATE INDEX IF NOT EXISTS idx_call_logs_api_key ON call_logs(api_key_id);
`

const sqlCircuitBreakers = `
CREATE TABLE IF NOT EXISTS circuit_breakers (
	provider TEXT PRIMARY KEY,
	state TEXT NOT NULL DEFAULT 'CLOSED',
	failures INTEGER NOT NULL DEFAULT 0,
	last_failure_at DATETIME,
	reset_at DATETIME,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

const sqlSettings = `
CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO settings (key, value) VALUES
	('require_login', 'false'),
	('has_password', 'false'),
	('fallback_strategy', 'priority'),
	('sticky_round_robin_limit', '5'),
	('request_retry', '3'),
	('max_retry_interval_sec', '30'),
	('mcp_enabled', 'false'),
	('mcp_transport', 'stdio'),
	('a2a_enabled', 'false'),
	('default_combo_strategy', 'priority'),
	('default_max_retries', '3'),
	('default_retry_delay_ms', '1000'),
	('default_timeout_ms', '60000'),
	('sse_heartbeat_interval_ms', '5000'),
	('compression_enabled', 'false'),
	('semantic_cache_enabled', 'false'),
	('pii_redaction_enabled', 'false');
`
