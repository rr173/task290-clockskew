package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"task290-clockskew/internal/model"
)

// ContentHash 计算验证快照的内容指纹：批次 + 豁免裁决 + 失效 + 锁定工艺角。
// 哈希固定后快照即不可变——任何被引用的数据变化都会导致哈希不匹配。
func ContentHash(batchID string, exemptions []*model.Exemption,
	failures []*model.ValidationFailure, corners []string) string {
	type digest struct {
		Batch     string   `json:"batch"`
		Exemptions []string `json:"exemptions"`
		Failures  []string `json:"failures"`
		Corners   []string `json:"corners"`
	}
	d := digest{
		Batch:     batchID,
		Exemptions: make([]string, 0, len(exemptions)),
		Failures:  make([]string, 0, len(failures)),
		Corners:   corners,
	}
	for _, ex := range exemptions {
		d.Exemptions = append(d.Exemptions, ex.ID+"|"+ex.Status)
	}
	for _, f := range failures {
		d.Failures = append(d.Failures, f.Type+"|"+f.Message)
	}
	raw, _ := json.Marshal(d)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
