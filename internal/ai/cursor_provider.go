package ai

import (
	"context"
	"reviewtask/internal/config"
)

// CursorProvider implements ClaudeClient using Cursor CLI
type CursorProvider struct {
	*BaseCLIClient
}

// NewCursorProvider creates a new Cursor provider
func NewCursorProvider(cfg *config.Config) (*CursorProvider, error) {
	providerConf := GetCursorProviderConfig()

	baseClient, err := NewBaseCLIClient(cfg, providerConf)
	if err != nil {
		return nil, err
	}

	return &CursorProvider{
		BaseCLIClient: baseClient,
	}, nil
}

// Execute runs Cursor with the given input (implements ClaudeClient interface)
func (p *CursorProvider) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	return p.BaseCLIClient.Execute(ctx, input, outputFormat)
}

// CheckAuthentication verifies Cursor authentication (implements ClaudeClient interface)
func (p *CursorProvider) CheckAuthentication() error {
	return p.BaseCLIClient.CheckAuthentication()
}