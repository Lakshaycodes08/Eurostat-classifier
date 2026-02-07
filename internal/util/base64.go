// base64.go decodes base64 or raw input for use in exec payloads (e.g. file content).
package util

import (
	"encoding/base64"
	"strings"
)

// DecodeBase64OrRaw decodes s as base64. If decoding fails (e.g. not base64),
// returns the original string as bytes so both base64 and plain content work.
// Whitespace (including newlines) is stripped before decoding so API-wrapped
// base64 still decodes.
func DecodeBase64OrRaw(s string) []byte {
	trimmed := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, s)
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		// Try without padding (some APIs omit it)
		decoded, err = base64.RawStdEncoding.DecodeString(trimmed)
		if err != nil {
			return []byte(s)
		}
	}
	return decoded
}
