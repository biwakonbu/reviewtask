package cmd

import (
	"fmt"
	"strings"
	"time"

	"reviewtask/internal/storage"

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
	// No command registration needed here - done in root.go
}

func runShow(cmd *cobra.Command, args []string) error {
	storageManager := storage.NewManager()

	if len(args) == 0 {
		// No task ID provided, show current or next task
		return showCurrentOrNextTask(storageManager)
	}

	// Task ID provided, show specific task
	taskID := args[0]
	return showSpecificTask(storageManager, taskID)
}

func showCurrentOrNextTask(storageManager *storage.Manager) error {
	allTasks, err := storageManager.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	if len(allTasks) == 0 {
		fmt.Println("No tasks found.")
		fmt.Println("Run 'reviewtask [PR_NUMBER]' to analyze PR reviews and generate tasks.")
		return nil
	}

	// Look for current task (doing status)
	for _, task := range allTasks {
		if task.Status == "doing" {
			fmt.Println("📍 Current Task (In Progress):")
			fmt.Println()
			return displayTaskDetails(task)
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
		fmt.Println("✅ No current or next tasks found.")
		fmt.Println("All tasks may be completed, cancelled, or pending.")
		fmt.Println("Run 'reviewtask status' to see overall task status.")
		return nil
	}

	fmt.Println("🎯 Next Task (Recommended):")
	fmt.Println()
	return displayTaskDetails(*nextTask)
}

func showSpecificTask(storageManager *storage.Manager, taskID string) error {
	allTasks, err := storageManager.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Find the specific task
	for _, task := range allTasks {
		if task.ID == taskID {
			return displayTaskDetails(task)
		}
	}

	return fmt.Errorf("task with ID '%s' not found", taskID)
}

func displayTaskDetails(task storage.Task) error {
	// Status indicator
	statusIndicator := getStatusIndicator(task.Status)

	// Priority indicator
	priorityIndicator := getPriorityIndicator(task.Priority)

	// Header
	fmt.Printf("Task ID: %s\n", task.ID)
	title := cases.Title(language.Und)
	fmt.Printf("Status: %s %s\n", statusIndicator, title.String(task.Status))
	fmt.Printf("Priority: %s %s\n", priorityIndicator, strings.ToUpper(task.Priority))

	// Implementation and Verification Status
	if task.ImplementationStatus != "" {
		implIndicator := getImplementationIndicator(task.ImplementationStatus)
		fmt.Printf("Implementation: %s %s\n", implIndicator, title.String(task.ImplementationStatus))
	}
	if task.VerificationStatus != "" {
		verifyIndicator := getVerificationIndicator(task.VerificationStatus)
		fmt.Printf("Verification: %s %s\n", verifyIndicator, title.String(task.VerificationStatus))
	}
	fmt.Println()

	// Task Description
	fmt.Println("📝 Task Description:")
	fmt.Printf("   %s\n", task.Description)
	fmt.Println()

	// Original Review Comment
	fmt.Println("💬 Original Review Comment:")
	originLines := strings.Split(task.OriginText, "\n")
	for _, line := range originLines {
		fmt.Printf("   %s\n", line)
	}
	fmt.Println()

	// File and Line Information
	if task.File != "" {
		fmt.Println("📂 Location:")
		fmt.Printf("   File: %s\n", task.File)
		if task.Line > 0 {
			fmt.Printf("   Line: %d\n", task.Line)
		}
		fmt.Println()
	}

	// PR and Review Information
	fmt.Println("🔗 Source Information:")
	fmt.Printf("   PR Number: #%d\n", task.PRNumber)
	fmt.Printf("   Review ID: %d\n", task.SourceReviewID)
	fmt.Printf("   Comment ID: %d\n", task.SourceCommentID)
	if task.TaskIndex > 0 {
		fmt.Printf("   Task Index: %d (multiple tasks from same comment)\n", task.TaskIndex)
	}
	fmt.Println()

	// Timestamps
	fmt.Println("🕒 Timeline:")
	if task.CreatedAt != "" {
		if createdTime, err := time.Parse("2006-01-02T15:04:05Z", task.CreatedAt); err == nil {
			fmt.Printf("   Created: %s\n", createdTime.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   Created: %s\n", task.CreatedAt)
		}
	}
	if task.UpdatedAt != "" && task.UpdatedAt != task.CreatedAt {
		if updatedTime, err := time.Parse("2006-01-02T15:04:05Z", task.UpdatedAt); err == nil {
			fmt.Printf("   Updated: %s\n", updatedTime.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   Updated: %s\n", task.UpdatedAt)
		}
	}

	// Last verification timestamp
	if task.LastVerificationAt != "" {
		if verifyTime, err := time.Parse("2006-01-02T15:04:05Z", task.LastVerificationAt); err == nil {
			fmt.Printf("   Last Verification: %s\n", verifyTime.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   Last Verification: %s\n", task.LastVerificationAt)
		}
	}
	fmt.Println()

	// Verification History
	if len(task.VerificationResults) > 0 {
		fmt.Println("🔍 Verification History:")
		for i, result := range task.VerificationResults {
			resultIndicator := "✅"
			if !result.Success {
				resultIndicator = "❌"
			}

			verifyTime := result.Timestamp
			if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", result.Timestamp); err == nil {
				verifyTime = parsedTime.Format("2006-01-02 15:04:05")
			}

			fmt.Printf("   %d. %s %s", i+1, resultIndicator, verifyTime)
			if len(result.ChecksRun) > 0 {
				fmt.Printf(" (checks: %s)", strings.Join(result.ChecksRun, ", "))
			}
			fmt.Println()

			if result.FailureReason != "" {
				fmt.Printf("      Reason: %s\n", result.FailureReason)
			}
		}
		fmt.Println()
	}

	// Action suggestions based on status
	fmt.Println("💡 Suggested Actions:")
	switch task.Status {
	case "todo":
		fmt.Printf("   Start working on this task:\n")
		fmt.Printf("   reviewtask update %s doing\n", task.ID)
	case "doing":
		fmt.Printf("   Verify and complete task:\n")
		fmt.Printf("   reviewtask complete %s\n", task.ID)
		fmt.Printf("   \n")
		fmt.Printf("   Or verify without completing:\n")
		fmt.Printf("   reviewtask verify %s\n", task.ID)
		fmt.Printf("   \n")
		fmt.Printf("   Or mark as done directly (skip verification):\n")
		fmt.Printf("   reviewtask update %s done\n", task.ID)
		fmt.Printf("   \n")
		fmt.Printf("   Or mark as pending if blocked:\n")
		fmt.Printf("   reviewtask update %s pending\n", task.ID)
	case "pending":
		fmt.Printf("   Resume work when unblocked:\n")
		fmt.Printf("   reviewtask update %s doing\n", task.ID)
		fmt.Printf("   \n")
		fmt.Printf("   Or cancel if no longer needed:\n")
		fmt.Printf("   reviewtask update %s cancel\n", task.ID)
	case "done":
		fmt.Printf("   Task completed! ✅\n")
	case "cancel", "cancelled":
		fmt.Printf("   Task cancelled. ❌\n")
	}

	return nil
}

func getStatusIndicator(status string) string {
	switch status {
	case "todo":
		return "⏳"
	case "doing":
		return "🚀"
	case "done":
		return "✅"
	case "pending":
		return "⏸️"
	case "cancel", "cancelled":
		return "❌"
	default:
		return "❓"
	}
}

func getPriorityIndicator(priority string) string {
	switch priority {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func getImplementationIndicator(status string) string {
	switch status {
	case "implemented":
		return "✅"
	case "not_implemented":
		return "❌"
	default:
		return "❓"
	}
}

func getVerificationIndicator(status string) string {
	switch status {
	case "verified":
		return "✅"
	case "failed":
		return "❌"
	case "not_verified":
		return "⏳"
	default:
		return "❓"
	}
}
