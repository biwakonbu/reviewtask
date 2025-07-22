package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRootCommandRegistration tests that commands are registered correctly
func TestRootCommandRegistration(t *testing.T) {
	// Get the root command
	root := rootCmd

	// Expected commands that should be registered
	expectedCommands := map[string]bool{
		"status":  false,
		"update":  false,
		"show":    false,
		"stats":   false,
		"version": false,
		"auth":    false,
		"init":    false,
		"claude":  false,
	}

	// Check all registered commands
	for _, cmd := range root.Commands() {
		if _, expected := expectedCommands[cmd.Name()]; expected {
			expectedCommands[cmd.Name()] = true
		} else if cmd.Name() != "completion" && cmd.Name() != "help" {
			// Ignore auto-generated commands
			t.Errorf("Unexpected command registered: %s", cmd.Name())
		}
	}

	// Verify all expected commands are registered
	for cmdName, registered := range expectedCommands {
		if !registered {
			t.Errorf("Expected command not registered: %s", cmdName)
		}
	}
}

// TestNoDuplicateCommands tests that no commands are registered multiple times
func TestNoDuplicateCommands(t *testing.T) {
	// Get the root command
	root := rootCmd

	// Map to track command names
	commandNames := make(map[string]int)

	// Count occurrences of each command
	for _, cmd := range root.Commands() {
		commandNames[cmd.Name()]++
	}

	// Check for duplicates
	for cmdName, count := range commandNames {
		if count > 1 {
			t.Errorf("Command '%s' is registered %d times (should be 1)", cmdName, count)
		}
	}
}

// TestCommandInitialization tests that all commands are properly initialized
func TestCommandInitialization(t *testing.T) {
	// Get the root command
	root := rootCmd

	// Test each command has proper configuration
	for _, cmd := range root.Commands() {
		t.Run(cmd.Name(), func(t *testing.T) {
			// Check command has a Use field
			if cmd.Use == "" {
				t.Errorf("Command '%s' has empty Use field", cmd.Name())
			}

			// Check command has a Short description
			if cmd.Short == "" {
				t.Errorf("Command '%s' has empty Short description", cmd.Name())
			}

			// Check command has either Run, RunE, or subcommands
			if cmd.Run == nil && cmd.RunE == nil && !cmd.HasSubCommands() {
				t.Errorf("Command '%s' has no Run function or subcommands", cmd.Name())
			}
		})
	}
}

// TestRootCommandHelp tests that the root command help is accessible
func TestRootCommandHelp(t *testing.T) {
	// Create a new root command instance to avoid side effects
	cmd := &cobra.Command{
		Use:   rootCmd.Use,
		Short: rootCmd.Short,
		Long:  rootCmd.Long,
	}

	// Test help flag
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()

	// Help should not return an error
	if err != nil {
		t.Errorf("Help command returned error: %v", err)
	}
}

// TestCommandStructure tests the overall command structure
func TestCommandStructure(t *testing.T) {
	root := rootCmd

	// Test root command properties
	if root.Use != "reviewtask [PR_NUMBER]" {
		t.Errorf("Unexpected root command Use: %s", root.Use)
	}

	if !strings.Contains(root.Short, "AI-powered PR review management tool") {
		t.Errorf("Root command Short description doesn't match expected: %s", root.Short)
	}

	// Test that root command accepts max 1 argument
	if root.Args == nil {
		t.Error("Root command should have Args validation")
	}

	// Test persistent flags
	refreshFlag := root.PersistentFlags().Lookup("refresh-cache")
	if refreshFlag == nil {
		t.Error("Root command should have 'refresh-cache' persistent flag")
	}
}

// TestSubcommandUniqueness tests that all subcommands have unique names and aliases
func TestSubcommandUniqueness(t *testing.T) {
	root := rootCmd
	
	// Track all command names and aliases
	usedNames := make(map[string]string) // name -> command that uses it
	
	for _, cmd := range root.Commands() {
		// Check main command name
		if existingCmd, exists := usedNames[cmd.Name()]; exists {
			t.Errorf("Duplicate command name '%s' used by both '%s' and current command", cmd.Name(), existingCmd)
		}
		usedNames[cmd.Name()] = cmd.Name()
		
		// Check aliases
		for _, alias := range cmd.Aliases {
			if existingCmd, exists := usedNames[alias]; exists {
				t.Errorf("Alias '%s' of command '%s' conflicts with '%s'", alias, cmd.Name(), existingCmd)
			}
			usedNames[alias] = cmd.Name()
		}
	}
}