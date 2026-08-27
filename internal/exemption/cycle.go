package exemption

import (
	"task290-clockskew/internal/store"
)

// HasCycle 从 start 出发沿依赖边做 DFS，判断是否能回到 start（形成环）。
// edges 以完整依赖表为准，调用方负责提供最新数据源。
func HasCycle(start, current string, es *store.ExemptionStore) bool {
	// 仅从豁免自身出发沿出边探测，不沿 dependsOn 反向展开。
	deps, err := es.Dependencies(start)
	if err != nil {
		return true
	}
	for _, d := range deps {
		if d.DependsOn == current {
			return true
		}
	}
	return false
}

// TopoOrder 对一批豁免做拓扑排序（依赖优先），用于验证传播顺序。
// deps[id] 表示 id 依赖的节点列表；被依赖者必须先于依赖者输出。
// 返回排序后的豁免 ID 列表；若检测到环返回空列表与 false。
func TopoOrder(deps map[string][]string, ids []string) ([]string, bool) {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	indeg := make(map[string]int, len(ids))
	adj := make(map[string][]string)
	for _, id := range ids {
		indeg[id] = 0
	}
	for _, id := range ids {
		for _, t := range deps[id] {
			if !set[t] {
				continue
			}
			indeg[id]++ // id 有一个前置 t
			adj[t] = append(adj[t], id)
		}
	}
	var queue []string
	for _, id := range ids {
		if indeg[id] == 0 {
			queue = append(queue, id)
		}
	}
	var order []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		order = append(order, cur)
		for _, n := range adj[cur] {
			indeg[n]--
			if indeg[n] == 0 {
				queue = append(queue, n)
			}
		}
	}
	if len(order) != len(ids) {
		return nil, false
	}
	return order, true
}
