package progress

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"reviewtask/internal/ui"
)

// Tracker provides a simple interface for updating progress from the fetch command
type Tracker struct {
	program *tea.Program
	model   Model
	mu      sync.Mutex
	done    chan struct{}
	isTTY   bool
	console *ui.Console
}

// NewTracker creates a new progress tracker
func NewTracker() *Tracker {
	isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	console := ui.NewConsole()

	if !isTTY {
		// Return a no-op tracker for non-TTY environments
		return &Tracker{
			isTTY:   false,
			done:    make(chan struct{}),
			console: console,
		}
	}

	model := New()
	program := tea.NewProgram(model)

	return &Tracker{
		program: program,
		model:   model,
		isTTY:   true,
		done:    make(chan struct{}),
		console: console,
	}
}

// NewTrackerForTesting creates a new progress tracker configured for testing
func NewTrackerForTesting(isTTY bool) *Tracker {
	console := ui.NewConsole()

	if !isTTY {
		return &Tracker{
			isTTY:   false,
			done:    make(chan struct{}),
			console: console,
		}
	}

	model := New()
	program := tea.NewProgram(model)

	return &Tracker{
		program: program,
		model:   model,
		isTTY:   true,
		done:    make(chan struct{}),
		console: console,
	}
}

// Start begins the progress display
func (t *Tracker) Start(ctx context.Context) error {
	if !t.isTTY {
		return nil
	}

	// Enable progress mode and buffering for synchronized output
	t.console.SetProgressActive(true)
	t.console.SetBufferEnabled(true)

	go func() {
		finalModel, err := t.program.Run()
		if err != nil {
			t.console.WriteWithSync(func(w io.Writer) {
				fmt.Fprintf(w, "Error running progress tracker: %v\n", err)
			})
		}
		close(t.done)

		// Check if user interrupted with Ctrl-C
		if model, ok := finalModel.(Model); ok && model.interrupted {
			// User pressed Ctrl-C, terminate the entire process immediately
			os.Exit(0)
		}
	}()

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		t.program.Quit()
	}()

	return nil
}

// Stop stops the progress display
func (t *Tracker) Stop() {
	if !t.isTTY {
		return
	}

	if t.program != nil {
		t.program.Quit()

		// Wait for program to finish with timeout to prevent hanging
		select {
		case <-t.done:
			// Program finished normally
		case <-time.After(500 * time.Millisecond):
			// Timeout - force exit without waiting
		}
	}

	// Disable progress mode and flush any buffered messages
	t.console.SetProgressActive(false)
}

// SetGitHubProgress updates GitHub API progress
func (t *Tracker) SetGitHubProgress(current, total int) {
	if !t.isTTY {
		if total > 0 {
			t.console.Printf("GitHub API: %d/%d\n", current, total)
		}
		return
	}

	if t.program != nil {
		t.program.Send(UpdateProgress("github", current, total))
	}
}

// SetAnalysisProgress updates AI analysis progress
func (t *Tracker) SetAnalysisProgress(current, total int) {
	if !t.isTTY {
		if total > 0 {
			t.console.Printf("AI Analysis: %d/%d\n", current, total)
		}
		return
	}

	if t.program != nil {
		t.program.Send(UpdateProgress("analysis", current, total))
	}
}

// SetSavingProgress updates data saving progress
func (t *Tracker) SetSavingProgress(current, total int) {
	if !t.isTTY {
		if total > 0 {
			t.console.Printf("Saving Data: %d/%d\n", current, total)
		}
		return
	}

	if t.program != nil {
		t.program.Send(UpdateProgress("saving", current, total))
	}
}

// SetStageStatus updates the status of a stage
func (t *Tracker) SetStageStatus(stage, status string) {
	if !t.isTTY {
		t.console.Printf("%s: %s\n", stage, status)
		return
	}

	if t.program != nil {
		t.program.Send(UpdateStatus(stage, status))
	}
}

// UpdateStatistics updates real-time statistics
func (t *Tracker) UpdateStatistics(commentsProcessed, totalComments, tasksGenerated int, currentOp string) {
	if !t.isTTY {
		if currentOp != "" {
			t.console.Printf("Processing: %s\n", currentOp)
		}
		return
	}

	if t.program != nil {
		stats := Statistics{
			CommentsProcessed: commentsProcessed,
			TotalComments:     totalComments,
			TasksGenerated:    tasksGenerated,
			CurrentOperation:  currentOp,
		}
		t.program.Send(UpdateStats(stats))
	}
}

// Simple progress callback for existing code
func (t *Tracker) OnProgress(processed, total int) {
	t.SetAnalysisProgress(processed, total)
	t.UpdateStatistics(processed, total, 0, fmt.Sprintf("Processing comment %d/%d", processed, total))
}

// AddError adds an error message to the progress display queue
func (t *Tracker) AddError(message string) {
	if !t.isTTY {
		t.console.Printf("⚠️  %s\n", message)
		return
	}

	if t.program != nil {
		t.program.Send(AddError(message))
	}
}
