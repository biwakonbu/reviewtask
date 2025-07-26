package ai

import (
	"reviewtask/internal/ui"
)

// printf is a wrapper around the synchronized console output
// This replaces direct fmt.Printf calls in AI processing to prevent display corruption
func printf(format string, args ...interface{}) {
	ui.Printf(format, args...)
}

// println is a wrapper around the synchronized console output
func println(msg string) {
	ui.Println(msg)
}

// print is a wrapper around the synchronized console output  
func print(msg string) {
	ui.Print(msg)
}