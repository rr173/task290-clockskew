package store

import "database/sql"

// migrate 执行全部建表 DDL。表结构以 batch 为根，外键级联清理测试数据。
func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS batches (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			default_max_skew_ps INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			opened_at TEXT,
			sealed_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS clock_endpoints (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			clock_name TEXT NOT NULL,
			clock_domain TEXT NOT NULL,
			x REAL NOT NULL,
			y REAL NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS modes (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS corners (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS batch_corners (
			batch_id TEXT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
			corner_id TEXT NOT NULL REFERENCES corners(id) ON DELETE CASCADE,
			PRIMARY KEY (batch_id, corner_id)
		)`,
		`CREATE TABLE IF NOT EXISTS skew_measurements (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
			seq INTEGER NOT NULL,
			endpoint_a TEXT NOT NULL,
			endpoint_b TEXT NOT NULL,
			mode_id TEXT NOT NULL,
			corner_id TEXT NOT NULL,
			skew_ps INTEGER NOT NULL,
			measured_at TEXT NOT NULL,
			checksum TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, seq)
		)`,
		`CREATE TABLE IF NOT EXISTS exemptions (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			from_endpoint TEXT NOT NULL,
			to_endpoint TEXT NOT NULL,
			max_skew_ps INTEGER NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS exemption_conditions (
			id TEXT PRIMARY KEY,
			exemption_id TEXT NOT NULL REFERENCES exemptions(id) ON DELETE CASCADE,
			mode_id TEXT NOT NULL,
			corner_id TEXT NOT NULL,
			max_distance_um REAL NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS exemption_dependencies (
			id TEXT PRIMARY KEY,
			exemption_id TEXT NOT NULL REFERENCES exemptions(id) ON DELETE CASCADE,
			depends_on TEXT NOT NULL REFERENCES exemptions(id) ON DELETE CASCADE,
			UNIQUE(exemption_id, depends_on)
		)`,
		`CREATE TABLE IF NOT EXISTS validation_runs (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
			status TEXT NOT NULL,
			cursor_measurement INTEGER NOT NULL DEFAULT 0,
			cursor_exemption INTEGER NOT NULL DEFAULT 0,
			failure_count INTEGER NOT NULL DEFAULT 0,
			started_at TEXT NOT NULL,
			finished_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS validation_failures (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL REFERENCES validation_runs(id) ON DELETE CASCADE,
			exemption_id TEXT,
			measurement_id TEXT,
			type TEXT NOT NULL,
			message TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS validation_snapshots (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			frozen_corners TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			published_at TEXT,
			UNIQUE(batch_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS snapshot_items (
			id TEXT PRIMARY KEY,
			snapshot_id TEXT NOT NULL REFERENCES validation_snapshots(id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			ref_id TEXT NOT NULL,
			summary TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_measurements_batch ON skew_measurements(batch_id, seq)`,
		`CREATE INDEX IF NOT EXISTS idx_conditions_exemption ON exemption_conditions(exemption_id)`,
		`CREATE INDEX IF NOT EXISTS idx_deps_exemption ON exemption_dependencies(exemption_id)`,
		`CREATE INDEX IF NOT EXISTS idx_failures_run ON validation_failures(run_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_snapshot ON snapshot_items(snapshot_id)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
