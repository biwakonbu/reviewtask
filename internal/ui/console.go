package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Console provides synchronized console output to prevent display corruption
// This ensures that progress displays and status messages don't interfere with each other
type Console struct {
	mu               sync.Mutex
	writer           io.Writer
	isProgressActive bool
	bufferEnabled    bool
	buffer           []string
}

// NewConsole creates a new console with synchronized output
func NewConsole() *Console {
	return &Console{
		writer: os.Stdout,
	}
}

// NewConsoleWithWriter creates a console with a custom writer (for testing)
func NewConsoleWithWriter(w io.Writer) *Console {
	return &Console{
		writer: w,
	}
}

// SetProgressActive sets whether progress display is currently active
// When active, other messages are buffered to prevent corruption
func (c *Console) SetProgressActive(active bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isProgressActive = active

	// If progress just stopped, flush any buffered messages
	if !active && c.bufferEnabled && len(c.buffer) > 0 {
		for _, msg := range c.buffer {
			fmt.Fprint(c.writer, msg)
		}
		c.buffer = nil
	}
}

// SetBufferEnabled controls whether messages should be buffered when progress is active
func (c *Console) SetBufferEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bufferEnabled = enabled
}

// Print writes a message to the console with synchronization
func (c *Console) Print(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If progress is active and buffering is enabled, buffer the message
	if c.isProgressActive && c.bufferEnabled {
		c.buffer = append(c.buffer, msg)
		return
	}

	// Otherwise, write immediately
	fmt.Fprint(c.writer, msg)
}

// Printf writes a formatted message to the console with synchronization
func (c *Console) Printf(format string, args ...interface{}) {
	c.Print(fmt.Sprintf(format, args...))
}

// Println writes a message with newline to the console with synchronization
func (c *Console) Println(msg string) {
	c.Print(msg + "\n")
}

// ClearLine clears the current line (useful before progress updates)
func (c *Console) ClearLine() {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprint(c.writer, "\r\033[K")
}

// WriteWithSync provides a thread-safe way to write directly to the console
func (c *Console) WriteWithSync(fn func(w io.Writer)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(c.writer)
}

// Global console instance for package-level functions
var globalConsole = NewConsole()

// Package-level functions for easy migration from fmt.Print* calls

// Print writes a synchronized message to stdout
func Print(msg string) {
	globalConsole.Print(msg)
}

// Printf writes a synchronized formatted message to stdout
func Printf(format string, args ...interface{}) {
	globalConsole.Printf(format, args...)
}

// Println writes a synchronized message with newline to stdout
func Println(msg string) {
	globalConsole.Println(msg)
}

// SetProgressActive sets the global progress state
func SetProgressActive(active bool) {
	globalConsole.SetProgressActive(active)
}

// SetBufferEnabled controls global message buffering
func SetBufferEnabled(enabled bool) {
	globalConsole.SetBufferEnabled(enabled)
}

// ClearLine clears the current line globally
func ClearLine() {
	globalConsole.ClearLine()
}
