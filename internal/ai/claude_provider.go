package ai

import (
	"context"
	"reviewtask/internal/config"
)

// ClaudeProvider implements ClaudeClient using Claude Code CLI
type ClaudeProvider struct {
	*BaseCLIClient
}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(cfg *config.Config) (*ClaudeProvider, error) {
	providerConf := GetClaudeProviderConfig()

	baseClient, err := NewBaseCLIClient(cfg, providerConf)
	if err != nil {
		return nil, err
	}

	return &ClaudeProvider{
		BaseCLIClient: baseClient,
	}, nil
}

// Execute runs Claude with the given input (implements ClaudeClient interface)
func (p *ClaudeProvider) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	return p.BaseCLIClient.Execute(ctx, input, outputFormat)
}

// CheckAuthentication verifies Claude authentication (implements ClaudeClient interface)
func (p *ClaudeProvider) CheckAuthentication() error {
	return p.BaseCLIClient.CheckAuthentication()
}