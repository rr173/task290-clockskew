package store

import (
	"database/sql"
	"fmt"

	"task290-clockskew/internal/model"
)

// MeasurementStore 管理偏斜测量：批量导入（幂等）、查询与状态更新。
type MeasurementStore struct{ db *sql.DB }

func NewMeasurementStore(db *sql.DB) *MeasurementStore { return &MeasurementStore{db: db} }

// NextSeq 返回批次内下一个测量序号（事务内保证并发安全）。
func (s *MeasurementStore) NextSeq(batchID string) (int64, error) {
	var seq sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(seq) FROM skew_measurements WHERE batch_id = ?`, batchID).Scan(&seq)
	if err != nil {
		return 0, err
	}
	if !seq.Valid {
		return 1, nil
	}
	return seq.Int64 + 1, nil
}

// ExistsChecksum 判断测量摘要是否已存在（幂等判重）。
func (s *MeasurementStore) ExistsChecksum(checksum string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM skew_measurements WHERE checksum = ?`, checksum).Scan(&n)
	return n > 0, err
}

// InsertWithAutoSeq 在单事务内分配 seq 并插入测量，保证并发导入序号唯一。
func (s *MeasurementStore) InsertWithAutoSeq(m *model.SkewMeasurement) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var seq sql.NullInt64
	if err := tx.QueryRow(`SELECT MAX(seq) FROM skew_measurements WHERE batch_id = ?`, m.BatchID).Scan(&seq); err != nil {
		return err
	}
	if !seq.Valid {
		m.Seq = 1
	} else {
		m.Seq = seq.Int64 + 1
	}
	if _, err := tx.Exec(
		`INSERT INTO skew_measurements (id, batch_id, seq, endpoint_a, endpoint_b, mode_id, corner_id, skew_ps, measured_at, checksum, status, reason, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		m.ID, m.BatchID, m.Seq, m.EndpointA, m.EndpointB, m.ModeID, m.CornerID,
		m.SkewPS, m.MeasuredAt, m.Checksum, m.Status, m.Reason, m.CreatedAt); err != nil {
		return fmt.Errorf("insert measurement: %w", err)
	}
	return tx.Commit()
}

// Insert 插入一条测量；校验和唯一冲突视为幂等重复。
func (s *MeasurementStore) Insert(m *model.SkewMeasurement) (bool, error) {
	_, err := s.db.Exec(
		`INSERT INTO skew_measurements (id, batch_id, seq, endpoint_a, endpoint_b, mode_id, corner_id, skew_ps, measured_at, checksum, status, reason, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		m.ID, m.BatchID, m.Seq, m.EndpointA, m.EndpointB, m.ModeID, m.CornerID,
		m.SkewPS, m.MeasuredAt, m.Checksum, m.Status, m.Reason, m.CreatedAt)
	if err != nil {
		return false, fmt.Errorf("insert measurement: %w", err)
	}
	return true, nil
}

// List 列出批次下全部测量（按 seq 升序）。
func (s *MeasurementStore) List(batchID string) ([]*model.SkewMeasurement, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, seq, endpoint_a, endpoint_b, mode_id, corner_id, skew_ps, measured_at, checksum, status, reason, created_at
		 FROM skew_measurements WHERE batch_id = ? ORDER BY seq`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.SkewMeasurement
	for rows.Next() {
		var m model.SkewMeasurement
		if err := rows.Scan(&m.ID, &m.BatchID, &m.Seq, &m.EndpointA, &m.EndpointB, &m.ModeID,
			&m.CornerID, &m.SkewPS, &m.MeasuredAt, &m.Checksum, &m.Status, &m.Reason, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// Get 按 ID 查询测量。
func (s *MeasurementStore) Get(id string) (*model.SkewMeasurement, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, seq, endpoint_a, endpoint_b, mode_id, corner_id, skew_ps, measured_at, checksum, status, reason, created_at
		 FROM skew_measurements WHERE id = ?`, id)
	var m model.SkewMeasurement
	if err := row.Scan(&m.ID, &m.BatchID, &m.Seq, &m.EndpointA, &m.EndpointB, &m.ModeID,
		&m.CornerID, &m.SkewPS, &m.MeasuredAt, &m.Checksum, &m.Status, &m.Reason, &m.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("measurement not found")
		}
		return nil, err
	}
	return &m, nil
}

// UpdateStatus 更新测量状态与原因。
func (s *MeasurementStore) UpdateStatus(id, status, reason string) error {
	_, err := s.db.Exec(`UPDATE skew_measurements SET status = ?, reason = ? WHERE id = ?`, status, reason, id)
	return err
}

// ResetStatuses 将批次内所有测量重置为 raw（供复验使用），并清空原因。
func (s *MeasurementStore) ResetStatuses(batchID string) error {
	_, err := s.db.Exec(`UPDATE skew_measurements SET status = ?, reason = '' WHERE batch_id = ?`, model.MeasurementStatusRaw, batchID)
	return err
}
