package ai

import (
	"os"
	"strings"
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

func TestSimpleTaskRequest_Structure(t *testing.T) {
	// Verify SimpleTaskRequest has only minimal fields
	task := SimpleTaskRequest{
		Description: "Test description",
		Priority:    "high",
	}

	if task.Description != "Test description" {
		t.Errorf("Expected description 'Test description', got %s", task.Description)
	}

	if task.Priority != "high" {
		t.Errorf("Expected priority 'high', got %s", task.Priority)
	}
}

func TestProcessCommentSimple(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage: "English",
			VerboseMode:  false,
		},
	}

	// Create a mock Claude client
	mockClient := &MockClaudeClient{
		ExecuteFunc: func(input, format string) (string, error) {
			// Return a simple JSON response
			return `[
				{"description": "Fix nil check in function", "priority": "high"},
				{"description": "Add error logging", "priority": "medium"}
			]`, nil
		},
	}

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	ctx := CommentContext{
		Comment: github.Comment{
			ID:     12345,
			File:   "test.go",
			Line:   42,
			Body:   "This function lacks error handling. Add nil check and error logging.",
			Author: "reviewer",
			URL:    "https://github.com/test/repo/pull/1#discussion_r12345",
		},
		SourceReview: github.Review{
			ID:       67890,
			Reviewer: "reviewer",
			State:    "CHANGES_REQUESTED",
		},
	}

	tasks, err := analyzer.processCommentSimple(ctx)
	if err != nil {
		t.Fatalf("processCommentSimple failed: %v", err)
	}

	// Verify we got 2 tasks
	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(tasks))
	}

	// Verify first task
	task1 := tasks[0]
	if task1.Description != "Fix nil check in function" {
		t.Errorf("Task 1 description mismatch: %s", task1.Description)
	}
	if task1.Priority != "high" {
		t.Errorf("Task 1 priority mismatch: %s", task1.Priority)
	}
	if task1.OriginText != ctx.Comment.Body {
		t.Errorf("Task 1 origin_text should be the original comment body")
	}
	if task1.SourceCommentID != ctx.Comment.ID {
		t.Errorf("Task 1 source_comment_id mismatch: %d", task1.SourceCommentID)
	}
	if task1.File != ctx.Comment.File {
		t.Errorf("Task 1 file mismatch: %s", task1.File)
	}
	if task1.Line != ctx.Comment.Line {
		t.Errorf("Task 1 line mismatch: %d", task1.Line)
	}
	if task1.URL != ctx.Comment.URL {
		t.Errorf("Task 1 URL mismatch: %s", task1.URL)
	}
	if task1.TaskIndex != 0 {
		t.Errorf("Task 1 index should be 0, got %d", task1.TaskIndex)
	}

	// Verify second task
	task2 := tasks[1]
	if task2.Description != "Add error logging" {
		t.Errorf("Task 2 description mismatch: %s", task2.Description)
	}
	if task2.Priority != "medium" {
		t.Errorf("Task 2 priority mismatch: %s", task2.Priority)
	}
	if task2.TaskIndex != 1 {
		t.Errorf("Task 2 index should be 1, got %d", task2.TaskIndex)
	}
}

func TestBuildSimpleCommentPrompt(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage: "Japanese",
		},
	}

	analyzer := NewAnalyzer(cfg)

	ctx := CommentContext{
		Comment: github.Comment{
			File:   "main.go",
			Line:   100,
			Body:   "Add validation here",
			Author: "alice",
		},
	}

	prompt := analyzer.buildSimpleCommentPrompt(ctx)

	// Check that prompt contains key elements
	if !strings.Contains(prompt, "Japanese") {
		t.Errorf("Prompt should mention Japanese language")
	}
	if !strings.Contains(prompt, "main.go:100") {
		t.Errorf("Prompt should contain file:line")
	}
	if !strings.Contains(prompt, "alice") {
		t.Errorf("Prompt should contain author")
	}
	if !strings.Contains(prompt, "Add validation here") {
		t.Errorf("Prompt should contain comment body")
	}
	if !strings.Contains(prompt, "```json") {
		t.Errorf("Prompt should have JSON code blocks")
	}
}

