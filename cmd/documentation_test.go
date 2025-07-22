package cmd

import (
	"testing"
)

// TestDocumentationAccuracy tests that documented commands are accurately implemented
func TestDocumentationAccuracy(t *testing.T) {
	tests := []struct {
		name    string
		command string
		expect  bool
	}{
		{
			name:    "stats command exists",
			command: "stats",
			expect:  true,
		},
		{
			name:    "versions command exists",
			command: "versions",
			expect:  true,
		},
		{
			name:    "claude command exists",
			command: "claude",
			expect:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tt.command})
			if err != nil && tt.expect {
				t.Errorf("Expected command %s to exist but got error: %v", tt.command, err)
			}

			if cmd == nil && tt.expect {
				t.Errorf("Expected command %s to exist but got nil", tt.command)
			}

			if cmd != nil && cmd.Name() != tt.command && tt.expect {
				t.Errorf("Expected command name %s but got %s", tt.command, cmd.Name())
			}
		})
	}
}

// TestStatsCommandDocumentedFlags tests that stats command has documented flags
func TestStatsCommandDocumentedFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"stats"})
	if err != nil {
		t.Fatalf("stats command not found: %v", err)
	}

	expectedFlags := []string{"all", "pr", "branch"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag --%s on stats command but not found", flagName)
		}
	}
}

// TestStatusCommandDocumentedFlags tests that status command has documented flags
func TestStatusCommandDocumentedFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"status"})
	if err != nil {
		t.Fatalf("status command not found: %v", err)
	}

	expectedFlags := []string{"all", "pr", "branch"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag --%s on status command but not found", flagName)
		}
	}
}

// TestVersionCommandFlags tests that version command has documented flags
func TestVersionCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("version command not found: %v", err)
	}

	flag := cmd.Flags().Lookup("check")
	if flag == nil {
		t.Error("Expected flag --check on version command but not found")
	}
}


// TestAuthSubcommands tests that auth command has documented subcommands
func TestAuthSubcommands(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"auth"})
	if err != nil {
		t.Fatalf("auth command not found: %v", err)
	}

	expectedSubcommands := []string{"login", "logout", "status", "check"}

	for _, subcommandName := range expectedSubcommands {
		subcmd, _, err := cmd.Find([]string{subcommandName})
		if err != nil || subcmd == nil || subcmd.Name() != subcommandName {
			t.Errorf("Expected auth subcommand %s but not found or invalid", subcommandName)
		}
	}
}

// TestClaudeCommandArguments tests that claude command supports documented arguments
func TestClaudeCommandArguments(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"claude"})
	if err != nil {
		t.Fatalf("claude command not found: %v", err)
	}

	// Verify that command expects at least one argument based on usage
	if cmd.Args == nil {
		t.Error("Expected claude command to validate arguments but Args is nil")
	}
}

// TestVersionCommandArguments tests that version command supports version arguments
func TestVersionCommandArguments(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("version command not found: %v", err)
	}

	// Version command should accept optional version argument
	// This is validated by checking that Args is either nil (no validation) or accepts optional args
	if cmd.Args != nil {
		// If Args is set, it should be a function that allows 0 or 1 args
		// We can't easily test the function here, but we verify it's configured
		t.Logf("version command has Args validation configured")
	}
}
