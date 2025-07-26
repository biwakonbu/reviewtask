package progress

import (
	"context"
	"fmt"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

// Tracker provides a simple interface for updating progress from the fetch command
type Tracker struct {
	program *tea.Program
	model   Model
	mu      sync.Mutex
	done    chan struct{}
	isTTY   bool
}

// NewTracker creates a new progress tracker
func NewTracker() *Tracker {
	isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	if !isTTY {
		// Return a no-op tracker for non-TTY environments
		return &Tracker{
			isTTY: false,
			done:  make(chan struct{}),
		}
	}

	model := New()
	program := tea.NewProgram(model)

	return &Tracker{
		program: program,
		model:   model,
		isTTY:   true,
		done:    make(chan struct{}),
	}
}

// Start begins the progress display
func (t *Tracker) Start(ctx context.Context) error {
	if !t.isTTY {
		return nil
	}

	go func() {
		if _, err := t.program.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running progress tracker: %v\n", err)
		}
		close(t.done)
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
		<-t.done
	}
}

// SetGitHubProgress updates GitHub API progress
func (t *Tracker) SetGitHubProgress(current, total int) {
	if !t.isTTY {
		if total > 0 {
			fmt.Printf("GitHub API: %d/%d\n", current, total)
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
			fmt.Printf("AI Analysis: %d/%d\n", current, total)
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
			fmt.Printf("Saving Data: %d/%d\n", current, total)
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
		fmt.Printf("%s: %s\n", stage, status)
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
			fmt.Printf("Processing: %s\n", currentOp)
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
