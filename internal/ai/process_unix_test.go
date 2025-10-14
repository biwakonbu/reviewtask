//go:build !windows

package ai

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestSetProcessGroup_Unix tests Unix process group setup
func TestSetProcessGroup_Unix(t *testing.T) {
	cmd := exec.Command("sleep", "5")

	// Should not panic or error
	setProcessGroup(cmd)

	// Verify SysProcAttr was configured
	if cmd.SysProcAttr == nil {
		t.Fatal("Expected SysProcAttr to be set")
	}

	if !cmd.SysProcAttr.Setpgid {
		t.Fatal("Expected Setpgid to be true")
	}
}

// TestKillProcessGroup_Unix_NoProcess tests killing with nil process
func TestKillProcessGroup_Unix_NoProcess(t *testing.T) {
	err := killProcessGroup(nil)
	if err != nil {
		t.Errorf("Expected no error for nil process, got: %v", err)
	}
}

// TestKillProcessGroup_Unix_Success tests successful process group killing
func TestKillProcessGroup_Unix_Success(t *testing.T) {
	// Create a simple command that will run
	cmd := exec.Command("sleep", "10")

	// Set up process group
	setProcessGroup(cmd)

	// Start the process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	pid := cmd.Process.Pid

	// Verify process is running
	if !isProcessRunning(pid) {
		t.Fatal("Process should be running")
	}

	// Kill the process group
	err := killProcessGroup(cmd)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Wait for process to be killed and reaped
	// Use cmd.Wait() to properly clean up the process
	cmd.Wait()

	// Give additional time for process to be fully terminated
	time.Sleep(200 * time.Millisecond)

	// Verify process is no longer running
	if isProcessRunning(pid) {
		t.Error("Process should not be running after killProcessGroup")
	}
}

// TestKillProcessGroup_Unix_WithChildren tests that child processes are killed
func TestKillProcessGroup_Unix_WithChildren(t *testing.T) {
	// Create a parent process that spawns children
	// Use bash to create a subprocess
	cmd := exec.Command("bash", "-c", "sleep 10 & sleep 10 & wait")

	// Set up process group
	setProcessGroup(cmd)

	// Start the parent process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start parent process: %v", err)
	}

	parentPID := cmd.Process.Pid

	// Give time for child processes to spawn
	time.Sleep(500 * time.Millisecond)

	// Get child processes before killing
	childPIDs := getChildProcesses(parentPID)
	if len(childPIDs) == 0 {
		t.Log("Warning: No child processes detected (may be timing issue)")
	}

	// Kill the process group
	err := killProcessGroup(cmd)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Wait for the command to fully exit
	cmd.Wait()

	// Give time for all processes to be killed
	time.Sleep(300 * time.Millisecond)

	// Verify parent is no longer running
	if isProcessRunning(parentPID) {
		t.Error("Parent process should not be running")
	}

	// Verify children are no longer running
	for _, childPID := range childPIDs {
		if isProcessRunning(childPID) {
			t.Errorf("Child process %d should not be running", childPID)
		}
	}
}

// TestProcessGroupIsolation_Unix tests that process groups are isolated
func TestProcessGroupIsolation_Unix(t *testing.T) {
	// Create two separate process groups
	cmd1 := exec.Command("sleep", "10")
	cmd2 := exec.Command("sleep", "10")

	setProcessGroup(cmd1)
	setProcessGroup(cmd2)

	if err := cmd1.Start(); err != nil {
		t.Fatalf("Failed to start first process: %v", err)
	}

	if err := cmd2.Start(); err != nil {
		t.Fatalf("Failed to start second process: %v", err)
	}

	pid1 := cmd1.Process.Pid
	pid2 := cmd2.Process.Pid

	// Verify both are running
	time.Sleep(100 * time.Millisecond)
	if !isProcessRunning(pid1) {
		t.Fatal("First process should be running")
	}
	if !isProcessRunning(pid2) {
		t.Fatal("Second process should be running")
	}

	// Kill first process group
	if err := killProcessGroup(cmd1); err != nil {
		t.Errorf("Failed to kill first process group: %v", err)
	}

	// Wait for first process to exit
	cmd1.Wait()
	time.Sleep(200 * time.Millisecond)

	// First process should be dead
	if isProcessRunning(pid1) {
		t.Error("First process should be dead")
	}

	// Second process should still be alive
	if !isProcessRunning(pid2) {
		t.Error("Second process should still be alive")
	}

	// Clean up second process
	killProcessGroup(cmd2)
	cmd2.Wait()
}

// TestKillProcessGroup_Unix_AlreadyExited tests killing an already-exited process
func TestKillProcessGroup_Unix_AlreadyExited(t *testing.T) {
	cmd := exec.Command("echo", "test")

	setProcessGroup(cmd)

	// Run the command (it will exit immediately)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}

	// Try to kill it (should handle gracefully)
	err := killProcessGroup(cmd)
	// We expect an error here since the process is already gone
	// but the function should not panic
	if err == nil {
		t.Log("No error returned when killing already-exited process (acceptable)")
	} else {
		t.Logf("Got error when killing already-exited process: %v (acceptable)", err)
	}
}

// Helper function to check if a process is running
func isProcessRunning(pid int) bool {
	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Helper function to get child processes of a parent PID
func getChildProcesses(parentPID int) []int {
	// Use pgrep to find child processes
	cmd := exec.Command("pgrep", "-P", fmt.Sprintf("%d", parentPID))
	output, err := cmd.Output()
	if err != nil {
		// No child processes or command failed
		return nil
	}

	// Parse PIDs from output (one per line)
	var pids []int
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			pids = append(pids, pid)
		}
	}

	return pids
}

// BenchmarkSetProcessGroup_Unix benchmarks Unix process group setup
func BenchmarkSetProcessGroup_Unix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("echo", "test")
		setProcessGroup(cmd)
	}
}

// BenchmarkKillProcessGroup_Unix benchmarks Unix process group killing
func BenchmarkKillProcessGroup_Unix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("sleep", "0.1")
		setProcessGroup(cmd)

		if err := cmd.Start(); err != nil {
			b.Fatalf("Failed to start process: %v", err)
		}

		if err := killProcessGroup(cmd); err != nil {
			b.Errorf("Failed to kill process: %v", err)
		}
	}
}
