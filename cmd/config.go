package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"reviewtask/internal/config"
	"reviewtask/internal/verification"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage reviewtask configuration",
	Long: `Manage reviewtask configuration settings including verification commands,
priority rules, and AI settings.

Examples:
  reviewtask config show                    # Show current configuration
  reviewtask config set-verifier build-task "npm run build && npm test"
  reviewtask config get-verifier build-task
  reviewtask config list-verifiers`,
	Args: cobra.MinimumNArgs(1),
	RunE: runConfig,
}

var setVerifierCmd = &cobra.Command{
	Use:   "set-verifier <task-type> <command>",
	Short: "Set custom verification command for task type",
	Long: `Set a custom verification command for a specific task type.

Task types are automatically inferred from task descriptions:
  - test-task     : Tasks containing "test" or "testing"
  - build-task    : Tasks containing "build" or "compile"  
  - style-task    : Tasks containing "lint" or "format"
  - bug-fix       : Tasks containing "bug" or "fix"
  - feature-task  : Tasks containing "feature" or "implement"
  - general-task  : All other tasks

Examples:
  reviewtask config set-verifier build-task "go build ./... && go test ./..."
  reviewtask config set-verifier test-task "go test -v ./..."
  reviewtask config set-verifier style-task "gofmt -l . && golangci-lint run"`,
	Args: cobra.ExactArgs(2),
	RunE: runSetVerifier,
}

var getVerifierCmd = &cobra.Command{
	Use:   "get-verifier <task-type>",
	Short: "Get verification command for task type",
	Long: `Get the current verification command for a specific task type.

Examples:
  reviewtask config get-verifier build-task
  reviewtask config get-verifier test-task`,
	Args: cobra.ExactArgs(1),
	RunE: runGetVerifier,
}

var listVerifiersCmd = &cobra.Command{
	Use:   "list-verifiers",
	Short: "List all configured verification commands",
	Long: `List all configured verification commands by task type.

This shows both default verification commands and any custom commands
that have been configured.`,
	Args: cobra.NoArgs,
	RunE: runListVerifiers,
}

var showConfigCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long: `Show the current reviewtask configuration including priority rules,
AI settings, and verification commands.`,
	Args: cobra.NoArgs,
	RunE: runShowConfig,
}

func init() {
	configCmd.AddCommand(setVerifierCmd)
	configCmd.AddCommand(getVerifierCmd)
	configCmd.AddCommand(listVerifiersCmd)
	configCmd.AddCommand(showConfigCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	// If no subcommand provided, show help
	return cmd.Help()
}

func runSetVerifier(cmd *cobra.Command, args []string) error {
	taskType := args[0]
	command := args[1]

	verifier, err := verification.NewVerifier()
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	err = verifier.SetVerificationCommand(taskType, command)
	if err != nil {
		return fmt.Errorf("failed to set verification command: %w", err)
	}

	fmt.Printf("✅ Set verification command for '%s':\n", taskType)
	fmt.Printf("   %s\n", command)
	fmt.Println("\n💡 This command will be run when verifying tasks of this type")
	return nil
}

func runGetVerifier(cmd *cobra.Command, args []string) error {
	taskType := args[0]

	verifier, err := verification.NewVerifier()
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	// Get the verification config
	config := verifier.GetConfig()
	if command, exists := config.CustomRules[taskType]; exists && command != "" {
		fmt.Printf("Verification command for '%s':\n", taskType)
		fmt.Printf("  %s\n", command)
	} else {
		fmt.Printf("No custom verification command configured for task type '%s'\n", taskType)
		fmt.Println("\nDefault verification commands:")
		fmt.Printf("  Build: %s\n", config.BuildCommand)
		fmt.Printf("  Test:  %s\n", config.TestCommand)
		fmt.Printf("  Lint:  %s\n", config.LintCommand)
	}
	return nil
}

func runListVerifiers(cmd *cobra.Command, args []string) error {
	verifier, err := verification.NewVerifier()
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	config := verifier.GetConfig()

	fmt.Println("Default Verification Commands:")
	fmt.Printf("  build:  %s\n", config.BuildCommand)
	fmt.Printf("  test:   %s\n", config.TestCommand)
	fmt.Printf("  lint:   %s\n", config.LintCommand)
	fmt.Printf("  format: %s\n", config.FormatCommand)

	if len(config.CustomRules) > 0 {
		fmt.Println("\nCustom Verification Commands:")
		for taskType, command := range config.CustomRules {
			if command != "" {
				fmt.Printf("  %-12s: %s\n", taskType, command)
			}
		}
	} else {
		fmt.Println("\nNo custom verification commands configured.")
		fmt.Println("\n💡 To add custom commands, use:")
		fmt.Println("   reviewtask config set-verifier <task-type> <command>")
	}

	fmt.Printf("\nMandatory checks: %s\n", strings.Join(verificationTypesToStrings(config.Mandatory), ", "))
	fmt.Printf("Optional checks:  %s\n", strings.Join(verificationTypesToStrings(config.Optional), ", "))
	fmt.Printf("Timeout:          %v\n", config.Timeout)

	return nil
}

func runShowConfig(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Println("=== ReviewTask Configuration ===")
	fmt.Println()

	fmt.Println("Priority Rules:")
	fmt.Printf("  Critical: %s\n", cfg.PriorityRules.Critical)
	fmt.Printf("  High:     %s\n", cfg.PriorityRules.High)
	fmt.Printf("  Medium:   %s\n", cfg.PriorityRules.Medium)
	fmt.Printf("  Low:      %s\n", cfg.PriorityRules.Low)
	fmt.Println()

	fmt.Println("Task Settings:")
	fmt.Printf("  Default Status:        %s\n", cfg.TaskSettings.DefaultStatus)
	fmt.Printf("  Auto Prioritize:       %t\n", cfg.TaskSettings.AutoPrioritize)
	fmt.Printf("  Low Priority Status:   %s\n", cfg.TaskSettings.LowPriorityStatus)
	fmt.Printf("  Low Priority Patterns: %s\n", strings.Join(cfg.TaskSettings.LowPriorityPatterns, ", "))
	fmt.Println()

	fmt.Println("AI Settings:")
	fmt.Printf("  User Language:           %s\n", cfg.AISettings.UserLanguage)
	fmt.Printf("  Max Retries:             %d\n", cfg.AISettings.MaxRetries)
	validationEnabled := false
	if cfg.AISettings.ValidationEnabled != nil {
		validationEnabled = *cfg.AISettings.ValidationEnabled
	}
	fmt.Printf("  Validation Enabled:      %t\n", validationEnabled)
	fmt.Printf("  Quality Threshold:       %.2f\n", cfg.AISettings.QualityThreshold)
	fmt.Printf("  Verbose Mode:            %t\n", cfg.AISettings.VerboseMode)
	fmt.Printf("  Deduplication Enabled:   %t\n", cfg.AISettings.DeduplicationEnabled)
	fmt.Printf("  Similarity Threshold:    %.2f\n", cfg.AISettings.SimilarityThreshold)
	fmt.Println()

	// Show verification settings
	if err := runListVerifiers(cmd, args); err != nil {
		fmt.Printf("Warning: Could not load verification settings: %v\n", err)
	}

	return nil
}

// Helper function to convert verification types to strings
func verificationTypesToStrings(types []verification.VerificationType) []string {
	strings := make([]string, len(types))
	for i, t := range types {
		strings[i] = string(t)
	}
	return strings
}
