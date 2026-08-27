package model

// SkewMeasurement 是一条时钟偏斜测量：记录端点 A→B 在某模式、某工艺角下的偏斜值。
// 摘要幂等：batch_id + endpoint 对 + mode + corner + skew + 时间 的哈希唯一。
type SkewMeasurement struct {
	ID          string `json:"id"`
	BatchID     string `json:"batch_id"`
	Seq         int64  `json:"seq"` // 批次内递增序号，配合 checksum 实现幂等
	EndpointA   string `json:"endpoint_a"`
	EndpointB   string `json:"endpoint_b"`
	ModeID      string `json:"mode_id"`
	CornerID    string `json:"corner_id"`
	SkewPS      int64  `json:"skew_ps"` // 偏斜值，皮秒
	MeasuredAt  string `json:"measured_at"`
	Checksum    string `json:"checksum"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// Exemption 是一条时钟偏斜豁免：设计者声明 A→B 路径的偏斜可超过默认上限，
// 但该声明仅在若干（模式, 工艺角, 物理距离上限）条件组合下成立。
// 验证器检查这些前提在测量数据下是否仍然成立。
type Exemption struct {
	ID            string `json:"id"`
	BatchID       string `json:"batch_id"`
	Name          string `json:"name"`
	FromEndpoint  string `json:"from_endpoint"`
	ToEndpoint    string `json:"to_endpoint"`
	MaxSkewPS     int64  `json:"max_skew_ps"` // 豁免允许的偏斜上限（皮秒）
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

// ExemptionCondition 是豁免的一个前提条件：
// 在 mode 下、corner 下、且物理距离 ≤ MaxDistanceUM 时，该豁免才成立。
type ExemptionCondition struct {
	ID            string  `json:"id"`
	ExemptionID   string  `json:"exemption_id"`
	ModeID        string  `json:"mode_id"`
	CornerID      string  `json:"corner_id"`
	MaxDistanceUM float64 `json:"max_distance_um"`
}

// ExemptionDependency 是豁免之间的依赖边：exemption 依赖 dependsOn 的豁免成立。
// 依赖用于传播失效——上游豁免失效时，下游豁免一并失效。依赖图必须无环。
type ExemptionDependency struct {
	ID          string `json:"id"`
	ExemptionID string `json:"exemption_id"`
	DependsOn   string `json:"depends_on"`
}
