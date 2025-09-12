package cmd

import (
	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
)

// DisplayAIModeContentForTest is a test helper that exposes displayAIModeContent
// for testing the status command output without running the full command.
func DisplayAIModeContentForTest(tasks []storage.Task, contextDescription string) error {
	return displayAIModeContent(tasks, contextDescription)
}

// DisplayTaskDetailsForTest is a test helper that exposes displayTaskDetails
// for testing the show command output without running the full command.
func DisplayTaskDetailsForTest(task storage.Task) error {
	cmd := &cobra.Command{}
	return displayTaskDetails(cmd, task, false, false)
}
