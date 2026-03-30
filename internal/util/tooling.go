// tooling.go loads project tooling.json in one place for consistent errors.
package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// LoadToolingJSON reads and parses .swytchcode/tooling.json under projectRoot.
// It returns a wrapped error if the file is missing, unreadable, or not valid JSON object.
func LoadToolingJSON(projectRoot string) (map[string]interface{}, error) {
	path := ToolingPath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("tooling.json not found; run 'swytchcode init' first: %w", err)
		}
		return nil, fmt.Errorf("read tooling.json: %w", err)
	}
	var tooling map[string]interface{}
	if err := json.Unmarshal(data, &tooling); err != nil {
		return nil, fmt.Errorf("parse tooling.json: %w", err)
	}
	if tooling == nil {
		return nil, fmt.Errorf("parse tooling.json: root must be a JSON object")
	}
	return tooling, nil
}
