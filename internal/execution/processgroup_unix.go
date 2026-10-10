//go:build linux || darwin

package execution

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts the child in its own process group so a timeout can
// terminate it together with any backgrounded grandchildren.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup terminates the child's whole process group, falling back to
// killing the direct child.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err == nil {
		return
	}
	_ = cmd.Process.Kill()
}
