package util

import (
	"runtime"
	"testing"
)

func TestExecJSONInvalidHint(t *testing.T) {
	t.Run("ampersand", func(t *testing.T) {
		h := ExecJSONInvalidHint([]byte(`{"a":"b&c"}`))
		if h == "" {
			t.Fatal("expected hint for & in payload")
		}
	})
	t.Run("windows_no_ampersand", func(t *testing.T) {
		h := ExecJSONInvalidHint([]byte(`{`))
		if runtime.GOOS == "windows" {
			if h == "" {
				t.Fatal("expected windows hint")
			}
		} else if h != "" {
			t.Fatalf("unexpected hint on non-windows: %q", h)
		}
	})
}
