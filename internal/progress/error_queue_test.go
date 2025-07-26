package progress

import (
	"fmt"
	"strings"
	"testing"
)

func TestModel_ErrorQueue(t *testing.T) {
	model := New()

	// Test adding single error
	msg := errorMsg{message: "Test error message"}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	if len(m.errorQueue) != 1 {
		t.Errorf("Expected 1 error in queue, got %d", len(m.errorQueue))
	}

	if m.errorQueue[0] != "Test error message" {
		t.Errorf("Expected 'Test error message', got %q", m.errorQueue[0])
	}
}

func TestModel_ErrorQueueLimit(t *testing.T) {
	model := New()
	model.maxErrors = 3 // Override default for testing

	// Add more errors than the limit
	for i := 0; i < 5; i++ {
		msg := errorMsg{message: fmt.Sprintf("Error %d", i)}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)
	}

	// Should only keep the last 3 errors
	if len(model.errorQueue) != 3 {
		t.Errorf("Expected 3 errors in queue, got %d", len(model.errorQueue))
	}

	// Should have errors 2, 3, 4 (the last 3)
	expected := []string{"Error 2", "Error 3", "Error 4"}
	for i, expectedErr := range expected {
		if model.errorQueue[i] != expectedErr {
			t.Errorf("Expected error %d to be %q, got %q", i, expectedErr, model.errorQueue[i])
		}
	}
}

func TestModel_ErrorsInView(t *testing.T) {
	model := New()
	model.isTTY = true // Ensure TTY mode for full view
	
	// Add an error
	msg := errorMsg{message: "Display corruption detected"}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Get the view output
	view := m.View()

	// Should contain the error message
	if !strings.Contains(view, "Display corruption detected") {
		t.Errorf("View should contain the error message. View: %s", view)
	}

	// Should contain the warning emoji
	if !strings.Contains(view, "⚠️") {
		t.Errorf("View should contain warning emoji. View: %s", view)
	}
}

func TestModel_NoErrorsInView(t *testing.T) {
	model := New()
	
	// Get the view output without errors
	view := model.View()

	// Should not contain error section
	if strings.Contains(view, "⚠️") {
		t.Error("View should not contain warning emoji when no errors")
	}
}

func TestAddErrorCommand(t *testing.T) {
	cmd := AddError("Test error")
	msg := cmd()
	
	errorMessage, ok := msg.(errorMsg)
	if !ok {
		t.Error("AddError should return an errorMsg")
	}

	if errorMessage.message != "Test error" {
		t.Errorf("Expected 'Test error', got %q", errorMessage.message)
	}
}