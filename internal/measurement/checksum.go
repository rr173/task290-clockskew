package measurement

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// Checksum 计算测量摘要：batch + 端点对 + 模式 + 工艺角 + 偏斜值 + 测量时间。
// 相同的测量内容得到相同摘要，用于幂等导入与重启恢复的去重。
func Checksum(batchID, endpointA, endpointB, modeID, cornerID string, skewPS int64, measuredAt string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%s|%s|%s",
		batchID, endpointA, endpointB, modeID, cornerID, strconv.FormatInt(skewPS, 10), measuredAt)
	return hex.EncodeToString(h.Sum(nil))
}
