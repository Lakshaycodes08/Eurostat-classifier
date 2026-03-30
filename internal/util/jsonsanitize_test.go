package util

import (
	"encoding/json"
	"math"
	"testing"
)

func TestSanitizeForJSON_NaN(t *testing.T) {
	in := map[string]interface{}{
		"x": math.NaN(),
		"y": 1.5,
	}
	out := SanitizeForJSON(in)
	m, ok := out.(map[string]interface{})
	if !ok {
		t.Fatalf("want map, got %T", out)
	}
	if m["x"] != nil {
		t.Errorf("x = %v, want nil", m["x"])
	}
	if m["y"] != 1.5 {
		t.Errorf("y = %v", m["y"])
	}
	if _, err := json.Marshal(out); err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
}
