# Project Structure

This document describes the organization of the reviewtask codebase.

## Directory Layout

```
reviewtask/
├── cmd/                    # CLI command implementations
│   ├── analyze.go         # Main analysis command (deprecated)
│   ├── auth.go            # Authentication management
│   ├── claude.go          # AI provider integration
│   ├── config.go          # Configuration management
│   ├── debug.go           # Debug and troubleshooting
│   ├── fetch.go           # PR fetching and analysis
│   ├── root.go            # Root command and global flags
│   ├── show.go            # Task display
│   ├── stats.go           # Statistics
│   ├── status.go          # Task status overview
│   ├── update.go          # Task updates
│   └── version.go         # Version information
├── internal/              # Private packages
│   ├── ai/               # AI integration
│   │   ├── analyzer.go   # Comment analysis logic
│   │   ├── claude_client.go # Claude CLI wrapper
│   │   ├── stream_processor.go # Parallel processing
│   │   └── simple_task_test.go # Task generation tests
│   ├── config/           # Configuration management
│   │   └── config.go     # Config structures
│   ├── github/           # GitHub integration
│   │   ├── client.go     # API client
│   │   ├── auth.go       # Authentication
│   │   ├── codex_parser.go # Codex embedded comment parser
│   │   ├── deduplication.go # Review deduplication
│   │   └── graphql.go    # GraphQL API (thread resolution)
│   ├── storage/          # Data persistence
│   │   ├── manager.go    # Storage operations
│   │   ├── write_worker.go # Concurrent writes
│   │   └── failed_comments.go # Retry handling
│   ├── tasks/            # Task utilities
│   │   └── formatter.go  # Task formatting
│   ├── verification/     # Data validation
│   │   └── validator.go  # Validation logic
│   └── version/          # Version checking
│       └── checker.go    # Update checks
├── prompts/              # AI prompt templates
│   ├── README.md         # Template documentation
│   └── simple_task_generation.md # Main template
├── scripts/              # Build and release scripts
│   ├── build.sh          # Cross-platform builds
│   ├── release.sh        # Release automation
│   └── version.sh        # Version management
├── docs/                 # Documentation
│   ├── user-guide/       # End-user documentation
│   └── developer-guide/  # Developer documentation
├── test/                 # Integration tests
└── .pr-review/           # Runtime data (gitignored)
    ├── config.json       # Project configuration
    ├── auth.json         # Authentication (gitignored)
    └── PR-{number}/      # Per-PR data
```

## Package Dependencies

### Command Layer (`cmd/`)

- Commands use Cobra for CLI framework
- Each command is self-contained in its own file
- Commands delegate business logic to internal packages
- No direct GitHub API or AI provider calls

### Internal Packages (`internal/`)

#### AI Package (`internal/ai`)
- Handles all AI-related operations
- Abstracts AI provider interactions
- Implements parallel processing
- Manages prompt templates

#### GitHub Package (`internal/github`)
- Wraps GitHub API client (REST and GraphQL)
- Manages authentication
- Handles rate limiting
- Provides caching
- **Multi-source review support**:
  - CodeRabbit integration (standard comments + nitpick detection)
  - Codex integration (embedded comment parsing)
  - Standard GitHub reviews
- **Review deduplication** (content-based fingerprinting)
- **Thread auto-resolution** (GraphQL API integration)

#### Storage Package (`internal/storage`)
- Manages file I/O operations
- Provides thread-safe writes
- Handles data persistence
- Manages PR-specific directories

#### Config Package (`internal/config`)
- Defines configuration structures
- Handles configuration loading/saving
- Provides defaults

## Design Principles

### 1. Single Responsibility
Each package has a clear, focused purpose.

### 2. Dependency Inversion
Commands depend on interfaces, not concrete implementations.

### 3. No Circular Dependencies
Strict hierarchy prevents circular imports.

### 4. Configuration Over Code
Behavior customizable through configuration files.

## Adding New Features

### Adding a New Command

1. Create new file in `cmd/` directory
2. Define command structure using Cobra
3. Implement command logic using internal packages
4. Add command to root in `cmd/root.go`
5. Update documentation

Example:
```go
// cmd/newfeature.go
package cmd

import (
    "github.com/spf13/cobra"
    "reviewtask/internal/storage"
)

var newFeatureCmd = &cobra.Command{
    Use:   "newfeature",
    Short: "Description of new feature",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(newFeatureCmd)
}
```

### Adding a New Internal Package

1. Create directory under `internal/`
2. Define package interfaces
3. Implement concrete types
4. Add tests
5. Update documentation

### Modifying AI Processing

1. Edit templates in `prompts/` directory
2. Update `internal/ai/analyzer.go` for logic changes
3. Add tests in `internal/ai/*_test.go`
4. Update golden tests if output format changes

## Testing Structure

### Unit Tests
- Located alongside source files (`*_test.go`)
- Focus on individual functions
- Mock external dependencies

### Golden Tests
- Compare output against known-good snapshots
- Located in `testdata/` directories
- Update with `UPDATE_GOLDEN=1`

### Integration Tests
- Located in `test/` directory
- Test end-to-end workflows
- Use real GitHub API (with mocking)

## Build System

### Local Development
```bash
go build -o reviewtask main.go
```

### Cross-Platform Builds
```bash
./scripts/build.sh all
```

### Release Process
```bash
./scripts/release.sh prepare minor
./scripts/release.sh release minor
```

## Code Style

### Go Standards
- Follow standard Go formatting (`gofmt`)
- Use `golint` for style checks
- Keep functions small and focused
- Document exported types and functions

### Error Handling
- Return errors, don't panic
- Wrap errors with context
- Log errors at appropriate levels

### Logging
- Use structured logging where possible
- Include context in log messages
- Use appropriate log levels (Error, Warn, Info, Debug)