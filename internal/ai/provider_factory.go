package ai

import (
	"fmt"
	"reviewtask/internal/config"
)

// AIProvider is the common interface for AI providers
// For backward compatibility, we use ClaudeClient interface name
type AIProvider = ClaudeClient

// NewAIProvider creates an AI provider based on configuration
func NewAIProvider(cfg *config.Config) (AIProvider, error) {
	aiProvider := cfg.AISettings.AIProvider
	if aiProvider == "" {
		aiProvider = "auto"
	}

	switch aiProvider {
	case "claude":
		return NewClaudeProvider(cfg)
	case "cursor":
		return NewCursorProvider(cfg)
	case "auto":
		return tryProviders(cfg)
	default:
		return nil, fmt.Errorf("unknown AI provider: %s", aiProvider)
	}
}

// tryProviders tries multiple providers in order and returns the first successful one
func tryProviders(cfg *config.Config) (AIProvider, error) {
	// Try Cursor first
	if provider, err := NewCursorProvider(cfg); err == nil {
		if cfg.AISettings.VerboseMode {
			fmt.Println("✓ Using Cursor CLI for AI analysis")
		}
		return provider, nil
	} else if cfg.AISettings.VerboseMode {
		fmt.Printf("ℹ️  Cursor not available: %v, trying Claude...\n", err)
	}

	// Fallback to Claude
	if provider, err := NewClaudeProvider(cfg); err == nil {
		if cfg.AISettings.VerboseMode {
			fmt.Println("✓ Using Claude Code CLI for AI analysis")
		}
		return provider, nil
	} else if cfg.AISettings.VerboseMode {
		fmt.Printf("⚠️  Claude also not available: %v\n", err)
	}

	return nil, fmt.Errorf("no AI provider available: tried cursor and claude")
}
