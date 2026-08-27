// Package measurement 负责偏斜测量的导入（幂等）、匹配与排除。
package measurement

import (
	"fmt"

	"task290-clockskew/internal/model"
	"task290-clockskew/internal/store"
)

// Input 是单条测量导入请求。
type Input struct {
	EndpointA  string `json:"endpoint_a"`
	EndpointB  string `json:"endpoint_b"`
	ModeID     string `json:"mode_id"`
	CornerID   string `json:"corner_id"`
	SkewPS     int64  `json:"skew_ps"`
	MeasuredAt string `json:"measured_at"`
}

// ImportResult 是批量导入的结果统计（幂等导入不中断）。
type ImportResult struct {
	Inserted int64 `json:"inserted"`
	Duplicates int64 `json:"duplicates"`
	Rejected int64 `json:"rejected"`
}

// Service 提供测量导入与状态管理。
type Service struct {
	ms *store.MeasurementStore
	rs *store.ReferenceStore
}

func NewService(ms *store.MeasurementStore, rs *store.ReferenceStore) *Service {
	return &Service{ms: ms, rs: rs}
}

// Import 批量导入测量。校验规则：
//   - 端点必须存在且时钟域已知（端点时钟域未知拒绝）；
//   - 端点对不得相同；
//   - 模式/工艺角必须存在（工艺角缺失拒绝）；
//   - skew 非负；
//   - 校验和唯一：重复导入同一测量（同一端点对/模式/角/值/时间）幂等跳过。
func (s *Service) Import(batchID string, inputs []Input) (*ImportResult, error) {
	res := &ImportResult{}
	for _, in := range inputs {
		if err := s.validate(batchID, &in); err != nil {
			res.Rejected++
			continue
		}
		cs := Checksum(batchID, in.EndpointA, in.EndpointB, in.ModeID, in.CornerID, in.SkewPS, in.MeasuredAt)
		dup, err := s.ms.ExistsChecksum(cs)
		if err != nil {
			return nil, err
		}
		if dup {
			res.Duplicates++
			continue
		}
		m := &model.SkewMeasurement{
			ID:         model.NewID("msr"),
			BatchID:    batchID,
			EndpointA:  in.EndpointA,
			EndpointB:  in.EndpointB,
			ModeID:     in.ModeID,
			CornerID:   in.CornerID,
			SkewPS:     in.SkewPS,
			MeasuredAt: in.MeasuredAt,
			Checksum:   cs,
			Status:     model.MeasurementStatusRaw,
			CreatedAt:  model.TimeNow(),
		}
		seq, err := s.ms.NextSeq(batchID)
		if err != nil {
			return nil, err
		}
		m.Seq = seq
		if _, err := s.ms.Insert(m); err != nil {
			return nil, err
		}
		res.Inserted++
	}
	return res, nil
}

// validate 校验单条测量的领域约束。
func (s *Service) validate(batchID string, in *Input) error {
	ea, err := s.rs.GetEndpoint(in.EndpointA)
	if err != nil {
		return err
	}
	eb, err := s.rs.GetEndpoint(in.EndpointB)
	if err != nil {
		return err
	}
	if ea.BatchID != batchID || eb.BatchID != batchID {
		return fmt.Errorf("endpoint belongs to another batch")
	}
	if ea.ID == eb.ID {
		return model.ErrSameEndpointPair
	}
	if ea.ClockDomain == model.UnknownClockDomain || ea.ClockDomain == "" ||
		eb.ClockDomain == model.UnknownClockDomain || eb.ClockDomain == "" {
		return model.ErrEndpointDomainKnown
	}
	if _, err := s.rs.GetMode(in.ModeID); err != nil {
		return err
	}
	if _, err := s.rs.GetCorner(in.CornerID); err != nil {
		return model.ErrCornerMissing
	}
	if in.SkewPS < 0 {
		return model.ErrInvalidSkew
	}
	return nil
}

// Exclude 将测量标记为排除（如已知噪声/错误标注），排除后不再参与豁免匹配。
func (s *Service) Exclude(id, reason string) error {
	if err := s.ms.UpdateStatus(id, model.MeasurementStatusExcluded, reason); err != nil {
		return err
	}
	return nil
}

// List 列出批次全部测量。
func (s *Service) List(batchID string) ([]*model.SkewMeasurement, error) {
	return s.ms.List(batchID)
}
