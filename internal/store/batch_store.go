package store

import (
	"database/sql"
	"fmt"

	"task290-clockskew/internal/model"
)

// BatchStore 负责签核批次的持久化与状态流转。
type BatchStore struct{ db *sql.DB }

func NewBatchStore(db *sql.DB) *BatchStore { return &BatchStore{db: db} }

// Create 创建批次（状态固定为 receiving），name 唯一。
func (s *BatchStore) Create(b *model.Batch) error {
	_, err := s.db.Exec(
		`INSERT INTO batches (id, name, status, default_max_skew_ps, created_at) VALUES (?,?,?,?,?)`,
		b.ID, b.Name, model.BatchStatusReceiving, b.DefaultMaxSkewPS, b.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert batch: %w", err)
	}
	return nil
}

// Get 按 ID 查询批次。
func (s *BatchStore) Get(id string) (*model.Batch, error) {
	row := s.db.QueryRow(
		`SELECT id, name, status, default_max_skew_ps, created_at, opened_at, sealed_at FROM batches WHERE id = ?`, id)
	var b model.Batch
	var opened, sealed sql.NullString
	if err := row.Scan(&b.ID, &b.Name, &b.Status, &b.DefaultMaxSkewPS, &b.CreatedAt, &opened, &sealed); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrBatchNotFound
		}
		return nil, err
	}
	b.OpenedAt, b.SealedAt = opened.String, sealed.String
	return &b, nil
}

// List 列出全部批次（按创建时间倒序）。
func (s *BatchStore) List() ([]*model.Batch, error) {
	rows, err := s.db.Query(
		`SELECT id, name, status, default_max_skew_ps, created_at, opened_at, sealed_at FROM batches ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Batch
	for rows.Next() {
		var b model.Batch
		var opened, sealed sql.NullString
		if err := rows.Scan(&b.ID, &b.Name, &b.Status, &b.DefaultMaxSkewPS, &b.CreatedAt, &opened, &sealed); err != nil {
			return nil, err
		}
		b.OpenedAt, b.SealedAt = opened.String, sealed.String
		out = append(out, &b)
	}
	return out, rows.Err()
}

// UpdateStatus 原子更新批次状态（CAS：期望旧状态，防并发乱序流转）。
func (s *BatchStore) UpdateStatus(id, from, to string) error {
	res, err := s.db.Exec(
		`UPDATE batches SET status = ? WHERE id = ? AND status = ?`, to, id, from)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("batch %s not in state %s", id, from)
	}
	return nil
}

// MarkSealed 记录封存时间并置为 sealed（publishable → sealed 的 CAS 路径）。
func (s *BatchStore) MarkSealed(id, at string) error {
	res, err := s.db.Exec(
		`UPDATE batches SET status = ?, sealed_at = ? WHERE id = ? AND status = ?`,
		model.BatchStatusSealed, at, id, model.BatchStatusPublishable)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("batch %s is not publishable", id)
	}
	return nil
}
