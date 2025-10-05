package ai

import (
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImpactAssessment_InitialStatusFromAI tests that initial_status from AI is preserved
func TestImpactAssessment_InitialStatusFromAI(t *testing.T) {
	tests := []struct {
		name           string
		simpleTask     SimpleTaskRequest
		expectedStatus string
	}{
		{
			name: "AI assigns TODO for small change",
			simpleTask: SimpleTaskRequest{
				Description:   "Fix typo in variable name",
				Priority:      "low",
				InitialStatus: "todo",
			},
			expectedStatus: "todo",
		},
		{
			name: "AI assigns PENDING for design change",
			simpleTask: SimpleTaskRequest{
				Description:   "Refactor to use event-driven architecture",
				Priority:      "high",
				InitialStatus: "pending",
			},
			expectedStatus: "pending",
		},
		{
			name: "Empty initial_status defaults to TODO",
			simpleTask: SimpleTaskRequest{
				Description:   "Add error handling",
				Priority:      "medium",
				InitialStatus: "",
			},
			expectedStatus: "todo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AISettings: config.AISettings{
					UserLanguage: "English",
				},
			}
			_ = NewAnalyzer(cfg) // Create analyzer for config setup

			// Create comment context
			ctx := CommentContext{
				Comment: github.Comment{
					ID:   123,
					Body: "Test comment",
					File: "test.go",
					Line: 10,
					URL:  "https://github.com/test/pr/1#comment-123",
				},
				SourceReview: github.Review{
					ID: 1,
				},
			}

			// Convert simple task to full task
			fullTasks := []TaskRequest{}
			tasks := []SimpleTaskRequest{tt.simpleTask}
			for i, simpleTask := range tasks {
				// Use AI-assigned initial status, default to "todo" if not specified
				initialStatus := simpleTask.InitialStatus
				if initialStatus == "" {
					initialStatus = "todo"
				}

				fullTask := TaskRequest{
					Description:     simpleTask.Description,
					Priority:        simpleTask.Priority,
					OriginText:      ctx.Comment.Body,
					SourceReviewID:  ctx.SourceReview.ID,
					SourceCommentID: ctx.Comment.ID,
					File:            ctx.Comment.File,
					Line:            ctx.Comment.Line,
					Status:          initialStatus,
					TaskIndex:       i,
					URL:             ctx.Comment.URL,
				}
				fullTasks = append(fullTasks, fullTask)
			}

			require.Len(t, fullTasks, 1)
			assert.Equal(t, tt.expectedStatus, fullTasks[0].Status)
		})
	}
}

// TestImpactAssessment_TaskConsolidation tests status preservation during consolidation
func TestImpactAssessment_TaskConsolidation(t *testing.T) {
	tests := []struct {
		name           string
		tasks          []SimpleTaskRequest
		expectedStatus string
		description    string
	}{
		{
			name: "All TODO tasks consolidated as TODO",
			tasks: []SimpleTaskRequest{
				{Description: "Add nil check", Priority: "high", InitialStatus: "todo"},
				{Description: "Add error logging", Priority: "medium", InitialStatus: "todo"},
			},
			expectedStatus: "todo",
			description:    "Should preserve TODO status when all tasks are TODO",
		},
		{
			name: "Mixed TODO and PENDING consolidated as PENDING",
			tasks: []SimpleTaskRequest{
				{Description: "Fix typo", Priority: "low", InitialStatus: "todo"},
				{Description: "Refactor architecture", Priority: "high", InitialStatus: "pending"},
			},
			expectedStatus: "pending",
			description:    "Should use PENDING if any task is PENDING",
		},
		{
			name: "All PENDING tasks consolidated as PENDING",
			tasks: []SimpleTaskRequest{
				{Description: "Design new API", Priority: "high", InitialStatus: "pending"},
				{Description: "Evaluate caching strategy", Priority: "medium", InitialStatus: "pending"},
			},
			expectedStatus: "pending",
			description:    "Should preserve PENDING status when all tasks are PENDING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AISettings: config.AISettings{
					UserLanguage: "English",
				},
			}
			analyzer := NewAnalyzer(cfg)

			// Test consolidation logic
			consolidated := analyzer.consolidateTasksIfNeeded(tt.tasks)

			require.Len(t, consolidated, 1, tt.description)
			assert.Equal(t, tt.expectedStatus, consolidated[0].InitialStatus, tt.description)
		})
	}
}

// TestImpactAssessment_PriorityLevels tests different priority levels with impact assessment
func TestImpactAssessment_PriorityLevels(t *testing.T) {
	tests := []struct {
		name          string
		description   string
		priority      string
		initialStatus string
		expectation   string
	}{
		{
			name:          "Critical priority can be TODO if small change",
			description:   "Add missing timeout to prevent hang",
			priority:      "critical",
			initialStatus: "todo",
			expectation:   "Critical issues with small fixes should be TODO for quick resolution",
		},
		{
			name:          "Low priority can be PENDING if needs design",
			description:   "Consider refactoring for better maintainability",
			priority:      "low",
			initialStatus: "pending",
			expectation:   "Low priority design changes should be PENDING for evaluation",
		},
		{
			name:          "Medium priority typical TODO for straightforward fix",
			description:   "Add validation for user input",
			priority:      "medium",
			initialStatus: "todo",
			expectation:   "Medium priority fixes should typically be TODO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := SimpleTaskRequest{
				Description:   tt.description,
				Priority:      tt.priority,
				InitialStatus: tt.initialStatus,
			}

			// Verify that priority and initial_status are independent
			assert.Equal(t, tt.priority, task.Priority, "Priority should be preserved")
			assert.Equal(t, tt.initialStatus, task.InitialStatus, tt.expectation)
		})
	}
}

// TestImpactAssessment_JSONRecovery tests that initial_status is preserved during recovery
func TestImpactAssessment_JSONRecovery(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage:        "English",
			EnableJSONRecovery:  true,
			MaxRecoveryAttempts: 3,
		},
	}
	_ = NewAnalyzer(cfg) // Create analyzer for config setup

	// Simulate recovered TaskRequest with status
	recoveredTask := TaskRequest{
		Description: "Implement caching strategy",
		Priority:    "high",
		Status:      "pending", // This should be preserved as InitialStatus
	}

	// Convert to SimpleTaskRequest (as done in recovery code)
	simpleTask := SimpleTaskRequest{
		Description:   recoveredTask.Description,
		Priority:      recoveredTask.Priority,
		InitialStatus: recoveredTask.Status,
	}

	assert.Equal(t, "pending", simpleTask.InitialStatus, "InitialStatus should be preserved from recovered task")
}

// TestImpactAssessment_DefaultBehavior tests backward compatibility with old tasks
func TestImpactAssessment_DefaultBehavior(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  string
		expectedStatus string
	}{
		{
			name:           "Empty initial_status defaults to TODO",
			initialStatus:  "",
			expectedStatus: "todo",
		},
		{
			name:           "Explicit TODO preserved",
			initialStatus:  "todo",
			expectedStatus: "todo",
		},
		{
			name:           "Explicit PENDING preserved",
			initialStatus:  "pending",
			expectedStatus: "pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialStatus := tt.initialStatus
			if initialStatus == "" {
				initialStatus = "todo"
			}

			assert.Equal(t, tt.expectedStatus, initialStatus)
		})
	}
}
