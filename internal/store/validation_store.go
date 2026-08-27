package store

import (
	"database/sql"
	"fmt"

	"task290-clockskew/internal/model"
)

// ValidationStore 管理验证运行（含断点游标）与验证失效。
type ValidationStore struct{ db *sql.DB }

func NewValidationStore(db *sql.DB) *ValidationStore { return &ValidationStore{db: db} }

// CreateRun 创建一次验证运行（running 状态）。
func (s *ValidationStore) CreateRun(r *model.ValidationRun) error {
	_, err := s.db.Exec(
		`INSERT INTO validation_runs (id, batch_id, status, cursor_measurement, cursor_exemption, failure_count, started_at)
		 VALUES (?,?,?,?,?,?,?)`,
		r.ID, r.BatchID, r.Status, r.CursorMeasurement, r.CursorExemption, r.FailureCount, r.StartedAt)
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}
	return nil
}

// BeginRun 在事务内声明批次验证运行：若已有 running 则返回该运行，否则创建新运行。
func (s *ValidationStore) BeginRun(batchID string, factory func() *model.ValidationRun) (*model.ValidationRun, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	row := tx.QueryRow(
		`SELECT id, batch_id, status, cursor_measurement, cursor_exemption, failure_count, started_at, finished_at
		 FROM validation_runs WHERE batch_id = ? AND status = ? ORDER BY started_at DESC LIMIT 1`,
		batchID, model.RunStatusRunning)
	var existing model.ValidationRun
	var fin sql.NullString
	err = row.Scan(&existing.ID, &existing.BatchID, &existing.Status, &existing.CursorMeasurement,
		&existing.CursorExemption, &existing.FailureCount, &existing.StartedAt, &fin)
	if err == nil {
		existing.FinishedAt = fin.String
		return &existing, true, tx.Commit()
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}
	run := factory()
	if _, err := tx.Exec(
		`INSERT INTO validation_runs (id, batch_id, status, cursor_measurement, cursor_exemption, failure_count, started_at)
		 VALUES (?,?,?,?,?,?,?)`,
		run.ID, run.BatchID, run.Status, run.CursorMeasurement, run.CursorExemption, run.FailureCount, run.StartedAt); err != nil {
		return nil, false, fmt.Errorf("insert run: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return run, false, nil
}

// ActiveRun 返回批次当前运行中的验证（无则返回 nil）。
func (s *ValidationStore) ActiveRun(batchID string) (*model.ValidationRun, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, status, cursor_measurement, cursor_exemption, failure_count, started_at, finished_at
		 FROM validation_runs WHERE batch_id = ? AND status = ? ORDER BY started_at DESC LIMIT 1`,
		batchID, model.RunStatusRunning)
	var r model.ValidationRun
	var fin sql.NullString
	if err := row.Scan(&r.ID, &r.BatchID, &r.Status, &r.CursorMeasurement, &r.CursorExemption,
		&r.FailureCount, &r.StartedAt, &fin); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	r.FinishedAt = fin.String
	return &r, nil
}

// UpdateCursor 推进断点游标（测量/豁免处理进度）。
func (s *ValidationStore) UpdateCursor(runID string, cursorMeasurement, cursorExemption int64) error {
	_, err := s.db.Exec(
		`UPDATE validation_runs SET cursor_measurement = ?, cursor_exemption = ? WHERE id = ?`,
		cursorMeasurement, cursorExemption, runID)
	return err
}

// CompleteRun 结束一次验证运行并记录失效数。
func (s *ValidationStore) CompleteRun(runID string, failureCount int64, at string) error {
	_, err := s.db.Exec(
		`UPDATE validation_runs SET status = ?, failure_count = ?, finished_at = ? WHERE id = ?`,
		model.RunStatusCompleted, failureCount, at, runID)
	return err
}

// Runs 列出批次全部验证运行（倒序）。
func (s *ValidationStore) Runs(batchID string) ([]*model.ValidationRun, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, status, cursor_measurement, cursor_exemption, failure_count, started_at, finished_at
		 FROM validation_runs WHERE batch_id = ? ORDER BY started_at DESC`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ValidationRun
	for rows.Next() {
		var r model.ValidationRun
		var fin sql.NullString
		if err := rows.Scan(&r.ID, &r.BatchID, &r.Status, &r.CursorMeasurement, &r.CursorExemption,
			&r.FailureCount, &r.StartedAt, &fin); err != nil {
			return nil, err
		}
		r.FinishedAt = fin.String
		out = append(out, &r)
	}
	return out, rows.Err()
}

// AddFailure 记录一条验证失效。
func (s *ValidationStore) AddFailure(f *model.ValidationFailure) error {
	_, err := s.db.Exec(
		`INSERT INTO validation_failures (id, run_id, exemption_id, measurement_id, type, message) VALUES (?,?,?,?,?,?)`,
		f.ID, f.RunID, f.ExemptionID, f.MeasurementID, f.Type, f.Message)
	if err != nil {
		return fmt.Errorf("insert failure: %w", err)
	}
	return nil
}

// Failures 返回一次运行的失效列表（按创建顺序）。
func (s *ValidationStore) Failures(runID string) ([]*model.ValidationFailure, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, exemption_id, measurement_id, type, message FROM validation_failures WHERE run_id = ? ORDER BY id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ValidationFailure
	for rows.Next() {
		var f model.ValidationFailure
		var ex, me sql.NullString
		if err := rows.Scan(&f.ID, &f.RunID, &ex, &me, &f.Type, &f.Message); err != nil {
			return nil, err
		}
		f.ExemptionID, f.MeasurementID = ex.String, me.String
		out = append(out, &f)
	}
	return out, rows.Err()
}

// ClearFailures 清空某运行的全部失效记录（新验证轮次开始时调用）。
func (s *ValidationStore) ClearFailures(runID string) error {
	_, err := s.db.Exec(`DELETE FROM validation_failures WHERE run_id = ?`, runID)
	return err
}

// LastFailures 返回批次最近一次完成的验证运行的失效（供快照与展示）。
func (s *ValidationStore) LastFailures(batchID string) ([]*model.ValidationFailure, error) {
	var runID sql.NullString
	err := s.db.QueryRow(
		`SELECT id FROM validation_runs WHERE batch_id = ? AND status = ? ORDER BY started_at DESC LIMIT 1`,
		batchID, model.RunStatusCompleted).Scan(&runID)
	if err == sql.ErrNoRows || !runID.Valid {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.Failures(runID.String)
}
