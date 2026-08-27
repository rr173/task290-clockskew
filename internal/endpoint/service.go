// Package endpoint 负责时钟端点、工作模式与工艺角的登记校验。
package endpoint

import (
	"fmt"

	"task290-clockskew/internal/model"
	"task290-clockskew/internal/store"
)

// Service 提供基准数据的注册入口，统一执行领域校验。
type Service struct {
	ref *store.ReferenceStore
}

func NewService(ref *store.ReferenceStore) *Service { return &Service{ref: ref} }

// RegisterEndpoint 注册一个时钟端点。时钟域必须显式提供（拒绝 UNKNOWN 与空值），
// 名称在同批次内唯一。
func (s *Service) RegisterEndpoint(batchID, name, clockName, clockDomain string, x, y float64) (*model.ClockEndpoint, error) {
	domain := NormalizeDomain(clockDomain)
	if domain == model.UnknownClockDomain || domain == "" {
		return nil, fmt.Errorf("%w: endpoint %s", model.ErrEndpointDomainKnown, name)
	}
	e := &model.ClockEndpoint{
		ID:          model.NewID("ep"),
		BatchID:     batchID,
		Name:        name,
		ClockName:   clockName,
		ClockDomain: domain,
		X:           x,
		Y:           y,
		CreatedAt:   model.TimeNow(),
	}
	if err := s.ref.CreateEndpoint(e); err != nil {
		return nil, err
	}
	return e, nil
}

// RegisterMode 注册工作模式。
func (s *Service) RegisterMode(batchID, name, description string) (*model.Mode, error) {
	m := &model.Mode{ID: model.NewID("mode"), BatchID: batchID, Name: name, Description: description, CreatedAt: model.TimeNow()}
	if err := s.ref.CreateMode(m); err != nil {
		return nil, err
	}
	return m, nil
}

// RegisterCorner 注册工艺角。
func (s *Service) RegisterCorner(batchID, name, description string) (*model.Corner, error) {
	c := &model.Corner{ID: model.NewID("corner"), BatchID: batchID, Name: name, Description: description, CreatedAt: model.TimeNow()}
	if err := s.ref.CreateCorner(c); err != nil {
		return nil, err
	}
	return c, nil
}

// LockBatchCorners 将一组工艺角绑定到批次（验证与快照发布时的角集合）。
func (s *Service) LockBatchCorners(batchID string, cornerIDs []string) error {
	return s.ref.LockCorners(batchID, cornerIDs)
}

// ListEndpoints 列出批次全部端点。
func (s *Service) ListEndpoints(batchID string) ([]*model.ClockEndpoint, error) {
	return s.ref.ListEndpoints(batchID)
}

// ListModes 列出批次全部模式。
func (s *Service) ListModes(batchID string) ([]*model.Mode, error) {
	return s.ref.ListModes(batchID)
}

// ListCorners 列出批次全部工艺角。
func (s *Service) ListCorners(batchID string) ([]*model.Corner, error) {
	return s.ref.ListCorners(batchID)
}

// BatchCorners 返回批次锁定的工艺角。
func (s *Service) BatchCorners(batchID string) ([]string, error) {
	return s.ref.BatchCorners(batchID)
}
