//go:build linux

package trickle

import "syscall"

func crossPlatformSelect(nfd int, r, w, e *syscall.FdSet, timeout *syscall.Timeval) (int, error) {
	return syscall.Select(nfd, r, w, e, timeout)
}
