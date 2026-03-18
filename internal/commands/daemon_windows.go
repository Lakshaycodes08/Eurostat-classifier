//go:build windows

package commands

import "syscall"

// daemonSysProcAttr returns nil on Windows (no equivalent of Setsid).
func daemonSysProcAttr() *syscall.SysProcAttr {
	return nil
}
