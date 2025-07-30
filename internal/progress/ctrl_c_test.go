package progress

import (
	"context"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"reviewtask/internal/ui"
)

// TestProgressModelCtrlCHandling tests the Ctrl-C key handling in the progress model
func TestProgressModelCtrlCHandling(t *testing.T) {
	model := New()

	tests := []struct {
		name     string
		keyInput string
		expect   string
	}{
		{
			name:     "Ctrl-C should set interrupted flag and quit",
			keyInput: "ctrl+c",
			expect:   "quit",
		},
		{
			name:     "Other keys should not trigger quit",
			keyInput: "enter",
			expect:   "continue",
		},
		{
			name:     "Regular characters should not trigger quit",
			keyInput: "a",
			expect:   "continue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a key message
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune(tt.keyInput),
			}

			// Handle Ctrl-C specially since it's a different key type
			if tt.keyInput == "ctrl+c" {
				keyMsg = tea.KeyMsg{
					Type: tea.KeyCtrlC,
				}
			}

			// Update the model with the key message
			updatedModel, cmd := model.Update(keyMsg)

			// Cast back to our model type
			progressModel, ok := updatedModel.(Model)
			if !ok {
				t.Fatal("Model update returned unexpected type")
			}

			if tt.expect == "quit" {
				// Should have set interrupted flag
				if !progressModel.interrupted {
					t.Error("Expected interrupted flag to be set after Ctrl-C")
				}

				// Should have returned quit command
				if cmd == nil {
					t.Error("Expected quit command after Ctrl-C, got nil")
				}

				// Execute the command to verify it's a quit command
				if cmd != nil {
					msg := cmd()
					if quitMsg, ok := msg.(tea.QuitMsg); !ok {
						t.Errorf("Expected tea.QuitMsg, got %T", quitMsg)
					}
				}
			} else {
				// Should not have set interrupted flag
				if progressModel.interrupted {
					t.Error("Interrupted flag should not be set for non-Ctrl-C keys")
				}

				// Should not have returned quit command
				if cmd != nil {
					msg := cmd()
					if _, ok := msg.(tea.QuitMsg); ok {
						t.Error("Should not have returned quit command for non-Ctrl-C keys")
					}
				}
			}
		})
	}
}

// TestProgressModelInterruptedFlag tests that the interrupted flag works correctly
func TestProgressModelInterruptedFlag(t *testing.T) {
	model := New()

	// Initially, interrupted should be false
	if model.interrupted {
		t.Error("Model should not be interrupted initially")
	}

	// Simulate Ctrl-C
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, _ := model.Update(keyMsg)

	progressModel, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("Model update returned unexpected type")
	}

	// After Ctrl-C, interrupted should be true
	if !progressModel.interrupted {
		t.Error("Model should be interrupted after Ctrl-C")
	}
}

// TestProgressTrackerCtrlCExit tests the progress tracker's response to Ctrl-C
func TestProgressTrackerCtrlCExit(t *testing.T) {
	// Skip this test in CI environments where TTY is not available
	if !isTerminalAvailable() {
		t.Skip("Skipping TTY-dependent test in non-terminal environment")
	}

	tracker := NewTracker()

	// Create a context that can be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start the tracker (this will be a no-op if not TTY)
	err := tracker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}

	// The tracker should handle the context cancellation gracefully
	<-ctx.Done()

	// Stop should complete without hanging
	done := make(chan struct{})
	go func() {
		tracker.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success - Stop completed
	case <-time.After(1 * time.Second):
		t.Error("Tracker.Stop() hung and did not complete within timeout")
	}
}

// TestProgressTrackerNonTTY tests progress tracker behavior in non-TTY environments
func TestProgressTrackerNonTTY(t *testing.T) {
	// Create a tracker that thinks it's not in a TTY
	tracker := &Tracker{
		isTTY:   false,
		done:    make(chan struct{}),
		console: ui.NewConsole(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start should return immediately for non-TTY
	err := tracker.Start(ctx)
	if err != nil {
		t.Errorf("Start failed in non-TTY mode: %v", err)
	}

	// Stop should also return immediately
	tracker.Stop()

	// Should be able to call methods without panicking
	tracker.SetGitHubProgress(1, 2)
	tracker.SetAnalysisProgress(1, 2)
	tracker.UpdateStatistics(1, 2, 3, "test")
}

// isTerminalAvailable checks if we're running in a terminal environment
func isTerminalAvailable() bool {
	// Check if stdout is a terminal
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// BenchmarkProgressModelUpdate benchmarks the Update method performance
func BenchmarkProgressModelUpdate(t *testing.B) {
	model := New()
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		model.Update(keyMsg)
	}
}

// TestProgressModelMessageTypes tests various message types
func TestProgressModelMessageTypes(t *testing.T) {
	model := New()

	// Test window size message
	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, _ := model.Update(windowMsg)
	progressModel := updatedModel.(Model)

	if progressModel.width != 100 || progressModel.height != 50 {
		t.Errorf("Window size not updated correctly: got %dx%d, want 100x50",
			progressModel.width, progressModel.height)
	}

	// Test progress message
	progressMsg := UpdateProgress("github", 1, 2)
	updatedModel, _ = model.Update(progressMsg)
	progressModel = updatedModel.(Model)

	if progressModel.activeStage != "github" {
		t.Errorf("Active stage not updated: got %s, want github", progressModel.activeStage)
	}

	// Test status message
	statusMsg := UpdateStatus("analysis", "completed")
	updatedModel, _ = model.Update(statusMsg)
	progressModel = updatedModel.(Model)

	if stage, exists := progressModel.stages["analysis"]; exists {
		if stage.Status != "completed" {
			t.Errorf("Status not updated: got %s, want completed", stage.Status)
		}
	}

	// Test error message
	errorMsg := AddError("test error")
	updatedModel, _ = model.Update(errorMsg)
	progressModel = updatedModel.(Model)

	if len(progressModel.errorQueue) == 0 {
		t.Error("Error message was not added to queue")
	} else if progressModel.errorQueue[0] != "test error" {
		t.Errorf("Wrong error message: got %s, want 'test error'", progressModel.errorQueue[0])
	}
}
