package ai

import (
	"os"
	"path/filepath"
	"runtime"
)

// ProviderConfig defines the configuration for an AI provider
type ProviderConfig struct {
	Name         string   // Provider name (e.g., "claude", "cursor")
	CommandName  string   // CLI command name (e.g., "claude", "cursor-agent")
	DefaultModel string   // Default model to use
	AuthEnvVar   string   // Environment variable to skip auth check
	LoginCommand string   // Command to authenticate
	CommonPaths  []string // Common installation paths to search
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to environment variables
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.Getenv("USERPROFILE") // Windows fallback
		}
	}
	return homeDir
}

// GetClaudeProviderConfig returns the configuration for Claude Code provider
func GetClaudeProviderConfig() ProviderConfig {
	homeDir := getHomeDir()
	var paths []string
	switch runtime.GOOS {
	case "windows":
		paths = []string{
			filepath.Join(homeDir, ".claude", "local", "claude.exe"),
			filepath.Join(homeDir, ".npm-global", "claude.cmd"),
			filepath.Join(homeDir, ".npm-global", "claude.exe"),
			filepath.Join(homeDir, ".volta", "bin", "claude.exe"),
			filepath.Join(homeDir, ".volta", "bin", "claude.cmd"),
			filepath.Join(homeDir, "AppData", "Local", "Programs", "claude", "claude.exe"),
			filepath.Join(homeDir, "AppData", "Roaming", "npm", "claude.cmd"),
			filepath.Join(homeDir, "AppData", "Roaming", "npm", "claude.exe"),
		}
	case "darwin":
		paths = []string{
			filepath.Join(homeDir, ".claude/local/claude"),
			filepath.Join(homeDir, ".npm-global/bin/claude"),
			filepath.Join(homeDir, ".volta/bin/claude"),
			"/usr/local/bin/claude",
			filepath.Join(homeDir, ".local/bin/claude"),
			"/opt/homebrew/bin/claude",
		}
	default:
		paths = []string{
			filepath.Join(homeDir, ".claude/local/claude"),
			filepath.Join(homeDir, ".npm-global/bin/claude"),
			filepath.Join(homeDir, ".volta/bin/claude"),
			"/usr/local/bin/claude",
			filepath.Join(homeDir, ".local/bin/claude"),
		}
	}

	return ProviderConfig{
		Name:         "claude",
		CommandName:  "claude",
		DefaultModel: "sonnet",
		AuthEnvVar:   "SKIP_CLAUDE_AUTH_CHECK",
		LoginCommand: "claude (then use /login command)",
		CommonPaths:  paths,
	}
}

// GetCursorProviderConfig returns the configuration for Cursor CLI provider
func GetCursorProviderConfig() ProviderConfig {
	homeDir := getHomeDir()
	var paths []string
	switch runtime.GOOS {
	case "windows":
		paths = []string{
			filepath.Join(homeDir, "AppData", "Roaming", "npm", "cursor-agent.cmd"),
			filepath.Join(homeDir, "AppData", "Roaming", "npm", "cursor-agent.exe"),
			filepath.Join(homeDir, ".volta", "bin", "cursor-agent.exe"),
			filepath.Join(homeDir, ".volta", "bin", "cursor-agent.cmd"),
		}
	case "darwin":
		paths = []string{
			filepath.Join(homeDir, ".npm-global/bin/cursor-agent"),
			filepath.Join(homeDir, ".volta/bin/cursor-agent"),
			"/usr/local/bin/cursor-agent",
			filepath.Join(homeDir, ".local/bin/cursor-agent"),
			"/opt/homebrew/bin/cursor-agent",
		}
	default:
		paths = []string{
			filepath.Join(homeDir, ".npm-global/bin/cursor-agent"),
			filepath.Join(homeDir, ".volta/bin/cursor-agent"),
			"/usr/local/bin/cursor-agent",
			filepath.Join(homeDir, ".local/bin/cursor-agent"),
		}
	}

	return ProviderConfig{
		Name:         "cursor",
		CommandName:  "cursor-agent",
		DefaultModel: "auto",
		AuthEnvVar:   "SKIP_CURSOR_AUTH_CHECK",
		LoginCommand: "cursor-agent login",
		CommonPaths:  paths,
	}
}
