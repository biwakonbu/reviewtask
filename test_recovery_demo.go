package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// MockFailingClaudeClient simulates the exact failure patterns from Issue #140
type MockFailingClaudeClient struct {
	CallCount int
}

func (m *MockFailingClaudeClient) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	m.CallCount++

	fmt.Printf("📡 Mock Claude API Call #%d\n", m.CallCount)
	fmt.Printf("📊 Input size: %d characters\n", len(input))

	// Simulate different failure patterns on different calls
	switch m.CallCount {
	case 1:
		// Simulate sudden truncation mid-JSON (the most common Issue #140 pattern)
		truncatedJSON := `[
			{
				"description": "Fix memory leak in HTTP client connection pool",
				"origin_text": "The HTTP client is not properly closing connections, causing memory leaks over time. This is especially problematic under high load.",
				"priority": "critical",
				"source_review_id": 123456,
				"source_comment_id": 789012,
				"file": "internal/client/http.go",
				"line": 45,
				"task_index": 0
			},
			{
				"description": "Add comprehensive error handling for database operations",
				"origin_text": "Database operations should have proper timeout and retry logic with exponential backoff.",
				"priority": "high",
				"source_review_id": 123456,
				"source_comment_id": 789013,
				"file": "internal/db/connection.go",
				"line": 78,
				"task_in` // Sudden truncation during field name

		// Return in Claude CLI wrapper format
		claudeResponse := fmt.Sprintf(`{
			"type": "code_execution_result",
			"subtype": "claude_execution",
			"is_error": false,
			"result": %q
		}`, truncatedJSON)

		fmt.Printf("🚨 Simulating Claude API truncation (Issue #140 pattern)\n")
		fmt.Printf("📄 Truncated response: %d chars\n", len(truncatedJSON))
		return claudeResponse, nil

	case 2:
		// Simulate successful retry with smaller prompt
		successJSON := `[
			{
				"description": "Fix memory leak in HTTP client connection pool",
				"origin_text": "The HTTP client is not properly closing connections, causing memory leaks over time. This is especially problematic under high load.",
				"priority": "critical",
				"source_review_id": 123456,
				"source_comment_id": 789012,
				"file": "internal/client/http.go",
				"line": 45,
				"task_index": 0
			}
		]`

		claudeResponse := fmt.Sprintf(`{
			"type": "code_execution_result",
			"subtype": "claude_execution",
			"is_error": false,
			"result": %q
		}`, successJSON)

		fmt.Printf("✅ Simulating successful retry with reduced prompt\n")
		return claudeResponse, nil

	default:
		claudeResponse := `{
			"type": "code_execution_result",
			"subtype": "claude_execution",
			"is_error": false,
			"result": "[]"
		}`
		return claudeResponse, nil
	}
}

func main() {
	fmt.Println("🧪 Testing Issue #140 Recovery Mechanism")
	fmt.Println("========================================")

	// Configure recovery mechanism
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        true,
			MaxRetries:         3,
		},
	}

	// Create analyzer with failing mock client
	mockClient := &MockFailingClaudeClient{}
	analyzer := ai.NewAnalyzerWithClient(cfg, mockClient)

	// Create test reviews that simulate the problematic scenario from Issue #140
	problemReviews := []github.Review{
		{
			ID:       123456,
			Reviewer: "senior-developer",
			Comments: []github.Comment{
				{
					ID:     789012,
					File:   "internal/client/http.go",
					Line:   45,
					Body:   "The HTTP client is not properly closing connections, causing memory leaks over time. This is especially problematic under high load.",
					Author: "senior-developer",
				},
				{
					ID:     789013,
					File:   "internal/db/connection.go",
					Line:   78,
					Body:   "Database operations should have proper timeout and retry logic with exponential backoff.",
					Author: "senior-developer",
				},
			},
		},
	}

	fmt.Printf("\n📋 Processing %d review comments...\n", len(problemReviews[0].Comments))

	// This would have failed completely in the original Issue #140
	// But should now succeed with recovery mechanism
	fmt.Println("\n🔧 Attempting task generation (this would fail without recovery)...")
	tasks, err := analyzer.GenerateTasks(problemReviews)

	if err != nil {
		log.Fatalf("❌ Task generation failed even with recovery: %v", err)
	}

	if len(tasks) == 0 {
		log.Fatalf("❌ No tasks generated - recovery mechanism failed")
	}

	// Success! The recovery mechanism worked
	fmt.Printf("\n🎉 SUCCESS: Recovery mechanism worked!\n")
	fmt.Printf("📊 Generated %d tasks from problematic Claude API responses\n", len(tasks))
	fmt.Printf("🔄 Mock API calls made: %d (shows retry mechanism worked)\n", mockClient.CallCount)

	fmt.Println("\n📝 Generated Tasks:")
	for i, task := range tasks {
		fmt.Printf("  %d. %s (Priority: %s)\n", i+1, task.Description, task.Priority)
		fmt.Printf("     File: %s:%d\n", task.File, task.Line)
		fmt.Printf("     Origin: %s\n", truncateString(task.OriginText, 60))
		fmt.Println()
	}

	fmt.Println("✅ Issue #140 recovery mechanism verification complete!")
	fmt.Println("📈 The system successfully recovered from Claude API failures that")
	fmt.Println("   would have previously caused complete task generation failure.")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return strings.TrimSpace(s[:maxLen]) + "..."
}
