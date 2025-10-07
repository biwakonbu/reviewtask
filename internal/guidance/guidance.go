// Package guidance provides context-aware guidance for user workflow.
package guidance

import (
	"fmt"
	"reviewtask/internal/ui"
	"strings"
)

// Context represents the current workflow state.
type Context struct {
	// Task counts
	TodoCount    int
	DoingCount   int
	DoneCount    int
	PendingCount int
	HoldCount    int

	// State flags
	HasUnresolvedComments bool
	AllTasksComplete      bool
	HasPendingTasks       bool

	// Next suggested task
	NextTaskID   string
	NextTaskDesc string

	// Language
	Language string
}

// Step represents a suggested next step.
type Step struct {
	Action      string
	Command     string
	Description string
}

// Guidance represents a set of guidance steps.
type Guidance struct {
	Title string
	Steps []Step
}

// Generate creates context-appropriate guidance.
func (c *Context) Generate() *Guidance {
	if c.Language == "" {
		c.Language = "en"
	}

	// Determine the appropriate guidance pattern
	if c.HasUnresolvedComments {
		return c.unresolvedCommentsGuidance()
	}

	if c.AllTasksComplete {
		return c.allCompleteGuidance()
	}

	if c.TodoCount == 0 && c.PendingCount > 0 {
		return c.pendingTasksGuidance()
	}

	if c.DoneCount > 0 && (c.TodoCount > 0 || c.DoingCount > 0) {
		return c.continueTasksGuidance()
	}

	if c.TodoCount > 0 {
		return c.startTasksGuidance()
	}

	return c.defaultGuidance()
}

// Format formats the guidance for output.
func (g *Guidance) Format() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(ui.SectionDivider("Next Steps"))
	sb.WriteString("\n")

	for _, step := range g.Steps {
		sb.WriteString(ui.Next(step.Action))
		sb.WriteString("\n")
		if step.Command != "" {
			sb.WriteString(ui.Command(step.Command, step.Description))
			sb.WriteString("\n")
		}
		if step != g.Steps[len(g.Steps)-1] {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (c *Context) unresolvedCommentsGuidance() *Guidance {
	return &Guidance{
		Title: "Next Steps",
		Steps: []Step{
			{
				Action:      ui.Warning("You have unresolved review comments"),
				Command:     "reviewtask analyze",
				Description: "Analyze new comments and create tasks",
			},
		},
	}
}

func (c *Context) allCompleteGuidance() *Guidance {
	steps := []Step{
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
	}

	return &Guidance{
		Title: "Next Steps",
		Steps: steps,
	}
}

func (c *Context) pendingTasksGuidance() *Guidance {
	message := fmt.Sprintf("! All TODO tasks completed\n\nYou have %d PENDING tasks requiring your decision:", c.PendingCount)

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

func (c *Context) continueTasksGuidance() *Guidance {
	steps := []Step{
		{
			Action:      "Continue with next task",
			Command:     "reviewtask show",
			Description: "See next recommended task",
		},
	}

	if c.NextTaskID != "" {
		steps = append(steps, Step{
			Action:      "Start immediately",
			Command:     fmt.Sprintf("reviewtask start %s", c.NextTaskID),
			Description: "",
		})
	}

	return &Guidance{
		Title: "Next Steps",
		Steps: steps,
	}
}

func (c *Context) startTasksGuidance() *Guidance {
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

	if c.HasPendingTasks {
		steps = append(steps, Step{
			Action:      ui.Warning(fmt.Sprintf("You have %d PENDING tasks requiring decision", c.PendingCount)),
			Command:     "",
			Description: "",
		})
		steps = append(steps, Step{
			Action:      "PENDING tasks need design decisions (complete TODO tasks first)",
			Command:     "",
			Description: "",
		})
	}

	return &Guidance{
		Title: "Next Steps",
		Steps: steps,
	}
}

func (c *Context) defaultGuidance() *Guidance {
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
