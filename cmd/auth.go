package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"reviewtask/internal/github"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication management",
	Long:  `Manage GitHub authentication for reviewtask.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub",
	Long: `Authenticate with GitHub by providing a personal access token.

The token will be saved locally for future use. You can create a token at:
https://github.com/settings/tokens

Required scopes: repo, pull_requests`,
	RunE: runAuthLogin,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  `Check current authentication status and show which token source is being used.`,
	RunE:  runAuthStatus,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove local authentication",
	Long:  `Remove locally stored authentication token.`,
	RunE:  runAuthLogout,
}

var authCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check authentication and permissions",
	Long: `Perform comprehensive authentication and permission checks for GitHub API access.

This command verifies:
- Token authentication
- Repository access permissions
- Pull request access permissions
- Required scopes and capabilities`,
	RunE: runAuthCheck,
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authCheckCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	// Force interactive token input
	token, err := github.PromptForTokenWithSave()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Skip token verification in test environment
	if os.Getenv("REVIEWTASK_TEST_MODE") == "true" {
		fmt.Println("✓ Test mode: skipping token verification")
		fmt.Println("✓ Token saved locally")
		return nil
	}

	// Test the token by making a simple API call
	client, err := github.NewClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("failed to verify token: %w", err)
	}

	fmt.Printf("✓ Authenticated as %s\n", user)
	fmt.Println("✓ Token saved locally")

	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	tokenSource, token, err := github.GetTokenWithSource()
	if err != nil {
		fmt.Println("✗ Not authenticated")
		fmt.Println()
		fmt.Println("To authenticate, run:")
		fmt.Println("  reviewtask auth login")
		return nil
	}

	// Skip token verification in test environment
	if os.Getenv("REVIEWTASK_TEST_MODE") == "true" {
		fmt.Printf("✓ Authentication configured (source: %s)\n", tokenSource)
		fmt.Println("✓ Test mode: skipping token verification")
		return nil
	}

	// Test the token
	client, err := github.NewClientWithToken(token)
	if err != nil {
		fmt.Printf("✗ Invalid token (source: %s)\n", tokenSource)
		return nil
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		fmt.Printf("✗ Token authentication failed (source: %s)\n", tokenSource)
		fmt.Println()
		fmt.Println("To re-authenticate, run:")
		fmt.Println("  reviewtask auth login")
		return nil
	}

	fmt.Printf("✓ Authenticated as %s\n", user)
	fmt.Printf("✓ Token source: %s\n", tokenSource)
	fmt.Println()
	fmt.Println("For comprehensive permission checking, run:")
	fmt.Println("  reviewtask auth check")

	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	if err := github.RemoveLocalToken(); err != nil {
		return fmt.Errorf("failed to remove local token: %w", err)
	}

	fmt.Println("✓ Local authentication removed")
	fmt.Println()
	fmt.Println("Note: This only removes the locally stored token.")
	fmt.Println("Your gh CLI authentication (if any) remains unchanged.")

	return nil
}

func runAuthCheck(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Comprehensive Authentication and Permission Check")
	fmt.Println()

	// 1. Check token existence and source
	fmt.Println("📝 Checking authentication...")
	tokenSource, token, err := github.GetTokenWithSource()
	if err != nil {
		fmt.Println("✗ No GitHub authentication found")
		fmt.Println()
		fmt.Println("To authenticate, run:")
		fmt.Println("  reviewtask auth login")
		return nil
	}

	fmt.Printf("✓ Token found (source: %s)\n", tokenSource)

	// Skip token verification in test environment
	if os.Getenv("REVIEWTASK_TEST_MODE") == "true" {
		fmt.Println("✓ Test mode: skipping detailed permission checks")
		fmt.Println("✓ All permissions OK (test mode)")
		return nil
	}

	// 2. Test basic authentication
	fmt.Println("🔐 Testing token authentication...")
	client, err := github.NewClientWithToken(token)
	if err != nil {
		fmt.Println("✗ Failed to create GitHub client")
		fmt.Println()
		fmt.Println("Please re-authenticate:")
		fmt.Println("  reviewtask auth login")
		return nil
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		fmt.Println("✗ Token authentication failed")
		fmt.Println()
		fmt.Println("Error details:", err)
		fmt.Println()
		fmt.Println("Please re-authenticate:")
		fmt.Println("  reviewtask auth login")
		return nil
	}

	fmt.Printf("✓ Authenticated as %s\n", user)

	// 3. Check token scopes
	fmt.Println("🔑 Checking token scopes...")
	scopes, err := client.GetTokenScopes()
	if err != nil {
		fmt.Printf("⚠️  Could not retrieve token scopes: %v\n", err)
	} else {
		fmt.Printf("✓ Token scopes: %v\n", scopes)
	}

	// 4. Check repository permissions
	fmt.Println("📁 Testing repository access...")
	var missingPermissions []string

	_, err = client.GetRepoInfo()
	if err != nil {
		fmt.Printf("✗ Repository access failed: %v\n", err)
		missingPermissions = append(missingPermissions, "repo (or public_repo for public repositories)")
	} else {
		fmt.Println("✓ Repository access verified")
	}

	// 5. Check pull request permissions
	fmt.Println("🔄 Testing pull request access...")
	_, err = client.GetPRList()
	if err != nil {
		fmt.Printf("✗ Pull request access failed: %v\n", err)
		missingPermissions = append(missingPermissions, "pull_requests")
	} else {
		fmt.Println("✓ Pull request access verified")
	}

	// 6. Check review access (try to get reviews for existing PR)
	fmt.Println("📝 Testing review access...")
	prs, err := client.GetPRList()
	if err == nil && len(prs) > 0 {
		_, err = client.GetPRReviews(context.Background(), prs[0].GetNumber())
		if err != nil {
			fmt.Printf("⚠️  Review access test failed: %v\n", err)
		} else {
			fmt.Println("✓ Review access verified")
		}
	} else {
		fmt.Println("⚠️  No PRs found for review access testing")
	}

	fmt.Println()

	// 7. Summary and recommendations
	if len(missingPermissions) > 0 {
		fmt.Println("❌ Missing Permissions Detected")
		fmt.Println()
		fmt.Println("Your token is missing the following required permissions:")
		for _, perm := range missingPermissions {
			fmt.Printf("  - %s\n", perm)
		}
		fmt.Println()
		fmt.Println("To fix this:")
		fmt.Println("1. Go to: https://github.com/settings/tokens")
		fmt.Println("2. Edit your token or create a new one")
		fmt.Println("3. Ensure the following scopes are selected:")
		fmt.Println("   ✓ repo (Full control of private repositories)")
		fmt.Println("   ✓ public_repo (Access public repositories)")
		fmt.Println("4. Run: reviewtask auth login")
		fmt.Println()
	} else {
		fmt.Println("✅ All Checks Passed!")
		fmt.Println()
		fmt.Println("Your authentication is properly configured.")
		fmt.Println("reviewtask should work correctly with this setup.")
		fmt.Println()
	}

	return nil
}
