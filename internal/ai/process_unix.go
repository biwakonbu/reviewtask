//go:build !windows

package ai

import (
	"os/exec"
	"syscall"
)

// setProcessGroup sets up process group for Unix-like systems
// This allows killing all child processes when the parent is killed
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}
}

// killProcessGroup kills the entire process group (parent + all children)
func killProcessGroup(process *exec.Cmd) error {
	if process == nil || process.Process == nil {
		return nil
	}

	// Get the process group ID (should be same as process PID if Setpgid was used)
	pgid := process.Process.Pid

	// Kill entire process group (negative PID kills the group)
	// SIGKILL (9) ensures immediate termination
	return syscall.Kill(-pgid, syscall.SIGKILL)
}
