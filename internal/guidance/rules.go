package guidance

import (
	"fmt"
	"reviewtask/internal/ui"
)

// Rule represents a guidance generation rule.
type Rule interface {
	// Matches returns true if this rule applies to the context.
	Matches(ctx *Context) bool
	// Priority returns the priority of this rule (higher = more important).
	Priority() int
	// Generate creates guidance for the context.
	Generate(ctx *Context) *Guidance
}

// RuleSet manages a collection of guidance rules.
type RuleSet struct {
	rules []Rule
}

// NewRuleSet creates a new rule set with default rules.
func NewRuleSet() *RuleSet {
	return &RuleSet{
		rules: []Rule{
			&UnresolvedCommentsRule{},
			&AllCompleteRule{},
			&PendingOnlyRule{},
			&ContinueTasksRule{},
			&StartTasksRule{},
			&DefaultRule{},
		},
	}
}

// AddRule adds a custom rule to the rule set.
func (rs *RuleSet) AddRule(rule Rule) {
	rs.rules = append(rs.rules, rule)
}

// Apply finds the best matching rule and generates guidance.
func (rs *RuleSet) Apply(ctx *Context) *Guidance {
	var bestRule Rule
	bestPriority := -1

	for _, rule := range rs.rules {
		if rule.Matches(ctx) && rule.Priority() > bestPriority {
			bestRule = rule
			bestPriority = rule.Priority()
		}
	}

	if bestRule != nil {
		return bestRule.Generate(ctx)
	}

	// Fallback to default
	defaultRule := &DefaultRule{}
	return defaultRule.Generate(ctx)
}

// UnresolvedCommentsRule handles unresolved review comments.
type UnresolvedCommentsRule struct{}

func (r *UnresolvedCommentsRule) Matches(ctx *Context) bool {
	return ctx.HasUnresolvedComments
}

func (r *UnresolvedCommentsRule) Priority() int {
	return 100 // Highest priority
}

func (r *UnresolvedCommentsRule) Generate(ctx *Context) *Guidance {
	return &Guidance{
		Title: "Next Steps",
		Steps: []Step{
			{
				Action:      ui.Warning("You have unresolved review comments"),
				Command:     "reviewtask",
				Description: "Analyze new comments and create tasks",
			},
		},
	}
}

// AllCompleteRule handles all tasks complete scenario.
type AllCompleteRule struct{}

func (r *AllCompleteRule) Matches(ctx *Context) bool {
	return ctx.AllTasksComplete
}

func (r *AllCompleteRule) Priority() int {
	return 90
}

func (r *AllCompleteRule) Generate(ctx *Context) *Guidance {
	return &Guidance{
		Title: "Next Steps",
		Steps: []Step{
			{
				Action:      "✓ All tasks completed",
				Command:     "",
				Description: "",
			},
			{
				Action:      "Push your changes",
				Command:     "git push",
				Description: "",
			},
			{
				Action:      "Check for new reviews",
				Command:     "reviewtask",
				Description: "",
			},
		},
	}
}

// PendingOnlyRule handles TODO complete but PENDING tasks remain.
type PendingOnlyRule struct{}

func (r *PendingOnlyRule) Matches(ctx *Context) bool {
	return ctx.TodoCount == 0 && ctx.DoingCount == 0 && ctx.PendingCount > 0
}

func (r *PendingOnlyRule) Priority() int {
	return 80
}

func (r *PendingOnlyRule) Generate(ctx *Context) *Guidance {
	message := fmt.Sprintf("! All TODO tasks completed\n\nYou have %d PENDING tasks requiring your decision:", ctx.PendingCount)

	return &Guidance{
		Title: "Next Steps",
		Steps: []Step{
			{
				Action:      ui.Warning(message),
				Command:     "",
				Description: "",
			},
			{
				Action:      "→ Review PENDING tasks",
				Command:     "reviewtask show <task-id>",
				Description: "View details and assess implementation effort",
			},
			{
				Action:      "→ Decide: start or cancel",
				Command:     "reviewtask start <task-id>",
				Description: "Start working on the task",
			},
			{
				Action:      "  or",
				Command:     "reviewtask cancel <task-id> --reason \"<explanation>\"",
				Description: "Skip this task with explanation",
			},
		},
	}
}

// ContinueTasksRule handles continuing work with existing progress.
type ContinueTasksRule struct{}

func (r *ContinueTasksRule) Matches(ctx *Context) bool {
	return ctx.DoneCount > 0 && (ctx.TodoCount > 0 || ctx.DoingCount > 0)
}

func (r *ContinueTasksRule) Priority() int {
	return 70
}

func (r *ContinueTasksRule) Generate(ctx *Context) *Guidance {
	steps := []Step{
		{
			Action:      "Continue with next task",
			Command:     "reviewtask show",
			Description: "See next recommended task",
		},
	}

	if ctx.NextTaskID != "" {
		steps = append(steps, Step{
			Action:      "Start immediately",
			Command:     fmt.Sprintf("reviewtask start %s", ctx.NextTaskID),
			Description: "",
		})
	}

	return &Guidance{
		Title: "Next Steps",
		Steps: steps,
	}
}

// StartTasksRule handles starting new tasks.
type StartTasksRule struct{}

func (r *StartTasksRule) Matches(ctx *Context) bool {
	return ctx.TodoCount > 0
}

func (r *StartTasksRule) Priority() int {
	return 60
}

func (r *StartTasksRule) Generate(ctx *Context) *Guidance {
	steps := []Step{
		{
			Action:      "Start working on TODO tasks",
			Command:     "reviewtask show",
			Description: "See next recommended task",
		},
		{
			Action:      "View all tasks",
			Command:     "reviewtask status",
			Description: "",
		},
	}

	if ctx.HasPendingTasks {
		steps = append(steps, Step{
			Action:      ui.Warning(fmt.Sprintf("! You have %d PENDING tasks requiring decision", ctx.PendingCount)),
			Command:     "",
			Description: "",
		})
		steps = append(steps, Step{
			Action:      "Review PENDING tasks after completing TODO tasks",
			Command:     "",
			Description: "",
		})
	}

	return &Guidance{
		Title: "Next Steps",
		Steps: steps,
	}
}

// DefaultRule provides fallback guidance.
type DefaultRule struct{}

func (r *DefaultRule) Matches(ctx *Context) bool {
	return true // Always matches as fallback
}

func (r *DefaultRule) Priority() int {
	return 0 // Lowest priority
}

func (r *DefaultRule) Generate(ctx *Context) *Guidance {
	return &Guidance{
		Title: "Next Steps",
		Steps: []Step{
			{
				Action:      "Check task status",
				Command:     "reviewtask status",
				Description: "",
			},
		},
	}
}
