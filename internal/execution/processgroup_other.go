//go:build !linux && !darwin && !windows

package execution

import "os/exec"

// setProcessGroup is a no-op on platforms without POSIX process groups.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup kills the direct child on platforms without process groups.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
