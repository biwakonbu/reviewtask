# Developer Guide

This documentation is for developers who want to contribute to reviewtask or understand its internal architecture.

## Supported Integration Points

### AI Providers
- **Cursor CLI** - Primary AI provider with automatic model selection
- **Claude Code** - Alternative AI provider via Anthropic's CLI
- **Extensible architecture** - Easy to add new AI providers

### Review Source Integrations
- **Standard GitHub Reviews** - REST API + GraphQL for thread resolution
- **CodeRabbit** (`coderabbitai[bot]`) - Nitpick detection, HTML cleaning
- **Codex** (`chatgpt-codex-connector`) - Embedded comment parser with priority badges

See [Architecture Overview](architecture.md) for detailed integration patterns.

## Architecture & Design

- [Architecture Overview](architecture.md) - System design and components
- [Project Structure](project-structure.md) - Code organization and packages
- [Data Models](data-models.md) - Storage schemas and structures

## Development

- [Development Setup](development-setup.md) - Setting up your development environment
- [Contributing Guidelines](contributing.md) - How to contribute to the project
- [Testing Strategy](testing.md) - Testing approach and guidelines
- [Versioning & Releases](versioning.md) - Release management process

## Technical Reference

- [API Documentation](api-reference.md) - Internal API documentation
- [Plugin System](plugin-system.md) - Extending reviewtask
- [Performance Optimization](performance.md) - Performance considerations

## Implementation Details

- [AI Integration](ai-integration.md) - How AI task generation works
- [GitHub API Client](github-client.md) - GitHub integration details
- [Storage System](storage-system.md) - Data persistence architecture
- [Concurrency Model](concurrency.md) - Parallel processing design

## Project Management

- [Product Requirements](prd.md) - Original product requirements document
- [Implementation Progress](implementation-progress.md) - Feature completion tracking
- [Known Issues](known-issues.md) - Current limitations and planned fixes