package test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestFetchCommandCtrlCHandling tests the complete Ctrl-C handling flow
func TestFetchCommandCtrlCHandling(t *testing.T) {
	// Skip in CI if TEST_SIGNAL_INTEGRATION is not set
	if os.Getenv("CI") == "true" && os.Getenv("TEST_SIGNAL_INTEGRATION") != "true" {
		t.Skip("Skipping signal integration test in CI (set TEST_SIGNAL_INTEGRATION=true to run)")
	}

	// Skip if we don't have the binary built
	binaryPath := getBinaryPath(t)
	if binaryPath == "" {
		t.Skip("reviewtask binary not found, skipping integration test")
	}

	tests := []struct {
		name           string
		signal         os.Signal
		delay          time.Duration
		expectExitCode bool // true if we expect a specific exit code
		maxWaitTime    time.Duration
	}{
		{
			name:           "SIGINT should terminate process gracefully",
			signal:         os.Interrupt,
			delay:          2 * time.Second,
			expectExitCode: false, // Process should exit cleanly
			maxWaitTime:    5 * time.Second,
		},
		{
			name:           "Quick SIGINT should still work",
			signal:         os.Interrupt,
			delay:          500 * time.Millisecond,
			expectExitCode: false,
			maxWaitTime:    3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context for the test
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Start the reviewtask fetch command
			cmd := exec.CommandContext(ctx, binaryPath, "fetch")
			cmd.Dir = getTestRepoDir(t)

			// Capture output for debugging
			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			// Start the command
			err := cmd.Start()
			if err != nil {
				t.Fatalf("Failed to start reviewtask command: %v", err)
			}

			// Wait for the specified delay
			time.Sleep(tt.delay)

			// Send the signal
			err = cmd.Process.Signal(tt.signal)
			if err != nil {
				t.Fatalf("Failed to send signal %v: %v", tt.signal, err)
			}

			// Wait for the process to exit
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			select {
			case err := <-done:
				// Process exited
				if err != nil {
					// Check if it's an expected signal-related exit
					var exitError *exec.ExitError
					if errors.As(err, &exitError) {
						// Signal-related exits are expected
						t.Logf("Process exited with signal: %v", exitError)
					} else {
						t.Logf("Process exited with error: %v", err)
					}
				} else {
					t.Log("Process exited cleanly")
				}

				// Log output for debugging
				if stdout.Len() > 0 {
					t.Logf("Stdout: %s", stdout.String())
				}
				if stderr.Len() > 0 {
					t.Logf("Stderr: %s", stderr.String())
				}

			case <-time.After(tt.maxWaitTime):
				// Process didn't exit in time - this is a failure
				t.Error("Process did not exit within the expected time after signal")

				// Force kill the process
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
			}
		})
	}
}

