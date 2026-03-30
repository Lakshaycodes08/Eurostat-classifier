// jsonhint.go adds contextual hints when JSON input fails to parse (e.g. Windows cmd and &).
package util

import (
	"runtime"
	"strings"
)

// ExecJSONInvalidHint returns a short hint for exec/body JSON parse failures.
// It triggers when the payload contains '&' (often mangled by cmd.exe) or on Windows
// where inline JSON is easy to break.
func ExecJSONInvalidHint(payload []byte) string {
	s := string(payload)
	if strings.Contains(s, "&") {
		return "If you used cmd.exe, `&` may have broken the command line — put the JSON in a file and use `swytchcode exec --body file.json`, or use PowerShell with proper quoting. See docs/windows-guide.md."
	}
	if runtime.GOOS == "windows" {
		return "On Windows, prefer `swytchcode exec --body payload.json` for complex JSON instead of inline strings. See docs/windows-guide.md."
	}
	return ""
}
