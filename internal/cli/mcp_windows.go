//go:build windows
// +build windows

package cli

import "syscall"

// getSysProcAttr returns platform-specific SysProcAttr for Windows
func getSysProcAttr() *syscall.SysProcAttr {
	// Windows doesn't support Setsid - return nil
	return nil
}
