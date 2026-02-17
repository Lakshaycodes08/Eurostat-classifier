//go:build !windows
// +build !windows

package cli

import "syscall"

// getSysProcAttr returns platform-specific SysProcAttr for Unix systems
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true, // Create new session (detach from terminal)
	}
}
