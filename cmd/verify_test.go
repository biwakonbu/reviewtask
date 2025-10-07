package cmd

import (
	"testing"
)

func TestVerifyCommand(t *testing.T) {
	cmd := verifyCmd

	// Test command properties
	if cmd.Use != "verify <task-id>" {
		t.Errorf("Expected Use 'verify <task-id>', got %q", cmd.Use)
	}

	if cmd.Short != "Verify task completion requirements" {
		t.Errorf("Expected Short description about verification, got %q", cmd.Short)
	}

	// Test args validation
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no arguments")
	}

	if err := cmd.Args(cmd, []string{"task-id"}); err != nil {
		t.Errorf("Expected no error for correct arguments, got: %v", err)
	}

	if err := cmd.Args(cmd, []string{"task-id", "extra"}); err == nil {
		t.Error("Expected error for too many arguments")
	}
}

func TestVerifyCommandFlags(t *testing.T) {
	cmd := verifyCmd

	// Test that verbose flag is properly defined
	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("Expected --verbose flag to be defined")
	} else {
		if verboseFlag.Shorthand != "v" {
			t.Errorf("Expected --verbose flag shorthand 'v', got %q", verboseFlag.Shorthand)
		}
		if verboseFlag.DefValue != "false" {
			t.Errorf("Expected --verbose flag default value 'false', got %q", verboseFlag.DefValue)
		}
	}
}

func TestVerifyCommandHelp(t *testing.T) {
	cmd := verifyCmd

	// Test that help text contains important information
	if cmd.Long == "" {
		t.Error("Expected Long description to be provided")
	}

	// Check for key phrases in help text
	expectedPhrases := []string{
		"verification checks",
		"build verification",
		"test execution",
		"lint/format checks",
		"custom verification",
		"Examples:",
	}

	for _, phrase := range expectedPhrases {
		if !containsPhrase(cmd.Long, phrase) {
			t.Errorf("Expected help text to contain %q", phrase)
		}
	}
}

func TestVerifyCommandExamples(t *testing.T) {
	cmd := verifyCmd

	// Test that examples are present in the help text
	expectedExamples := []string{
		"reviewtask verify task-1",
		"reviewtask verify task-2 --verbose",
	}

	for _, example := range expectedExamples {
		if !containsPhrase(cmd.Long, example) {
			t.Errorf("Expected help text to contain example %q", example)
		}
	}
}

func TestVerifyCommandStructure(t *testing.T) {
	cmd := verifyCmd

	// Test that the command has proper structure
	if cmd.RunE == nil {
		t.Error("Expected RunE function to be defined")
	}

	// Test that the command accepts exactly one argument
	if cmd.Args == nil {
		t.Error("Expected Args validation to be defined")
	}

	// Test command hierarchy
	if cmd.Parent() != nil && cmd.Parent().Name() != "reviewtask" {
		t.Errorf("Expected parent to be nil or 'reviewtask', got %q", cmd.Parent().Name())
	}
}

func TestIndentOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Single line",
			input:    "Hello world",
			expected: "     Hello world",
		},
		{
			name:     "Multiple lines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "     Line 1\n     Line 2\n     Line 3",
		},
		{
			name:     "Line with trailing newline",
			input:    "Hello\n",
			expected: "     Hello",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := indentOutput(test.input)
			if result != test.expected {
				t.Errorf("indentOutput(%q) = %q, expected %q", test.input, result, test.expected)
			}
		})
	}
}

func TestVerifyCommandInit(t *testing.T) {
	// Test that the command initialization sets up flags correctly
	cmd := verifyCmd

	// Check that the init function has properly set up the verbose flag
	flag := cmd.Flags().Lookup("verbose")
	if flag == nil {
		t.Fatal("verbose flag should be initialized")
	}

	// Test flag properties
	if flag.Usage != "Show detailed verification output" {
		t.Errorf("Expected verbose flag usage description, got %q", flag.Usage)
	}
}

func TestVerifyCommandUsage(t *testing.T) {
	cmd := verifyCmd

	// Test that usage information is comprehensive
	usage := cmd.UsageString()
	if len(usage) == 0 {
		t.Error("Expected usage string to be generated")
	}

	// Test that the command can generate help
	help := cmd.Long
	if len(help) == 0 {
		t.Error("Expected help text to be available")
	}

	// Verify that help contains verification types
	verificationTypes := []string{"build", "test", "lint", "format", "custom"}
	for _, vType := range verificationTypes {
		if !containsPhrase(help, vType) {
			t.Errorf("Expected help to mention verification type %q", vType)
		}
	}
}

// Helper function to check if text contains a phrase
func containsPhrase(text, phrase string) bool {
	if len(text) == 0 || len(phrase) == 0 {
		return false
	}

	// Use simple substring checking
	for i := 0; i <= len(text)-len(phrase); i++ {
		if text[i:i+len(phrase)] == phrase {
			return true
		}
	}
	return false
}
