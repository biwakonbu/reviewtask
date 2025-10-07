package guidance

import (
	"strings"
	"testing"
)

func TestRuleSet_Apply(t *testing.T) {
	tests := []struct {
		name        string
		context     *Context
		wantRule    string // Name identifier for expected rule type
		wantSteps   int    // Expected number of steps
		wantContain string // String that should appear in guidance
	}{
		{
			name: "unresolved comments takes highest priority",
			context: &Context{
				HasUnresolvedComments: true,
				TodoCount:             5,
			},
			wantRule:    "unresolved",
			wantSteps:   1,
			wantContain: "unresolved",
		},
		{
			name: "all complete guidance",
			context: &Context{
				AllTasksComplete: true,
				DoneCount:        10,
			},
			wantRule:    "complete",
			wantSteps:   3,
			wantContain: "git push",
		},
		{
			name: "pending only guidance",
			context: &Context{
				TodoCount:    0,
				DoingCount:   0,
				PendingCount: 3,
			},
			wantRule:    "pending",
			wantSteps:   4,
			wantContain: "PENDING",
		},
		{
			name: "continue tasks guidance",
			context: &Context{
				TodoCount:    2,
				DoneCount:    3,
				NextTaskID:   "abc123",
				NextTaskDesc: "Fix bug",
			},
			wantRule:    "continue",
			wantSteps:   2,
			wantContain: "Continue",
		},
		{
			name: "start tasks guidance",
			context: &Context{
				TodoCount:       5,
				HasPendingTasks: true,
				PendingCount:    2,
			},
			wantRule:    "start",
			wantSteps:   4,
			wantContain: "TODO",
		},
		{
			name: "start tasks without pending",
			context: &Context{
				TodoCount:       3,
				HasPendingTasks: false,
			},
			wantRule:    "start",
			wantSteps:   2,
			wantContain: "TODO",
		},
		{
			name:        "default fallback",
			context:     &Context{},
			wantRule:    "default",
			wantSteps:   1,
			wantContain: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleSet := NewRuleSet()
			guidance := ruleSet.Apply(tt.context)

			if guidance == nil {
				t.Fatal("Apply() returned nil guidance")
			}

			if len(guidance.Steps) != tt.wantSteps {
				t.Errorf("Steps count = %d, want %d", len(guidance.Steps), tt.wantSteps)
			}

			formatted := guidance.Format()
			if !strings.Contains(strings.ToLower(formatted), strings.ToLower(tt.wantContain)) {
				t.Errorf("Guidance should contain %q, got:\n%s", tt.wantContain, formatted)
			}
		})
	}
}

func TestUnresolvedCommentsRule(t *testing.T) {
	rule := &UnresolvedCommentsRule{}

	// Test Matches
	ctx := &Context{HasUnresolvedComments: true}
	if !rule.Matches(ctx) {
		t.Error("UnresolvedCommentsRule should match when HasUnresolvedComments is true")
	}

	ctx = &Context{HasUnresolvedComments: false}
	if rule.Matches(ctx) {
		t.Error("UnresolvedCommentsRule should not match when HasUnresolvedComments is false")
	}

	// Test Priority
	if rule.Priority() != 100 {
		t.Errorf("Priority = %d, want 100", rule.Priority())
	}

	// Test Generate
	ctx = &Context{HasUnresolvedComments: true}
	guidance := rule.Generate(ctx)
	if guidance == nil {
		t.Fatal("Generate() returned nil")
	}
	if len(guidance.Steps) == 0 {
		t.Error("Generate() should return at least one step")
	}
}

func TestAllCompleteRule(t *testing.T) {
	rule := &AllCompleteRule{}

	// Test Matches
	ctx := &Context{AllTasksComplete: true}
	if !rule.Matches(ctx) {
		t.Error("AllCompleteRule should match when AllTasksComplete is true")
	}

	ctx = &Context{AllTasksComplete: false}
	if rule.Matches(ctx) {
		t.Error("AllCompleteRule should not match when AllTasksComplete is false")
	}

	// Test Priority
	if rule.Priority() != 90 {
		t.Errorf("Priority = %d, want 90", rule.Priority())
	}

	// Test Generate
	ctx = &Context{AllTasksComplete: true, DoneCount: 5}
	guidance := rule.Generate(ctx)
	if guidance == nil {
		t.Fatal("Generate() returned nil")
	}
	formatted := guidance.Format()
	if !strings.Contains(formatted, "git push") {
		t.Error("All complete guidance should suggest git push")
	}
}

