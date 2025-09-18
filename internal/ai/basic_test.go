package ai

import (
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// TestBasicAnalyzerFunctionality tests the core analyzer functionality
// without requiring external dependencies like Claude Code CLI
func TestBasicAnalyzerFunctionality(t *testing.T) {
	t.Log("Testing basic AI analyzer functionality...")

	// Create basic configuration
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
		AISettings: config.AISettings{
			UserLanguage: "English",
			VerboseMode:  false,
		},
	}

	// Create analyzer
	analyzer := NewAnalyzer(cfg)

	// Test basic prompt generation
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "PENDING",
			Comments: []github.Comment{
				{
					ID:     1,
					Author: "reviewer1",
					Body:   "This function needs error handling",
					File:   "test.go",
					Line:   42,
				},
			},
		},
	}

	// Test prompt building (this should work without external dependencies)
	prompt := analyzer.buildAnalysisPrompt(reviews)
	if prompt == "" {
		t.Error("Generated prompt should not be empty")
	}

	t.Logf("✓ Prompt generation works (length: %d chars)", len(prompt))

	// Test UUID generation for tasks
	testTasks := []TaskRequest{
		{
			Description:     "Test task",
			OriginText:      "Test comment",
			Priority:        "high",
			SourceReviewID:  12345,
			SourceCommentID: 67890,
			File:            "test.go",
			Line:            42,
			Status:          "todo",
			TaskIndex:       0,
		},
	}

	storageTasks := analyzer.convertToStorageTasks(testTasks)
	if len(storageTasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(storageTasks))
	}

	task := storageTasks[0]
	if task.ID == "" {
		t.Error("Task ID should not be empty")
	}

	t.Logf("✓ Task conversion works (ID: %s)", task.ID)

	// Test low priority comment detection
	lowPriorityComment := "nit: Fix indentation"
	isLowPriority := analyzer.isLowPriorityComment(lowPriorityComment)
	if !isLowPriority {
		t.Error("Should detect nit: comments as low priority")
	}

	t.Log("✓ Low priority comment detection works")

	// Test CodeRabbit nitpick detection
	codeRabbitComment := `<details>
<summary>🧹 Nitpick comments (3)</summary>
<blockquote>
Some nitpick content here
</blockquote>
</details>`
	isCodeRabbitNitpick := analyzer.isLowPriorityComment(codeRabbitComment)
	if !isCodeRabbitNitpick {
		t.Error("Should detect CodeRabbit nitpick comments")
	}

	t.Log("✓ CodeRabbit nitpick detection works")

	t.Log("All basic tests passed!")
}

// TestAnalyzerConfiguration tests that the analyzer can be configured properly
func TestAnalyzerConfiguration(t *testing.T) {
	t.Log("Testing analyzer configuration...")

	// Test with different configurations
	configs := []*config.Config{
		{
			TaskSettings: config.TaskSettings{
				DefaultStatus: "todo",
			},
			AISettings: config.AISettings{
				UserLanguage: "English",
			},
		},
		{
			TaskSettings: config.TaskSettings{
				DefaultStatus: "pending",
			},
			AISettings: config.AISettings{
				UserLanguage: "Japanese",
			},
		},
	}

	for i, cfg := range configs {
		analyzer := NewAnalyzer(cfg)
		if analyzer == nil {
			t.Errorf("Config %d: Analyzer should not be nil", i)
		}
		t.Logf("✓ Config %d: Analyzer created successfully", i)
	}

	t.Log("All configuration tests passed!")
}

// TestPromptProfiles tests different prompt profiles
func TestPromptProfiles(t *testing.T) {
	t.Log("Testing prompt profiles...")

	profiles := []string{"legacy", "v2", "compact", "minimal"}

	for _, profile := range profiles {
		cfg := &config.Config{
			AISettings: config.AISettings{
				PromptProfile: profile,
				UserLanguage:  "English",
			},
		}

		analyzer := NewAnalyzer(cfg)
		if analyzer == nil {
			t.Errorf("Profile %s: Analyzer should not be nil", profile)
		}

		// Test that we can build a prompt with this profile
		reviews := []github.Review{
			{
				ID:       1,
				Reviewer: "test-reviewer",
				State:    "PENDING",
				Comments: []github.Comment{
					{
						ID:     1,
						Author: "reviewer1",
						Body:   "Test comment",
						File:   "test.go",
						Line:   42,
					},
				},
			},
		}

		prompt := analyzer.buildAnalysisPrompt(reviews)
		if prompt == "" {
			t.Errorf("Profile %s: Generated prompt should not be empty", profile)
		}

		t.Logf("✓ Profile %s: Prompt generation works", profile)
	}

	t.Log("All prompt profile tests passed!")
}
