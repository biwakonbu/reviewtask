package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptStdout_PRReview_Golden(t *testing.T) {
	root := NewRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"prompt", "stdout", "pr-review"})

	if err := root.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	got := buf.String()
	goldenPath := filepath.Join("testdata", "pr-review.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	// Normalize line endings and trim trailing whitespace for robust comparison
	gotN := strings.TrimSpace(strings.ReplaceAll(got, "\r\n", "\n"))
	wantN := strings.TrimSpace(strings.ReplaceAll(string(want), "\r\n", "\n"))
	if gotN != wantN {
		t.Fatalf("stdout prompt mismatch\n--- got ---\n%s\n--- want ---\n%s", gotN, wantN)
	}
}
