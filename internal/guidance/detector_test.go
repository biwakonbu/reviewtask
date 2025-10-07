package guidance

import (
	"os"
	"path/filepath"
	"testing"

	"reviewtask/internal/storage"
)

func TestDetector_DetectContext(t *testing.T) {
	// Create temporary directory for test storage
	tmpDir, err := os.MkdirTemp("", "guidance-detector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create storage directory structure
	storageDir := filepath.Join(tmpDir, ".pr-review", "PR-123")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		t.Fatalf("failed to create storage dir: %v", err)
	}

	// Initialize storage manager
	mgr := storage.NewManager()
	detector := NewDetector(mgr)

	tests := []struct {
		name       string
		setupTasks []storage.Task
		wantCounts struct {
			todo    int
			doing   int
			done    int
			pending int
			hold    int
		}
		wantFlags struct {
			hasPending      bool
			allComplete     bool
			hasUnresolved   bool
			hasNextTask     bool
			nextTaskNonZero bool
		}
	}{
		{
			name:       "empty state",
			setupTasks: []storage.Task{},
			wantCounts: struct {
				todo    int
				doing   int
				done    int
				pending int
				hold    int
			}{0, 0, 0, 0, 0},
			wantFlags: struct {
				hasPending      bool
				allComplete     bool
				hasUnresolved   bool
				hasNextTask     bool
				nextTaskNonZero bool
			}{false, false, false, false, false},
		},
		{
			name: "todo tasks only",
			setupTasks: []storage.Task{
				{ID: "task1", Status: StatusTodo, Description: "First task", PRNumber: 123},
				{ID: "task2", Status: StatusTodo, Description: "Second task", PRNumber: 123},
			},
			wantCounts: struct {
				todo    int
				doing   int
				done    int
				pending int
				hold    int
			}{2, 0, 0, 0, 0},
			wantFlags: struct {
				hasPending      bool
				allComplete     bool
				hasUnresolved   bool
				hasNextTask     bool
				nextTaskNonZero bool
			}{false, false, false, true, true},
		},
		{
			name: "all tasks done",
			setupTasks: []storage.Task{
				{ID: "task1", Status: StatusDone, Description: "Completed task", PRNumber: 123},
				{ID: "task2", Status: StatusDone, Description: "Another completed", PRNumber: 123},
			},
			wantCounts: struct {
				todo    int
				doing   int
				done    int
				pending int
				hold    int
			}{0, 0, 2, 0, 0},
			wantFlags: struct {
				hasPending      bool
				allComplete     bool
				hasUnresolved   bool
				hasNextTask     bool
				nextTaskNonZero bool
			}{false, true, false, false, false},
		},
		{
			name: "mixed statuses with pending",
			setupTasks: []storage.Task{
				{ID: "task1", Status: StatusDone, Description: "Done", PRNumber: 123},
				{ID: "task2", Status: StatusTodo, Description: "Todo", PRNumber: 123},
				{ID: "task3", Status: StatusPending, Description: "Pending", PRNumber: 123},
				{ID: "task4", Status: StatusDoing, Description: "Doing", PRNumber: 123},
			},
			wantCounts: struct {
				todo    int
				doing   int
				done    int
				pending int
				hold    int
			}{1, 1, 1, 1, 0},
			wantFlags: struct {
				hasPending      bool
				allComplete     bool
				hasUnresolved   bool
				hasNextTask     bool
				nextTaskNonZero bool
			}{true, false, false, true, true},
		},
		{
			name: "only pending tasks remain",
			setupTasks: []storage.Task{
				{ID: "task1", Status: StatusDone, Description: "Done", PRNumber: 123},
				{ID: "task2", Status: StatusPending, Description: "Pending", PRNumber: 123},
			},
			wantCounts: struct {
				todo    int
				doing   int
				done    int
				pending int
				hold    int
			}{0, 0, 1, 1, 0},
			wantFlags: struct {
				hasPending      bool
				allComplete     bool
				hasUnresolved   bool
				hasNextTask     bool
				nextTaskNonZero bool
			}{true, false, false, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup: Save tasks
			if len(tt.setupTasks) > 0 {
				if err := mgr.SaveTasks(123, tt.setupTasks); err != nil {
					t.Fatalf("failed to save tasks: %v", err)
				}
			} else {
				// Clean up any existing tasks
				if err := mgr.SaveTasks(123, []storage.Task{}); err != nil {
					t.Fatalf("failed to clean tasks: %v", err)
				}
			}

			// Execute
			ctx, err := detector.DetectContext()
			if err != nil {
				t.Fatalf("DetectContext() error = %v", err)
			}

			// Verify counts
			if ctx.TodoCount != tt.wantCounts.todo {
				t.Errorf("TodoCount = %d, want %d", ctx.TodoCount, tt.wantCounts.todo)
			}
			if ctx.DoingCount != tt.wantCounts.doing {
				t.Errorf("DoingCount = %d, want %d", ctx.DoingCount, tt.wantCounts.doing)
			}
			if ctx.DoneCount != tt.wantCounts.done {
				t.Errorf("DoneCount = %d, want %d", ctx.DoneCount, tt.wantCounts.done)
			}
			if ctx.PendingCount != tt.wantCounts.pending {
				t.Errorf("PendingCount = %d, want %d", ctx.PendingCount, tt.wantCounts.pending)
			}
			if ctx.HoldCount != tt.wantCounts.hold {
				t.Errorf("HoldCount = %d, want %d", ctx.HoldCount, tt.wantCounts.hold)
			}

			// Verify flags
			if ctx.HasPendingTasks != tt.wantFlags.hasPending {
				t.Errorf("HasPendingTasks = %v, want %v", ctx.HasPendingTasks, tt.wantFlags.hasPending)
			}
			if ctx.AllTasksComplete != tt.wantFlags.allComplete {
				t.Errorf("AllTasksComplete = %v, want %v", ctx.AllTasksComplete, tt.wantFlags.allComplete)
			}
			if ctx.HasUnresolvedComments != tt.wantFlags.hasUnresolved {
				t.Errorf("HasUnresolvedComments = %v, want %v", ctx.HasUnresolvedComments, tt.wantFlags.hasUnresolved)
			}

			// Verify next task
			hasNext := ctx.NextTaskID != ""
			if hasNext != tt.wantFlags.hasNextTask {
				t.Errorf("has next task = %v, want %v", hasNext, tt.wantFlags.hasNextTask)
			}
			if tt.wantFlags.nextTaskNonZero && ctx.NextTaskDesc == "" {
				t.Error("NextTaskDesc should be non-empty when next task exists")
			}
		})
	}
}

func TestDetector_DetectContextWithConfig(t *testing.T) {
	// Create temporary directory for test storage
	tmpDir, err := os.MkdirTemp("", "guidance-detector-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create storage directory structure
	storageDir := filepath.Join(tmpDir, ".pr-review", "PR-123")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		t.Fatalf("failed to create storage dir: %v", err)
	}

	mgr := storage.NewManager()
	detector := NewDetector(mgr)

	// Test with custom language
	ctx, err := detector.DetectContextWithConfig("ja")
	if err != nil {
		t.Fatalf("DetectContextWithConfig() error = %v", err)
	}

	if ctx.Language != "ja" {
		t.Errorf("Language = %s, want ja", ctx.Language)
	}

	// Test with empty language (should use default)
	ctx, err = detector.DetectContextWithConfig("")
	if err != nil {
		t.Fatalf("DetectContextWithConfig() error = %v", err)
	}

	if ctx.Language != "en" {
		t.Errorf("Language = %s, want en (default)", ctx.Language)
	}
}
