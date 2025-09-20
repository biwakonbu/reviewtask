package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/setup"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize reviewtask for this repository",
	Long: `Initialize reviewtask for this repository by:
- Creating .pr-review directory
- Generating default configuration
- Adding .pr-review to .gitignore
- Checking GitHub authentication and permissions`,
	RunE: runInit,
}

func init() {
	// Command registration moved to root.go
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(cmd.InOrStdin())

	// Interactive setup wizard
	fmt.Println("Welcome to reviewtask setup!")
	fmt.Println()

	// 1. Check if already initialized
	if setup.IsInitialized() {
		fmt.Println("✓ Repository is already initialized")
		fmt.Println()

		fmt.Print("Reinitialize? This will recreate configuration files (y/N): ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Initialization cancelled")
			return nil
		}
		fmt.Println()
	}

	// Start interactive setup
	fmt.Println("Let's configure reviewtask with minimal settings.")
	fmt.Println("You can customize more settings later if needed.")
	fmt.Println()

	// Ask for language preference
	fmt.Print("What language do you prefer for task descriptions? [English/Japanese] (default: English): ")
	langInput, _ := reader.ReadString('\n')
	langInput = strings.TrimSpace(langInput)
	if langInput == "" {
		langInput = "English"
	}

	// Detect available AI providers
	fmt.Println()
	fmt.Println("Detecting AI providers...")

	cursorAvailable := config.CheckCursorAvailable()
	claudeAvailable := config.CheckClaudeAvailable()

	var provider string
	if cursorAvailable && claudeAvailable {
		fmt.Println("Found: Cursor CLI and Claude Code")
		fmt.Print("Which AI provider would you like to use? [cursor/claude/auto] (default: auto): ")
		providerInput, _ := reader.ReadString('\n')
		provider = strings.TrimSpace(strings.ToLower(providerInput))
		if provider == "" {
			provider = "auto"
		}
		// Validate provider input
		switch provider {
		case "cursor", "claude", "auto":
			// Valid input
		default:
			fmt.Printf("Unrecognized provider '%s'; defaulting to 'auto'.\n", provider)
			provider = "auto"
		}
	} else if cursorAvailable {
		fmt.Println("Found: Cursor CLI")
		fmt.Print("Use Cursor CLI as AI provider? [Y/n]: ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response == "n" || response == "no" {
			provider = "auto"
		} else {
			provider = "cursor"
		}
	} else if claudeAvailable {
		fmt.Println("Found: Claude Code")
		fmt.Print("Use Claude Code as AI provider? [Y/n]: ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response == "n" || response == "no" {
			provider = "auto"
		} else {
			provider = "claude"
		}
	} else {
		fmt.Println("No AI providers detected. Will use auto-detection mode.")
		provider = "auto"
	}

	fmt.Println()

	// 2. Create .pr-review directory
	fmt.Println("📁 Creating .pr-review directory...")
	if err := setup.CreateDirectory(); err != nil {
		return fmt.Errorf("failed to create .pr-review directory: %w", err)
	}
	fmt.Println("✓ .pr-review directory created")

	// 3. Generate minimal configuration
	fmt.Println("⚙️  Generating minimal configuration...")

	// Create simplified config
	simplifiedConfig := config.SimplifiedConfig{
		Language:   langInput,
		AIProvider: provider,
	}

	if err := config.CreateSimplifiedConfig(&simplifiedConfig); err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}
	fmt.Println("✓ Minimal configuration created at .pr-review/config.json")

	// 4. Add to .gitignore
	fmt.Println("📝 Adding .pr-review to .gitignore...")
	if err := setup.UpdateGitignore(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to update .gitignore: %v\n", err)
		fmt.Println("Please manually add '.pr-review/' to your .gitignore file")
	} else {
		fmt.Println("✓ .pr-review added to .gitignore")
	}

	fmt.Println()
	fmt.Println("🔐 Checking GitHub authentication...")

	// 5. Check authentication
	tokenSource, token, err := github.GetTokenWithSource()
	if err != nil {
		fmt.Println("✗ No GitHub authentication found")
		fmt.Println()
		fmt.Println("To complete setup, please authenticate:")
		fmt.Println("  reviewtask auth login")
		fmt.Println()
		return nil
	}

	fmt.Printf("✓ Authenticated as ")

	// 6. Verify authentication and permissions
	client, err := github.NewClientWithToken(token)
	if err != nil {
		fmt.Printf("✗ Failed to create GitHub client\n")
		fmt.Println()
		fmt.Println("Please check your authentication:")
		fmt.Println("  reviewtask auth status")
		return nil
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		fmt.Printf("✗ Failed to verify authentication\n")
		fmt.Println()
		fmt.Println("Please re-authenticate:")
		fmt.Println("  reviewtask auth login")
		return nil
	}

	fmt.Printf("%s (source: %s)\n", user, tokenSource)

	// 7. Check repository permissions
	fmt.Println("🔍 Checking repository permissions...")
	if err := checkRepositoryPermissions(client); err != nil {
		fmt.Printf("⚠️  Permission check failed: %v\n", err)
		fmt.Println()
		fmt.Println("Please ensure your token has the following scopes:")
		fmt.Println("  - repo (or public_repo for public repositories)")
		fmt.Println("  - pull_requests")
		fmt.Println()
		fmt.Println("You can create a new token at: https://github.com/settings/tokens")
		return nil
	}

	fmt.Println("✓ Repository permissions verified")
	fmt.Println()
	fmt.Println("🎉 Initialization complete!")
	fmt.Println()
	fmt.Println("You can now use reviewtask:")
	fmt.Println("  reviewtask <PR_NUMBER>  # Analyze specific PR")
	fmt.Println("  reviewtask status       # View current tasks")
	fmt.Println()

	return nil
}

func checkRepositoryPermissions(client *github.Client) error {
	// Try to get repository info to test read permissions
	_, err := client.GetRepoInfo()
	if err != nil {
		return fmt.Errorf("failed to access repository: %w", err)
	}

	// Try to list PRs to test PR permissions
	_, err = client.GetPRList()
	if err != nil {
		return fmt.Errorf("failed to access pull requests: %w", err)
	}

	return nil
}
