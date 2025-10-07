// Package guidance provides context-aware guidance for user workflow.
package guidance

import (
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

// Generate creates context-appropriate guidance using the rule system.
func (c *Context) Generate() *Guidance {
	if c.Language == "" {
		c.Language = "en"
	}

	ruleSet := NewRuleSet()
	return ruleSet.Apply(c)
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
