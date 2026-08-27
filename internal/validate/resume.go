package validate

// Resume 返回批次当前是否有一个可续跑的 running 运行。
func (s *Service) Resume(batchID string) (bool, error) {
	run, err := s.vs.ActiveRun(batchID)
	if err != nil {
		return false, err
	}
	return run != nil, nil
}

// Progress 返回批次最近一次运行的处理进度（供展示与续跑判断）。
type Progress struct {
	Running           bool   `json:"running"`
	RunID             string `json:"run_id,omitempty"`
	CursorMeasurement int64  `json:"cursor_measurement"`
	CursorExemption   int64  `json:"cursor_exemption"`
}

// ProgressOf 查询批次当前的验证进度。
func (s *Service) ProgressOf(batchID string) (*Progress, error) {
	run, err := s.vs.ActiveRun(batchID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return &Progress{Running: false}, nil
	}
	return &Progress{
		Running:           true,
		RunID:             run.ID,
		CursorMeasurement: run.CursorMeasurement,
		CursorExemption:   run.CursorExemption,
	}, nil
}

// updateCursors 在验证过程中推进断点游标。measurementMaxSeq 为已处理的测量最大序号，
// exemptionCount 为已处理的豁免数。进度落库后，进程即使崩溃也能从该处恢复。
func (s *Service) updateCursors(runID string, measurementMaxSeq int64, exemptionCount int64) error {
	return s.vs.UpdateCursor(runID, measurementMaxSeq, exemptionCount)
}
