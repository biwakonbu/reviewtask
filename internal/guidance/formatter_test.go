package guidance

import (
	"strings"
	"testing"
)

func TestFormatter_Format(t *testing.T) {
	tests := []struct {
		name        string
		language    string
		guidance    *Guidance
		wantContain []string
	}{
		{
			name:     "basic formatting",
			language: "en",
			guidance: &Guidance{
				Title: "Next Steps",
				Steps: []Step{
					{
						Action:      "Do something",
						Command:     "reviewtask do",
						Description: "Does a thing",
					},
				},
			},
			wantContain: []string{"Next Steps", "Do something", "reviewtask do"},
		},
		{
			name:     "multiple steps",
			language: "en",
			guidance: &Guidance{
				Title: "Next Steps",
				Steps: []Step{
					{
						Action:      "First action",
						Command:     "reviewtask first",
						Description: "First desc",
					},
					{
						Action:      "Second action",
						Command:     "reviewtask second",
						Description: "Second desc",
					},
				},
			},
			wantContain: []string{"First action", "Second action", "reviewtask first", "reviewtask second"},
		},
		{
			name:     "step without command",
			language: "en",
			guidance: &Guidance{
				Title: "Next Steps",
				Steps: []Step{
					{
						Action:      "Just a message",
						Command:     "",
						Description: "",
					},
				},
			},
			wantContain: []string{"Just a message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.language)
			result := formatter.Format(tt.guidance)

			for _, want := range tt.wantContain {
				if !strings.Contains(result, want) {
					t.Errorf("Format() should contain %q, got:\n%s", want, result)
				}
			}
		})
	}
}

func TestFormatter_FormatCompact(t *testing.T) {
	formatter := NewFormatter("en")
	guidance := &Guidance{
		Title: "Next Steps",
		Steps: []Step{
			{
				Action:  "Do something",
				Command: "reviewtask do",
			},
			{
				Action:  "Do another",
				Command: "reviewtask another",
			},
		},
	}

	result := formatter.FormatCompact(guidance)

	// Should contain basic elements
	if !strings.Contains(result, "Next Steps") {
		t.Error("FormatCompact() should contain title")
	}
	if !strings.Contains(result, "Do something") {
		t.Error("FormatCompact() should contain action")
	}
	if !strings.Contains(result, "reviewtask do") {
		t.Error("FormatCompact() should contain command")
	}

	// Should have compact formatting
	if !strings.Contains(result, "→") {
		t.Error("FormatCompact() should use arrow symbol")
	}
}

func TestFormatter_FormatWithContext(t *testing.T) {
	formatter := NewFormatter("en")
	guidance := &Guidance{
		Title: "Next Steps",
		Steps: []Step{
			{
				Action:  "Do something",
				Command: "reviewtask do",
			},
		},
	}

	tests := []struct {
		name        string
		context     *Context
		wantContain []string
	}{
		{
			name: "context with todo tasks",
			context: &Context{
				TodoCount:    5,
				DoingCount:   2,
				PendingCount: 1,
				DoneCount:    3,
			},
			wantContain: []string{"Task Summary", "TODO", "DOING", "PENDING", "DONE"},
		},
		{
			name: "context without tasks",
			context: &Context{
				TodoCount:    0,
				DoingCount:   0,
				PendingCount: 0,
				DoneCount:    0,
			},
			wantContain: []string{"Next Steps", "Do something"},
		},
		{
			name: "context with only done tasks",
			context: &Context{
				DoneCount: 10,
			},
			wantContain: []string{"Next Steps"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatWithContext(guidance, tt.context)

			for _, want := range tt.wantContain {
				if !strings.Contains(result, want) {
					t.Errorf("FormatWithContext() should contain %q, got:\n%s", want, result)
				}
			}
		})
	}
}

func TestNewFormatter_DefaultLanguage(t *testing.T) {
	// Test with empty language
	formatter := NewFormatter("")
	if formatter.language != "en" {
		t.Errorf("NewFormatter(\"\") should default to en, got %s", formatter.language)
	}

	// Test with explicit language
	formatter = NewFormatter("ja")
	if formatter.language != "ja" {
		t.Errorf("NewFormatter(\"ja\") should set language to ja, got %s", formatter.language)
	}
}

func TestFormatter_EmptyGuidance(t *testing.T) {
	formatter := NewFormatter("en")
	guidance := &Guidance{
		Title: "Next Steps",
		Steps: []Step{},
	}

	result := formatter.Format(guidance)

	// Should still contain title even with no steps
	if !strings.Contains(result, "Next Steps") {
		t.Error("Format() should contain title even with empty steps")
	}
}

func TestFormatter_LongStepDescription(t *testing.T) {
	formatter := NewFormatter("en")
	longDesc := strings.Repeat("very long description ", 50)
	guidance := &Guidance{
		Title: "Next Steps",
		Steps: []Step{
			{
				Action:      "Do something complex",
				Command:     "reviewtask complex",
				Description: longDesc,
			},
		},
	}

	result := formatter.Format(guidance)

	// Should handle long descriptions without crashing
	if !strings.Contains(result, "Do something complex") {
		t.Error("Format() should handle long descriptions")
	}
	if !strings.Contains(result, "reviewtask complex") {
		t.Error("Format() should contain command even with long description")
	}
}
