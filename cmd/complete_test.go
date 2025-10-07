package cmd

import (
	"testing"
)

func TestCompleteCommand(t *testing.T) {
	cmd := completeCmd

	// Test command properties
	if cmd.Use != "complete <task-id>" {
		t.Errorf("Expected Use 'complete <task-id>', got %q", cmd.Use)
	}

	if cmd.Short != "[DEPRECATED] Use 'reviewtask done' instead" {
		t.Errorf("Expected deprecation message in Short description, got %q", cmd.Short)
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

func TestCompleteCommandFlags(t *testing.T) {
	cmd := completeCmd

	// Test that flags are properly defined
	verifyFlag := cmd.Flags().Lookup("verify")
	if verifyFlag == nil {
		t.Error("Expected --verify flag to be defined")
	} else {
		if verifyFlag.DefValue != "true" {
			t.Errorf("Expected --verify flag default value 'true', got %q", verifyFlag.DefValue)
		}
	}

	skipFlag := cmd.Flags().Lookup("skip-verification")
	if skipFlag == nil {
		t.Error("Expected --skip-verification flag to be defined")
	} else {
		if skipFlag.DefValue != "false" {
			t.Errorf("Expected --skip-verification flag default value 'false', got %q", skipFlag.DefValue)
		}
	}

	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("Expected --verbose flag to be defined")
	} else {
		if verboseFlag.Shorthand != "v" {
			t.Errorf("Expected --verbose flag shorthand 'v', got %q", verboseFlag.Shorthand)
		}
	}
}

func TestCompleteCommandHelp(t *testing.T) {
	cmd := completeCmd

	// Test that help text contains important information
	if cmd.Long == "" {
		t.Error("Expected Long description to be provided")
	}

	// Check for key phrases in help text
	expectedPhrases := []string{
		"DEPRECATION NOTICE",
		"reviewtask done",
		"Migration:",
	}

	for _, phrase := range expectedPhrases {
		if !containsPhrase(cmd.Long, phrase) {
			t.Errorf("Expected help text to contain %q", phrase)
		}
	}
}

func TestCompleteCommandExamples(t *testing.T) {
	cmd := completeCmd

	// Test that migration examples are present in the help text
	expectedExamples := []string{
		"reviewtask done <task-id>",
		"reviewtask done task-1",
		"--skip-verification",
	}

	for _, example := range expectedExamples {
		if !containsPhrase(cmd.Long, example) {
			t.Errorf("Expected help text to contain example %q", example)
		}
	}
}

func TestRunCompleteLogic(t *testing.T) {
	// Test the logic of runComplete function flag handling
	// Note: This is testing the flag logic, not the actual command execution

	tests := []struct {
		name           string
		skipVerify     bool
		withVerify     bool
		expectedDirect bool
	}{
		{
			name:           "Skip verification takes precedence",
			skipVerify:     true,
			withVerify:     true,
			expectedDirect: true,
		},
		{
			name:           "Default with verification",
			skipVerify:     false,
			withVerify:     true,
			expectedDirect: false,
		},
		{
			name:           "With verification disabled",
			skipVerify:     false,
			withVerify:     false,
			expectedDirect: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Reset global flags
			completeSkipVerify = test.skipVerify
			completeWithVerify = test.withVerify

			// Test the logic that determines whether to skip verification
			shouldSkip := completeSkipVerify || (!completeWithVerify && !completeSkipVerify)

			if shouldSkip != test.expectedDirect {
				t.Errorf("Expected direct completion: %t, got: %t", test.expectedDirect, shouldSkip)
			}
		})
	}
}

func TestCompleteCommandStructure(t *testing.T) {
	cmd := completeCmd

	// Test that the command has proper structure
	if cmd.RunE == nil {
		t.Error("Expected RunE function to be defined")
	}

	// Test that parent command relationship is not set (will be set by root)
	if cmd.Parent() != nil && cmd.Parent().Name() != "reviewtask" {
		t.Errorf("Expected parent to be nil or 'reviewtask', got %q", cmd.Parent().Name())
	}

	// Test that the command accepts exactly one argument
	if cmd.Args == nil {
		t.Error("Expected Args validation to be defined")
	}
}

// Helper function to check if text contains a phrase
func containsPhrase(text, phrase string) bool {
	if len(text) == 0 || len(phrase) == 0 {
		return false
	}

	// Use strings.Contains for actual substring checking
	for i := 0; i <= len(text)-len(phrase); i++ {
		if text[i:i+len(phrase)] == phrase {
			return true
		}
	}
	return false
}
