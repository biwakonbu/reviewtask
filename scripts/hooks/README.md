# Git Hooks for reviewtask

## Pre-push Hook

The pre-push hook automatically runs quality checks before allowing a push to proceed.

### What it checks:

1. **Code Formatting**: Ensures all Go files are properly formatted using `gofmt`
2. **Linting**: Runs `golangci-lint` on project files (if available)
3. **Tests**: Executes all tests using `go test ./...`
4. **Build**: Verifies the project builds successfully

### Setup

The pre-push hook is automatically installed in this repository. To verify it's working:

```bash
# Test the hook (without actually pushing)
./.git/hooks/pre-push origin https://github.com/example/test.git
```

### Requirements

- **Go**: Required for formatting, testing, and building
- **golangci-lint** (optional): Install with:
  ```bash
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  ```

### Bypassing the Hook

In emergency situations, you can bypass the hook with:
```bash
git push --no-verify
```

**Note**: This is strongly discouraged as it skips quality checks.

### Hook Behavior

- **Format Check**: If formatting issues are found, the hook will fail and show which files need formatting
- **Linting**: Warnings are allowed, but serious issues will cause failure
- **Tests**: All tests must pass for the push to proceed  
- **Build**: The project must compile successfully

### Troubleshooting

If the hook fails:

1. **Formatting Issues**: Run `gofmt -w .` to fix formatting
2. **Test Failures**: Run `go test ./...` to see detailed test results
3. **Build Issues**: Run `go build .` to identify compilation problems
4. **Linter Issues**: Run `golangci-lint run` to see detailed linting results

The hook is designed to catch issues early and maintain code quality standards.