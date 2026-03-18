//go:build !windows

package commands

import "syscall"

// daemonSysProcAttr returns process attributes that detach the child from the
// parent's terminal session (new session on Unix).
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
