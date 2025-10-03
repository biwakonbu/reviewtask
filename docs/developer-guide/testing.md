# Testing Strategy

This document outlines the testing approach and guidelines for reviewtask.

## Testing Philosophy

- **Test real workflows** over isolated units
- **Mock external dependencies** (GitHub API, Claude CLI)
- **Use golden tests** for output stability
- **Maintain high coverage** for critical paths

## Test Types

### 1. Unit Tests

Located alongside source files as `*_test.go`.

**Purpose**: Test individual functions and methods in isolation.

**Example**:
```go
// internal/ai/simple_task_test.go
func TestSimpleTaskRequest_Structure(t *testing.T) {
    task := SimpleTaskRequest{
        Description: "Test description",
        Priority:    "high",
    }

    if task.Description != "Test description" {
        t.Errorf("Expected description 'Test description', got %s", task.Description)
    }
}
```

### 2. Golden Tests

Compare output against known-good snapshots.

**Purpose**: Detect unintended changes in output format.

**Example**:
```go
// internal/ai/simple_prompt_golden_test.go
func TestBuildSimpleCommentPrompt_Golden(t *testing.T) {
    got := analyzer.buildSimpleCommentPrompt(ctx)

    goldenPath := "testdata/prompts/simple/english.golden"
    if updateGoldenEnabled() {
        writeGolden(t, goldenPath, got)
    }

    want := loadGolden(t, goldenPath)
    if got != want {
        t.Fatalf("Prompt mismatch")
    }
}
```

**Updating Golden Files**:
```bash
UPDATE_GOLDEN=1 go test ./internal/ai -run Golden
```

### 3. Integration Tests

Test complete workflows end-to-end.

**Purpose**: Verify components work together correctly.

**Location**: `test/` directory

**Example**:
```go
// test/workflow_test.go
func TestCompleteWorkflow(t *testing.T) {
    // Setup
    client := setupMockGitHub()
    ai := setupMockAI()

    // Execute workflow
    err := FetchAndAnalyze(123)

    // Verify results
    tasks := LoadTasks(123)
    assert.Equal(t, 5, len(tasks))
}
```

### 4. Concurrent Tests

Test thread safety and race conditions.

**Purpose**: Ensure concurrent operations are safe.

**Example**:
```go
// internal/storage/write_worker_test.go
func TestWriteWorker_ConcurrentWrites(t *testing.T) {
    worker := NewWriteWorker(manager, 100, false)

    var wg sync.WaitGroup
    for i := 0; i < 20; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            worker.QueueTask(task)
        }()
    }

    wg.Wait()
    // Verify all tasks written correctly
}
```

**Run with Race Detector**:
```bash
go test -race ./...
```

## Test Organization

### Directory Structure
```
reviewtask/
├── internal/
│   ├── ai/
│   │   ├── analyzer.go
│   │   ├── analyzer_test.go          # Unit tests
│   │   ├── simple_task_test.go       # Focused tests
│   │   ├── simple_prompt_golden_test.go # Golden tests
│   │   └── testdata/                 # Test fixtures
│   │       └── prompts/
│   │           └── simple/
│   │               ├── english.golden
│   │               └── japanese.golden
│   └── storage/
│       ├── manager.go
│       ├── manager_test.go
│       └── write_worker_test.go
└── test/                             # Integration tests
    ├── workflow_test.go
    └── fixtures/
```

### Test Naming Conventions

- **Unit tests**: `Test{FunctionName}_{Scenario}`
- **Golden tests**: `Test{Function}_Golden`
- **Integration tests**: `Test{Feature}Integration_{Scenario}`
- **Benchmarks**: `Benchmark{Operation}`
- **Examples**: `Example{Function}`

### Test Coverage for Review Sources

#### Codex Integration Tests

Tests for Codex embedded comment parsing and conversion:

**Location**: `internal/github/codex_integration_test.go`

**Test Scenarios**:
- Real-world Codex review parsing (from biwakonbu/pylay PR #26)
- Empty review body handling
- Mixed reviewer detection (Codex vs. regular reviewers)
- Thread ID tracking for embedded comments
- Priority badge extraction (P1/P2/P3)
- GitHub permalink parsing
- Title and description extraction

**Running Codex Tests**:
```bash
# Run all Codex integration tests
go test -v ./internal/github -run TestCodexIntegration

# Run specific Codex test
go test -v ./internal/github -run TestCodexIntegration_RealWorldScenario

# Skip integration tests (short mode)
go test -short ./internal/github
```

#### Review Deduplication Tests

Tests for duplicate review detection:

**Location**: `internal/github/deduplication_test.go`

**Test Scenarios**:
- Content-based fingerprinting
- Duplicate review detection from same reviewer
- Chronological ordering preservation
- Similar content detection
- Multiple reviewer handling

**Running Deduplication Tests**:
```bash
go test -v ./internal/github -run TestDeduplicateReviews
go test -v ./internal/github -run TestIsSimilarContent
```

## Writing Tests

### Test Structure

```go
func TestFunctionName_Scenario(t *testing.T) {
    // Arrange
    input := setupTestData()
    expected := expectedResult()

    // Act
    result := FunctionUnderTest(input)

    // Assert
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Table-Driven Tests

```go
func TestExtractJSON(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "Plain JSON",
            input:    `{"test": "value"}`,
            expected: `{"test": "value"}`,
            wantErr:  false,
        },
        {
            name:     "Markdown wrapped",
            input:    "```json\n{\"test\": \"value\"}\n```",
            expected: `{"test": "value"}`,
            wantErr:  false,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := ExtractJSON(tc.input)
            if tc.wantErr && err == nil {
                t.Error("Expected error but got none")
            }
            if result != tc.expected {
                t.Errorf("Expected %s, got %s", tc.expected, result)
            }
        })
    }
}
```

### Mocking External Dependencies

#### Mock Claude Client
```go
type MockClaudeClient struct {
    ExecuteFunc func(input, format string) (string, error)
}