// TestDoubleCtrlCForceExit tests the double Ctrl-C force exit behavior
func TestDoubleCtrlCForceExit(t *testing.T) {
	// Skip in CI if TEST_SIGNAL_INTEGRATION is not set
	if os.Getenv("CI") == "true" && os.Getenv("TEST_SIGNAL_INTEGRATION") != "true" {
		t.Skip("Skipping signal integration test in CI (set TEST_SIGNAL_INTEGRATION=true to run)")
	}

	binaryPath := getBinaryPath(t)
	if binaryPath == "" {
		t.Skip("reviewtask binary not found, skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Start the reviewtask fetch command
	cmd := exec.CommandContext(ctx, binaryPath, "fetch")
	cmd.Dir = getTestRepoDir(t)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start reviewtask command: %v", err)
	}

	// Wait a moment for startup
	time.Sleep(1 * time.Second)

	// Send first SIGINT
	err = cmd.Process.Signal(os.Interrupt)
	if err != nil {
		t.Fatalf("Failed to send first SIGINT: %v", err)
	}

	// Wait briefly, then send second SIGINT
	time.Sleep(200 * time.Millisecond)
	err = cmd.Process.Signal(os.Interrupt)
	if err != nil {
		t.Logf("Second signal failed (process may have already exited): %v", err)
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process should exit quickly with double Ctrl-C
		t.Logf("Process exited after double Ctrl-C: %v", err)

		// Log output
		if stdout.Len() > 0 {
			t.Logf("Stdout: %s", stdout.String())
		}
		if stderr.Len() > 0 {
			t.Logf("Stderr: %s", stderr.String())
		}

	case <-time.After(5 * time.Second):
		t.Error("Process did not exit quickly after double Ctrl-C")
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
}

// TestCtrlCDuringDifferentPhases tests Ctrl-C during different execution phases
func TestCtrlCDuringDifferentPhases(t *testing.T) {
	// Skip in CI if TEST_SIGNAL_INTEGRATION is not set
	if os.Getenv("CI") == "true" && os.Getenv("TEST_SIGNAL_INTEGRATION") != "true" {
		t.Skip("Skipping signal integration test in CI (set TEST_SIGNAL_INTEGRATION=true to run)")
	}

	binaryPath := getBinaryPath(t)
	if binaryPath == "" {
		t.Skip("reviewtask binary not found, skipping integration test")
	}

	phases := []struct {
		name  string
		delay time.Duration // How long to wait before sending Ctrl-C
	}{
		{"early_phase", 500 * time.Millisecond},
		{"github_api_phase", 2 * time.Second},
		{"potential_ai_phase", 5 * time.Second},
	}

	for _, phase := range phases {
		t.Run(phase.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binaryPath, "fetch")
			cmd.Dir = getTestRepoDir(t)

			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Start()
			if err != nil {
				t.Fatalf("Failed to start command: %v", err)
			}

			// Wait for the specified phase
			time.Sleep(phase.delay)

			// Send Ctrl-C
			err = cmd.Process.Signal(os.Interrupt)
			if err != nil {
				t.Fatalf("Failed to send SIGINT: %v", err)
			}

			// Wait for exit
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			select {
			case err := <-done:
				t.Logf("Process exited during %s: %v", phase.name, err)

				// Verify output doesn't indicate hanging
				output := stdout.String() + stderr.String()
				if strings.Contains(output, "hang") || strings.Contains(output, "deadlock") {
					t.Errorf("Output suggests hanging: %s", output)
				}

			case <-time.After(8 * time.Second):
				t.Errorf("Process hung during %s phase", phase.name)
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
			}
		})
	}
}

// getBinaryPath finds the reviewtask binary for testing
func getBinaryPath(t testing.TB) string {
	t.Helper()

	// Try current directory first
	if _, err := os.Stat("./reviewtask"); err == nil {
		absPath, _ := filepath.Abs("./reviewtask")
		t.Logf("Found binary at: %s", absPath)
		return "./reviewtask"
	}

	// Try parent directory
	if _, err := os.Stat("../reviewtask"); err == nil {
		absPath, _ := filepath.Abs("../reviewtask")
		t.Logf("Found binary at: %s", absPath)
		return "../reviewtask"
	}

	// Try to build it from parent directory
	t.Log("Binary not found, attempting to build...")

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Logf("Failed to get working directory: %v", err)
		return ""
	}

	// If we're in the test directory, go to parent
	var parentDir string
	if filepath.Base(wd) == "test" {
		parentDir = filepath.Dir(wd)
	} else {
		parentDir = wd
	}

	// Build the binary in the current test directory
	buildPath := "./reviewtask"
	if filepath.Base(wd) == "test" {
		// We're in the test directory, build to current directory
		cmd := exec.Command("go", "build", "-o", buildPath, "..")
		cmd.Dir = wd

		if output, err := cmd.CombinedOutput(); err != nil {
			t.Logf("Failed to build reviewtask: %v", err)
			if len(output) > 0 {
				t.Logf("Build output: %s", output)
			}
			return ""
		}
	} else {
		// We're in the parent directory
		buildPath = "./test/reviewtask"
		cmd := exec.Command("go", "build", "-o", buildPath, ".")
		cmd.Dir = parentDir

		if output, err := cmd.CombinedOutput(); err != nil {
			t.Logf("Failed to build reviewtask: %v", err)
			if len(output) > 0 {
				t.Logf("Build output: %s", output)
			}
			return ""
		}
	}

	t.Log("Successfully built reviewtask binary")

	// Make the binary executable
	absPath, _ := filepath.Abs(buildPath)
	if err := os.Chmod(absPath, 0755); err != nil {
		t.Logf("Failed to make binary executable: %v", err)
	}

	// Log the built binary location
	t.Logf("Built binary at: %s (absolute: %s)", buildPath, absPath)

	// Verify the file exists
	if _, err := os.Stat(buildPath); err != nil {
		t.Logf("WARNING: Built binary not found at %s: %v", buildPath, err)
	}

	// Return the built path
	return buildPath
}

