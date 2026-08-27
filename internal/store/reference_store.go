package store

import (
	"database/sql"
	"fmt"

	"task290-clockskew/internal/model"
)

// ReferenceStore 管理端点、模式、工艺角等基准数据，以及批次锁定的工艺角集合。
type ReferenceStore struct{ db *sql.DB }

func NewReferenceStore(db *sql.DB) *ReferenceStore { return &ReferenceStore{db: db} }

// CreateEndpoint 注册时钟端点；同批次重名拒绝。
func (s *ReferenceStore) CreateEndpoint(e *model.ClockEndpoint) error {
	_, err := s.db.Exec(
		`INSERT INTO clock_endpoints (id, batch_id, name, clock_name, clock_domain, x, y, created_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		e.ID, e.BatchID, e.Name, e.ClockName, e.ClockDomain, e.X, e.Y, e.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert endpoint: %w", err)
	}
	return nil
}

// GetEndpoint 按 ID 查询端点。
func (s *ReferenceStore) GetEndpoint(id string) (*model.ClockEndpoint, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, name, clock_name, clock_domain, x, y, created_at FROM clock_endpoints WHERE id = ?`, id)
	var e model.ClockEndpoint
	if err := row.Scan(&e.ID, &e.BatchID, &e.Name, &e.ClockName, &e.ClockDomain, &e.X, &e.Y, &e.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrEndpointNotFound
		}
		return nil, err
	}
	return &e, nil
}

// ListEndpoints 列出批次下全部端点。
func (s *ReferenceStore) ListEndpoints(batchID string) ([]*model.ClockEndpoint, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, name, clock_name, clock_domain, x, y, created_at FROM clock_endpoints WHERE batch_id = ? ORDER BY name`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ClockEndpoint
	for rows.Next() {
		var e model.ClockEndpoint
		if err := rows.Scan(&e.ID, &e.BatchID, &e.Name, &e.ClockName, &e.ClockDomain, &e.X, &e.Y, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

// CreateMode 注册工作模式。
func (s *ReferenceStore) CreateMode(m *model.Mode) error {
	_, err := s.db.Exec(
		`INSERT INTO modes (id, batch_id, name, description, created_at) VALUES (?,?,?,?,?)`,
		m.ID, m.BatchID, m.Name, m.Description, m.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert mode: %w", err)
	}
	return nil
}

// GetMode 按 ID 查询模式。
func (s *ReferenceStore) GetMode(id string) (*model.Mode, error) {
	row := s.db.QueryRow(`SELECT id, batch_id, name, description, created_at FROM modes WHERE id = ?`, id)
	var m model.Mode
	if err := row.Scan(&m.ID, &m.BatchID, &m.Name, &m.Description, &m.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrModeNotFound
		}
		return nil, err
	}
	return &m, nil
}

// ListModes 列出批次下全部模式。
func (s *ReferenceStore) ListModes(batchID string) ([]*model.Mode, error) {
	rows, err := s.db.Query(`SELECT id, batch_id, name, description, created_at FROM modes WHERE batch_id = ? ORDER BY name`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Mode
	for rows.Next() {
		var m model.Mode
		if err := rows.Scan(&m.ID, &m.BatchID, &m.Name, &m.Description, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// CreateCorner 注册工艺角。
func (s *ReferenceStore) CreateCorner(c *model.Corner) error {
	_, err := s.db.Exec(
		`INSERT INTO corners (id, batch_id, name, description, created_at) VALUES (?,?,?,?,?)`,
		c.ID, c.BatchID, c.Name, c.Description, c.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert corner: %w", err)
	}
	return nil
}

// GetCorner 按 ID 查询工艺角。
func (s *ReferenceStore) GetCorner(id string) (*model.Corner, error) {
	row := s.db.QueryRow(`SELECT id, batch_id, name, description, created_at FROM corners WHERE id = ?`, id)
	var c model.Corner
	if err := row.Scan(&c.ID, &c.BatchID, &c.Name, &c.Description, &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrCornerNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ListCorners 列出批次下全部工艺角。
func (s *ReferenceStore) ListCorners(batchID string) ([]*model.Corner, error) {
	rows, err := s.db.Query(`SELECT id, batch_id, name, description, created_at FROM corners WHERE batch_id = ? ORDER BY name`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Corner
	for rows.Next() {
		var c model.Corner
		if err := rows.Scan(&c.ID, &c.BatchID, &c.Name, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// LockCorners 将一组工艺角绑定到批次（快照发布时固定角集合）。
func (s *ReferenceStore) LockCorners(batchID string, cornerIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, cid := range cornerIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO batch_corners (batch_id, corner_id) VALUES (?,?)`, batchID, cid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// BatchCorners 返回批次锁定的工艺角 ID 列表。
func (s *ReferenceStore) BatchCorners(batchID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT corner_id FROM batch_corners WHERE batch_id = ? ORDER BY corner_id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
