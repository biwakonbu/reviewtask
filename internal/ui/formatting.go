package ui

import (
	"fmt"
	"strings"
)

// Header creates a prominent header section.
func Header(title string) string {
	return Bold(title)
}

// ListItem creates a formatted list item.
func ListItem(text string) string {
	return fmt.Sprintf("  • %s", text)
}

// KeyValue formats a key-value pair.
func KeyValue(key, value string) string {
	return fmt.Sprintf("  %s: %s", Bold(key), value)
}

// CodeBlock formats text as a code block with indentation.
func CodeBlock(code string) string {
	return Indent(Dim(code), 2)
}

// Command formats a command suggestion.
func Command(cmd string, description string) string {
	if description != "" {
		return fmt.Sprintf("  %s    %s", Bold(cmd), Dim("# "+description))
	}
	return fmt.Sprintf("  %s", Bold(cmd))
}

// Table creates a simple aligned table.
func Table(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var result strings.Builder

	// Header
	for i, header := range headers {
		result.WriteString(fmt.Sprintf("%-*s", widths[i], header))
		if i < len(headers)-1 {
			result.WriteString("  ")
		}
	}
	result.WriteString("\n")

	// Separator
	for i, width := range widths {
		result.WriteString(strings.Repeat("─", width))
		if i < len(widths)-1 {
			result.WriteString("  ")
		}
	}
	result.WriteString("\n")

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				result.WriteString(fmt.Sprintf("%-*s", widths[i], cell))
				if i < len(row)-1 {
					result.WriteString("  ")
				}
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}