// getTestRepoDir returns a directory suitable for testing
func getTestRepoDir(t testing.TB) string {
	t.Helper()

	// Use current working directory or a test directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// If we're in the test directory, go up one level
	if filepath.Base(wd) == "test" {
		return filepath.Dir(wd)
	}

	return wd
}

// BenchmarkCtrlCResponseTime benchmarks how quickly Ctrl-C is handled
func BenchmarkCtrlCResponseTime(b *testing.B) {
	binaryPath := getBinaryPath(b)
	if binaryPath == "" {
		b.Skip("reviewtask binary not found")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		cmd := exec.Command(binaryPath, "fetch")
		cmd.Dir = getTestRepoDir(b)

		err := cmd.Start()
		if err != nil {
			b.Fatalf("Failed to start command: %v", err)
		}

		// Wait a brief moment for startup
		time.Sleep(100 * time.Millisecond)

		// Send Ctrl-C
		signalTime := time.Now()
		cmd.Process.Signal(os.Interrupt)

		// Wait for exit
		cmd.Wait()
		exitTime := time.Now()

		responseTime := exitTime.Sub(signalTime)
		b.Logf("Response time for iteration %d: %v", i, responseTime)

		totalTime := exitTime.Sub(start)
		b.Logf("Total time for iteration %d: %v", i, totalTime)
	}
}

// TestProcessCleanupAfterCtrlC verifies proper cleanup after Ctrl-C
func TestProcessCleanupAfterCtrlC(t *testing.T) {
	// Skip in CI if TEST_SIGNAL_INTEGRATION is not set
	if os.Getenv("CI") == "true" && os.Getenv("TEST_SIGNAL_INTEGRATION") != "true" {
		t.Skip("Skipping signal integration test in CI (set TEST_SIGNAL_INTEGRATION=true to run)")
	}

	binaryPath := getBinaryPath(t)
	if binaryPath == "" {
		t.Skip("reviewtask binary not found, skipping integration test")
	}

	// Start the process
	cmd := exec.Command(binaryPath, "fetch")
	cmd.Dir = getTestRepoDir(t)

	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	originalPID := cmd.Process.Pid

	// Wait for startup
	time.Sleep(1 * time.Second)

	// Send Ctrl-C
	err = cmd.Process.Signal(os.Interrupt)
	if err != nil {
		t.Fatalf("Failed to send SIGINT: %v", err)
	}

	// Wait for process to exit
	cmd.Wait()

	// Verify the process is actually gone
	time.Sleep(500 * time.Millisecond)

	// Try to signal the process - it should fail if the process is gone
	process, err := os.FindProcess(originalPID)
	if err == nil {
		err = process.Signal(syscall.Signal(0)) // Signal 0 tests if process exists
		if err == nil {
			t.Error("Process still exists after Ctrl-C termination")
		}
	}
}
