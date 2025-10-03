package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"

	"github.com/spf13/cobra"
)

var (
	batchSize  int
	maxBatches int
	async      bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [PR_NUMBER]",
	Short: "Analyze PR reviews and generate tasks in batches",
	Long: `Analyze GitHub Pull Request reviews and generate tasks using AI.
This command processes comments in small batches to provide better
control and progress visibility.

The command will process a limited number of batches per execution
and automatically save progress. Run the command again to continue
processing remaining comments.

Examples:
  reviewtask analyze        # Analyze current branch's PR
  reviewtask analyze 123    # Analyze PR #123
  reviewtask analyze 123 --batch-size 3 --max-batches 2`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAnalyzeCommand,
}

func init() {
	analyzeCmd.Flags().IntVar(&batchSize, "batch-size", 1, "Number of comments to process per batch (recommended: 1 for stability)")
	analyzeCmd.Flags().IntVar(&maxBatches, "max-batches", 1, "Maximum number of batches to process per command")
	analyzeCmd.Flags().BoolVar(&async, "async", false, "Run analysis in background (output will be logged to stderr)")
	rootCmd.AddCommand(analyzeCmd)
}

// runAnalysis executes the analysis logic (used by both async and sync paths)
func runAnalysis(cfg *config.Config, analyzer *ai.Analyzer, reviews []github.Review, prNumber int, storageManager *storage.Manager, batchSize, maxBatches int) ([]storage.Task, error) {
	// Validate input parameters
	if batchSize <= 0 {
		return nil, fmt.Errorf("batch-size must be > 0")
	}
	if maxBatches < 0 {
		return nil, fmt.Errorf("max-batches must be >= 0")
	}

	var tasks []storage.Task
	var err error

	// Check if real-time saving is enabled
	if cfg.AISettings.RealtimeSavingEnabled {
		// Use real-time saving for immediate task persistence
		tasks, err = analyzer.GenerateTasksWithRealtimeSaving(reviews, prNumber, storageManager)
	} else {
		// Set up incremental processing options
		incrementalOpts := ai.IncrementalOptions{
			BatchSize:           batchSize,
			MaxBatchesToProcess: maxBatches,
			Resume:              true,
			FastMode:            false,
			MaxTimeout:          10 * time.Minute,
			ShowProgress:        true,
			OnProgress: func(processed, total int) {
				if processed > 0 && processed%batchSize == 0 {
					fmt.Printf("  📝 Processed %d/%d comments...\n", processed, total)
				}
			},
		}

		// Generate tasks incrementally
		tasks, err = analyzer.GenerateTasksIncremental(reviews, prNumber, storageManager, incrementalOpts)
	}

	return tasks, err
}

func runAnalyzeCommand(cmd *cobra.Command, args []string) error {
	// Display AI provider info and load configuration
	cfg, err := DisplayAIProviderIfNeeded()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get PR number
	var prNumber int
	if len(args) > 0 {
		prNumber, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}
	} else {
		// Try to get current branch's PR
		ghClient, err := github.NewClient()
		if err != nil {
			return fmt.Errorf("failed to initialize GitHub client: %w", err)
		}
		prNumber, err = ghClient.GetCurrentBranchPR(context.Background())
		if err != nil {
			return fmt.Errorf("failed to determine PR number from current branch: %w", err)
		}
	}

	// Initialize storage manager
	storageManager := storage.NewManager()

	// Check if reviews exist
	reviewsExist, err := storageManager.ReviewsExist(prNumber)
	if err != nil {
		return fmt.Errorf("failed to check reviews: %w", err)
	}

	if !reviewsExist {
		fmt.Printf("❌ No reviews found for PR #%d\n", prNumber)
		fmt.Printf("🔄 Run 'reviewtask fetch %d' first to download reviews\n", prNumber)
		return fmt.Errorf("reviews not found for PR #%d", prNumber)
	}

	// Load existing reviews
	fmt.Printf("📊 Analyzing PR #%d reviews...\n", prNumber)
	reviews, err := storageManager.LoadReviews(prNumber)
	if err != nil {
		return fmt.Errorf("failed to load reviews: %w", err)
	}

	// Display AI provider information
	providerDisplay := config.GetProviderDisplayName(cfg.AISettings.AIProvider, cfg.AISettings.Model)
	fmt.Printf("🤖 Using AI Provider: %s\n", providerDisplay)
	fmt.Println()

	// Pre-flight check: Verify AI provider is available
	_, err = ai.NewAIProvider(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "authentication") {
			fmt.Println()
			fmt.Println("❌ AI Provider Authentication Required")
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()
			fmt.Println("The Claude CLI is not authenticated. To fix this:")
			fmt.Println()
			fmt.Println("1. Open Claude by running:")
			fmt.Println("   $ claude")
			fmt.Println()
			fmt.Println("2. In the Claude interface, use the login command:")
			fmt.Println("   /login")
			fmt.Println()
			fmt.Println("3. Follow the authentication prompts")
			fmt.Println()
			fmt.Println("4. Once authenticated, run this command again")
			fmt.Println()
			fmt.Println("Or if Claude Code logs out frequently, you can skip this check:")
			fmt.Println("- Set environment variable: SKIP_CLAUDE_AUTH_CHECK=true")
			fmt.Println("- Or in config: \"skip_claude_auth_check\": true")
			fmt.Println()
			return fmt.Errorf("claude CLI authentication required")
		}
		return fmt.Errorf("failed to initialize Claude client: %w", err)
	}

	// Initialize AI analyzer
	analyzer := ai.NewAnalyzer(cfg)

	if async {
		// Run analysis in background
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Background analysis failed: %v\n", r)
				}
			}()

			tasks, err := runAnalysis(cfg, analyzer, reviews, prNumber, storageManager, batchSize, maxBatches)

			if err != nil {
				if strings.Contains(err.Error(), "timed out") {
					fmt.Println("\n⚠️  Background processing timed out. Run the command again to resume from where it left off.")
				} else {
					fmt.Printf("Background analysis failed: %v\n", err)
				}
				return
			}

			// Merge tasks with existing ones (preserves task statuses)
			if err := storageManager.MergeTasks(prNumber, tasks); err != nil {
				fmt.Printf("Failed to save tasks: %v\n", err)
				return
			}

			// Show results
			if len(tasks) > 0 {
				fmt.Printf("✅ Background analysis completed: generated %d tasks\n", len(tasks))
			}
		}()

		fmt.Printf("🔄 Analysis started in background. Check progress with: reviewtask status\n")
		return nil
	}

	tasks, err := runAnalysis(cfg, analyzer, reviews, prNumber, storageManager, batchSize, maxBatches)

	if err != nil {
		if strings.Contains(err.Error(), "timed out") {
			fmt.Println("\n⚠️  Processing timed out. Run the command again to resume from where it left off.")
		}
		return fmt.Errorf("failed to generate tasks: %w", err)
	}

	// Merge tasks with existing ones (preserves task statuses)
	if err := storageManager.MergeTasks(prNumber, tasks); err != nil {
		return fmt.Errorf("failed to merge tasks: %w", err)
	}

	// Show results
	if len(tasks) > 0 {
		fmt.Printf("✅ Generated %d tasks and saved to .pr-review/PR-%d/tasks.json\n", len(tasks), prNumber)
	}

	return nil
}
