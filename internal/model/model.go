// Package model 定义芯片时钟树偏斜豁免依赖验证服务的领域实体、状态机常量与领域错误。
package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// NewID 生成 32 位十六进制随机 ID，用于所有实体的主键。
func NewID(prefix string) string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// 理论上 crypto/rand 不会失败；兜底使用纳秒时间戳。
		return prefix + "-" + time.Now().Format("150405.000000000")
	}
	return prefix + "-" + hex.EncodeToString(buf)
}

// TimeNow 返回统一的 RFC3339 时间字符串，保证 SQLite 中时间可排序。
func TimeNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// 签核批次状态机：receiving → pending → has_failures | publishable → sealed。
const (
	BatchStatusReceiving   = "receiving"
	BatchStatusPending     = "pending"
	BatchStatusFailures    = "has_failures"
	BatchStatusPublishable = "publishable"
	BatchStatusSealed      = "sealed"
)

// 偏斜测量状态机：raw → matched | mode_mismatch | excluded。
const (
	MeasurementStatusRaw      = "raw"
	MeasurementStatusMatched  = "matched"
	MeasurementStatusMismatch = "mode_mismatch"
	MeasurementStatusExcluded = "excluded"
)

// 豁免关系状态机：candidate → valid | invalid → revoked → confirmed。
const (
	ExemptionStatusCandidate = "candidate"
	ExemptionStatusValid     = "valid"
	ExemptionStatusInvalid   = "invalid"
	ExemptionStatusRevoked   = "revoked"
	ExemptionStatusConfirmed = "confirmed"
)

// 验证快照状态机：draft → published → superseded。
const (
	SnapshotStatusDraft      = "draft"
	SnapshotStatusPublished  = "published"
	SnapshotStatusSuperseded = "superseded"
)

// 验证运行状态。
const (
	RunStatusRunning   = "running"
	RunStatusCompleted = "completed"
)

// 验证失效类型：定位一条豁免/测量为何不被信任。
const (
	FailureTypeSkewExceeded      = "skew_exceeded"
	FailureTypeDistanceViolation = "distance_violation"
	FailureTypeCornerMissing     = "corner_missing"
	FailureTypeDependencyInvalid = "dependency_invalid"
	FailureTypeRevokedUsed       = "revoked_used"
)

// 时钟域未知时使用的占位域，导入端点必须显式提供时钟域。
const UnknownClockDomain = "UNKNOWN"
