package ui

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConsole_BasicOutput(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsoleWithWriter(&buf)

	console.Print("Hello")
	console.Printf(" %s", "World")
	console.Println("!")

	expected := "Hello World!\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestConsole_ProgressBuffering(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsoleWithWriter(&buf)
	console.SetBufferEnabled(true)

	// Start progress mode
	console.SetProgressActive(true)
	console.Print("Message 1")
	console.Print("Message 2")

	// Messages should be buffered, nothing written yet
	if buf.String() != "" {
		t.Errorf("Expected empty buffer during progress, got %q", buf.String())
	}

	// Stop progress mode - messages should be flushed
	console.SetProgressActive(false)
	expected := "Message 1Message 2"
	if buf.String() != expected {
		t.Errorf("Expected %q after flush, got %q", expected, buf.String())
	}
}

func TestConsole_NoBuffering(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsoleWithWriter(&buf)
	console.SetBufferEnabled(false)

	// Start progress mode but with buffering disabled
	console.SetProgressActive(true)
	console.Print("Immediate message")

	// Message should be written immediately even during progress
	expected := "Immediate message"
	if buf.String() != expected {
		t.Errorf("Expected %q immediately, got %q", expected, buf.String())
	}
}

func TestConsole_ConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsoleWithWriter(&buf)

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 100

	// Start multiple goroutines writing concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				console.Printf("G%d-M%d\n", id, j)
			}
		}(i)
	}

	wg.Wait()

	// Verify we got the expected number of lines
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	expectedLines := numGoroutines * messagesPerGoroutine

	if len(lines) != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, len(lines))
	}

	// Verify no corruption by checking each line has expected format
	for _, line := range lines {
		if !strings.HasPrefix(line, "G") || !strings.Contains(line, "-M") {
			t.Errorf("Corrupted line detected: %q", line)
		}
	}
}

func TestConsole_ProgressModeTransitions(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsoleWithWriter(&buf)
	console.SetBufferEnabled(true)

	// Normal mode
	console.Print("Before progress")

	// Progress mode
	console.SetProgressActive(true)
	console.Print("During progress 1")
	console.Print("During progress 2")

	// Back to normal mode
	console.SetProgressActive(false)
	console.Print("After progress")

	expected := "Before progressDuring progress 1During progress 2After progress"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestConsole_ClearLine(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsoleWithWriter(&buf)

	console.ClearLine()
	expected := "\r\033[K"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestConsole_WriteWithSync(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsoleWithWriter(&buf)

	console.WriteWithSync(func(w io.Writer) {
		w.Write([]byte("Custom write"))
	})

	expected := "Custom write"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Test that global functions work (we can't easily test output without affecting other tests)
	// So we just verify they don't panic
	SetProgressActive(true)
	SetBufferEnabled(true)
	SetProgressActive(false)
	ClearLine()
	// These would normally write to stdout, but we're just testing they don't panic
}

func TestConsole_RaceConditionStress(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsoleWithWriter(&buf)
	console.SetBufferEnabled(true)

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Progress toggler
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		active := false
		for {
			select {
			case <-ticker.C:
				active = !active
				console.SetProgressActive(active)
			case <-done:
				return
			}
		}
	}()

	// Multiple writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				console.Printf("Writer%d-Msg%d\n", id, j)
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)
	close(done)
	wg.Wait()

	// Just verify we didn't crash and got some output
	if buf.Len() == 0 {
		t.Error("Expected some output but got none")
	}
}