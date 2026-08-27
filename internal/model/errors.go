package model

import "fmt"

// 领域错误：所有业务校验失败统一返回可识别的错误类型，HTTP 层据此映射状态码。
var (
	ErrBatchNotFound       = fmt.Errorf("batch not found")
	ErrEndpointNotFound    = fmt.Errorf("endpoint not found")
	ErrModeNotFound        = fmt.Errorf("mode not found")
	ErrCornerNotFound      = fmt.Errorf("corner not found")
	ErrExemptionNotFound   = fmt.Errorf("exemption not found")
	ErrRunNotFound         = fmt.Errorf("validation run not found")
	ErrSnapshotNotFound    = fmt.Errorf("snapshot not found")
	ErrEndpointDomainKnown = fmt.Errorf("endpoint clock domain unknown")

	ErrCornerMissing       = fmt.Errorf("corner missing")
	ErrRuleCycle           = fmt.Errorf("exemption dependency cycle detected")
	ErrSealedBatchModify   = fmt.Errorf("batch is sealed, modification rejected")
	ErrSnapshotFrozen      = fmt.Errorf("snapshot is frozen, modification rejected")
	ErrBatchNotOpen        = fmt.Errorf("batch not in pending state")
	ErrBatchAlreadySealed  = fmt.Errorf("batch already sealed")
	ErrDuplicateMeasurement = fmt.Errorf("duplicate measurement checksum (idempotent)")
	ErrSameEndpointPair    = fmt.Errorf("measurement endpoints must differ")
	ErrInvalidSkew         = fmt.Errorf("skew must be non-negative")
	ErrConditionConflict   = fmt.Errorf("exemption condition conflicts with existing condition")
	ErrRevokedExemption    = fmt.Errorf("exemption already revoked")
	ErrMissingDefaultSkew  = fmt.Errorf("batch default skew not set")
)

// ErrKind 返回错误对应的 HTTP 语义类别，便于上层统一处理。
// 返回 "not_found"、"conflict"、"bad_request"、"frozen" 或 "internal"。
func ErrKind(err error) string {
	if err == nil {
		return "internal"
	}
	switch err {
	case ErrBatchNotFound, ErrEndpointNotFound, ErrModeNotFound, ErrCornerNotFound,
		ErrExemptionNotFound, ErrRunNotFound, ErrSnapshotNotFound:
		return "not_found"
	case ErrSealedBatchModify, ErrSnapshotFrozen, ErrBatchAlreadySealed,
		ErrDuplicateMeasurement, ErrConditionConflict, ErrRevokedExemption:
		return "conflict"
	case ErrEndpointDomainKnown, ErrCornerMissing, ErrRuleCycle, ErrBatchNotOpen,
		ErrSameEndpointPair, ErrInvalidSkew, ErrMissingDefaultSkew:
		return "bad_request"
	default:
		return "internal"
	}
}
