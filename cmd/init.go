package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/setup"
	"github.com/spf13/cobra"
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
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Initializing reviewtask for this repository...")
	fmt.Println()

	// 1. Check if already initialized
	if setup.IsInitialized() {
		fmt.Println("✓ Repository is already initialized")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Reinitialize? This will recreate configuration files (y/N): ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Initialization cancelled")
			return nil
		}
		fmt.Println()
	}

	// 2. Create .pr-review directory
	fmt.Println("📁 Creating .pr-review directory...")
	if err := setup.CreateDirectory(); err != nil {
		return fmt.Errorf("failed to create .pr-review directory: %w", err)
	}
	fmt.Println("✓ .pr-review directory created")

	// 3. Generate default configuration
	fmt.Println("⚙️  Generating default configuration...")
	if err := config.CreateDefault(); err != nil {
		return fmt.Errorf("failed to create default configuration: %w", err)
	}
	fmt.Println("✓ Default configuration created at .pr-review/config.json")

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
