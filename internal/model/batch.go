package model

// Batch 是一个芯片签核批次：承接一组端点、测量、豁免规则，最终产出验证快照。
type Batch struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	DefaultMaxSkewPS int64  `json:"default_max_skew_ps"` // 未豁免路径的默认偏斜上限（皮秒）
	CreatedAt        string `json:"created_at"`
	OpenedAt         string `json:"opened_at,omitempty"`
	SealedAt         string `json:"sealed_at,omitempty"`
}

// CanTransition 校验批次状态流转合法性：
// receiving→pending（开启验证）、pending→has_failures/publishable（验证结果）、
// publishable→sealed（封存）、has_failures→pending（复验）。
func (b *Batch) CanTransition(next string) bool {
	switch b.Status {
	case BatchStatusReceiving:
		return next == BatchStatusPending
	case BatchStatusPending:
		return next == BatchStatusFailures || next == BatchStatusPublishable
	case BatchStatusFailures:
		return next == BatchStatusPending
	case BatchStatusPublishable:
		return next == BatchStatusSealed
	case BatchStatusSealed:
		return false
	default:
		return false
	}
}

// Sealed 返回批次是否已封存（封存后所有写操作被拒绝）。
func (b *Batch) Sealed() bool { return b.Status == BatchStatusSealed }

// Open 返回批次是否处于待验证状态。
func (b *Batch) Open() bool { return b.Status == BatchStatusPending }
