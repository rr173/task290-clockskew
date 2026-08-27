package endpoint

import "strings"

// NormalizeDomain 规整时钟域名称：去空格、统一小写。
// 返回空串表示未提供时钟域。
func NormalizeDomain(d string) string {
	return strings.ToLower(strings.TrimSpace(d))
}

// DomainGroups 将一批端点的时钟域归类为「已声明域」与「未知域」，
// 用于批量导入前的整体校验，避免逐条回滚。
func DomainGroups(domains []string) (declared []string, unknown []string) {
	for _, d := range domains {
		n := NormalizeDomain(d)
		if n == "" {
			unknown = append(unknown, d)
			continue
		}
		declared = append(declared, n)
	}
	return declared, unknown
}

// SameDomain 判断两个时钟域是否属于同一域（域比较不区分大小写）。
func SameDomain(a, b string) bool {
	return NormalizeDomain(a) == NormalizeDomain(b)
}