func TestBuildSimpleCommentPromptFromTemplate(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage: "English",
			VerboseMode:  false,
		},
	}

	analyzer := NewAnalyzer(cfg)

	ctx := CommentContext{
		Comment: github.Comment{
			File:   "test.go",
			Line:   42,
			Body:   "Fix this bug",
			Author: "bob",
		},
	}

	// Test with template file present
	prompt := analyzer.buildSimpleCommentPromptFromTemplate(ctx)

	// Should either load from template or fall back to hardcoded
	if prompt == "" {
		t.Errorf("Prompt should not be empty")
	}

	// Verify key content is present
	if !strings.Contains(prompt, "test.go") {
		t.Errorf("Prompt should contain file name")
	}
	if !strings.Contains(prompt, "42") {
		t.Errorf("Prompt should contain line number")
	}
	if !strings.Contains(prompt, "Fix this bug") {
		t.Errorf("Prompt should contain comment body")
	}
}

func TestCallClaudeForSimpleTasks(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			VerboseMode: false,
		},
	}

	testCases := []struct {
		name     string
		response string
		expected int
		wantErr  bool
	}{
		{
			name:     "Valid JSON array",
			response: `[{"description": "Task 1", "priority": "high"}]`,
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "JSON wrapped in code block",
			response: "```json\n[{\"description\": \"Task 1\", \"priority\": \"high\"}]\n```",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "JSON wrapped in response tags",
			response: "<response>[{\"description\": \"Task 1\", \"priority\": \"high\"}]</response>",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "JSON with prefix text",
			response: "Here's my response: [{\"description\": \"Task 1\", \"priority\": \"high\"}]",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "Empty array",
			response: "[]",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Invalid JSON",
			response: "Not JSON at all",
			expected: 0,
			wantErr:  false, // Returns empty array on parse error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockClaudeClient{
				Responses: map[string]string{
					"default": tc.response,
				},
			}

			analyzer := NewAnalyzerWithClient(cfg, mockClient)
			tasks, err := analyzer.callClaudeForSimpleTasks("test prompt")

			if tc.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(tasks) != tc.expected {
				t.Errorf("Expected %d tasks, got %d", tc.expected, len(tasks))
			}
		})
	}
}

func TestExtractJSONFromVariousFormats(t *testing.T) {
	analyzer := NewAnalyzer(&config.Config{})

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain JSON",
			input:    `[{"test": "value"}]`,
			expected: `[{"test": "value"}]`,
		},
		{
			name:     "Markdown code block with json",
			input:    "```json\n[{\"test\": \"value\"}]\n```",
			expected: `[{"test": "value"}]`,
		},
		{
			name:     "Markdown code block without language",
			input:    "```\n[{\"test\": \"value\"}]\n```",
			expected: `[{"test": "value"}]`,
		},
		{
			name:     "JSON with surrounding text",
			input:    "Here is the result:\n[{\"test\": \"value\"}]\nEnd of result",
			expected: `[{"test": "value"}]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.extractJSON(tc.input)
			if result != tc.expected {
				t.Errorf("extractJSON failed\nInput: %s\nExpected: %s\nGot: %s",
					tc.input, tc.expected, result)
			}
		})
	}
}

func TestLoadPromptTemplate(t *testing.T) {
	// Create a temporary template file for testing
	tmpDir := t.TempDir()
	templatePath := tmpDir + "/test_template.md"
	templateContent := `Test Template
{{.LanguageInstruction}}
File: {{.File}}:{{.Line}}
Author: {{.Author}}
Comment: {{.Comment}}`

	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	analyzer := NewAnalyzer(&config.Config{})

	data := struct {
		LanguageInstruction string
		File                string
		Line                int
		Author              string
		Comment             string
	}{
		LanguageInstruction: "Use English",
		File:                "test.go",
		Line:                42,
		Author:              "alice",
		Comment:             "Fix this",
	}

	// This will fail as it won't find the file in the default locations
	// but that's expected for this unit test
	result, err := analyzer.loadPromptTemplate("nonexistent.md", data)
	if err == nil {
		t.Errorf("Expected error for nonexistent template, got result: %s", result)
	}
}