func TestPendingOnlyRule(t *testing.T) {
	rule := &PendingOnlyRule{}

	// Test Matches - should match when only pending tasks remain
	ctx := &Context{
		TodoCount:    0,
		DoingCount:   0,
		PendingCount: 3,
	}
	if !rule.Matches(ctx) {
		t.Error("PendingOnlyRule should match when only pending tasks remain")
	}

	// Should not match if there are todo tasks
	ctx = &Context{
		TodoCount:    1,
		DoingCount:   0,
		PendingCount: 3,
	}
	if rule.Matches(ctx) {
		t.Error("PendingOnlyRule should not match when todo tasks exist")
	}

	// Test Priority
	if rule.Priority() != 80 {
		t.Errorf("Priority = %d, want 80", rule.Priority())
	}

	// Test Generate
	ctx = &Context{
		TodoCount:    0,
		DoingCount:   0,
		PendingCount: 5,
	}
	guidance := rule.Generate(ctx)
	if guidance == nil {
		t.Fatal("Generate() returned nil")
	}
	formatted := guidance.Format()
	if !strings.Contains(formatted, "PENDING") {
		t.Error("Pending only guidance should mention PENDING tasks")
	}
	if !strings.Contains(formatted, "5") {
		t.Error("Pending only guidance should show count")
	}
}

func TestContinueTasksRule(t *testing.T) {
	rule := &ContinueTasksRule{}

	// Test Matches - should match when there are done tasks and remaining work
	ctx := &Context{
		DoneCount:  3,
		TodoCount:  2,
		DoingCount: 0,
	}
	if !rule.Matches(ctx) {
		t.Error("ContinueTasksRule should match when done > 0 and work remains")
	}

	// Should not match if no done tasks
	ctx = &Context{
		DoneCount: 0,
		TodoCount: 5,
	}
	if rule.Matches(ctx) {
		t.Error("ContinueTasksRule should not match when no done tasks")
	}

	// Test Priority
	if rule.Priority() != 70 {
		t.Errorf("Priority = %d, want 70", rule.Priority())
	}

	// Test Generate with next task
	ctx = &Context{
		DoneCount:    3,
		TodoCount:    2,
		NextTaskID:   "task-123",
		NextTaskDesc: "Fix bug",
	}
	guidance := rule.Generate(ctx)
	if guidance == nil {
		t.Fatal("Generate() returned nil")
	}
	if len(guidance.Steps) < 2 {
		t.Error("Continue guidance with next task should have at least 2 steps")
	}
	formatted := guidance.Format()
	if !strings.Contains(formatted, "task-123") {
		t.Error("Continue guidance should include next task ID")
	}

	// Test Generate without next task
	ctx = &Context{
		DoneCount: 3,
		TodoCount: 2,
	}
	guidance = rule.Generate(ctx)
	if guidance == nil {
		t.Fatal("Generate() returned nil")
	}
	if len(guidance.Steps) != 1 {
		t.Errorf("Continue guidance without next task should have 1 step, got %d", len(guidance.Steps))
	}
}

func TestStartTasksRule(t *testing.T) {
	rule := &StartTasksRule{}

	// Test Matches
	ctx := &Context{TodoCount: 5}
	if !rule.Matches(ctx) {
		t.Error("StartTasksRule should match when todo tasks exist")
	}

	ctx = &Context{TodoCount: 0}
	if rule.Matches(ctx) {
		t.Error("StartTasksRule should not match when no todo tasks")
	}

	// Test Priority
	if rule.Priority() != 60 {
		t.Errorf("Priority = %d, want 60", rule.Priority())
	}

	// Test Generate with pending tasks
	ctx = &Context{
		TodoCount:       3,
		HasPendingTasks: true,
		PendingCount:    2,
	}
	guidance := rule.Generate(ctx)
	if guidance == nil {
		t.Fatal("Generate() returned nil")
	}
	if len(guidance.Steps) != 4 {
		t.Errorf("Start guidance with pending should have 4 steps, got %d", len(guidance.Steps))
	}
	formatted := guidance.Format()
	if !strings.Contains(formatted, "PENDING") {
		t.Error("Start guidance with pending should warn about pending tasks")
	}

	// Test Generate without pending tasks
	ctx = &Context{
		TodoCount:       3,
		HasPendingTasks: false,
	}
	guidance = rule.Generate(ctx)
	if guidance == nil {
		t.Fatal("Generate() returned nil")
	}
	if len(guidance.Steps) != 2 {
		t.Errorf("Start guidance without pending should have 2 steps, got %d", len(guidance.Steps))
	}
}

func TestDefaultRule(t *testing.T) {
	rule := &DefaultRule{}

	// Test Matches - should always match
	ctx := &Context{}
	if !rule.Matches(ctx) {
		t.Error("DefaultRule should always match")
	}

	// Test Priority
	if rule.Priority() != 0 {
		t.Errorf("Priority = %d, want 0 (lowest)", rule.Priority())
	}

	// Test Generate
	guidance := rule.Generate(ctx)
	if guidance == nil {
		t.Fatal("Generate() returned nil")
	}
	if len(guidance.Steps) == 0 {
		t.Error("Default guidance should have at least one step")
	}
	formatted := guidance.Format()
	if !strings.Contains(formatted, "status") {
		t.Error("Default guidance should suggest checking status")
	}
}

func TestRuleSet_AddRule(t *testing.T) {
	ruleSet := NewRuleSet()
	initialCount := len(ruleSet.rules)

	// Add a custom rule
	customRule := &DefaultRule{}
	ruleSet.AddRule(customRule)

	if len(ruleSet.rules) != initialCount+1 {
		t.Errorf("AddRule() should increase rule count, got %d want %d", len(ruleSet.rules), initialCount+1)
	}
}
