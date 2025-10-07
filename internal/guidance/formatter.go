package guidance

import (
	"fmt"
	"reviewtask/internal/ui"
	"strings"
)

// Formatter handles the presentation of guidance to users.
type Formatter struct {
	language string
}

// NewFormatter creates a new guidance formatter.
func NewFormatter(language string) *Formatter {
	if language == "" {
		language = "en"
	}
	return &Formatter{
		language: language,
	}
}

// Format formats the guidance for output.
func (f *Formatter) Format(g *Guidance) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(ui.SectionDivider("Next Steps"))
	sb.WriteString("\n")

	for i, step := range g.Steps {
		sb.WriteString(ui.Next(step.Action))
		sb.WriteString("\n")
		if step.Command != "" {
			sb.WriteString(ui.Command(step.Command, step.Description))
			sb.WriteString("\n")
		}
		if i < len(g.Steps)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// FormatCompact provides a more concise guidance format.
func (f *Formatter) FormatCompact(g *Guidance) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("Next Steps\n")
	sb.WriteString("──────────\n")

	for _, step := range g.Steps {
		sb.WriteString("→ ")
		sb.WriteString(step.Action)
		sb.WriteString("\n")
		if step.Command != "" {
			sb.WriteString("  ")
			sb.WriteString(step.Command)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// FormatWithContext adds additional context information to the guidance.
func (f *Formatter) FormatWithContext(g *Guidance, ctx *Context) string {
	var sb strings.Builder

	// Add task summary
	if ctx.TodoCount > 0 || ctx.DoingCount > 0 || ctx.PendingCount > 0 {
		sb.WriteString("\n")
		sb.WriteString(ui.SectionDivider("Task Summary"))
		sb.WriteString("\n")
		if ctx.TodoCount > 0 {
			sb.WriteString(ui.InfoText(fmt.Sprintf("TODO: %d", ctx.TodoCount)))
			sb.WriteString("\n")
		}
		if ctx.DoingCount > 0 {
			sb.WriteString(ui.InfoText(fmt.Sprintf("DOING: %d", ctx.DoingCount)))
			sb.WriteString("\n")
		}
		if ctx.PendingCount > 0 {
			sb.WriteString(ui.WarningText(fmt.Sprintf("PENDING: %d", ctx.PendingCount)))
			sb.WriteString("\n")
		}
		if ctx.DoneCount > 0 {
			sb.WriteString(ui.SuccessText(fmt.Sprintf("DONE: %d", ctx.DoneCount)))
			sb.WriteString("\n")
		}
	}

	// Add guidance
	sb.WriteString(f.Format(g))

	return sb.String()
}
