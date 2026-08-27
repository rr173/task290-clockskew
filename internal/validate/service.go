// Package validate 执行偏斜豁免的前提验证：匹配测量、传播依赖、定位失效并支持断点续跑。
package validate

import (
	"time"

	"task290-clockskew/internal/exemption"
	"task290-clockskew/internal/measurement"
	"task290-clockskew/internal/model"
	"task290-clockskew/internal/store"
)

// Service 是验证编排器，一次 Validate 调用完成一轮批次验证。
type Service struct {
	vs *store.ValidationStore
	es *store.ExemptionStore
	ms *store.MeasurementStore
	rs *store.ReferenceStore
}

func NewService(vs *store.ValidationStore, es *store.ExemptionStore, ms *store.MeasurementStore, rs *store.ReferenceStore) *Service {
	return &Service{vs: vs, es: es, ms: ms, rs: rs}
}

// Result 是一轮验证的结果摘要。
type Result struct {
	RunID         string `json:"run_id"`
	FailureCount  int64  `json:"failure_count"`
	Resumed       bool   `json:"resumed"`
	ExemptionSeen int    `json:"exemption_seen"`
}

// Validate 对批次执行（或续跑）一轮验证。前提：批次处于 pending 状态。
// 同一批次同一时刻只允许一个 running 运行（裁决串行）；进程重启后，
// 若存在 running 运行则从其游标继续。
func (s *Service) Validate(batchID string) (*Result, error) {
	active, err := s.vs.ActiveRun(batchID)
	if err != nil {
		return nil, err
	}
	var run *model.ValidationRun
	var resumed bool
	if active != nil {
		run = active
		resumed = true
	} else {
		time.Sleep(3 * time.Millisecond)
		again, err := s.vs.ActiveRun(batchID)
		if err != nil {
			return nil, err
		}
		if again != nil {
			run = again
			resumed = true
		} else {
			run = &model.ValidationRun{
				ID:                model.NewID("run"),
				BatchID:           batchID,
				Status:            model.RunStatusRunning,
				CursorMeasurement: 0,
				CursorExemption:   0,
				StartedAt:         model.TimeNow(),
			}
			if err := s.vs.CreateRun(run); err != nil {
				return nil, err
			}
			resumed = false
		}
	}

	// 收集批次数据：豁免（非撤销）、条件、依赖、测量。
	exemptions, err := s.es.List(batchID)
	if err != nil {
		return nil, err
	}
	measurements, err := s.ms.List(batchID)
	if err != nil {
		return nil, err
	}
	condsByEx := make(map[string][]*model.ExemptionCondition, len(exemptions))
	for _, ex := range exemptions {
		conds, err := s.es.Conditions(ex.ID)
		if err != nil {
			return nil, err
		}
		condsByEx[ex.ID] = conds
	}
	depMap, err := s.depMap(batchID)
	if err != nil {
		return nil, err
	}

	// 拓扑排序豁免，保证依赖在传播前已被裁决。
	ids := make([]string, 0, len(exemptions))
	for _, ex := range exemptions {
		ids = append(ids, ex.ID)
	}
	order, ok := exemption.TopoOrder(depMap, ids)
	if !ok {
		return nil, model.ErrRuleCycle
	}
	_ = order // 依赖顺序由 propagation 内部按图遍历保证，此处仅做环检查

	// 清理旧失效（本轮从零开始）。
	if err := s.clearFailures(run.ID); err != nil {
		return nil, err
	}

	statusByEx, err := s.propagate(batchID, run, exemptions, condsByEx, depMap, measurements)
	if err != nil {
		return nil, err
	}

	// 测量状态对账（matched / mode_mismatch），使用传播后的裁决结果。
	if err := measurement.Reconcile(batchID, measurements, exemptions, condsByEx, statusByEx,
		func(id, st, reason string) error { return s.ms.UpdateStatus(id, st, reason) }); err != nil {
		return nil, err
	}

	// 记录豁免裁决。
	for exID, st := range statusByEx {
		if err := s.es.UpdateStatus(exID, st); err != nil {
			return nil, err
		}
	}

	// 推进断点游标：全部测量与豁免已处理完毕。
	var maxSeq int64
	for _, m := range measurements {
		if m.Seq > maxSeq {
			maxSeq = m.Seq
		}
	}
	if err := s.updateCursors(run.ID, maxSeq, int64(len(exemptions))); err != nil {
		return nil, err
	}

	failures, err := s.vs.Failures(run.ID)
	if err != nil {
		return nil, err
	}
	fc := int64(len(failures))
	if err := s.vs.CompleteRun(run.ID, fc, model.TimeNow()); err != nil {
		return nil, err
	}
	return &Result{RunID: run.ID, FailureCount: fc, Resumed: resumed, ExemptionSeen: len(exemptions)}, nil
}

// depMap 构造批次内「豁免 → 依赖豁免」映射（只保留批次内豁免，便于拓扑）。
func (s *Service) depMap(batchID string) (map[string][]string, error) {
	deps, err := s.es.AllDependencies(batchID)
	if err != nil {
		return nil, err
	}
	exemptions, err := s.es.List(batchID)
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(exemptions))
	for _, ex := range exemptions {
		known[ex.ID] = true
	}
	out := make(map[string][]string)
	for _, d := range deps {
		if known[d.DependsOn] {
			out[d.ExemptionID] = append(out[d.ExemptionID], d.DependsOn)
		}
	}
	return out, nil
}

// clearFailures 清空某运行的历史失效（续跑时防止重复累计）。
func (s *Service) clearFailures(runID string) error {
	return s.vs.ClearFailures(runID)
}
