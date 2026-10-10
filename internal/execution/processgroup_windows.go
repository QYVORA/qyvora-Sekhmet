//go:build windows

package execution

import "os/exec"

// setProcessGroup is a no-op on Windows: SysProcAttr has no Setpgid there, and
// exec.CommandContext already terminates the child on cancellation.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup kills the direct child; Windows has no POSIX process-group
// signal to deliver.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
