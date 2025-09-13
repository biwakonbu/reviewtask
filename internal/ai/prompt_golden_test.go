package ai

import (
	"os"
	"path/filepath"
	"testing"

	cfgpkg "reviewtask/internal/config"
	gh "reviewtask/internal/github"
)

// helper to write golden when UPDATE_GOLDEN=1
func updateGoldenEnabled() bool {
	return os.Getenv("UPDATE_GOLDEN") == "1"
}

func loadGolden(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden: %v", err)
	}
	return string(b)
}

func writeGolden(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to mkdir for golden: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write golden: %v", err)
	}
}

func basicConfigWithProfile(profile string) *cfgpkg.Config {
	return &cfgpkg.Config{
		PriorityRules: cfgpkg.PriorityRules{
			Critical: "Security vulnerabilities, authentication bypasses, data exposure risks",
			High:     "Performance bottlenecks, memory leaks, database optimization issues",
			Medium:   "Functional bugs, logic improvements, error handling",
			Low:      "Code style, naming conventions, comment improvements",
		},
		TaskSettings: cfgpkg.TaskSettings{
			DefaultStatus:       "todo",
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
			LowPriorityStatus:   "pending",
		},
		AISettings: cfgpkg.AISettings{
			UserLanguage:           "English",
			PromptProfile:          profile,
			ProcessNitpickComments: true,
			NitpickPriority:        "low",
			DeduplicationEnabled:   true,
		},
	}
}

func basicReviews() []gh.Review {
	return []gh.Review{
		{
			ID:          1,
			Reviewer:    "alice",
			State:       "APPROVED",
			Body:        "Looks good overall.",
			SubmittedAt: "2025-01-01T00:00:00Z",
			Comments: []gh.Comment{
				{ID: 101, File: "internal/foo.go", Line: 42, Body: "Please validate input length.", Author: "bob", CreatedAt: "2025-01-01T00:00:00Z"},
			},
		},
	}
}

func TestBuildAnalysisPrompt_Golden(t *testing.T) {
	cases := []struct {
		profile string
		golden  string
	}{
		{profile: "legacy", golden: "testdata/prompts/legacy/basic.golden"},
		{profile: "v2", golden: "testdata/prompts/v2/basic.golden"},
		{profile: "compact", golden: "testdata/prompts/compact/basic.golden"},
		{profile: "minimal", golden: "testdata/prompts/minimal/basic.golden"},
	}

	reviews := basicReviews()

	for _, tc := range cases {
		t.Run(tc.profile, func(t *testing.T) {
			cfg := basicConfigWithProfile(tc.profile)
			a := NewAnalyzerWithClient(cfg, nil)
			got := a.buildAnalysisPrompt(reviews)

			goldenPath := filepath.Clean(tc.golden)
			if updateGoldenEnabled() {
				writeGolden(t, goldenPath, got)
			}

			want := loadGolden(t, goldenPath)
			if got != want {
				t.Fatalf("prompt mismatch for profile %s\n--- got ---\n%s\n--- want ---\n%s", tc.profile, got, want)
			}
		})
	}
}
