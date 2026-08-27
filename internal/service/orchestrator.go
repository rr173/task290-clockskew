// Package service 编排批次生命周期：创建 → 开启 → 验证 → 发布快照 → 封存。
package service

import (
	"fmt"

	"task290-clockskew/internal/exemption"
	"task290-clockskew/internal/model"
	"task290-clockskew/internal/snapshot"
	"task290-clockskew/internal/store"
	"task290-clockskew/internal/validate"
)

// Orchestrator 是批次级门禁与状态机编排器。
type Orchestrator struct {
	bs          *store.BatchStore
	ms          *store.MeasurementStore
	vs          *store.ValidationStore
	validate    *validate.Service
	snapshot    *snapshot.Service
	exemption   *exemption.Service
}

func NewOrchestrator(bs *store.BatchStore, ms *store.MeasurementStore, vs *store.ValidationStore,
	validateSvc *validate.Service, snapSvc *snapshot.Service, exSvc *exemption.Service) *Orchestrator {
	return &Orchestrator{bs: bs, ms: ms, vs: vs, validate: validateSvc, snapshot: snapSvc, exemption: exSvc}
}

// CreateBatch 创建签核批次（receiving 状态）。
func (o *Orchestrator) CreateBatch(name string, defaultMaxSkewPS int64) (*model.Batch, error) {
	if name == "" {
		return nil, fmt.Errorf("batch name required")
	}
	if defaultMaxSkewPS < 0 {
		return nil, model.ErrMissingDefaultSkew
	}
	b := &model.Batch{
		ID: model.NewID("batch"), Name: name, Status: model.BatchStatusReceiving,
		DefaultMaxSkewPS: defaultMaxSkewPS, CreatedAt: model.TimeNow(),
	}
	if err := o.bs.Create(b); err != nil {
		return nil, err
	}
	return b, nil
}

// OpenBatch 将批次从 receiving 推进到 pending（开始可验证）。
func (o *Orchestrator) OpenBatch(id string) (*model.Batch, error) {
	b, err := o.bs.Get(id)
	if err != nil {
		return nil, err
	}
	if !b.CanTransition(model.BatchStatusPending) {
		return nil, fmt.Errorf("batch %s cannot open from %s", id, b.Status)
	}
	if err := o.bs.UpdateStatus(id, b.Status, model.BatchStatusPending); err != nil {
		return nil, err
	}
	return o.bs.Get(id)
}

// ValidateBatch 执行一轮验证并按结果流转批次状态：
// failures > 0 → has_failures；否则 → publishable。
func (o *Orchestrator) ValidateBatch(id string) (*validate.Result, error) {
	b, err := o.bs.Get(id)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchStatusPending {
		return nil, model.ErrBatchNotOpen
	}
	res, err := o.validate.Validate(id)
	if err != nil {
		return nil, err
	}
	next := model.BatchStatusPublishable
	if res.FailureCount > 0 {
		next = model.BatchStatusFailures
	}
	if err := o.bs.UpdateStatus(id, model.BatchStatusPending, next); err != nil {
		return nil, err
	}
	return res, nil
}

// RevokeExemption 撤销豁免并让批次回到待验证（若此前存在失效）。
func (o *Orchestrator) RevokeExemption(exemptionID string) (*model.Exemption, error) {
	ex, err := o.exemptionService().Get(exemptionID)
	if err != nil {
		return nil, err
	}
	if err := o.exemption.Revoke(exemptionID); err != nil {
		return nil, err
	}
	// 若批次因该豁免失效而停在 has_failures，撤销后回到 pending 并重置测量派生状态供复验。
	b, err := o.bs.Get(ex.BatchID)
	if err == nil && b.Status == model.BatchStatusFailures && !b.Sealed() {
		if err := o.bs.UpdateStatus(ex.BatchID, model.BatchStatusFailures, model.BatchStatusPending); err != nil {
			return nil, err
		}
	}
	// 返回撤销后的最新状态。
	return o.exemptionService().Get(exemptionID)
}

// SealBatch 封存批次：要求 publishable 且存在已发布快照；封存后所有写操作被拒绝。
func (o *Orchestrator) SealBatch(id string) (*model.Batch, error) {
	b, err := o.bs.Get(id)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchStatusPublishable {
		return nil, fmt.Errorf("batch %s is not publishable (current %s)", id, b.Status)
	}
	pub, err := o.snapshot.Published(id)
	if err != nil {
		return nil, err
	}
	if pub == nil {
		return nil, fmt.Errorf("batch %s has no published snapshot to seal", id)
	}
	if err := o.bs.MarkSealed(id, model.TimeNow()); err != nil {
		return nil, err
	}
	return o.bs.Get(id)
}

// GetBatch 查询批次。
func (o *Orchestrator) GetBatch(id string) (*model.Batch, error) { return o.bs.Get(id) }

// ListBatches 列出全部批次。
func (o *Orchestrator) ListBatches() ([]*model.Batch, error) { return o.bs.List() }

// exemptionService 返回底层豁免服务（供撤销前查询）。
func (o *Orchestrator) exemptionService() *exemption.Service { return o.exemption }