func (m *MockClaudeClient) Execute(input, format string) (string, error) {
    if m.ExecuteFunc != nil {
        return m.ExecuteFunc(input, format)
    }
    return "", nil
}
```

#### Mock GitHub Client
```go
type MockGitHubClient struct {
    PRs     map[int]*github.PullRequest
    Reviews map[int][]*github.Review
}

func (m *MockGitHubClient) GetPR(number int) (*github.PullRequest, error) {
    if pr, ok := m.PRs[number]; ok {
        return pr, nil
    }
    return nil, fmt.Errorf("PR not found")
}
```

### Testing Error Cases

```go
func TestHandleError(t *testing.T) {
    testCases := []struct {
        name        string
        setupMock   func() *MockClient
        expectedErr string
    }{
        {
            name: "Network error",
            setupMock: func() *MockClient {
                return &MockClient{
                    ReturnError: errors.New("network timeout"),
                }
            },
            expectedErr: "network timeout",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            mock := tc.setupMock()
            err := ProcessWithClient(mock)
            if err == nil || !strings.Contains(err.Error(), tc.expectedErr) {
                t.Errorf("Expected error containing %q, got %v", tc.expectedErr, err)
            }
        })
    }
}
```

## Test Coverage

### Measuring Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Coverage for specific package
go test -cover ./internal/ai
```

### Coverage Goals

- **Overall**: Aim for >80% coverage
- **Critical paths**: 95%+ coverage
- **AI processing**: 90%+ coverage
- **Storage operations**: 90%+ coverage
- **Error handling**: 100% coverage

### Excluding from Coverage

```go
// Some legitimate exclusions:
// - Panic recovery code
// - Unreachable defensive code
// - Generated code
// - Test helpers
```

## Testing Commands

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/ai

# Verbose output
go test -v ./...

# Short tests only (skip slow tests)
go test -short ./...

# With timeout
go test -timeout 30s ./...

# Parallel execution
go test -parallel 4 ./...
```

### Test Caching

```bash
# Clear test cache
go clean -testcache

# Force test run (skip cache)
go test -count=1 ./...
```

## Continuous Integration

### GitHub Actions Workflow

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - run: go test -v -race -cover ./...
```

### Pre-commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: go-test
        name: go test
        entry: go test ./...
        language: system
        pass_filenames: false
```

## Best Practices

### 1. Test Behavior, Not Implementation

❌ Bad:
```go
func TestInternalState(t *testing.T) {
    obj := NewObject()
    if obj.internalField != 42 {
        t.Error("Internal field not set")
    }
}
```

✅ Good:
```go
func TestObjectBehavior(t *testing.T) {
    obj := NewObject()
    result := obj.Process()
    if result != expected {
        t.Error("Unexpected behavior")
    }
}
```

### 2. Use Descriptive Test Names

❌ Bad: `TestProcess`
✅ Good: `TestProcess_WithEmptyInput_ReturnsError`

### 3. Keep Tests Independent

Each test should:
- Set up its own data
- Clean up after itself
- Not depend on test execution order

### 4. Use t.Helper() for Test Helpers

```go
func assertTaskEqual(t *testing.T, expected, actual Task) {
    t.Helper() // Reports errors at call site
    if expected != actual {
        t.Errorf("Tasks not equal\nExpected: %+v\nActual: %+v", expected, actual)
    }
}
```

### 5. Test Edge Cases

Always test:
- Empty inputs
- Nil values
- Maximum values
- Concurrent access
- Error conditions

## Debugging Failed Tests

### Verbose Output

```bash
go test -v ./internal/ai
```

### Run Single Test

```bash
go test -run TestSpecificFunction ./internal/ai
```

### Debug with Print Statements

```go
t.Logf("Debug: value = %+v", value)
```

### Use Delve Debugger

```bash
dlv test ./internal/ai -- -test.run TestSpecificFunction
```

## Performance Testing

### Writing Benchmarks

```go
func BenchmarkTaskGeneration(b *testing.B) {
    comment := setupTestComment()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        GenerateTasks(comment)
    }
}
```

### Running Benchmarks

```bash
# Run benchmarks
go test -bench=. ./internal/ai

# With memory allocation stats
go test -bench=. -benchmem ./internal/ai

# Compare benchmarks
go test -bench=. -count=10 ./internal/ai | tee old.txt
# Make changes
go test -bench=. -count=10 ./internal/ai | tee new.txt
benchstat old.txt new.txt
```

## Test Maintenance

### Updating Tests

When changing functionality:
1. Update tests first (TDD approach)
2. Make code changes
3. Verify tests pass
4. Update golden files if needed
5. Update documentation

### Removing Obsolete Tests

Periodically review and remove:
- Tests for deleted features
- Duplicate tests
- Tests that no longer provide value

### Test Documentation

Document complex tests:
```go
// TestComplexScenario verifies that when multiple comments
// are processed concurrently with some failing, the system
// correctly saves successful tasks while tracking failures
// for retry with exponential backoff.
func TestComplexScenario(t *testing.T) {
    // ... test implementation
}
```