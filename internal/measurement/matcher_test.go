package measurement

import (
	"testing"

	"task290-clockskew/internal/model"
)

// TestChecksumDeterministic 验证摘要幂等：相同内容产生相同校验和。
func TestChecksumDeterministic(t *testing.T) {
	a := Checksum("b1", "ea", "eb", "m1", "c1", 50, "2026-08-27T08:00:00Z")
	b := Checksum("b1", "ea", "eb", "m1", "c1", 50, "2026-08-27T08:00:00Z")
	if a != b {
		t.Fatalf("checksum not deterministic: %s vs %s", a, b)
	}
	c := Checksum("b1", "ea", "eb", "m1", "c1", 51, "2026-08-27T08:00:00Z")
	if a == c {
		t.Fatalf("checksum should differ when skew differs")
	}
}

// TestMatch 验证测量↔豁免匹配：模式+角一致才覆盖；端点对双向匹配。
func TestMatch(t *testing.T) {
	ex := &model.Exemption{ID: "ex1", FromEndpoint: "ea", ToEndpoint: "eb", MaxSkewPS: 100}
	conds := []*model.ExemptionCondition{
		{ExemptionID: "ex1", ModeID: "functional", CornerID: "typ", MaxDistanceUM: 500},
	}

	m := &model.SkewMeasurement{
		ID: "m1", EndpointA: "ea", EndpointB: "eb", ModeID: "functional", CornerID: "typ", SkewPS: 50,
	}
	if got := Match(m, ex, conds); got == nil {
		t.Fatalf("expected match for functional/typ")
	}

	// 反向端点对也应匹配（双向路径）。
	m2 := &model.SkewMeasurement{
		ID: "m2", EndpointA: "eb", EndpointB: "ea", ModeID: "functional", CornerID: "typ", SkewPS: 50,
	}
	if got := Match(m2, ex, conds); got == nil {
		t.Fatalf("expected match for reversed endpoint pair")
	}

	// 模式不匹配 → 不覆盖。
	m3 := &model.SkewMeasurement{
		ID: "m3", EndpointA: "ea", EndpointB: "eb", ModeID: "scan", CornerID: "typ", SkewPS: 950,
	}
	if got := Match(m3, ex, conds); got != nil {
		t.Fatalf("expected no match for scan mode")
	}
}

// TestCovers 验证模式/工艺角覆盖判断。
func TestCovers(t *testing.T) {
	conds := []*model.ExemptionCondition{
		{ModeID: "functional", CornerID: "typ"},
	}
	if !CoversMode(conds, "functional") {
		t.Fatalf("expected covers functional")
	}
	if CoversMode(conds, "scan") {
		t.Fatalf("expected not covers scan")
	}
	if !CoversCorner(conds, "typ") {
		t.Fatalf("expected covers typ")
	}
}
