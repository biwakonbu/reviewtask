# Development Setup

This guide helps you set up your development environment for contributing to reviewtask.

## Prerequisites

### Required Software

1. **Go 1.21+**
   ```bash
   go version  # Should show go1.21 or higher
   ```

2. **Git**
   ```bash
   git --version
   ```

3. **GitHub CLI (gh)**
   ```bash
   gh --version
   ```

4. **Claude CLI**
   ```bash
   claude --version
   ```

### Optional Tools

- **Make** - For using Makefile commands
- **jq** - For JSON processing in scripts
- **golangci-lint** - For code linting

## Setting Up the Development Environment

### 1. Fork and Clone the Repository

```bash
# Fork on GitHub first, then:
git clone https://github.com/YOUR_USERNAME/reviewtask.git
cd reviewtask
git remote add upstream https://github.com/biwakonbu/reviewtask.git
```

### 2. Install Dependencies

```bash
go mod download
go mod verify
```

### 3. Set Up Pre-commit Hooks (Optional)

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install
```

### 4. Configure GitHub Access

```bash
# Using GitHub CLI (recommended)
gh auth login

# Or set environment variable
export GITHUB_TOKEN="your_github_token"
```

### 5. Configure Claude CLI

```bash
# Install Claude CLI if not already installed
# Follow instructions at https://claude.ai/code

# Verify installation
claude --version
```

## Building the Project

### Standard Build

```bash
go build -o reviewtask main.go
```

### Development Build with Version Info

```bash
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "\
  -X main.version=$VERSION \
  -X main.commitHash=$COMMIT \
  -X main.buildDate=$DATE" \
  -o reviewtask main.go
```

### Cross-Platform Build

```bash
# Build for all platforms
./scripts/build.sh all

# Build for specific platform
./scripts/build.sh linux-amd64
```

## Running Tests

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/ai

# Verbose output
go test -v ./...
```

### Golden Tests

```bash
# Run golden tests
go test ./internal/ai -run Golden

# Update golden files when needed
UPDATE_GOLDEN=1 go test ./internal/ai -run Golden
```

### Integration Tests

```bash
# Run integration tests
go test ./test -tags integration

# With real GitHub API (requires token)
GITHUB_TOKEN=$YOUR_TOKEN go test ./test -tags integration
```

### Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out
```

## Debugging

### Running in Debug Mode

```bash
# Use debug commands
reviewtask debug fetch review 123
reviewtask debug fetch task 123
reviewtask debug prompt 123
```

### Using Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug main.go -- fetch 123

# Set breakpoints
(dlv) break internal/ai/analyzer.go:100
(dlv) continue
```

### Logging

Enable verbose mode in configuration:
```json
{
  "ai_settings": {
    "verbose_mode": true
  }
}
```

## Code Style and Linting

### Format Code

```bash
# Format all Go files
go fmt ./...

# Or use gofmt directly
gofmt -w .
```

### Run Linters

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linters
golangci-lint run

# Fix issues automatically
golangci-lint run --fix
```

### Pre-commit Checks

```bash
# Run all pre-commit hooks
pre-commit run --all-files

# Run specific hook
pre-commit run go-fmt --all-files
```

## Common Development Tasks

### Adding a New Command

1. Create file in `cmd/` directory
2. Implement command using Cobra
3. Add to root command
4. Add tests
5. Update documentation

### Modifying AI Prompts

1. Edit template in `prompts/` directory
2. Test with debug command:
   ```bash
   reviewtask debug prompt 123
   ```
3. Update golden tests if needed
4. Test with real PR

### Adding a New Configuration Option

1. Update `internal/config/config.go`
2. Add default value
3. Update `docs/user-guide/configuration.md`
4. Add migration logic if needed

### Working with Storage

1. Understand PR-specific directory structure
2. Use WriteWorker for concurrent writes
3. Always use mutex for file operations
4. Test with multiple PRs

## Troubleshooting Development Issues

### Module Issues

```bash
# Clean module cache
go clean -modcache

# Update dependencies
go get -u ./...

# Tidy modules
go mod tidy
```

### Build Issues

```bash
# Clean build cache
go clean -cache

# Verbose build
go build -v -o reviewtask main.go
```

### Test Issues

```bash
# Skip cache
go test -count=1 ./...

# Run with race detector
go test -race ./...
```

## Development Workflow

### 1. Create Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes

- Write code
- Add tests
- Update documentation

### 3. Test Locally

```bash
# Run tests
go test ./...

# Build and test binary
go build -o reviewtask main.go
./reviewtask YOUR_PR_NUMBER
```

### 4. Commit Changes

```bash
git add .
git commit -m "feat: your feature description"
```

### 5. Push and Create PR

```bash
git push origin feature/your-feature-name
gh pr create
```

## Useful Make Commands

If using the Makefile:

```bash
make build        # Build binary
make test         # Run tests
make test-cover   # Run tests with coverage
make lint         # Run linters
make fmt          # Format code
make clean        # Clean build artifacts
```

## Environment Variables

### Development Variables

```bash
# Enable debug logging
export REVIEWTASK_DEBUG=true

# Skip version check
export REVIEWTASK_SKIP_VERSION_CHECK=true

# Use specific config file
export REVIEWTASK_CONFIG=/path/to/config.json
```

### Testing Variables

```bash
# Update golden test files
export UPDATE_GOLDEN=1

# Skip integration tests
export SKIP_INTEGRATION=1

# Use test GitHub token
export TEST_GITHUB_TOKEN=your_test_token
```

## IDE Setup

### Visual Studio Code

`.vscode/settings.json`:
```json
{
  "go.lintTool": "golangci-lint",
  "go.formatTool": "gofmt",
  "go.testFlags": ["-v"],
  "go.testTimeout": "30s"
}
```

### GoLand / IntelliJ IDEA

1. Open project
2. Configure Go SDK
3. Enable Go modules
4. Set up run configurations

## Getting Help

- Check [Troubleshooting Guide](../user-guide/troubleshooting.md)
- Search [GitHub Issues](https://github.com/biwakonbu/reviewtask/issues)
- Ask in [Discussions](https://github.com/biwakonbu/reviewtask/discussions)
- Review [Contributing Guidelines](contributing.md)