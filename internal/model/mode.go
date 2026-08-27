package model

// Mode 定义一个芯片工作模式，如 functional（功能模式）、scan（扫描测试模式）、
// shift（移位模式）、capture（捕获模式）。豁免条件按模式声明其适用性。
type Mode struct {
	ID          string `json:"id"`
	BatchID     string `json:"batch_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// Corner 定义一个工艺角（process/voltage/temperature corner），
// 如 typ（典型）、ss（慢慢）、ff（快快）。快照发布时固定工艺角集合。
type Corner struct {
	ID          string `json:"id"`
	BatchID     string `json:"batch_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}
