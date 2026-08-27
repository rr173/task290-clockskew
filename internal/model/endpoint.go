package model

// ClockEndpoint 是时钟树上的一个终点（通常是触发器/寄存器时钟引脚），
// 带物理坐标用于校验豁免的物理距离前提。
type ClockEndpoint struct {
	ID          string  `json:"id"`
	BatchID     string  `json:"batch_id"`
	Name        string  `json:"name"`
	ClockName   string  `json:"clock_name"`   // 所属时钟名，如 clk1
	ClockDomain string  `json:"clock_domain"` // 时钟域，必须显式提供，未知域拒绝
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	CreatedAt   string  `json:"created_at"`
}

// DistanceTo 计算两个端点间的欧氏物理距离（微米）。
func (e *ClockEndpoint) DistanceTo(o *ClockEndpoint) float64 {
	dx := e.X - o.X
	dy := e.Y - o.Y
	return sqrt(dx*dx + dy*dy)
}

// sqrt 提供无 math 依赖的平方根，避免 float64 溢出处理复杂化。
func sqrt(v float64) float64 {
	if v <= 0 {
		return 0
	}
	// 牛顿迭代 8 轮已足够达到 float64 精度。
	x := v
	for i := 0; i < 8; i++ {
		x = (x + v/x) / 2
	}
	return x
}
