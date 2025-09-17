package ai

import (
	"path/filepath"
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

func TestBuildSimpleCommentPrompt_Golden(t *testing.T) {
	cases := []struct {
		name     string
		language string
		golden   string
	}{
		{
			name:     "english",
			language: "English",
			golden:   "testdata/prompts/simple/english.golden",
		},
		{
			name:     "japanese",
			language: "Japanese",
			golden:   "testdata/prompts/simple/japanese.golden",
		},
	}

	ctx := CommentContext{
		Comment: github.Comment{
			ID:     12345,
			File:   "internal/test.go",
			Line:   42,
			Body:   "This function lacks error handling. Please add nil check and error logging.",
			Author: "reviewer",
			URL:    "https://github.com/test/repo/pull/1#discussion_r12345",
		},
		SourceReview: github.Review{
			ID:       67890,
			Reviewer: "reviewer",
			State:    "CHANGES_REQUESTED",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				AISettings: config.AISettings{
					UserLanguage: tc.language,
				},
			}

			analyzer := NewAnalyzer(cfg)
			got := analyzer.buildSimpleCommentPrompt(ctx)

			goldenPath := filepath.Clean(tc.golden)
			if updateGoldenEnabled() {
				writeGolden(t, goldenPath, got)
				t.Logf("Updated golden file: %s", goldenPath)
			}

			want := loadGolden(t, goldenPath)
			if got != want {
				t.Fatalf("Prompt mismatch for language %s\n--- got ---\n%s\n--- want ---\n%s",
					tc.language, got, want)
			}
		})
	}
}

func TestBuildSimpleCommentPromptFromTemplate_Golden(t *testing.T) {
	// This test verifies that template-based prompts match expected output
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage: "English",
			VerboseMode:  false,
		},
	}

	ctx := CommentContext{
		Comment: github.Comment{
			ID:     98765,
			File:   "cmd/main.go",
			Line:   100,
			Body:   "Add validation for user input to prevent injection attacks.",
			Author: "security-reviewer",
			URL:    "https://github.com/test/repo/pull/10#discussion_r98765",
		},
		SourceReview: github.Review{
			ID:       54321,
			Reviewer: "security-reviewer",
			State:    "CHANGES_REQUESTED",
		},
	}

	analyzer := NewAnalyzer(cfg)
	got := analyzer.buildSimpleCommentPromptFromTemplate(ctx)

	goldenPath := "testdata/prompts/simple/template_based.golden"
	if updateGoldenEnabled() {
		writeGolden(t, goldenPath, got)
		t.Logf("Updated golden file: %s", goldenPath)
	}

	want := loadGolden(t, goldenPath)
	if got != want {
		t.Fatalf("Template-based prompt mismatch\n--- got ---\n%s\n--- want ---\n%s",
			got, want)
	}
}
