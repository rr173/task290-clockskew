package validate

import (
	"task290-clockskew/internal/model"
)

// exemptionStatus 记录单条豁免的裁决结果。
type exemptionStatus struct {
	ex       *model.Exemption
	status   string
	failures []*model.ValidationFailure
}

// propagate 对批次内全部豁免执行前提验证并记录失效。
//
// 每条豁免的裁决顺序：
//  1. 撤销豁免直接跳过（保持 revoked，不产生失效）；
//  2. 依赖传播：被依赖的豁免若已失效/撤销，本豁免记 dependency_invalid；
//  3. 条件校验：条件的工艺角必须属于批次锁定角集合，否则记 corner_missing；
//  4. 覆盖测量校验：匹配条件（模式+角+端点对）的测量若偏斜超限或物理距离超限，
//     记 skew_exceeded / distance_violation；
//  5. 有任一失效 → invalid，否则 → valid。
//
// 返回 豁免 ID → 最终状态 映射。
func (s *Service) propagate(batchID string, run *model.ValidationRun,
	exemptions []*model.Exemption,
	condsByEx map[string][]*model.ExemptionCondition,
	depMap map[string][]string,
	measurements []*model.SkewMeasurement) (map[string]string, error) {

	locked, err := s.rs.BatchCorners(batchID)
	if err != nil {
		return nil, err
	}
	lockedSet := make(map[string]bool, len(locked))
	for _, c := range locked {
		lockedSet[c] = true
	}

	endpoints := make(map[string]*model.ClockEndpoint)
	for _, ex := range exemptions {
		ea, err1 := s.rs.GetEndpoint(ex.FromEndpoint)
		eb, err2 := s.rs.GetEndpoint(ex.ToEndpoint)
		if err1 != nil || err2 != nil {
			continue // 端点被清理的异常情形：跳过，避免传播阻塞
		}
		endpoints[ex.FromEndpoint] = ea
		endpoints[ex.ToEndpoint] = eb
	}

	// 依赖状态表：先裁决被依赖者。
	dependStatus := make(map[string]string, len(exemptions))
	statusByEx := make(map[string]string, len(exemptions))
	seen := make(map[string]bool)

	var visit func(id string) (string, error)
	visit = func(id string) (string, error) {
		if st, ok := dependStatus[id]; ok {
			return st, nil
		}
		if seen[id] {
			return model.ExemptionStatusInvalid, model.ErrRuleCycle
		}
		seen[id] = true

		ex := findExemption(exemptions, id)
		if ex == nil {
			return "", model.ErrExemptionNotFound
		}
		if ex.Status == model.ExemptionStatusRevoked {
			dependStatus[id] = model.ExemptionStatusRevoked
			statusByEx[id] = model.ExemptionStatusRevoked
			return model.ExemptionStatusRevoked, nil
		}

		// 依赖传播。
		failures := []*model.ValidationFailure{}
		for _, dep := range depMap[id] {
			dst, err := visit(dep)
			if err != nil {
				return "", err
			}
			if dst == model.ExemptionStatusInvalid || dst == model.ExemptionStatusRevoked {
				failures = append(failures, &model.ValidationFailure{
					ID: model.NewID("fail"), RunID: run.ID, ExemptionID: id,
					Type: model.FailureTypeDependencyInvalid,
					Message: "dependency exemption " + dep + " is " + dst,
				})
			}
		}

		// 条件校验 + 覆盖测量校验。
		conds := condsByEx[id]
		for _, cond := range conds {
			if !lockedSet[cond.CornerID] {
				failures = append(failures, &model.ValidationFailure{
					ID: model.NewID("fail"), RunID: run.ID, ExemptionID: id,
					Type: model.FailureTypeCornerMissing,
					Message: "condition corner " + cond.CornerID + " not locked in batch",
				})
				continue
			}
			ea, eb := endpoints[ex.FromEndpoint], endpoints[ex.ToEndpoint]
			distance := ea.DistanceTo(eb)
			for _, m := range measurements {
				if m.Status == model.MeasurementStatusExcluded {
					continue
				}
				if !pairMatches(m, ex) {
					continue
				}
				if m.ModeID != cond.ModeID || m.CornerID != cond.CornerID {
					continue
				}
				if m.SkewPS > ex.MaxSkewPS {
					failures = append(failures, &model.ValidationFailure{
						ID: model.NewID("fail"), RunID: run.ID, ExemptionID: id, MeasurementID: m.ID,
						Type: model.FailureTypeSkewExceeded,
						Message: "skew " + int64s(m.SkewPS) + "ps exceeds exemption limit " + int64s(ex.MaxSkewPS) + "ps under mode/corner",
					})
				}
				if distance > cond.MaxDistanceUM {
					failures = append(failures, &model.ValidationFailure{
						ID: model.NewID("fail"), RunID: run.ID, ExemptionID: id, MeasurementID: m.ID,
						Type: model.FailureTypeDistanceViolation,
						Message: "physical distance exceeds condition limit",
					})
				}
			}
		}

		final := model.ExemptionStatusValid
		if len(failures) > 0 {
			final = model.ExemptionStatusInvalid
		}
		for _, f := range failures {
			if err := s.vs.AddFailure(f); err != nil {
				return "", err
			}
		}
		dependStatus[id] = final
		statusByEx[id] = final
		return final, nil
	}

	for _, ex := range exemptions {
		if _, err := visit(ex.ID); err != nil {
			return nil, err
		}
	}
	return statusByEx, nil
}

// pairMatches 判断测量端点对是否与豁免端点对一致（双向）。
func pairMatches(m *model.SkewMeasurement, ex *model.Exemption) bool {
	return (m.EndpointA == ex.FromEndpoint && m.EndpointB == ex.ToEndpoint) ||
		(m.EndpointA == ex.ToEndpoint && m.EndpointB == ex.FromEndpoint)
}

func findExemption(list []*model.Exemption, id string) *model.Exemption {
	for _, e := range list {
		if e.ID == id {
			return e
		}
	}
	return nil
}

func int64s(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
