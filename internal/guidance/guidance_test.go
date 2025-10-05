package guidance

import (
	"strings"
	"testing"
)

func TestContextGenerate_UnresolvedComments(t *testing.T) {
	ctx := &Context{
		HasUnresolvedComments: true,
		TodoCount:             5,
	}

	guidance := ctx.Generate()

	if guidance.Title != "Next Steps" {
		t.Errorf("Expected title 'Next Steps', got %q", guidance.Title)
	}

	if len(guidance.Steps) == 0 {
		t.Error("Expected at least one step for unresolved comments")
	}

	formatted := guidance.Format()
	if !strings.Contains(formatted, "unresolved") {
		t.Error("Guidance should mention unresolved comments")
	}
}

func TestContextGenerate_AllComplete(t *testing.T) {
	ctx := &Context{
		AllTasksComplete: true,
		DoneCount:        10,
	}

	guidance := ctx.Generate()

	formatted := guidance.Format()
	if !strings.Contains(formatted, "git push") {
		t.Error("All complete guidance should suggest git push")
	}
}

func TestContextGenerate_PendingTasks(t *testing.T) {
	ctx := &Context{
		TodoCount:    0,
		PendingCount: 3,
		DoneCount:    5,
	}

	guidance := ctx.Generate()

	formatted := guidance.Format()
	if !strings.Contains(formatted, "PENDING") {
		t.Error("Guidance should mention PENDING tasks")
	}
}

func TestContextGenerate_ContinueTasks(t *testing.T) {
	ctx := &Context{
		TodoCount:    3,
		DoneCount:    2,
		NextTaskID:   "abc123",
		NextTaskDesc: "Fix bug",
	}

	guidance := ctx.Generate()

	formatted := guidance.Format()
	if !strings.Contains(formatted, "Continue") || !strings.Contains(formatted, "next") {
		t.Error("Guidance should suggest continuing with tasks")
	}
	if !strings.Contains(formatted, ctx.NextTaskID) {
		t.Error("Guidance should include next task ID")
	}
}

func TestContextGenerate_StartTasks(t *testing.T) {
	ctx := &Context{
		TodoCount:       5,
		HasPendingTasks: true,
	}

	guidance := ctx.Generate()

	formatted := guidance.Format()
	if !strings.Contains(formatted, "TODO") {
		t.Error("Guidance should mention TODO tasks")
	}
	if !strings.Contains(formatted, "PENDING") {
		t.Error("Guidance should warn about PENDING tasks")
	}
}

func TestContextGenerate_DefaultLanguage(t *testing.T) {
	ctx := &Context{
		TodoCount: 1,
	}

	guidance := ctx.Generate()

	// Should not panic and should generate guidance
	if guidance == nil {
		t.Error("Generate() should return guidance even with default language")
	}
}

func TestGuidanceFormat(t *testing.T) {
	guidance := &Guidance{
		Title: "Next Steps",
		Steps: []Step{
			{
				Action:      "Do something",
				Command:     "reviewtask do",
				Description: "Does a thing",
			},
		},
	}

	formatted := guidance.Format()

	if !strings.Contains(formatted, "Next Steps") {
		t.Error("Formatted output should contain title")
	}
	if !strings.Contains(formatted, "Do something") {
		t.Error("Formatted output should contain action")
	}
	if !strings.Contains(formatted, "reviewtask do") {
		t.Error("Formatted output should contain command")
	}
}
