package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"reviewtask/internal/storage"
	"reviewtask/internal/ui"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var showCmd = &cobra.Command{
	Use:   "show [TASK_ID]",
	Short: "Show detailed information about a specific task",
	Long: `Display detailed information about a specific task including:
- Task description and priority
- Original review comment
- File and line information
- Status and timestamps
- Associated PR and review information

If no TASK_ID is provided, shows the current task (doing status) or next task (todo status with highest priority).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runShow,
}

func init() {
	// Register output format flags
	showCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
	showCmd.Flags().BoolP("brief", "b", false, "Output only key fields (brief mode)")
}

func runShow(cmd *cobra.Command, args []string) error {
	// Display AI provider info
	_, err := DisplayAIProviderIfNeeded()
	if err != nil {
		// Continue without config - show can work without it
	}

	// Read flags to determine output format
	jsonOut, _ := cmd.Flags().GetBool("json")
	briefOut, _ := cmd.Flags().GetBool("brief")

	storageManager := storage.NewManager()

	if len(args) == 0 {
		// No task ID provided, show current or next task
		return showCurrentOrNextTask(cmd, storageManager, jsonOut, briefOut)
	}

	// Task ID provided, show specific task
	taskID := args[0]
	return showSpecificTask(cmd, storageManager, taskID, jsonOut, briefOut)
}

func showCurrentOrNextTask(cmd *cobra.Command, storageManager *storage.Manager, jsonOut, briefOut bool) error {
	allTasks, err := storageManager.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	if len(allTasks) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No tasks found.")
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'reviewtask [PR_NUMBER]' to analyze PR reviews and generate tasks.")
		return nil
	}

	// Look for current task (doing status)
	for _, task := range allTasks {
		if task.Status == "doing" {
			if !jsonOut && !briefOut {
				fmt.Fprintln(cmd.OutOrStdout(), "Current Task (In Progress)")
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return displayTaskDetails(cmd, task, jsonOut, briefOut)
		}
	}

	// No current task, find next task (todo with highest priority)
	var nextTask *storage.Task
	priorityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
	highestPriority := 0

	for _, task := range allTasks {
		if task.Status == "todo" {
			if priority, exists := priorityOrder[task.Priority]; exists && priority > highestPriority {
				highestPriority = priority
				taskCopy := task
				nextTask = &taskCopy
			}
		}
	}

	if nextTask == nil {
		if !jsonOut && !briefOut {
			fmt.Fprintln(cmd.OutOrStdout(), "No current or next tasks found.")
			fmt.Fprintln(cmd.OutOrStdout(), "All tasks may be completed, cancelled, or pending.")
			fmt.Fprintln(cmd.OutOrStdout(), "Run 'reviewtask status' to see overall task status.")
		}
		return nil
	}

	if !jsonOut && !briefOut {
		fmt.Fprintln(cmd.OutOrStdout(), "Next Task (Recommended)")
		fmt.Fprintln(cmd.OutOrStdout())
	}
	return displayTaskDetails(cmd, *nextTask, jsonOut, briefOut)
}

func showSpecificTask(cmd *cobra.Command, storageManager *storage.Manager, taskID string, jsonOut, briefOut bool) error {
	allTasks, err := storageManager.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Find the specific task
	for _, task := range allTasks {
		if task.ID == taskID {
			return displayTaskDetails(cmd, task, jsonOut, briefOut)
		}
	}

	return fmt.Errorf("task with ID '%s' not found", taskID)
}

func displayTaskDetails(cmd *cobra.Command, task storage.Task, jsonOut, briefOut bool) error {
	// Handle JSON output
	if jsonOut {
		return displayTaskAsJSON(cmd, task)
	}

	// Handle brief output
	if briefOut {
		return displayTaskBrief(cmd, task)
	}

	// Default detailed output
	out := cmd.OutOrStdout()
	title := cases.Title(language.Und)

	// Task Overview section
	fmt.Fprintln(out, ui.SectionDivider("Task Overview"))
	fmt.Fprintf(out, "ID: %s\n", task.ID)
	fmt.Fprintf(out, "Status: %s\n", strings.ToUpper(task.Status))
	fmt.Fprintf(out, "Priority: %s\n", strings.ToUpper(task.Priority))

	// Implementation and Verification Status
	if task.ImplementationStatus != "" {
		fmt.Fprintf(out, "Implementation: %s\n", title.String(task.ImplementationStatus))
	}
	if task.VerificationStatus != "" {
		fmt.Fprintf(out, "Verification: %s\n", title.String(task.VerificationStatus))
	}
	fmt.Fprintln(out)

	// Task Description
	fmt.Fprintln(out, ui.SectionDivider("Description"))
	fmt.Fprintf(out, "%s\n", task.Description)
	fmt.Fprintln(out)

	// Original Review Comment
	fmt.Fprintln(out, ui.SectionDivider("Original Comment"))
	originLines := strings.Split(task.OriginText, "\n")
	for _, line := range originLines {
		fmt.Fprintf(out, "%s\n", line)
	}
	fmt.Fprintln(out)

	// File and Line Information
	if task.File != "" {
		fmt.Fprintln(out, ui.SectionDivider("Location"))
		fmt.Fprintf(out, "File: %s\n", task.File)
		if task.Line > 0 {
			fmt.Fprintf(out, "Line: %d\n", task.Line)
		}
		fmt.Fprintln(out)
	}

	// PR and Review Information
	fmt.Fprintln(out, ui.SectionDivider("Source"))
	fmt.Fprintf(out, "PR Number: #%d\n", task.PRNumber)
	fmt.Fprintf(out, "Review ID: %d\n", task.SourceReviewID)
	fmt.Fprintf(out, "Comment ID: %d\n", task.SourceCommentID)
	if task.TaskIndex > 0 {
		fmt.Fprintf(out, "Task Index: %d (multiple tasks from same comment)\n", task.TaskIndex)
	}
	if task.URL != "" {
		fmt.Fprintf(out, "URL: %s\n", task.URL)
	}
	fmt.Fprintln(out)

	// Timestamps
	fmt.Fprintln(out, ui.SectionDivider("Timeline"))
	if task.CreatedAt != "" {
		if createdTime, err := time.Parse("2006-01-02T15:04:05Z", task.CreatedAt); err == nil {
			fmt.Fprintf(out, "Created: %s\n", createdTime.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Fprintf(out, "Created: %s\n", task.CreatedAt)
		}
	}
	if task.UpdatedAt != "" && task.UpdatedAt != task.CreatedAt {
		if updatedTime, err := time.Parse("2006-01-02T15:04:05Z", task.UpdatedAt); err == nil {
			fmt.Fprintf(out, "Updated: %s\n", updatedTime.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Fprintf(out, "Updated: %s\n", task.UpdatedAt)
		}
	}

	// Last verification timestamp
	if task.LastVerificationAt != "" {
		if verifyTime, err := time.Parse("2006-01-02T15:04:05Z", task.LastVerificationAt); err == nil {
			fmt.Fprintf(out, "Last Verification: %s\n", verifyTime.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Fprintf(out, "Last Verification: %s\n", task.LastVerificationAt)
		}
	}
	fmt.Fprintln(out)

	// Verification History
	if len(task.VerificationResults) > 0 {
		fmt.Fprintln(out, ui.SectionDivider("Verification History"))
		for i, result := range task.VerificationResults {
			resultIndicator := ui.SymbolSuccess
			if !result.Success {
				resultIndicator = ui.SymbolError
			}

			verifyTime := result.Timestamp
			if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", result.Timestamp); err == nil {
				verifyTime = parsedTime.Format("2006-01-02 15:04:05")
			}

			fmt.Fprintf(out, "%d. %s %s", i+1, resultIndicator, verifyTime)
			if len(result.ChecksRun) > 0 {
				fmt.Fprintf(out, " (checks: %s)", strings.Join(result.ChecksRun, ", "))
			}
			fmt.Fprintln(out)

			if result.FailureReason != "" {
				fmt.Fprintf(out, "   Reason: %s\n", result.FailureReason)
			}
		}
		fmt.Fprintln(out)
	}

	// Action suggestions based on status
	fmt.Fprintln(out, ui.SectionDivider("Next Steps"))
	switch task.Status {
	case "todo":
		fmt.Fprintln(out, ui.Next("Start working on this task"))
		fmt.Fprintf(out, "  reviewtask start %s\n", task.ID)
	case "doing":
		fmt.Fprintln(out, ui.Next("Complete the task"))
		fmt.Fprintf(out, "  reviewtask done %s\n", task.ID)
		fmt.Fprintln(out)
		fmt.Fprintln(out, ui.Next("Or verify without completing"))
		fmt.Fprintf(out, "  reviewtask verify %s\n", task.ID)
	case "pending":
		fmt.Fprintln(out, ui.Next("Resume work when unblocked"))
		fmt.Fprintf(out, "  reviewtask start %s\n", task.ID)
		fmt.Fprintln(out)
		fmt.Fprintln(out, ui.Next("Or cancel if no longer needed"))
		fmt.Fprintf(out, "  reviewtask cancel %s\n", task.ID)
	case "done":
		fmt.Fprintln(out, ui.Success("Task completed!"))
	case "cancel", "cancelled":
		fmt.Fprintln(out, "Task cancelled")
	}
	fmt.Fprintln(out)

	return nil
}

// displayTaskAsJSON outputs the task in JSON format
func displayTaskAsJSON(cmd *cobra.Command, task storage.Task) error {
	// Create a simplified JSON representation of the task
	jsonTask := map[string]interface{}{
		"id":                    task.ID,
		"status":                task.Status,
		"priority":              task.Priority,
		"description":           task.Description,
		"origin_text":           task.OriginText,
		"file":                  task.File,
		"line":                  task.Line,
		"pr_number":             task.PRNumber,
		"source_review_id":      task.SourceReviewID,
		"source_comment_id":     task.SourceCommentID,
		"task_index":            task.TaskIndex,
		"created_at":            task.CreatedAt,
		"updated_at":            task.UpdatedAt,
		"implementation_status": task.ImplementationStatus,
		"verification_status":   task.VerificationStatus,
		"last_verification_at":  task.LastVerificationAt,
		"verification_results":  task.VerificationResults,
		"url":                   task.URL,
	}

	jsonData, err := json.MarshalIndent(jsonTask, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task to JSON: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(jsonData))
	return nil
}

// displayTaskBrief outputs the task in brief format (limited lines)
func displayTaskBrief(cmd *cobra.Command, task storage.Task) error {
	out := cmd.OutOrStdout()
	// Brief format: just essential info, max ~5 lines
	fmt.Fprintf(out, "Task: %s | %s | %s\n", task.ID, task.Status, task.Priority)
	fmt.Fprintf(out, "Description: %s\n", task.Description)
	if task.File != "" {
		fmt.Fprintf(out, "File: %s", task.File)
		if task.Line > 0 {
			fmt.Fprintf(out, ":%d", task.Line)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "PR: #%d | Comment: %d\n", task.PRNumber, task.SourceCommentID)
	return nil
}
