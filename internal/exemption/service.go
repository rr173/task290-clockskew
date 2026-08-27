// Package exemption 负责豁免规则、前提条件与依赖边的维护。
package exemption

import (
	"fmt"

	"task290-clockskew/internal/model"
	"task290-clockskew/internal/store"
)

// Service 提供豁免的创建、条件添加、依赖添加与状态处置。
type Service struct {
	es *store.ExemptionStore
	rs *store.ReferenceStore
}

func NewService(es *store.ExemptionStore, rs *store.ReferenceStore) *Service {
	return &Service{es: es, rs: rs}
}

// Create 创建一条豁免（状态 candidate）。端点必须存在且不同，max_skew 非负。
func (s *Service) Create(batchID, name, fromEndpoint, toEndpoint string, maxSkewPS int64) (*model.Exemption, error) {
	ea, err := s.rs.GetEndpoint(fromEndpoint)
	if err != nil {
		return nil, err
	}
	eb, err := s.rs.GetEndpoint(toEndpoint)
	if err != nil {
		return nil, err
	}
	if ea.BatchID != batchID || eb.BatchID != batchID {
		return nil, fmt.Errorf("endpoint belongs to another batch")
	}
	if ea.ID == eb.ID {
		return nil, model.ErrSameEndpointPair
	}
	if maxSkewPS < 0 {
		return nil, model.ErrInvalidSkew
	}
	ex := &model.Exemption{
		ID:           model.NewID("ex"),
		BatchID:      batchID,
		Name:         name,
		FromEndpoint: fromEndpoint,
		ToEndpoint:   toEndpoint,
		MaxSkewPS:    maxSkewPS,
		Status:       model.ExemptionStatusCandidate,
		CreatedAt:    model.TimeNow(),
	}
	if err := s.es.Create(ex); err != nil {
		return nil, err
	}
	return ex, nil
}

// AddCondition 为豁免添加前提条件（模式, 工艺角, 物理距离上限）。
// 模式与工艺角必须存在；同一豁免下（模式, 工艺角）组合不得重复。
func (s *Service) AddCondition(exemptionID, modeID, cornerID string, maxDistanceUM float64) (*model.ExemptionCondition, error) {
	if _, err := s.rs.GetMode(modeID); err != nil {
		return nil, err
	}
	if _, err := s.rs.GetCorner(cornerID); err != nil {
		return nil, model.ErrCornerMissing
	}
	ex, err := s.es.Get(exemptionID)
	if err != nil {
		return nil, err
	}
	if ex.Status == model.ExemptionStatusRevoked || ex.Status == model.ExemptionStatusConfirmed {
		return nil, fmt.Errorf("exemption %s is closed for modification", exemptionID)
	}
	existing, err := s.es.Conditions(exemptionID)
	if err != nil {
		return nil, err
	}
	for _, c := range existing {
		if c.ModeID == modeID && c.CornerID == cornerID {
			return nil, model.ErrConditionConflict
		}
	}
	cond := &model.ExemptionCondition{
		ID:            model.NewID("cond"),
		ExemptionID:   exemptionID,
		ModeID:        modeID,
		CornerID:      cornerID,
		MaxDistanceUM: maxDistanceUM,
	}
	if err := s.es.AddCondition(cond); err != nil {
		return nil, err
	}
	return cond, nil
}

// AddDependency 添加豁免依赖边：exemption 依赖 dependsOn 成立才成立。
// 添加前执行环检测，命中环即拒绝（规则循环）。
func (s *Service) AddDependency(exemptionID, dependsOn string) error {
	if exemptionID == dependsOn {
		return model.ErrRuleCycle
	}
	if _, err := s.es.Get(exemptionID); err != nil {
		return err
	}
	if _, err := s.es.Get(dependsOn); err != nil {
		return model.ErrExemptionNotFound
	}
	if HasCycle(exemptionID, dependsOn, s.es) {
		return model.ErrRuleCycle
	}
	d := &model.ExemptionDependency{ID: model.NewID("dep"), ExemptionID: exemptionID, DependsOn: dependsOn}
	if err := s.es.AddDependency(d); err != nil {
		return err
	}
	return nil
}

// Get 按 ID 查询豁免（含其条件与依赖由调用方按需加载）。
func (s *Service) Get(id string) (*model.Exemption, error) {
	return s.es.Get(id)
}

// List 列出批次全部豁免。
func (s *Service) List(batchID string) ([]*model.Exemption, error) {
	return s.es.List(batchID)
}

// Revoke 撤销豁免：状态 candidate/valid/invalid → revoked。
// 撤销后豁免不再参与后续验证，但历史失效记录保留。
func (s *Service) Revoke(id string) error {
	ex, err := s.es.Get(id)
	if err != nil {
		return err
	}
	if ex.Status == model.ExemptionStatusRevoked {
		return model.ErrRevokedExemption
	}
	return s.es.UpdateStatus(id, model.ExemptionStatusRevoked)
}

// Confirm 人工确认豁免（仅对 valid 的豁免开放）。
func (s *Service) Confirm(id string) error {
	ex, err := s.es.Get(id)
	if err != nil {
		return err
	}
	if ex.Status != model.ExemptionStatusValid {
		return fmt.Errorf("only valid exemption can be confirmed, current %s", ex.Status)
	}
	return s.es.UpdateStatus(id, model.ExemptionStatusConfirmed)
}
