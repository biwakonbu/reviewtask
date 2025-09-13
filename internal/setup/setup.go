package setup

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	PRReviewDir = ".pr-review"
)

// IsInitialized checks if the repository is already initialized
func IsInitialized() bool {
	// Check if .pr-review directory exists and has config.json
	configPath := filepath.Join(PRReviewDir, "config.json")
	_, err := os.Stat(configPath)
	return err == nil
}

// CreateDirectory creates the .pr-review directory
func CreateDirectory() error {
	return os.MkdirAll(PRReviewDir, 0755)
}

// UpdateGitignore adds .pr-review/ to .gitignore if not already present
func UpdateGitignore() error {
	gitignorePath := ".gitignore"

	// Read existing .gitignore
	var lines []string
	if file, err := os.Open(gitignorePath); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
	}

	// Check if .pr-review/ is already in .gitignore
	prReviewPattern := ".pr-review/"
	for _, line := range lines {
		if strings.TrimSpace(line) == prReviewPattern {
			// Already exists
			return nil
		}
	}

	// Add .pr-review/ to .gitignore
	file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	// Add a comment and the entry
	if len(lines) > 0 && lines[len(lines)-1] != "" {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline to .gitignore: %w", err)
		}
	}
	if _, err := file.WriteString("# reviewtask data directory\n"); err != nil {
		return fmt.Errorf("failed to write comment to .gitignore: %w", err)
	}
	if _, err := file.WriteString(".pr-review/\n"); err != nil {
		return fmt.Errorf("failed to write entry to .gitignore: %w", err)
	}

	return nil
}

// ShouldPromptInit checks if init should be prompted for new users
func ShouldPromptInit() bool {
	return !IsInitialized()
}
