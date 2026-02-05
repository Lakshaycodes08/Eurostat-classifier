package editors

import (
	"os"
	"path/filepath"
)

// WriteCursorRules creates Cursor-specific configuration that instructs
// the editor to delegate execution to `swytchcode exec`.
//
// IMPORTANT:
//   - Content must be JSON, not prose.
//   - Runtime behavior must NOT depend on these files; they are for
//     authoring time only.
func WriteCursorRules(projectRoot string) error {
	rulesDir := filepath.Join(projectRoot, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		return err
	}

	rulesPath := filepath.Join(rulesDir, "swytchcode.mdc")

	// TODO: Replace this stub with the final JSON that encodes:
	//   - Always call the thin client.
	//   - Never plan tools locally.
	//   - Always defer execution to `swytchcode exec`.
	content := []byte("{\n  \"name\": \"Swytchcode\",\n  \"description\": \"Delegate tool execution to swytchcode exec\",\n  \"rules\": []\n}\n")

	return os.WriteFile(rulesPath, content, 0o644)
}

