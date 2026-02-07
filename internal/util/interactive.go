// interactive.go detects TTY for setup commands only; exec must never branch on TTY.
package util

import "os"

// IsInteractive reports whether stdin is connected to a TTY.
//
// This is used ONLY by setup commands (init/get) to decide whether
// they are allowed to prompt. The execution path (swytchcode exec)
// must never branch on TTY presence.
func IsInteractive() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

