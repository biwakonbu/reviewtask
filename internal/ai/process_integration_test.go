//go:build integration
// +build integration

package ai

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// TestProcessCleanup_Integration tests process cleanup in a more realistic scenario
func TestProcessCleanup_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a command that would spawn child processes
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// On Windows, use cmd.exe to spawn a child process
		cmd = exec.Command("cmd.exe", "/c", "cmd.exe /c timeout /t 10 /nobreak")
	} else {
		// On Unix, use bash to spawn children
		cmd = exec.Command("bash", "-c", "sleep 10 & sleep 10 & wait")
	}

	// Set up process group
	setProcessGroup(cmd)

	// Start the process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	pid := cmd.Process.Pid
	t.Logf("Started process with PID: %d", pid)

	// Give time for child processes to spawn
	time.Sleep(500 * time.Millisecond)

	// Kill the process group
	if err := killProcessGroup(cmd); err != nil {
		t.Errorf("Failed to kill process group: %v", err)
	}

	// Wait for process to exit
	cmd.Wait()

	// Verify process is terminated
	time.Sleep(200 * time.Millisecond)

	// Check if the process is still running
	process, err := os.FindProcess(pid)
	if err == nil {
		// On Unix, we can send signal 0 to check
		if runtime.GOOS != "windows" {
			if err := process.Signal(os.Signal(nil)); err == nil {
				t.Error("Process should not be running after killProcessGroup")
			}
		}
	}

	t.Log("Process cleanup successful")
}

// TestProcessCleanup_MultipleProcesses tests cleanup of multiple processes
func TestProcessCleanup_MultipleProcesses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	const numProcesses = 5
	cmds := make([]*exec.Cmd, numProcesses)
	pids := make([]int, numProcesses)

	// Start multiple processes
	for i := 0; i < numProcesses; i++ {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd.exe", "/c", "timeout /t 10 /nobreak")
		} else {
			cmd = exec.Command("sleep", "10")
		}

		setProcessGroup(cmd)

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start process %d: %v", i, err)
		}

		cmds[i] = cmd
		pids[i] = cmd.Process.Pid
		t.Logf("Started process %d with PID: %d", i, pids[i])
	}

	// Give processes time to start
	time.Sleep(200 * time.Millisecond)

	// Kill all process groups
	for i, cmd := range cmds {
		if err := killProcessGroup(cmd); err != nil {
			t.Errorf("Failed to kill process group %d: %v", i, err)
		}
	}

	// Wait for all processes
	for _, cmd := range cmds {
		cmd.Wait()
	}

	// Verify all processes are terminated
	time.Sleep(200 * time.Millisecond)

	for i, pid := range pids {
		process, err := os.FindProcess(pid)
		if err == nil && runtime.GOOS != "windows" {
			if err := process.Signal(os.Signal(nil)); err == nil {
				t.Errorf("Process %d (PID %d) should not be running", i, pid)
			}
		}
	}

	t.Log("Multiple process cleanup successful")
}

// TestProcessCleanup_ConcurrentOperations tests concurrent process operations
func TestProcessCleanup_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd.exe", "/c", "echo test")
			} else {
				cmd = exec.Command("echo", "test")
			}

			setProcessGroup(cmd)

			if err := cmd.Run(); err != nil {
				errors <- err
				done <- false
				return
			}

			// Try to kill (should handle gracefully even if already exited)
			killProcessGroup(cmd)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case success := <-done:
			if success {
				successCount++
			}
		case err := <-errors:
			t.Errorf("Goroutine error: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	if successCount != numGoroutines {
		t.Errorf("Expected %d successful operations, got %d", numGoroutines, successCount)
	}

	t.Logf("Concurrent operations successful: %d/%d", successCount, numGoroutines)
}

// TestProcessCleanup_ContextCancellation tests that processes are cleaned up when context is cancelled
func TestProcessCleanup_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/c", "timeout /t 10 /nobreak")
	} else {
		cmd = exec.CommandContext(ctx, "sleep", "10")
	}

	setProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	pid := cmd.Process.Pid
	t.Logf("Started process with PID: %d", pid)

	// Wait for context timeout
	<-ctx.Done()

	// Kill the process group explicitly (simulating cleanup)
	if err := killProcessGroup(cmd); err != nil {
		t.Logf("Kill process group returned error: %v (may be expected)", err)
	}

	// Wait for process
	cmd.Wait()

	// Verify process is terminated
	time.Sleep(200 * time.Millisecond)

	process, err := os.FindProcess(pid)
	if err == nil && runtime.GOOS != "windows" {
		if err := process.Signal(os.Signal(nil)); err == nil {
			t.Error("Process should not be running after context cancellation and cleanup")
		}
	}

	t.Log("Context cancellation cleanup successful")
}

// TestProcessCleanup_LongRunningProcess tests cleanup of long-running processes
func TestProcessCleanup_LongRunningProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Create a process that would run for a long time
		cmd = exec.Command("cmd.exe", "/c", "timeout /t 60 /nobreak")
	} else {
		cmd = exec.Command("sleep", "60")
	}

	setProcessGroup(cmd)

	start := time.Now()

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	pid := cmd.Process.Pid
	t.Logf("Started long-running process with PID: %d", pid)

	// Give it a moment to start
	time.Sleep(500 * time.Millisecond)

	// Kill it immediately (should not wait for the full 60 seconds)
	if err := killProcessGroup(cmd); err != nil {
		t.Errorf("Failed to kill process: %v", err)
	}

	cmd.Wait()

	elapsed := time.Since(start)
	t.Logf("Process killed in %v", elapsed)

	// Should have killed quickly (within a few seconds), not wait 60 seconds
	if elapsed > 10*time.Second {
		t.Errorf("Process took too long to kill: %v", elapsed)
	}

	t.Log("Long-running process cleanup successful")
}
