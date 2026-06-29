package db

import (
	"context"
	"time"

	"github.com/omniroute/go-backend/internal/core/domain"
	"github.com/omniroute/go-backend/internal/core/ports/secondary"
)

type UsageRepository struct {
	db *DB
}

func NewUsageRepository(db *DB) *UsageRepository {
	return &UsageRepository{db: db}
}

func (r *UsageRepository) SaveCallLog(ctx context.Context, log *domain.CallLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO call_logs
		(id, request_id, api_key_id, provider, model, connection_id,
		 input_tokens, output_tokens, total_tokens, cost_usd, status_code,
		 error_type, error_message, duration_ms, is_streaming, cache_hit, combo_id, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.RequestID, nullStr(log.APIKeyID), log.Provider, log.Model,
		nullStr(log.ConnectionID), log.InputTokens, log.OutputTokens, log.TotalTokens,
		log.CostUSD, log.StatusCode, nullStr(log.ErrorType), nullStr(log.ErrorMessage),
		log.DurationMs, boolToInt(log.IsStreaming), boolToInt(log.CacheHit),
		nullStr(log.ComboID), log.Timestamp)
	return err
}

func (r *UsageRepository) FindCallLogs(ctx context.Context, filter secondary.CallLogFilter) ([]*domain.CallLog, error) {
	query := `SELECT id, request_id, api_key_id, provider, model, connection_id,
	                 input_tokens, output_tokens, total_tokens, cost_usd, status_code,
	                 error_type, error_message, duration_ms, is_streaming, cache_hit,
	                 combo_id, timestamp
	          FROM call_logs WHERE 1=1`
	args := []any{}

	if filter.Provider != "" {
		query += ` AND provider = ?`
		args = append(args, filter.Provider)
	}
	if filter.Model != "" {
		query += ` AND model = ?`
		args = append(args, filter.Model)
	}
	if filter.APIKeyID != "" {
		query += ` AND api_key_id = ?`
		args = append(args, filter.APIKeyID)
	}
	if filter.StartTime != nil {
		query += ` AND timestamp >= ?`
		args = append(args, filter.StartTime)
	}
	if filter.EndTime != nil {
		query += ` AND timestamp <= ?`
		args = append(args, filter.EndTime)
	}

	query += ` ORDER BY timestamp DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.CallLog
	for rows.Next() {
		var (
			log     domain.CallLog
			apiKeyID, connID, errType, errMsg, comboID nullableStr
			isStreaming, cacheHit                      int
		)
		if err := rows.Scan(
			&log.ID, &log.RequestID, &apiKeyID, &log.Provider, &log.Model, &connID,
			&log.InputTokens, &log.OutputTokens, &log.TotalTokens, &log.CostUSD,
			&log.StatusCode, &errType, &errMsg, &log.DurationMs, &isStreaming, &cacheHit,
			&comboID, &log.Timestamp,
		); err != nil {
			return nil, err
		}
		log.APIKeyID = string(apiKeyID)
		log.ConnectionID = string(connID)
		log.ErrorType = string(errType)
		log.ErrorMessage = string(errMsg)
		log.ComboID = string(comboID)
		log.IsStreaming = isStreaming == 1
		log.CacheHit = cacheHit == 1
		logs = append(logs, &log)
	}
	return logs, rows.Err()
}

func (r *UsageRepository) GetUsageStats(ctx context.Context, filter secondary.UsageStatsFilter) ([]*domain.UsageStats, error) {
	query := `
		SELECT provider, model,
		       SUM(input_tokens) as input_tokens, SUM(output_tokens) as output_tokens,
		       SUM(total_tokens) as total_tokens, SUM(cost_usd) as total_cost,
		       COUNT(*) as call_count, AVG(duration_ms) as avg_latency_ms,
		       AVG(CASE WHEN status_code >= 400 THEN 1.0 ELSE 0.0 END) as error_rate
		FROM call_logs WHERE 1=1`
	args := []any{}

	if filter.Provider != "" {
		query += ` AND provider = ?`
		args = append(args, filter.Provider)
	}
	if filter.StartTime != nil {
		query += ` AND timestamp >= ?`
		args = append(args, filter.StartTime)
	}
	if filter.EndTime != nil {
		query += ` AND timestamp <= ?`
		args = append(args, filter.EndTime)
	}
	query += ` GROUP BY provider, model ORDER BY total_tokens DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*domain.UsageStats
	for rows.Next() {
		var s domain.UsageStats
		if err := rows.Scan(
			&s.Provider, &s.Model, &s.InputTokens, &s.OutputTokens,
			&s.TotalCost, &s.CallCount, &s.AvgLatencyMs, &s.ErrorRate,
			&s.TotalTokens,
		); err != nil {
			return nil, err
		}
		stats = append(stats, &s)
	}
	return stats, rows.Err()
}

func (r *UsageRepository) GetTotalCost(ctx context.Context, apiKeyID string, since time.Time) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cost_usd), 0) FROM call_logs WHERE api_key_id = ? AND timestamp >= ?`,
		apiKeyID, since).Scan(&total)
	return total, err
}

func (r *UsageRepository) GetTokenCount(ctx context.Context, apiKeyID string, windowStart time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(total_tokens), 0) FROM call_logs WHERE api_key_id = ? AND timestamp >= ?`,
		apiKeyID, windowStart).Scan(&count)
	return count, err
}

type nullableStr string

func (n *nullableStr) Scan(value any) error {
	if value == nil {
		*n = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*n = nullableStr(v)
	case []byte:
		*n = nullableStr(v)
	}
	return nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
