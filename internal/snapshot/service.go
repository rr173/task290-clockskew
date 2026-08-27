// Package snapshot 负责验证快照的创建（草稿）、发布与不可变封存。
package snapshot

import (
	"fmt"

	"task290-clockskew/internal/model"
	"task290-clockskew/internal/store"
)

// Service 管理验证快照生命周期：draft → published → superseded。
type Service struct {
	ss *store.SnapshotStore
	vs *store.ValidationStore
	es *store.ExemptionStore
	rs *store.ReferenceStore
}

func NewService(ss *store.SnapshotStore, vs *store.ValidationStore, es *store.ExemptionStore, rs *store.ReferenceStore) *Service {
	return &Service{ss: ss, vs: vs, es: es, rs: rs}
}

// CreateDraft 从批次当前验证结论生成草稿快照：
//   - 计算内容哈希（豁免裁决 + 失效 + 测量状态 + 锁定工艺角）；
//   - 将每条豁免/失效/测量摘要写入快照条目；
//   - 批次必须已有完成的验证运行，否则拒绝（无结论可固化）。
func (s *Service) CreateDraft(batchID, name string) (*model.ValidationSnapshot, error) {
	failures, err := s.vs.LastFailures(batchID)
	if err != nil {
		return nil, err
	}
	exemptions, err := s.es.List(batchID)
	if err != nil {
		return nil, err
	}
	corners, err := s.rs.BatchCorners(batchID)
	if err != nil {
		return nil, err
	}
	if len(exemptions) == 0 && len(failures) == 0 {
		return nil, fmt.Errorf("batch has no validation conclusion to snapshot")
	}

	hash := ContentHash(batchID, exemptions, failures, corners)
	snap := &model.ValidationSnapshot{
		ID:            model.NewID("snap"),
		BatchID:       batchID,
		Name:          name,
		Status:        model.SnapshotStatusDraft,
		ContentHash:   hash,
		FrozenCorners: join(corners),
		CreatedAt:     model.TimeNow(),
	}
	var items []*model.SnapshotItem
	for _, ex := range exemptions {
		items = append(items, &model.SnapshotItem{
			ID: model.NewID("item"), SnapshotID: snap.ID, Kind: "exemption",
			RefID: ex.ID, Summary: ex.Name + ":" + ex.Status,
		})
	}
	for _, f := range failures {
		items = append(items, &model.SnapshotItem{
			ID: model.NewID("item"), SnapshotID: snap.ID, Kind: "failure",
			RefID: f.ID, Summary: f.Type + ":" + f.Message,
		})
	}
	if err := s.ss.CreateWithItems(snap, items); err != nil {
		return nil, err
	}
	return snap, nil
}

// Publish 发布草稿快照。发布后：
//   - 原已发布快照自动置为 superseded（替代语义）；
//   - 内容哈希不可变（frozen）。
func (s *Service) Publish(id string) (*model.ValidationSnapshot, error) {
	snap, err := s.ss.Get(id)
	if err != nil {
		return nil, err
	}
	if snap.Status != model.SnapshotStatusDraft {
		return nil, model.ErrSnapshotFrozen
	}
	if err := s.ss.Publish(id, model.TimeNow()); err != nil {
		return nil, err
	}
	return s.ss.Get(id)
}

// List 列出批次全部快照。
func (s *Service) List(batchID string) ([]*model.ValidationSnapshot, error) {
	return s.ss.List(batchID)
}

// Get 查询快照及其条目。
func (s *Service) Get(id string) (*model.ValidationSnapshot, []*model.SnapshotItem, error) {
	snap, err := s.ss.Get(id)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.ss.Items(id)
	if err != nil {
		return nil, nil, err
	}
	return snap, items, nil
}

// Published 返回批次最新已发布快照。
func (s *Service) Published(batchID string) (*model.ValidationSnapshot, error) {
	return s.ss.Published(batchID)
}

func join(ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += id
	}
	return out
}
