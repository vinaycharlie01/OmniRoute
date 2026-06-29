package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/omniroute/go-backend/internal/core/domain"
)

type ComboRepository struct {
	db *DB
}

func NewComboRepository(db *DB) *ComboRepository {
	return &ComboRepository{db: db}
}

func (r *ComboRepository) FindAll(ctx context.Context) ([]*domain.Combo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, model, strategy, is_active, max_retries, retry_delay_ms,
		       fallback_delay_ms, timeout_ms, health_check_enabled, max_combo_depth,
		       track_metrics, reasoning_token_buffer, fusion_judge_model,
		       fusion_max_candidates, created_at, updated_at
		FROM combos ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var combos []*domain.Combo
	for rows.Next() {
		combo, err := scanCombo(rows)
		if err != nil {
			return nil, err
		}
		nodes, err := r.findNodes(ctx, combo.ID)
		if err != nil {
			return nil, err
		}
		combo.Nodes = nodes
		combos = append(combos, combo)
	}
	return combos, rows.Err()
}

func (r *ComboRepository) FindByID(ctx context.Context, id string) (*domain.Combo, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, model, strategy, is_active, max_retries, retry_delay_ms,
		       fallback_delay_ms, timeout_ms, health_check_enabled, max_combo_depth,
		       track_metrics, reasoning_token_buffer, fusion_judge_model,
		       fusion_max_candidates, created_at, updated_at
		FROM combos WHERE id = ?`, id)
	combo, err := scanComboRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	combo.Nodes, err = r.findNodes(ctx, combo.ID)
	return combo, err
}

func (r *ComboRepository) FindByModel(ctx context.Context, model string) (*domain.Combo, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, model, strategy, is_active, max_retries, retry_delay_ms,
		       fallback_delay_ms, timeout_ms, health_check_enabled, max_combo_depth,
		       track_metrics, reasoning_token_buffer, fusion_judge_model,
		       fusion_max_candidates, created_at, updated_at
		FROM combos WHERE model = ? AND is_active = 1`, model)
	combo, err := scanComboRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	combo.Nodes, err = r.findNodes(ctx, combo.ID)
	return combo, err
}

func (r *ComboRepository) Save(ctx context.Context, combo *domain.Combo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO combos
		(id, name, model, strategy, is_active, max_retries, retry_delay_ms,
		 fallback_delay_ms, timeout_ms, health_check_enabled, max_combo_depth,
		 track_metrics, reasoning_token_buffer, fusion_judge_model,
		 fusion_max_candidates, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		combo.ID, combo.Name, combo.Model, combo.Strategy, boolToInt(combo.IsActive),
		combo.MaxRetries, combo.RetryDelayMs, combo.FallbackDelayMs, combo.TimeoutMs,
		boolToInt(combo.HealthCheckEnabled), combo.MaxComboDepth, boolToInt(combo.TrackMetrics),
		boolToInt(combo.ReasoningTokenBuffer), combo.FusionJudgeModel, combo.FusionMaxCandidates,
		combo.CreatedAt, combo.UpdatedAt)
	if err != nil {
		return err
	}

	if err := r.saveNodes(ctx, tx, combo.ID, combo.Nodes); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ComboRepository) Update(ctx context.Context, combo *domain.Combo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE combos SET
		  name = ?, model = ?, strategy = ?, is_active = ?, max_retries = ?,
		  retry_delay_ms = ?, fallback_delay_ms = ?, timeout_ms = ?,
		  health_check_enabled = ?, max_combo_depth = ?, track_metrics = ?,
		  reasoning_token_buffer = ?, fusion_judge_model = ?,
		  fusion_max_candidates = ?, updated_at = ?
		WHERE id = ?`,
		combo.Name, combo.Model, combo.Strategy, boolToInt(combo.IsActive),
		combo.MaxRetries, combo.RetryDelayMs, combo.FallbackDelayMs, combo.TimeoutMs,
		boolToInt(combo.HealthCheckEnabled), combo.MaxComboDepth, boolToInt(combo.TrackMetrics),
		boolToInt(combo.ReasoningTokenBuffer), combo.FusionJudgeModel, combo.FusionMaxCandidates,
		time.Now(), combo.ID)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM combo_nodes WHERE combo_id = ?`, combo.ID); err != nil {
		return err
	}
	if err := r.saveNodes(ctx, tx, combo.ID, combo.Nodes); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ComboRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM combos WHERE id = ?`, id)
	return err
}

func (r *ComboRepository) findNodes(ctx context.Context, comboID string) ([]domain.ComboNode, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, combo_id, connection_id, provider, model, weight, priority, is_active
		FROM combo_nodes WHERE combo_id = ? ORDER BY priority ASC`, comboID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []domain.ComboNode
	for rows.Next() {
		var (
			node     domain.ComboNode
			isActive int
		)
		if err := rows.Scan(&node.ID, &node.ComboID, &node.ConnectionID, &node.Provider,
			&node.Model, &node.Weight, &node.Priority, &isActive); err != nil {
			return nil, err
		}
		node.IsActive = isActive == 1
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func (r *ComboRepository) saveNodes(ctx context.Context, tx *sql.Tx, comboID string, nodes []domain.ComboNode) error {
	for _, node := range nodes {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO combo_nodes (id, combo_id, connection_id, provider, model, weight, priority, is_active)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			node.ID, comboID, node.ConnectionID, node.Provider, node.Model,
			node.Weight, node.Priority, boolToInt(node.IsActive))
		if err != nil {
			return err
		}
	}
	return nil
}

func scanCombo(rows *sql.Rows) (*domain.Combo, error) {
	return scanComboRow(rows)
}

type comboScanner interface {
	Scan(dest ...any) error
}

func scanComboRow(s comboScanner) (*domain.Combo, error) {
	var (
		combo              domain.Combo
		isActive           int
		healthCheck        int
		trackMetrics       int
		reasoningBuffer    int
		fusionJudgeModel   sql.NullString
	)
	err := s.Scan(
		&combo.ID, &combo.Name, &combo.Model, &combo.Strategy, &isActive,
		&combo.MaxRetries, &combo.RetryDelayMs, &combo.FallbackDelayMs, &combo.TimeoutMs,
		&healthCheck, &combo.MaxComboDepth, &trackMetrics, &reasoningBuffer,
		&fusionJudgeModel, &combo.FusionMaxCandidates, &combo.CreatedAt, &combo.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	combo.IsActive = isActive == 1
	combo.HealthCheckEnabled = healthCheck == 1
	combo.TrackMetrics = trackMetrics == 1
	combo.ReasoningTokenBuffer = reasoningBuffer == 1
	combo.FusionJudgeModel = fusionJudgeModel.String
	return &combo, nil
}
