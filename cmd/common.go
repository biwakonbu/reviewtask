package cmd

import (
	"fmt"
	"reviewtask/internal/config"
)

// DisplayAIProvider shows the current AI provider and model configuration
// This should be called at the start of every major command
func DisplayAIProvider(cfg *config.Config) {
	// Apply smart defaults to get actual provider if auto
	config.ApplySmartDefaults(cfg)

	// Get display name after smart defaults are applied
	displayName := config.GetProviderDisplayName(cfg.AISettings.AIProvider, cfg.AISettings.Model)

	fmt.Printf("🤖 AI Provider: %s\n", displayName)
}

// DisplayAIProviderIfNeeded shows AI provider for commands that use AI
// Returns the config for further use
func DisplayAIProviderIfNeeded() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	DisplayAIProvider(cfg)
	return cfg, nil
}
