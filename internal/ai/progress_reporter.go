package ai

import (
	"sync"
	"sync/atomic"
)

// ProcessingStep represents a single step in comment processing
type ProcessingStep string

const (
	StepPreparePrompt   ProcessingStep = "prepare_prompt"
	StepCallClaude      ProcessingStep = "call_claude"
	StepParseResponse   ProcessingStep = "parse_response"
	StepValidateFormat  ProcessingStep = "validate_format"
	StepValidateContent ProcessingStep = "validate_content"
	StepDeduplication   ProcessingStep = "deduplication"
	StepComplete        ProcessingStep = "complete"
)

// StepWeight defines the relative weight of each step for progress calculation
var StepWeights = map[ProcessingStep]int{
	StepPreparePrompt:   5,  // 5% of total work
	StepCallClaude:      70, // 70% of total work (most time-consuming)
	StepParseResponse:   5,  // 5% of total work
	StepValidateFormat:  5,  // 5% of total work
	StepValidateContent: 10, // 10% of total work
	StepDeduplication:   5,  // 5% of total work
}

// ProgressReporter manages fine-grained progress reporting
type ProgressReporter struct {
	totalComments    int
	totalSteps       int
	completedSteps   int32 // Use atomic for thread-safe updates
	commentProgress  map[int]*CommentProgress
	mu               sync.RWMutex
	onProgressUpdate func(current, total int)
}

// CommentProgress tracks progress for a single comment
type CommentProgress struct {
	CommentID      int
	CurrentStep    ProcessingStep
	CompletedSteps []ProcessingStep
	IsComplete     bool
}

// NewProgressReporter creates a new progress reporter
func NewProgressReporter(totalComments int, onProgressUpdate func(current, total int)) *ProgressReporter {
	// Calculate total steps based on comments and step weights
	totalSteps := 0
	for _, weight := range StepWeights {
		totalSteps += weight * totalComments
	}

	pr := &ProgressReporter{
		totalComments:    totalComments,
		totalSteps:       totalSteps,
		commentProgress:  make(map[int]*CommentProgress),
		onProgressUpdate: onProgressUpdate,
	}

	// Initialize comment progress
	for i := 0; i < totalComments; i++ {
		pr.commentProgress[i] = &CommentProgress{
			CommentID:      i,
			CompletedSteps: []ProcessingStep{},
		}
	}

	return pr
}

// ReportStepProgress reports progress for a specific step of a comment
func (pr *ProgressReporter) ReportStepProgress(commentIndex int, step ProcessingStep) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if cp, exists := pr.commentProgress[commentIndex]; exists {
		// Check if step was already completed
		for _, completedStep := range cp.CompletedSteps {
			if completedStep == step {
				return // Already reported
			}
		}

		cp.CurrentStep = step
		cp.CompletedSteps = append(cp.CompletedSteps, step)

		// Update completed steps count
		weight := StepWeights[step]
		newCompleted := atomic.AddInt32(&pr.completedSteps, int32(weight))

		// Call progress update callback
		if pr.onProgressUpdate != nil {
			pr.onProgressUpdate(int(newCompleted), pr.totalSteps)
		}
	}
}

// ReportCommentComplete marks a comment as fully processed
func (pr *ProgressReporter) ReportCommentComplete(commentIndex int) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if cp, exists := pr.commentProgress[commentIndex]; exists {
		cp.IsComplete = true
		cp.CurrentStep = StepComplete
	}
}

// GetProgress returns current progress as a percentage
func (pr *ProgressReporter) GetProgress() float64 {
	completed := atomic.LoadInt32(&pr.completedSteps)
	if pr.totalSteps == 0 {
		return 0
	}
	return float64(completed) / float64(pr.totalSteps) * 100
}

// GetCommentProgress returns progress for a specific comment
func (pr *ProgressReporter) GetCommentProgress(commentIndex int) *CommentProgress {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.commentProgress[commentIndex]
}
