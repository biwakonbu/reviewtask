package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"reviewtask/internal/tasks"
)

// Progress bar color styles for different task states
// Colors use ANSI 256-color palette for broad terminal compatibility:
// - Basic colors (8-15) work across most terminal themes
// - Color 240 provides subtle contrast for empty states
var (
	TodoProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")) // Gray for TODO - neutral waiting state

	DoingProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")) // Yellow for DOING - active work in progress

	DoneProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")) // Green for DONE - completed successfully

	PendingProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")) // Red for PENDING - blocked/needs attention

	EmptyProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")) // Dark gray for empty/cancelled - de-emphasized
)

// GenerateColoredProgressBar creates a progress bar with colors representing different task states
func GenerateColoredProgressBar(stats tasks.TaskStats, width int) string {
	// Validate width parameter
	if width <= 0 {
		return ""
	}

	total := stats.StatusCounts["todo"] + stats.StatusCounts["doing"] +
		stats.StatusCounts["done"] + stats.StatusCounts["pending"] + stats.StatusCounts["cancel"]

	if total == 0 {
		// Empty progress bar
		emptyBar := strings.Repeat("░", width)
		return EmptyProgressStyle.Render(emptyBar)
	}

	// Calculate completion rate
	completed := stats.StatusCounts["done"] + stats.StatusCounts["cancel"]
	completionRate := float64(completed) / float64(total)

	// Calculate widths based on completion vs remaining
	filledWidth := int(completionRate * float64(width))
	emptyWidth := width - filledWidth

	// For filled portion, show proportional colors for done/cancel
	var segments []string

	if filledWidth > 0 {
		// Within filled portion, show proportions of done vs cancel
		if completed > 0 {
			doneInFilled := int(float64(stats.StatusCounts["done"]) / float64(completed) * float64(filledWidth))
			cancelInFilled := filledWidth - doneInFilled

			if doneInFilled > 0 {
				segments = append(segments, DoneProgressStyle.Render(strings.Repeat("█", doneInFilled)))
			}
			if cancelInFilled > 0 {
				segments = append(segments, EmptyProgressStyle.Render(strings.Repeat("█", cancelInFilled)))
			}
		}
	}

	// For empty portion, show remaining work with status colors
	if emptyWidth > 0 {
		remaining := stats.StatusCounts["todo"] + stats.StatusCounts["doing"] + stats.StatusCounts["pending"]
		if remaining > 0 {
			// Proportional representation of remaining work
			doingInEmpty := int(float64(stats.StatusCounts["doing"]) / float64(remaining) * float64(emptyWidth))
			pendingInEmpty := int(float64(stats.StatusCounts["pending"]) / float64(remaining) * float64(emptyWidth))
			todoInEmpty := emptyWidth - doingInEmpty - pendingInEmpty

			if doingInEmpty > 0 {
				segments = append(segments, DoingProgressStyle.Render(strings.Repeat("░", doingInEmpty)))
			}
			if pendingInEmpty > 0 {
				segments = append(segments, PendingProgressStyle.Render(strings.Repeat("░", pendingInEmpty)))
			}
			if todoInEmpty > 0 {
				segments = append(segments, TodoProgressStyle.Render(strings.Repeat("░", todoInEmpty)))
			}
		} else {
			// Just empty gray
			segments = append(segments, EmptyProgressStyle.Render(strings.Repeat("░", emptyWidth)))
		}
	}

	return strings.Join(segments, "")
}

// ProgressBar creates a progress bar with the given parameters.
// Example: Progress [████████░░░░░░░░░░░░] 40% (4/10)
func ProgressBar(completed, total int, width int) string {
	if total == 0 {
		return "Progress [" + strings.Repeat("░", width) + "] 0% (0/0)"
	}

	percentage := float64(completed) / float64(total) * 100
	filledWidth := int(float64(completed) / float64(total) * float64(width))
	emptyWidth := width - filledWidth

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", emptyWidth)
	return fmt.Sprintf("Progress [%s] %.0f%% (%d/%d)", bar, percentage, completed, total)
}

// SimpleProgress creates a simple progress indicator without a bar.
func SimpleProgress(completed, total int) string {
	if total == 0 {
		return "0 of 0 complete (0%)"
	}
	percentage := float64(completed) / float64(total) * 100
	return fmt.Sprintf("%d of %d complete (%.0f%%)", completed, total, percentage)
}
