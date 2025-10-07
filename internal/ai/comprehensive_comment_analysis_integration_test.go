package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// TestComprehensiveCommentAnalysis_AllCommentsProcessed tests that ALL review comments
// are processed, including minor suggestions, nitpicks, and questions
func TestComprehensiveCommentAnalysis_AllCommentsProcessed(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:       "todo",
			LowPriorityPatterns: []string{"nit:", "minor:", "suggestion:"},
			LowPriorityStatus:   "pending",
		},
		AISettings: config.AISettings{
			UserLanguage:           "English",
			ProcessNitpickComments: true,
			DeduplicationEnabled:   false,
		},
	}

	mockClient := NewMockClaudeClient()

	// Setup mock responses with AI-assigned initial_status
	// Use unique substrings to avoid conflicts
	mockClient.Responses["processCount"] = `[{
		"description": "Fix typo in variable name",
		"priority": "low",
		"initial_status": "todo"
	}]`

	mockClient.Responses["better maintainability"] = `[{
		"description": "Consider refactoring this module for better maintainability",
		"priority": "medium",
		"initial_status": "pending"
	}]`

	mockClient.Responses["authentication approach"] = `[{
		"description": "Clarify design decision for authentication approach",
		"priority": "low",
		"initial_status": "todo"
	}]`

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "reviewer1",
			State:    "COMMENTED",
			Body:     "",
			Comments: []github.Comment{
				{
					ID:   101,
					Body: "nit: Fix typo in variable name 'processCount'",
					File: "main.go",
					Line: 42,
				},
				{
					ID:   102,
					Body: "Consider refactoring this module for better maintainability",
					File: "auth.go",
					Line: 100,
				},
				{
					ID:   103,
					Body: "Why did you choose this authentication approach?",
					File: "auth.go",
					Line: 50,
				},
			},
		},
	}

	tasks, err := analyzer.GenerateTasks(reviews)
	require.NoError(t, err)

	// Verify all comments generated tasks
	assert.Len(t, tasks, 3, "All comments should generate tasks")

	// Find each task by comment ID
	tasksByCommentID := make(map[int64]*struct {
		description string
		status      string
		priority    string
	})

	for _, task := range tasks {
		tasksByCommentID[task.SourceCommentID] = &struct {
			description string
			status      string
			priority    string
		}{
			description: task.Description,
			status:      task.Status,
			priority:    task.Priority,
		}
	}

	// Verify comment 101 (nitpick) - AI assigned TODO (small change)
	assert.Contains(t, tasksByCommentID, int64(101), "Nitpick comment should generate task")
	task101 := tasksByCommentID[101]
	assert.Equal(t, "todo", task101.status, "Small nitpick should be TODO")
	assert.Contains(t, task101.description, "typo", "Task should reference typo fix")

	// Verify comment 102 (refactoring suggestion) - AI assigned PENDING (design change)
	assert.Contains(t, tasksByCommentID, int64(102), "Refactoring suggestion should generate task")
	task102 := tasksByCommentID[102]
	assert.Equal(t, "pending", task102.status, "Refactoring should be PENDING")
	assert.Contains(t, task102.description, "refactoring", "Task should reference refactoring")

	// Verify comment 103 (question) - AI assigned TODO
	assert.Contains(t, tasksByCommentID, int64(103), "Question should generate task")
	task103 := tasksByCommentID[103]
	assert.Equal(t, "todo", task103.status, "Clarification question should be TODO")
	assert.Contains(t, task103.description, "Clarify", "Task should reference clarification")
}

// TestComprehensiveCommentAnalysis_ImpactAssessmentAccuracy tests AI impact assessment
func TestComprehensiveCommentAnalysis_ImpactAssessmentAccuracy(t *testing.T) {
	tests := []struct {
		name                string
		commentBody         string
		mockResponse        string
		expectedStatus      string
		expectedPriority    string
		impactJustification string
	}{
		{
			name:        "Small change - typo fix",
			commentBody: "Fix typo: 'recieve' should be 'receive'",
			mockResponse: `[{
				"description": "Fix typo: 'recieve' should be 'receive'",
				"priority": "low",
				"initial_status": "todo"
			}]`,
			expectedStatus:      "todo",
			expectedPriority:    "low",
			impactJustification: "< 50 lines, no architecture change",
		},
		{
			name:        "Medium change - add validation",
			commentBody: "Add input validation for email field",
			mockResponse: `[{
				"description": "Add input validation for email field",
				"priority": "medium",
				"initial_status": "todo"
			}]`,
			expectedStatus:      "todo",
			expectedPriority:    "medium",
			impactJustification: "< 50 lines, straightforward implementation",
		},
		{
			name:        "Large change - architecture refactor",
			commentBody: "Refactor to use event-driven architecture",
			mockResponse: `[{
				"description": "Refactor to use event-driven architecture",
				"priority": "high",
				"initial_status": "pending"
			}]`,
			expectedStatus:      "pending",
			expectedPriority:    "high",
			impactJustification: "> 50 lines, architecture change required",
		},
		{
			name:        "Large change - new feature",
			commentBody: "Implement caching layer with Redis",
			mockResponse: `[{
				"description": "Implement caching layer with Redis",
				"priority": "high",
				"initial_status": "pending"
			}]`,
			expectedStatus:      "pending",
			expectedPriority:    "high",
			impactJustification: "New dependency, significant implementation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TaskSettings: config.TaskSettings{
					DefaultStatus: "todo",
				},
				AISettings: config.AISettings{
					UserLanguage:         "English",
					DeduplicationEnabled: false,
				},
			}

			mockClient := NewMockClaudeClient()
			mockClient.Responses["default"] = tt.mockResponse

			analyzer := NewAnalyzerWithClient(cfg, mockClient)

			reviews := []github.Review{
				{
					ID:       1,
					Reviewer: "reviewer",
					State:    "COMMENTED",
					Comments: []github.Comment{
						{
							ID:   1,
							Body: tt.commentBody,
							File: "test.go",
							Line: 10,
						},
					},
				},
			}

			tasks, err := analyzer.GenerateTasks(reviews)
			require.NoError(t, err)
			require.Len(t, tasks, 1)

			task := tasks[0]
			assert.Equal(t, tt.expectedStatus, task.Status,
				"Impact assessment incorrect for: %s (%s)", tt.name, tt.impactJustification)
			assert.Equal(t, tt.expectedPriority, task.Priority,
				"Priority incorrect for: %s", tt.name)
		})
	}
}

// TestComprehensiveCommentAnalysis_FallbackBehavior tests pattern-based fallback
// when AI doesn't provide initial_status
func TestComprehensiveCommentAnalysis_FallbackBehavior(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:       "todo",
			LowPriorityPatterns: []string{"nit:", "minor:", "suggestion:"},
			LowPriorityStatus:   "pending",
		},
		AISettings: config.AISettings{
			UserLanguage:         "English",
			DeduplicationEnabled: false,
		},
	}

	mockClient := NewMockClaudeClient()

	// Mock response WITHOUT initial_status (tests fallback logic)
	mockClient.Responses["nit:"] = `[{
		"description": "Fix minor issue",
		"priority": "low"
	}]`

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "reviewer",
			State:    "COMMENTED",
			Comments: []github.Comment{
				{
					ID:   1,
					Body: "nit: Consider using const instead of var",
					File: "main.go",
					Line: 10,
				},
			},
		},
	}

	tasks, err := analyzer.GenerateTasks(reviews)
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// Should fall back to pattern-based detection (nit: → pending)
	assert.Equal(t, "pending", tasks[0].Status,
		"Should fall back to pattern-based detection when AI doesn't provide initial_status")
}
