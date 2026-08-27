package exemption

import (
	"task290-clockskew/internal/model"
)

// MatchesMeasurement 判断一条测量是否违反某豁免（在条件匹配前提下）：
//   - 偏斜超过豁免上限 → skew_exceeded；
//   - 物理距离超过条件上限 → distance_violation。
//
// 返回 (失败类型, 失败消息, 是否命中)。未命中返回 ("", "", false)。
func MatchesMeasurement(ex *model.Exemption, cond *model.ExemptionCondition, m *model.SkewMeasurement, distance float64) (string, string, bool) {
	if m.SkewPS > ex.MaxSkewPS {
		return model.FailureTypeSkewExceeded,
			"skew " + itoa(m.SkewPS) + "ps exceeds exemption limit " + itoa(ex.MaxSkewPS) + "ps", true
	}
	if distance > cond.MaxDistanceUM {
		return model.FailureTypeDistanceViolation,
			"physical distance " + ftoa(distance) + "um exceeds condition limit " + ftoa(cond.MaxDistanceUM) + "um", true
	}
	return "", "", false
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
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

func ftoa(v float64) string {
	// 保留两位小数的简易格式化，避免引入 strconv 依赖之外的差异。
	neg := ""
	if v < 0 {
		neg = "-"
		v = -v
	}
	whole := int64(v)
	frac := int64((v - float64(whole)) * 100)
	if frac >= 100 {
		whole++
		frac = 0
	}
	return neg + itoa(whole) + "." + itoa(frac)
}
