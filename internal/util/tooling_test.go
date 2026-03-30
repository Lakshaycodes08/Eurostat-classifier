package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
)

func TestLoadToolingJSON(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		dir := t.TempDir()
		_, err := LoadToolingJSON(dir)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ok", func(t *testing.T) {
		dir := t.TempDir()
		sw := filepath.Join(dir, constants.SwytchDirName)
		if err := os.MkdirAll(sw, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(sw, constants.ToolingJSONFile)
		obj := map[string]string{"version": "1"}
		raw, _ := json.Marshal(obj)
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		m, err := LoadToolingJSON(dir)
		if err != nil {
			t.Fatal(err)
		}
		if m["version"] != "1" {
			t.Fatalf("got %#v", m)
		}
	})

	t.Run("invalid_json", func(t *testing.T) {
		dir := t.TempDir()
		sw := filepath.Join(dir, constants.SwytchDirName)
		if err := os.MkdirAll(sw, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(sw, constants.ToolingJSONFile)
		if err := os.WriteFile(p, []byte(`not json`), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadToolingJSON(dir)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
