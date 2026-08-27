package exemption

import (
	"testing"

	"task290-clockskew/internal/model"
)

// TestMatchesMeasurement 验证测量违反豁免的判定：skew 超限与距离超限。
func TestMatchesMeasurement(t *testing.T) {
	ex := &model.Exemption{ID: "ex1", MaxSkewPS: 100}
	cond := &model.ExemptionCondition{MaxDistanceUM: 500}

	mOK := &model.SkewMeasurement{ID: "m1", SkewPS: 50}
	if typ, msg, hit := MatchesMeasurement(ex, cond, mOK, 141.4); hit {
		t.Fatalf("expected no violation, got %s: %s", typ, msg)
	}

	mSkew := &model.SkewMeasurement{ID: "m2", SkewPS: 950}
	typ, _, hit := MatchesMeasurement(ex, cond, mSkew, 141.4)
	if !hit || typ != model.FailureTypeSkewExceeded {
		t.Fatalf("expected skew_exceeded, got %s hit=%v", typ, hit)
	}

	mDist := &model.SkewMeasurement{ID: "m3", SkewPS: 50}
	typ, _, hit = MatchesMeasurement(ex, cond, mDist, 999)
	if !hit || typ != model.FailureTypeDistanceViolation {
		t.Fatalf("expected distance_violation, got %s hit=%v", typ, hit)
	}
}

// TestTopoOrder 验证依赖拓扑排序：有环返回 false。
func TestTopoOrder(t *testing.T) {
	deps := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {},
	}
	order, ok := TopoOrder(deps, []string{"a", "b", "c"})
	if !ok {
		t.Fatalf("expected acyclic topo order")
	}
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes ordered, got %d", len(order))
	}
	pos := map[string]int{}
	for i, id := range order {
		pos[id] = i
	}
	if pos["b"] > pos["a"] {
		t.Fatalf("a depends on b, b must come before a")
	}
	if pos["c"] > pos["b"] {
		t.Fatalf("b depends on c, c must come before b")
	}

	cyclic := map[string][]string{
		"a": {"b"},
		"b": {"a"},
	}
	if _, ok := TopoOrder(cyclic, []string{"a", "b"}); ok {
		t.Fatalf("expected cycle detection to fail topo order")
	}
}
