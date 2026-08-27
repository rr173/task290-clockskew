package model

// ValidationRun 是一次验证运行：对一个批次的所有豁免执行前提验证。
// 支持断点续跑：cursorMeasurement / cursorExemption 记录已处理进度，
// 进程重启后可恢复未完成的运行。
type ValidationRun struct {
	ID               string `json:"id"`
	BatchID          string `json:"batch_id"`
	Status           string `json:"status"` // running / completed
	CursorMeasurement int64 `json:"cursor_measurement"` // 已处理的测量最大 seq
	CursorExemption  int64  `json:"cursor_exemption"`   // 已处理的豁免序号（创建顺序）
	FailureCount     int64  `json:"failure_count"`
	StartedAt        string `json:"started_at"`
	FinishedAt       string `json:"finished_at,omitempty"`
}

// ValidationFailure 是一条验证失效：精确指向豁免/测量/失效类型与原因。
type ValidationFailure struct {
	ID           string `json:"id"`
	RunID        string `json:"run_id"`
	ExemptionID  string `json:"exemption_id,omitempty"`
	MeasurementID string `json:"measurement_id,omitempty"`
	Type         string `json:"type"`
	Message      string `json:"message"`
}

// ValidationSnapshot 是验证快照：冻结一批验证结论（含失效豁免清单与封存工艺角），
// 内容哈希固定后不可修改；发布后原快照不可变，新快照替代旧快照。
type ValidationSnapshot struct {
	ID          string `json:"id"`
	BatchID     string `json:"batch_id"`
	Name        string `json:"name"`
	Status      string `json:"status"` // draft / published / superseded
	ContentHash string `json:"content_hash"`
	FrozenCorners string `json:"frozen_corners"` // 封存的工艺角列表（逗号分隔）
	CreatedAt   string `json:"created_at"`
	PublishedAt string `json:"published_at,omitempty"`
}

// SnapshotItem 是快照内的摘要条目：记录某豁免的最终裁决或某测量的匹配结果。
type SnapshotItem struct {
	ID         string `json:"id"`
	SnapshotID string `json:"snapshot_id"`
	Kind       string `json:"kind"` // exemption / measurement
	RefID      string `json:"ref_id"`
	Summary    string `json:"summary"`
}
