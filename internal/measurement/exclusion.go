package measurement

import (
	"fmt"

	"task290-clockskew/internal/model"
)

// Reason 是测量状态标记的原因常量，写入 SQLite 便于追溯。
const (
	ReasonExcludedByEngineer = "excluded by engineer"
	ReasonModeMismatch       = "mode not covered by exemption"
	ReasonMatched            = "covered by exemption"
)

// Reconcile 在验证后重置并重算测量状态：
//   - 被排除的测量保持 excluded；
//   - 落入某豁免覆盖范围的测量标 matched；
//   - 端点对属于某豁免、但模式未被任何条件声明的测量标 mode_mismatch；
//   - 其余测量保持 raw。
//
// 该函数以整批方式写回，保证验证与测量状态在同一轮内一致。
func Reconcile(batchID string, ms []*model.SkewMeasurement, exemptions []*model.Exemption,
	condsByExemption map[string][]*model.ExemptionCondition,
	updater func(id, status, reason string) error) error {
	for _, m := range ms {
		if m.Status == model.MeasurementStatusExcluded {
			continue
		}
		covered := false
		for _, ex := range exemptions {
			if ex.Status == model.ExemptionStatusRevoked || ex.Status == model.ExemptionStatusInvalid {
				continue
			}
			conds := condsByExemption[ex.ID]
			if Match(m, ex, conds) != nil {
				covered = true
				break
			}
		}
		switch {
		case covered:
			if err := updater(m.ID, model.MeasurementStatusMatched, ReasonMatched); err != nil {
				return fmt.Errorf("mark matched %s: %w", m.ID, err)
			}
		case hasExemptionPair(m, exemptions):
			if err := updater(m.ID, model.MeasurementStatusMismatch, ReasonModeMismatch); err != nil {
				return fmt.Errorf("mark mismatch %s: %w", m.ID, err)
			}
		}
	}
	return nil
}

// hasExemptionPair 判断是否存在（非撤销）豁免覆盖该测量的端点对。
func hasExemptionPair(m *model.SkewMeasurement, exemptions []*model.Exemption) bool {
	for _, ex := range exemptions {
		if ex.Status == model.ExemptionStatusRevoked || ex.Status == model.ExemptionStatusInvalid {
			continue
		}
		if (m.EndpointA == ex.FromEndpoint && m.EndpointB == ex.ToEndpoint) ||
			(m.EndpointA == ex.ToEndpoint && m.EndpointB == ex.FromEndpoint) {
			return true
		}
	}
	return false
}
