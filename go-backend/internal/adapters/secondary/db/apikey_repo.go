package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type ApiKeyRepository struct {
	db *DB
}

func NewApiKeyRepository(db *DB) *ApiKeyRepository {
	return &ApiKeyRepository{db: db}
}

func (r *ApiKeyRepository) FindAll(ctx context.Context) ([]*domain.ApiKey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, key_hash, name, group_id, is_active, created_at, last_used_at,
		       expires_at, restricted_to, rate_limit_rpm, rate_limit_tpm, rate_limit_daily
		FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.ApiKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *ApiKeyRepository) FindByID(ctx context.Context, id string) (*domain.ApiKey, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, key_hash, name, group_id, is_active, created_at, last_used_at,
		       expires_at, restricted_to, rate_limit_rpm, rate_limit_tpm, rate_limit_daily
		FROM api_keys WHERE id = ?`, id)
	k, err := scanAPIKeyRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return k, err
}

func (r *ApiKeyRepository) FindByHash(ctx context.Context, hash string) (*domain.ApiKey, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, key_hash, name, group_id, is_active, created_at, last_used_at,
		       expires_at, restricted_to, rate_limit_rpm, rate_limit_tpm, rate_limit_daily
		FROM api_keys WHERE key_hash = ? AND is_active = 1`, hash)
	k, err := scanAPIKeyRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return k, err
}

func (r *ApiKeyRepository) Save(ctx context.Context, key *domain.ApiKey) error {
	restricted, _ := json.Marshal(key.RestrictedTo)
	var rpm, tpm, daily sql.NullInt64
	if key.RateLimit != nil {
		if key.RateLimit.RPM > 0 {
			rpm = sql.NullInt64{Int64: int64(key.RateLimit.RPM), Valid: true}
		}
		if key.RateLimit.TPM > 0 {
			tpm = sql.NullInt64{Int64: int64(key.RateLimit.TPM), Valid: true}
		}
		if key.RateLimit.DailyLimit > 0 {
			daily = sql.NullInt64{Int64: int64(key.RateLimit.DailyLimit), Valid: true}
		}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO api_keys (id, key_hash, name, group_id, is_active, created_at,
		  expires_at, restricted_to, rate_limit_rpm, rate_limit_tpm, rate_limit_daily)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		key.ID, key.KeyHash, key.Name, key.GroupID, boolToInt(key.IsActive),
		key.CreatedAt, key.ExpiresAt, string(restricted), rpm, tpm, daily)
	return err
}

func (r *ApiKeyRepository) Update(ctx context.Context, key *domain.ApiKey) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE api_keys SET is_active = ?, name = ? WHERE id = ?`,
		boolToInt(key.IsActive), key.Name, key.ID)
	return err
}

func (r *ApiKeyRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
	return err
}

func (r *ApiKeyRepository) UpdateLastUsed(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, at, id)
	return err
}

type apiKeyScanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(rows *sql.Rows) (*domain.ApiKey, error) {
	return scanAPIKeyRow(rows)
}

func scanAPIKeyRow(s apiKeyScanner) (*domain.ApiKey, error) {
	var (
		key          domain.ApiKey
		groupID      sql.NullString
		lastUsedAt   sql.NullTime
		expiresAt    sql.NullTime
		restrictedTo sql.NullString
		rpm          sql.NullInt64
		tpm          sql.NullInt64
		daily        sql.NullInt64
		isActive     int
	)
	err := s.Scan(
		&key.ID, &key.KeyHash, &key.Name, &groupID, &isActive, &key.CreatedAt,
		&lastUsedAt, &expiresAt, &restrictedTo, &rpm, &tpm, &daily,
	)
	if err != nil {
		return nil, err
	}
	key.IsActive = isActive == 1
	key.GroupID = groupID.String
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if restrictedTo.Valid && restrictedTo.String != "" {
		_ = json.Unmarshal([]byte(restrictedTo.String), &key.RestrictedTo)
	}
	if rpm.Valid || tpm.Valid || daily.Valid {
		key.RateLimit = &domain.RateLimit{
			RPM:        int(rpm.Int64),
			TPM:        int(tpm.Int64),
			DailyLimit: int(daily.Int64),
		}
	}
	return &key, nil
}
