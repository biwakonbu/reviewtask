package ui

import (
	"strings"
	"testing"
)

func TestSectionDivider(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "simple title",
			title:    "Next Steps",
			expected: "Next Steps\n──────────",
		},
		{
			name:     "long title",
			title:    "Implementation Progress",
			expected: "Implementation Progress\n───────────────────────",
		},
		{
			name:     "empty title",
			title:    "",
			expected: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SectionDivider(tt.title)
			if result != tt.expected {
				t.Errorf("SectionDivider(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

func TestSuccess(t *testing.T) {
	result := Success("Build passed")
	if !strings.Contains(result, SymbolSuccess) {
		t.Errorf("Success() should contain success symbol %s", SymbolSuccess)
	}
	if !strings.Contains(result, "Build passed") {
		t.Error("Success() should contain the message")
	}
}

func TestError(t *testing.T) {
	result := Error("Build failed")
	if !strings.Contains(result, SymbolError) {
		t.Errorf("Error() should contain error symbol %s", SymbolError)
	}
	if !strings.Contains(result, "Build failed") {
		t.Error("Error() should contain the message")
	}
}

func TestNext(t *testing.T) {
	result := Next("Run tests")
	if !strings.Contains(result, SymbolNext) {
		t.Errorf("Next() should contain next symbol %s", SymbolNext)
	}
	if !strings.Contains(result, "Run tests") {
		t.Error("Next() should contain the message")
	}
}

func TestWarning(t *testing.T) {
	result := Warning("Pending tasks exist")
	if !strings.Contains(result, SymbolWarning) {
		t.Errorf("Warning() should contain warning symbol %s", SymbolWarning)
	}
	if !strings.Contains(result, "Pending tasks exist") {
		t.Error("Warning() should contain the message")
	}
}

func TestIndent(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		spaces   int
		expected string
	}{
		{
			name:     "single line with 2 spaces",
			message:  "Hello",
			spaces:   2,
			expected: "  Hello",
		},
		{
			name:     "multiple lines with 4 spaces",
			message:  "Line 1\nLine 2",
			spaces:   4,
			expected: "    Line 1\n    Line 2",
		},
		{
			name:     "zero spaces",
			message:  "Test",
			spaces:   0,
			expected: "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Indent(tt.message, tt.spaces)
			if result != tt.expected {
				t.Errorf("Indent(%q, %d) = %q, want %q", tt.message, tt.spaces, result, tt.expected)
			}
		})
	}
}
