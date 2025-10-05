// Package ui provides modern, clean UI components for reviewtask output.
// Inspired by GitHub CLI design principles: minimal, clean, clear, and modern.
package ui

import (
	"fmt"
	"strings"
)

// Status symbols
const (
	SymbolSuccess = "✓"
	SymbolError   = "✗"
	SymbolNext    = "→"
	SymbolWarning = "!"
)

// SectionDivider creates a section divider with the given title.
func SectionDivider(title string) string {
	divider := strings.Repeat("─", len(title))
	return fmt.Sprintf("%s\n%s", title, divider)
}

// Success formats a success message with the success symbol.
func Success(message string) string {
	return fmt.Sprintf("  %s %s", SymbolSuccess, message)
}

// Error formats an error message with the error symbol.
func Error(message string) string {
	return fmt.Sprintf("  %s %s", SymbolError, message)
}

// Next formats a next step message with the next symbol.
func Next(message string) string {
	return fmt.Sprintf("%s %s", SymbolNext, message)
}

// Warning formats a warning message with the warning symbol.
func Warning(message string) string {
	return fmt.Sprintf("%s %s", SymbolWarning, message)
}

// Indent adds indentation to a message.
func Indent(message string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(message, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}
