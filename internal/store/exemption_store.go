package store

import (
	"database/sql"
	"fmt"

	"task290-clockskew/internal/model"
)

// ExemptionStore 管理豁免规则、条件与依赖边。
type ExemptionStore struct{ db *sql.DB }

func NewExemptionStore(db *sql.DB) *ExemptionStore { return &ExemptionStore{db: db} }

// Create 创建豁免（状态 candidate），同批次重名拒绝。
func (s *ExemptionStore) Create(e *model.Exemption) error {
	_, err := s.db.Exec(
		`INSERT INTO exemptions (id, batch_id, name, from_endpoint, to_endpoint, max_skew_ps, status, created_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		e.ID, e.BatchID, e.Name, e.FromEndpoint, e.ToEndpoint, e.MaxSkewPS, e.Status, e.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert exemption: %w", err)
	}
	return nil
}

// Get 按 ID 查询豁免。
func (s *ExemptionStore) Get(id string) (*model.Exemption, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, name, from_endpoint, to_endpoint, max_skew_ps, status, created_at FROM exemptions WHERE id = ?`, id)
	var e model.Exemption
	if err := row.Scan(&e.ID, &e.BatchID, &e.Name, &e.FromEndpoint, &e.ToEndpoint, &e.MaxSkewPS, &e.Status, &e.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrExemptionNotFound
		}
		return nil, err
	}
	return &e, nil
}

// List 列出批次下全部豁免（按创建时间升序）。
func (s *ExemptionStore) List(batchID string) ([]*model.Exemption, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, name, from_endpoint, to_endpoint, max_skew_ps, status, created_at
		 FROM exemptions WHERE batch_id = ? ORDER BY created_at`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Exemption
	for rows.Next() {
		var e model.Exemption
		if err := rows.Scan(&e.ID, &e.BatchID, &e.Name, &e.FromEndpoint, &e.ToEndpoint, &e.MaxSkewPS, &e.Status, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

// UpdateStatus 更新豁免状态。
func (s *ExemptionStore) UpdateStatus(id, status string) error {
	_, err := s.db.Exec(`UPDATE exemptions SET status = ? WHERE id = ?`, status, id)
	return err
}

// AddCondition 为豁免添加一个前提条件；同一（豁免, 模式, 工艺角）组合冲突拒绝。
func (s *ExemptionStore) AddCondition(c *model.ExemptionCondition) error {
	_, err := s.db.Exec(
		`INSERT INTO exemption_conditions (id, exemption_id, mode_id, corner_id, max_distance_um) VALUES (?,?,?,?,?)`,
		c.ID, c.ExemptionID, c.ModeID, c.CornerID, c.MaxDistanceUM)
	if err != nil {
		return fmt.Errorf("insert condition: %w", err)
	}
	return nil
}

// Conditions 返回豁免的全部条件。
func (s *ExemptionStore) Conditions(exemptionID string) ([]*model.ExemptionCondition, error) {
	rows, err := s.db.Query(
		`SELECT id, exemption_id, mode_id, corner_id, max_distance_um FROM exemption_conditions WHERE exemption_id = ? ORDER BY id`, exemptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ExemptionCondition
	for rows.Next() {
		var c model.ExemptionCondition
		if err := rows.Scan(&c.ID, &c.ExemptionID, &c.ModeID, &c.CornerID, &c.MaxDistanceUM); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// AddDependency 添加依赖边；依赖环由上层 cycle 检查器负责，这里仅落库。
func (s *ExemptionStore) AddDependency(d *model.ExemptionDependency) error {
	_, err := s.db.Exec(
		`INSERT INTO exemption_dependencies (id, exemption_id, depends_on) VALUES (?,?,?)`,
		d.ID, d.ExemptionID, d.DependsOn)
	if err != nil {
		return fmt.Errorf("insert dependency: %w", err)
	}
	return nil
}

// Dependencies 返回某豁免的全部依赖边。
func (s *ExemptionStore) Dependencies(exemptionID string) ([]*model.ExemptionDependency, error) {
	rows, err := s.db.Query(
		`SELECT id, exemption_id, depends_on FROM exemption_dependencies WHERE exemption_id = ?`, exemptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ExemptionDependency
	for rows.Next() {
		var d model.ExemptionDependency
		if err := rows.Scan(&d.ID, &d.ExemptionID, &d.DependsOn); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

// AllDependencies 返回批次内全部依赖边（供环检测与传播使用）。
func (s *ExemptionStore) AllDependencies(batchID string) ([]*model.ExemptionDependency, error) {
	rows, err := s.db.Query(
		`SELECT d.id, d.exemption_id, d.depends_on
		 FROM exemption_dependencies d JOIN exemptions e ON e.id = d.exemption_id
		 WHERE e.batch_id = ?`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ExemptionDependency
	for rows.Next() {
		var d model.ExemptionDependency
		if err := rows.Scan(&d.ID, &d.ExemptionID, &d.DependsOn); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}
