package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type ProviderRepository struct {
	db *DB
}

func NewProviderRepository(db *DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

func (r *ProviderRepository) FindAll(ctx context.Context) ([]*domain.ProviderConnection, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, provider, label, base_url, auth_type, status, is_active,
		       priority, weight, rate_limited_until, last_error, last_error_type,
		       error_code, backoff_level, max_concurrent, email, provider_specific,
		       created_at, updated_at
		FROM provider_connections ORDER BY priority ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProviders(rows)
}

func (r *ProviderRepository) FindByID(ctx context.Context, id string) (*domain.ProviderConnection, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, provider, label, base_url, auth_type, status, is_active,
		       priority, weight, rate_limited_until, last_error, last_error_type,
		       error_code, backoff_level, max_concurrent, email, provider_specific,
		       created_at, updated_at
		FROM provider_connections WHERE id = ?`, id)
	return scanProvider(row)
}

func (r *ProviderRepository) FindByProvider(ctx context.Context, provider domain.ProviderType) ([]*domain.ProviderConnection, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, provider, label, base_url, auth_type, status, is_active,
		       priority, weight, rate_limited_until, last_error, last_error_type,
		       error_code, backoff_level, max_concurrent, email, provider_specific,
		       created_at, updated_at
		FROM provider_connections WHERE provider = ? ORDER BY priority ASC`, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProviders(rows)
}

func (r *ProviderRepository) FindAvailable(ctx context.Context, provider domain.ProviderType) ([]*domain.ProviderConnection, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, provider, label, base_url, auth_type, status, is_active,
		       priority, weight, rate_limited_until, last_error, last_error_type,
		       error_code, backoff_level, max_concurrent, email, provider_specific,
		       created_at, updated_at
		FROM provider_connections
		WHERE provider = ?
		  AND is_active = 1
		  AND status IN ('active', 'unavailable')
		ORDER BY priority ASC`, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProviders(rows)
}

func (r *ProviderRepository) Save(ctx context.Context, conn *domain.ProviderConnection) error {
	ps, _ := json.Marshal(conn.ProviderSpecific)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO provider_connections
		(id, provider, label, base_url, auth_type, status, is_active, priority, weight,
		 max_concurrent, email, provider_specific, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		conn.ID, conn.Provider, conn.Label, conn.BaseURL, conn.AuthType,
		conn.Status, boolToInt(conn.IsActive), conn.Priority, conn.Weight,
		conn.MaxConcurrent, conn.Email, string(ps), conn.CreatedAt, conn.UpdatedAt)
	return err
}

func (r *ProviderRepository) Update(ctx context.Context, conn *domain.ProviderConnection) error {
	ps, _ := json.Marshal(conn.ProviderSpecific)
	_, err := r.db.ExecContext(ctx, `
		UPDATE provider_connections SET
		  label = ?, base_url = ?, auth_type = ?, status = ?, is_active = ?,
		  priority = ?, weight = ?, max_concurrent = ?, email = ?,
		  provider_specific = ?, updated_at = ?
		WHERE id = ?`,
		conn.Label, conn.BaseURL, conn.AuthType, conn.Status, boolToInt(conn.IsActive),
		conn.Priority, conn.Weight, conn.MaxConcurrent, conn.Email,
		string(ps), time.Now(), conn.ID)
	return err
}

func (r *ProviderRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM provider_connections WHERE id = ?`, id)
	return err
}

func (r *ProviderRepository) UpdateCooldown(ctx context.Context, id string, until time.Time, level int, errType, errCode, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE provider_connections SET
		  rate_limited_until = ?, backoff_level = ?, last_error = ?,
		  last_error_type = ?, error_code = ?, status = 'unavailable', updated_at = ?
		WHERE id = ?`,
		until, level, errMsg, errType, errCode, time.Now(), id)
	return err
}

func (r *ProviderRepository) ClearCooldown(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE provider_connections SET
		  rate_limited_until = NULL, backoff_level = 0, last_error = NULL,
		  last_error_type = NULL, error_code = NULL, status = 'active', updated_at = ?
		WHERE id = ?`, time.Now(), id)
	return err
}

func (r *ProviderRepository) UpdateStatus(ctx context.Context, id string, status domain.ConnectionStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE provider_connections SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), id)
	return err
}

func scanProviders(rows *sql.Rows) ([]*domain.ProviderConnection, error) {
	var results []*domain.ProviderConnection
	for rows.Next() {
		conn, err := scanProviderRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, conn)
	}
	return results, rows.Err()
}

func scanProvider(row *sql.Row) (*domain.ProviderConnection, error) {
	conn, err := scanProviderRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return conn, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProviderRow(s scanner) (*domain.ProviderConnection, error) {
	var (
		conn           domain.ProviderConnection
		rateLimitedUntil sql.NullTime
		oauthExpiresAt sql.NullTime
		lastError      sql.NullString
		lastErrorType  sql.NullString
		errorCode      sql.NullString
		email          sql.NullString
		providerSpec   sql.NullString
		isActive       int
	)

	err := s.Scan(
		&conn.ID, &conn.Provider, &conn.Label, &conn.BaseURL, &conn.AuthType,
		&conn.Status, &isActive, &conn.Priority, &conn.Weight,
		&rateLimitedUntil, &lastError, &lastErrorType, &errorCode,
		&conn.BackoffLevel, &conn.MaxConcurrent, &email, &providerSpec,
		&conn.CreatedAt, &conn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	conn.IsActive = isActive == 1
	if rateLimitedUntil.Valid {
		conn.RateLimitedUntil = &rateLimitedUntil.Time
	}
	if oauthExpiresAt.Valid {
		conn.OAuthExpiresAt = &oauthExpiresAt.Time
	}
	conn.LastError = lastError.String
	conn.LastErrorType = lastErrorType.String
	conn.ErrorCode = errorCode.String
	conn.Email = email.String
	if providerSpec.Valid && providerSpec.String != "" {
		_ = json.Unmarshal([]byte(providerSpec.String), &conn.ProviderSpecific)
	}
	return &conn, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
