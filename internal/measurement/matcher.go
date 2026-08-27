package measurement

import (
	"task290-clockskew/internal/model"
)

// Match 判断一条测量是否落入某豁免的覆盖范围：
//   - 端点对与豁免的 from→to 匹配（支持 A→B 与 B→A 双向视为同一路径）；
//   - 测量的模式在豁免任一条件声明的模式中；
//   - 测量的工艺角在豁免任一条件的工艺角中。
//
// 返回命中的条件（nil 表示不覆盖该测量）。
func Match(m *model.SkewMeasurement, ex *model.Exemption, conds []*model.ExemptionCondition) *model.ExemptionCondition {
	pairOK := (m.EndpointA == ex.FromEndpoint && m.EndpointB == ex.ToEndpoint) ||
		(m.EndpointA == ex.ToEndpoint && m.EndpointB == ex.FromEndpoint)
	if !pairOK {
		return nil
	}
	for _, c := range conds {
		if c.ModeID == m.ModeID && c.CornerID == m.CornerID {
			return c
		}
	}
	return nil
}

// CoversMode 判断豁免是否声明了某模式（用于把「豁免外模式测量」标记为 mode_mismatch）。
func CoversMode(conds []*model.ExemptionCondition, modeID string) bool {
	for _, c := range conds {
		if c.ModeID == modeID {
			return true
		}
	}
	return false
}

// CoversCorner 判断豁免是否声明了某工艺角。
func CoversCorner(conds []*model.ExemptionCondition, cornerID string) bool {
	for _, c := range conds {
		if c.CornerID == cornerID {
			return true
		}
	}
	return false
}
