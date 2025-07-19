package ai

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FindClaudeCommand searches for Claude CLI in order of preference:
// 1. Custom path from config (claude_path)
// 2. Environment variable CLAUDE_PATH
// 3. PATH environment variable (exec.LookPath)
// 4. Common installation locations
func FindClaudeCommand(claudePath string) (string, error) {
	// 1. Check custom path in config
	if claudePath != "" {
		if _, err := os.Stat(claudePath); err == nil {
			return claudePath, nil
		}
		return "", fmt.Errorf("custom claude path not found: %s", claudePath)
	}
	
	// 2. Check environment variable
	if envPath := os.Getenv("CLAUDE_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", fmt.Errorf("CLAUDE_PATH environment variable points to non-existent file: %s", envPath)
	}
	
	// 3. Check PATH
	if claudePath, err := exec.LookPath("claude"); err == nil {
		return claudePath, nil
	}
	
	// 4. Check common installation locations
	homeDir := os.Getenv("HOME")
	commonPaths := []string{
		filepath.Join(homeDir, ".claude/local/claude"),           // Local installation
		filepath.Join(homeDir, ".local/bin/claude"),             // User local bin
		filepath.Join(homeDir, ".npm-global/bin/claude"),        // npm global with custom prefix
		"/usr/local/bin/claude",                                 // System-wide installation
		"/opt/claude/bin/claude",                                // Alternative system location
	}
	
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	return "", fmt.Errorf("claude command not found in any search location")
}