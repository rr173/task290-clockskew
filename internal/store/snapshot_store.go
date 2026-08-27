package store

import (
	"database/sql"
	"fmt"

	"task290-clockskew/internal/model"
)

// SnapshotStore 管理验证快照及其摘要条目。
type SnapshotStore struct{ db *sql.DB }

func NewSnapshotStore(db *sql.DB) *SnapshotStore { return &SnapshotStore{db: db} }

// CreateWithItems 在单事务内创建草稿快照及其全部摘要条目。
func (s *SnapshotStore) CreateWithItems(snap *model.ValidationSnapshot, items []*model.SnapshotItem) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO validation_snapshots (id, batch_id, name, status, content_hash, frozen_corners, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		snap.ID, snap.BatchID, snap.Name, snap.Status, snap.ContentHash, snap.FrozenCorners, snap.CreatedAt); err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	for _, item := range items {
		if _, err := tx.Exec(
			`INSERT INTO snapshot_items (id, snapshot_id, kind, ref_id, summary) VALUES (?,?,?,?,?)`,
			item.ID, item.SnapshotID, item.Kind, item.RefID, item.Summary); err != nil {
			return fmt.Errorf("insert snapshot item: %w", err)
		}
	}
	return tx.Commit()
}

// Create 创建草稿快照；同批次同名冲突拒绝。
func (s *SnapshotStore) Create(snap *model.ValidationSnapshot) error {
	_, err := s.db.Exec(
		`INSERT INTO validation_snapshots (id, batch_id, name, status, content_hash, frozen_corners, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		snap.ID, snap.BatchID, snap.Name, snap.Status, snap.ContentHash, snap.FrozenCorners, snap.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

// Get 按 ID 查询快照。
func (s *SnapshotStore) Get(id string) (*model.ValidationSnapshot, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, name, status, content_hash, frozen_corners, created_at, published_at
		 FROM validation_snapshots WHERE id = ?`, id)
	var snap model.ValidationSnapshot
	var pub sql.NullString
	if err := row.Scan(&snap.ID, &snap.BatchID, &snap.Name, &snap.Status, &snap.ContentHash,
		&snap.FrozenCorners, &snap.CreatedAt, &pub); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrSnapshotNotFound
		}
		return nil, err
	}
	snap.PublishedAt = pub.String
	return &snap, nil
}

// List 列出批次全部快照（创建时间倒序）。
func (s *SnapshotStore) List(batchID string) ([]*model.ValidationSnapshot, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, name, status, content_hash, frozen_corners, created_at, published_at
		 FROM validation_snapshots WHERE batch_id = ? ORDER BY created_at DESC`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ValidationSnapshot
	for rows.Next() {
		var snap model.ValidationSnapshot
		var pub sql.NullString
		if err := rows.Scan(&snap.ID, &snap.BatchID, &snap.Name, &snap.Status, &snap.ContentHash,
			&snap.FrozenCorners, &snap.CreatedAt, &pub); err != nil {
			return nil, err
		}
		snap.PublishedAt = pub.String
		out = append(out, &snap)
	}
	return out, rows.Err()
}

// Published 返回批次已发布且未被替代的最新快照。
func (s *SnapshotStore) Published(batchID string) (*model.ValidationSnapshot, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, name, status, content_hash, frozen_corners, created_at, published_at
		 FROM validation_snapshots WHERE batch_id = ? AND status = ? ORDER BY created_at DESC LIMIT 1`,
		batchID, model.SnapshotStatusPublished)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	var snap model.ValidationSnapshot
	var pub sql.NullString
	if err := rows.Scan(&snap.ID, &snap.BatchID, &snap.Name, &snap.Status, &snap.ContentHash,
		&snap.FrozenCorners, &snap.CreatedAt, &pub); err != nil {
		return nil, err
	}
	snap.PublishedAt = pub.String
	return &snap, rows.Err()
}

// Publish 将草稿置为已发布；此前已发布的快照自动标记为替代（superseded）。
func (s *SnapshotStore) Publish(id, at string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var batchID string
	if err := tx.QueryRow(`SELECT batch_id FROM validation_snapshots WHERE id = ?`, id).Scan(&batchID); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE validation_snapshots SET status = ? WHERE batch_id = ? AND status = ? AND id <> ?`,
		model.SnapshotStatusSuperseded, batchID, model.SnapshotStatusPublished, id); err != nil {
		return err
	}
	res, err := tx.Exec(
		`UPDATE validation_snapshots SET status = ?, published_at = ? WHERE id = ? AND status = ?`,
		model.SnapshotStatusPublished, at, id, model.SnapshotStatusDraft)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("snapshot %s is not draft", id)
	}
	return tx.Commit()
}

// AddItem 追加快照摘要条目。
func (s *SnapshotStore) AddItem(item *model.SnapshotItem) error {
	_, err := s.db.Exec(
		`INSERT INTO snapshot_items (id, snapshot_id, kind, ref_id, summary) VALUES (?,?,?,?,?)`,
		item.ID, item.SnapshotID, item.Kind, item.RefID, item.Summary)
	if err != nil {
		return fmt.Errorf("insert snapshot item: %w", err)
	}
	return nil
}

// Items 返回快照的摘要条目。
func (s *SnapshotStore) Items(snapshotID string) ([]*model.SnapshotItem, error) {
	rows, err := s.db.Query(
		`SELECT id, snapshot_id, kind, ref_id, summary FROM snapshot_items WHERE snapshot_id = ? ORDER BY id`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.SnapshotItem
	for rows.Next() {
		var it model.SnapshotItem
		if err := rows.Scan(&it.ID, &it.SnapshotID, &it.Kind, &it.RefID, &it.Summary); err != nil {
			return nil, err
		}
		out = append(out, &it)
	}
	return out, rows.Err()
}
