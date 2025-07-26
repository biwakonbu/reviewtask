package progress

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTracker(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout

	t.Run("TTY environment", func(t *testing.T) {
		// Restore stdout after test
		defer func() { os.Stdout = oldStdout }()

		// Create a pipe to simulate TTY
		r, w, _ := os.Pipe()
		os.Stdout = w

		tracker := NewTracker()
		assert.NotNil(t, tracker)
		assert.NotNil(t, tracker.done)

		// Clean up
		w.Close()
		r.Close()
	})

	t.Run("Non-TTY environment", func(t *testing.T) {
		// Non-TTY is the default in test environment
		tracker := NewTracker()
		assert.NotNil(t, tracker)
		assert.False(t, tracker.isTTY)
		assert.NotNil(t, tracker.done)
	})
}

func TestTrackerNonTTY(t *testing.T) {
	// Capture stdout
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewTracker()
	// Force non-TTY mode for testing
	tracker.isTTY = false

	// Test GitHub progress
	tracker.SetGitHubProgress(1, 2)

	// Test Analysis progress
	tracker.SetAnalysisProgress(5, 10)

	// Test Saving progress
	tracker.SetSavingProgress(2, 2)

	// Test stage status
	tracker.SetStageStatus("github", "completed")

	// Test statistics
	tracker.UpdateStatistics(5, 10, 3, "Processing review")

	// Close writer and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output synchronously
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output
	assert.Contains(t, output, "GitHub API: 1/2")
	assert.Contains(t, output, "AI Analysis: 5/10")
	assert.Contains(t, output, "Saving Data: 2/2")
	assert.Contains(t, output, "github: completed")
	assert.Contains(t, output, "Processing: Processing review")
}

func TestTrackerStartStop(t *testing.T) {
	t.Run("Non-TTY Start/Stop", func(t *testing.T) {
		tracker := &Tracker{
			isTTY: false,
			done:  make(chan struct{}),
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Start should return immediately for non-TTY
		err := tracker.Start(ctx)
		assert.NoError(t, err)

		// Stop should also complete immediately
		tracker.Stop()

		cancel()
	})
}

func TestOnProgress(t *testing.T) {
	// Capture stdout for non-TTY test
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewTracker()
	// Force non-TTY mode for testing
	tracker.isTTY = false

	// Call OnProgress
	tracker.OnProgress(3, 10)

	// Close writer and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output synchronously
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify both SetAnalysisProgress and UpdateStatistics were called
	assert.Contains(t, output, "AI Analysis: 3/10")
	assert.Contains(t, output, "Processing: Processing comment 3/10")
}

func TestTrackerMethodsWithNilProgram(t *testing.T) {
	// Test that methods handle nil program gracefully
	tracker := &Tracker{
		isTTY:   true,
		program: nil, // Simulate uninitialized program
		done:    make(chan struct{}),
	}

	// These should not panic
	assert.NotPanics(t, func() {
		tracker.SetGitHubProgress(1, 2)
		tracker.SetAnalysisProgress(5, 10)
		tracker.SetSavingProgress(2, 2)
		tracker.SetStageStatus("github", "completed")
		tracker.UpdateStatistics(5, 10, 3, "Test")
		tracker.OnProgress(3, 10)
	})
}

func TestConcurrentAccess(t *testing.T) {
	tracker := NewTracker()
	// Force non-TTY mode for testing
	tracker.isTTY = false

	// Test concurrent access doesn't cause issues
	done := make(chan bool)

	// Multiple goroutines updating progress
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				tracker.SetGitHubProgress(j, 10)
				tracker.SetAnalysisProgress(j, 10)
				tracker.UpdateStatistics(j, 10, j/2, "Concurrent test")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Should complete without deadlock or panic
	assert.True(t, true)
}

func TestProgressPercentageCalculation(t *testing.T) {
	// Test UpdateProgress percentage calculation
	tests := []struct {
		current int
		total   int
		want    float64
	}{
		{0, 10, 0.0},
		{5, 10, 0.5},
		{10, 10, 1.0},
		{0, 0, 0.0}, // Division by zero case
	}

	for _, tt := range tests {
		cmd := UpdateProgress("test", tt.current, tt.total)
		msg := cmd()
		progressMsg, ok := msg.(progressMsg)
		assert.True(t, ok)
		assert.Equal(t, tt.want, progressMsg.percentage)
	}
}

func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestTrackerOutput(t *testing.T) {
	t.Run("GitHub Progress Output", func(t *testing.T) {
		output := captureOutput(func() {
			tracker := NewTracker()
			tracker.isTTY = false
			tracker.SetGitHubProgress(1, 2)
		})

		assert.True(t, strings.Contains(output, "GitHub API: 1/2"))
	})

	t.Run("Empty Progress Output", func(t *testing.T) {
		output := captureOutput(func() {
			tracker := NewTracker()
			tracker.isTTY = false
			tracker.SetGitHubProgress(0, 0)
		})

		// Should not output anything for 0/0 progress
		assert.Empty(t, strings.TrimSpace(output))
	})
}
