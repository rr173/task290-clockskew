package service

import (
	"task290-clockskew/internal/store"
)

// Stats 是全库统计快照。
type Stats struct {
	BatchCount        int            `json:"batch_count"`
	EndpointCount     int            `json:"endpoint_count"`
	ModeCount         int            `json:"mode_count"`
	CornerCount       int            `json:"corner_count"`
	MeasurementCount  int            `json:"measurement_count"`
	ExemptionCount    int            `json:"exemption_count"`
	SnapshotCount     int            `json:"snapshot_count"`
	ByBatchStatus     map[string]int `json:"by_batch_status"`
	ByExemptionStatus map[string]int `json:"by_exemption_status"`
}

// StatsService 聚合统计指标（读取各 store 的 List 结果计数，数据量级小、无额外 SQL）。
type StatsService struct {
	bs *store.BatchStore
	rs *store.ReferenceStore
	ms *store.MeasurementStore
	es *store.ExemptionStore
	ss *store.SnapshotStore
}

func NewStatsService(bs *store.BatchStore, rs *store.ReferenceStore,
	ms *store.MeasurementStore, es *store.ExemptionStore, ss *store.SnapshotStore) *StatsService {
	return &StatsService{bs: bs, rs: rs, ms: ms, es: es, ss: ss}
}

// Collect 汇总全库指标。
func (s *StatsService) Collect() (*Stats, error) {
	st := &Stats{
		ByBatchStatus:     map[string]int{},
		ByExemptionStatus: map[string]int{},
	}
	batches, err := s.bs.List()
	if err != nil {
		return nil, err
	}
	st.BatchCount = len(batches)
	for _, b := range batches {
		st.ByBatchStatus[b.Status]++
		eps, err := s.rs.ListEndpoints(b.ID)
		if err != nil {
			return nil, err
		}
		st.EndpointCount += len(eps)
		modes, err := s.rs.ListModes(b.ID)
		if err != nil {
			return nil, err
		}
		st.ModeCount += len(modes)
		corners, err := s.rs.ListCorners(b.ID)
		if err != nil {
			return nil, err
		}
		st.CornerCount += len(corners)
		ms, err := s.ms.List(b.ID)
		if err != nil {
			return nil, err
		}
		st.MeasurementCount += len(ms)
		exs, err := s.es.List(b.ID)
		if err != nil {
			return nil, err
		}
		st.ExemptionCount += len(exs)
		for _, ex := range exs {
			st.ByExemptionStatus[ex.Status]++
		}
		snaps, err := s.ss.List(b.ID)
		if err != nil {
			return nil, err
		}
		st.SnapshotCount += len(snaps)
	}
	return st, nil
}

// BatchSummary 返回单批次的轻量摘要（供列表页展示）。
func (s *StatsService) BatchSummary(batchID string) (map[string]int, error) {
	out := map[string]int{}
	ms, err := s.ms.List(batchID)
	if err != nil {
		return nil, err
	}
	out["measurements"] = len(ms)
	exs, err := s.es.List(batchID)
	if err != nil {
		return nil, err
	}
	out["exemptions"] = len(exs)
	return out, nil
}
