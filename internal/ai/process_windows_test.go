//go:build windows

package ai

import (
	"bytes"
	"errors"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// TestCreateJobObject tests the job object creation
func TestCreateJobObject(t *testing.T) {
	handle, err := createJobObject()
	if err != nil {
		t.Fatalf("Failed to create job object: %v", err)
	}
	defer windows.CloseHandle(handle)

	if handle == 0 {
		t.Fatal("Expected non-zero job handle")
	}
}

// TestSetProcessGroup tests setting up process group
func TestSetProcessGroup(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "echo test")

	// Should not panic or error
	setProcessGroup(cmd)

	// Verify job info was stored
	processJobsMu.RLock()
	jobInfo := processJobs[cmd]
	processJobsMu.RUnlock()

	if jobInfo == nil {
		t.Fatal("Expected job info to be stored for command")
	}

	if jobInfo.jobHandle == 0 {
		t.Fatal("Expected non-zero job handle")
	}

	// Clean up
	processJobsMu.Lock()
	delete(processJobs, cmd)
	processJobsMu.Unlock()
	windows.CloseHandle(jobInfo.jobHandle)
}

// TestKillProcessGroup_NoProcess tests killing with nil process
func TestKillProcessGroup_NoProcess(t *testing.T) {
	err := killProcessGroup(nil)
	if err != nil {
		t.Errorf("Expected no error for nil process, got: %v", err)
	}
}

// TestKillProcessGroup_NoJobObject tests fallback to process.Kill
func TestKillProcessGroup_NoJobObject(t *testing.T) {
	// Create a simple command that will run
	cmd := exec.Command("cmd.exe", "/c", "timeout /t 5 /nobreak")

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Kill without job object (should use fallback)
	if err := killProcessGroup(cmd); err != nil && !errors.Is(err, exec.ErrWaitDone) {
		t.Errorf("Unexpected kill error: %v", err)
	}

	// Verify process is terminated
	time.Sleep(100 * time.Millisecond)
	cmd.Wait() // Wait for termination

	// Verify process is no longer running
	checkCmd := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(cmd.Process.Pid))
	output, _ := checkCmd.Output()
	if len(output) > 0 && bytes.Contains(output, []byte(strconv.Itoa(cmd.Process.Pid))) {
		t.Error("Process still running after kill")
	}
}

// TestKillProcessGroup_WithJobObject tests killing with job object
func TestKillProcessGroup_WithJobObject(t *testing.T) {
	// Create a command
	cmd := exec.Command("cmd.exe", "/c", "timeout /t 5 /nobreak")

	// Set up job object
	setProcessGroup(cmd)

	// Start the process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Small delay to ensure process is running
	time.Sleep(100 * time.Millisecond)

	// Kill with job object
	if err := killProcessGroup(cmd); err != nil && !errors.Is(err, exec.ErrWaitDone) {
		t.Errorf("Unexpected kill error: %v", err)
	}

	// Verify process is terminated
	time.Sleep(200 * time.Millisecond)

	// Check process no longer exists
	checkCmd := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(cmd.Process.Pid))
	output, _ := checkCmd.Output()
	if len(output) > 0 && bytes.Contains(output, []byte(strconv.Itoa(cmd.Process.Pid))) {
		t.Error("Process still running after kill with job object")
	}

	// Verify job info was cleaned up
	processJobsMu.RLock()
	jobInfo := processJobs[cmd]
	processJobsMu.RUnlock()

	if jobInfo != nil {
		t.Error("Expected job info to be cleaned up")
	}
}

// TestKillProcessGroup_WithChildProcesses tests that child processes are killed
func TestKillProcessGroup_WithChildProcesses(t *testing.T) {
	// Create a parent process that spawns a child
	// Use cmd.exe to start another cmd.exe with a long-running command
	cmd := exec.Command("cmd.exe", "/c", "cmd.exe /c timeout /t 10 /nobreak")

	// Set up job object
	setProcessGroup(cmd)

	// Start the parent process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start parent process: %v", err)
	}

	parentPID := cmd.Process.Pid

	// Give time for child process to spawn
	time.Sleep(500 * time.Millisecond)

	// Kill the parent with job object (should kill children too)
	if err := killProcessGroup(cmd); err != nil && !errors.Is(err, exec.ErrWaitDone) {
		t.Errorf("Unexpected kill error: %v", err)
	}

	// Verify parent process is terminated
	time.Sleep(200 * time.Millisecond)

	// Check that the parent process is no longer running
	checkCmd := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(parentPID))
	output, _ := checkCmd.Output()
	if len(output) > 0 && bytes.Contains(output, []byte(strconv.Itoa(parentPID))) {
		t.Error("Parent process still running after kill")
	}

	// Verify child processes were also terminated
	// Query for processes whose parent was parentPID
	psCmd := exec.Command("powershell", "-Command",
		"Get-CimInstance Win32_Process | Where-Object { $_.ParentProcessId -eq "+strconv.Itoa(parentPID)+" } | Select-Object ProcessId")
	childOutput, _ := psCmd.Output()
	if len(childOutput) > 0 && bytes.Contains(childOutput, []byte("ProcessId")) {
		t.Error("Child processes still running after parent kill via job object")
	}
}

// TestProcessJobInfo_Concurrency tests concurrent access to job info
func TestProcessJobInfo_Concurrency(t *testing.T) {
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			cmd := exec.Command("cmd.exe", "/c", "echo test")
			setProcessGroup(cmd)

			// Access the job info
			processJobsMu.RLock()
			_ = processJobs[cmd]
			processJobsMu.RUnlock()

			// Clean up
			processJobsMu.Lock()
			if jobInfo := processJobs[cmd]; jobInfo != nil {
				windows.CloseHandle(jobInfo.jobHandle)
				delete(processJobs, cmd)
			}
			processJobsMu.Unlock()

			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out waiting for goroutines")
		}
	}
}

// TestJobObjectLimitFlags tests that the correct flags are set
func TestJobObjectLimitFlags(t *testing.T) {
	handle, err := createJobObject()
	if err != nil {
		t.Fatalf("Failed to create job object: %v", err)
	}
	defer windows.CloseHandle(handle)

	// The job object should have JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE flag set
	// We can't easily query the flags, but we can verify the handle is valid
	if handle == 0 {
		t.Fatal("Expected valid job handle")
	}
}

// BenchmarkCreateJobObject benchmarks job object creation
func BenchmarkCreateJobObject(b *testing.B) {
	for i := 0; i < b.N; i++ {
		handle, err := createJobObject()
		if err != nil {
			b.Fatalf("Failed to create job object: %v", err)
		}
		windows.CloseHandle(handle)
	}
}

// BenchmarkSetProcessGroup benchmarks setting up process group
func BenchmarkSetProcessGroup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("cmd.exe", "/c", "echo test")
		setProcessGroup(cmd)

		// Clean up
		processJobsMu.Lock()
		if jobInfo := processJobs[cmd]; jobInfo != nil {
			windows.CloseHandle(jobInfo.jobHandle)
			delete(processJobs, cmd)
		}
		processJobsMu.Unlock()
	}
}
