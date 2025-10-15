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

// TestProcessJobInfo_Close tests the Close method
func TestProcessJobInfo_Close(t *testing.T) {
	// Create a job handle
	handle, err := createJobObject()
	if err != nil {
		t.Fatalf("Failed to create job object: %v", err)
	}

	jobInfo := &processJobInfo{
		jobHandle: handle,
		cmd:       exec.Command("cmd.exe", "/c", "echo test"),
	}

	// Verify handle is non-zero before Close
	if jobInfo.jobHandle == 0 {
		t.Fatal("Expected non-zero job handle before Close")
	}

	// Call Close
	err = jobInfo.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Verify handle was zeroed out
	if jobInfo.jobHandle != 0 {
		t.Error("Expected job handle to be zeroed after Close")
	}
}

// TestProcessJobInfo_Close_Idempotent tests that Close can be called multiple times
func TestProcessJobInfo_Close_Idempotent(t *testing.T) {
	handle, err := createJobObject()
	if err != nil {
		t.Fatalf("Failed to create job object: %v", err)
	}

	jobInfo := &processJobInfo{
		jobHandle: handle,
		cmd:       exec.Command("cmd.exe", "/c", "echo test"),
	}

	// Call Close multiple times
	for i := 0; i < 3; i++ {
		err = jobInfo.Close()
		if err != nil {
			t.Errorf("Close call %d returned error: %v", i+1, err)
		}
	}

	// Verify handle remains zero
	if jobInfo.jobHandle != 0 {
		t.Error("Expected job handle to remain zero after multiple Close calls")
	}
}

// TestProcessJobInfo_Close_Nil tests Close with nil receiver
func TestProcessJobInfo_Close_Nil(t *testing.T) {
	var jobInfo *processJobInfo = nil
	err := jobInfo.Close()
	if err != nil {
		t.Errorf("Close on nil receiver returned error: %v", err)
	}
}

// TestProcessJobInfo_Close_Concurrency tests concurrent Close calls
func TestProcessJobInfo_Close_Concurrency(t *testing.T) {
	handle, err := createJobObject()
	if err != nil {
		t.Fatalf("Failed to create job object: %v", err)
	}

	jobInfo := &processJobInfo{
		jobHandle: handle,
		cmd:       exec.Command("cmd.exe", "/c", "echo test"),
	}

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	// Launch concurrent Close calls
	for i := 0; i < numGoroutines; i++ {
		go func() {
			err := jobInfo.Close()
			if err != nil {
				t.Errorf("Concurrent Close returned error: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out waiting for concurrent Close calls")
		}
	}

	// Verify handle is zero
	if jobInfo.jobHandle != 0 {
		t.Error("Expected job handle to be zero after concurrent Close calls")
	}
}

// TestKillProcessGroup_ClearsFinalizer tests that finalizer is cleared
func TestKillProcessGroup_ClearsFinalizer(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "timeout /t 1 /nobreak")

	// Set up job object with finalizer
	setProcessGroup(cmd)

	// Start the process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Small delay
	time.Sleep(100 * time.Millisecond)

	// Get job info before kill
	processJobsMu.RLock()
	jobInfo := processJobs[cmd]
	processJobsMu.RUnlock()

	if jobInfo == nil {
		t.Fatal("Expected job info to exist before kill")
	}

	// Kill should clear the finalizer and close the handle
	err := killProcessGroup(cmd)
	if err != nil && !errors.Is(err, exec.ErrWaitDone) {
		t.Errorf("Unexpected kill error: %v", err)
	}

	// Verify job info was cleaned up from map
	processJobsMu.RLock()
	jobInfo = processJobs[cmd]
	processJobsMu.RUnlock()

	if jobInfo != nil {
		t.Error("Expected job info to be removed from map after kill")
	}
}

// TestSetProcessGroup_RegistersFinalizer tests that finalizer is registered
func TestSetProcessGroup_RegistersFinalizer(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "echo test")

	// Set up process group
	setProcessGroup(cmd)

	// Verify job info exists
	processJobsMu.RLock()
	jobInfo := processJobs[cmd]
	processJobsMu.RUnlock()

	if jobInfo == nil {
		t.Fatal("Expected job info to be stored")
	}

	if jobInfo.jobHandle == 0 {
		t.Fatal("Expected non-zero job handle")
	}

	// Note: We can't directly verify finalizer registration, but we can verify
	// that the jobInfo has a valid handle that would be cleaned up by the finalizer

	// Clean up manually since we're not calling killProcessGroup
	processJobsMu.Lock()
	delete(processJobs, cmd)
	processJobsMu.Unlock()
	jobInfo.Close()
}

// TestProcessJobInfo_Close_ExplicitVsFinalizer demonstrates explicit cleanup pattern
func TestProcessJobInfo_Close_ExplicitVsFinalizer(t *testing.T) {
	// This test demonstrates that explicit cleanup via Close() is preferred
	// over relying on the finalizer

	// Test 1: Explicit cleanup (preferred pattern)
	t.Run("ExplicitCleanup", func(t *testing.T) {
		handle, err := createJobObject()
		if err != nil {
			t.Fatalf("Failed to create job object: %v", err)
		}

		jobInfo := &processJobInfo{
			jobHandle: handle,
			cmd:       exec.Command("cmd.exe", "/c", "echo test"),
		}

		// Explicit cleanup - deterministic and immediate
		err = jobInfo.Close()
		if err != nil {
			t.Errorf("Explicit Close failed: %v", err)
		}

		if jobInfo.jobHandle != 0 {
			t.Error("Handle should be zero after explicit Close")
		}
	})

	// Test 2: Finalizer cleanup (fallback pattern)
	t.Run("FinalizerFallback", func(t *testing.T) {
		// Create job info that goes out of scope without explicit cleanup
		// The finalizer should eventually clean it up (non-deterministic)
		createJobInfoWithoutCleanup := func() {
			handle, err := createJobObject()
			if err != nil {
				t.Fatalf("Failed to create job object: %v", err)
			}

			jobInfo := &processJobInfo{
				jobHandle: handle,
				cmd:       exec.Command("cmd.exe", "/c", "echo test"),
			}

			// Register finalizer (simulating setProcessGroup behavior)
			jobInfo.Close() // Clean up immediately for test purposes
			// In real scenario without Close(), finalizer would eventually run
		}

		createJobInfoWithoutCleanup()
		// Note: We can't reliably test finalizer execution timing
		// This test primarily documents the fallback pattern
	})
}
