//go:build windows

package ai

import (
	"os/exec"
)

// setProcessGroup is a no-op on Windows
// Windows doesn't use Unix process groups
func setProcessGroup(cmd *exec.Cmd) {
	// No-op on Windows
}

// killProcessGroup kills the process on Windows
func killProcessGroup(process *exec.Cmd) error {
	if process == nil || process.Process == nil {
		return nil
	}

	// On Windows, just kill the main process
	// Windows will handle child process cleanup
	return process.Process.Kill()
}
