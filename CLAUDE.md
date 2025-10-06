## Project Language

- Memories and documents should be written in English
- Conversations should be conducted in the language specified by the user

## Documentation Rules

### Public Documentation Standards
- **User-facing documentation must only describe implemented features**
- **Never include future plans, roadmaps, or unimplemented features in user guides**
- **Internal development documents (PRD, implementation progress) must not appear in public docs**
- **Clear separation between user guide and developer guide content**

### Documentation Site Structure
- **User Guide**: Only production-ready features and actual capabilities
- **Developer Guide**: Architecture, setup, testing, contributing guidelines
- **Excluded from Public Site**:
  - Product Requirements Documents (PRD)
  - Implementation progress tracking
  - Future enhancement plans
  - Internal prompt templates documentation

### Content Guidelines
- **Focus on what IS, not what WILL BE**
- **Remove sections like "Future Enhancements" or "Planned Features" from user docs**
- **Keep developer-centric discussions in internal documents only**
- **Ensure all documented features are actually available to users**

# AI-Powered PR Review Management Tool

## Project Vision

Transform GitHub Pull Request reviews into a structured, trackable workflow that ensures no feedback is lost and every actionable comment becomes a managed task. This tool bridges the gap between code review feedback and actual implementation work.

## Core Values and Operating Principles

### 1. Zero Feedback Loss Policy
- **Every actionable review comment must be captured and tracked**
- **No developer should need to manually track what needs to be done**
- **Review discussions should translate directly into work items**

### 2. State Preservation is Sacred
- **Developer work progress is never lost due to tool operations**
- **Task statuses reflect real work and must be preserved across all operations**
- **Tool should adapt to developer workflow, not force workflow changes**

### 3. AI-Assisted, Human-Controlled
- **AI provides intelligent task generation and prioritization**
- **Developers maintain full control over task status and workflow**
- **Automation reduces cognitive overhead without removing agency**

### 4. Simplicity Over Features
- **Core workflow should be immediately intuitive**
- **Advanced features are optional and discoverable**
- **CLI commands follow standard patterns and conventions**

## Ideal Developer Workflow

### 1. PR Review Response Workflow

**The Golden Path for handling PR reviews:**

1. **Receive Review Notification**
   - PR receives reviews with comments and feedback
   - Developer needs to address feedback systematically

2. **Generate Actionable Tasks**
   ```bash
   # Convert all review feedback into tracked tasks
   reviewtask
   ```

3. **Review What Needs to be Done**
   ```bash
   # See current work or next recommended task
   reviewtask show
   ```

4. **Work on Tasks Systematically**
   ```bash
   # Start working on a task (v3.0.0: intuitive commands)
   reviewtask start <task-id>

   # Complete implementation (v3.0.0)
   reviewtask done <task-id>

   # Traditional commands still supported:
   # reviewtask update <task-id> doing
   # reviewtask update <task-id> done
   ```

5. **Handle Updated Reviews**
   ```bash
   # Re-run when reviewers add new comments
   reviewtask
   # Tool automatically preserves your work progress
   ```

### 2. Daily Development Routine

**Morning Startup:**
```bash
reviewtask show           # What should I work on today?
reviewtask status         # Overall progress across all PRs
```

**During Implementation:**
```bash
reviewtask show <task-id>  # Full context for current task
reviewtask start <task-id> # Start working (v3.0.0)
# Work on the task...
reviewtask done <task-id>  # Mark completed (v3.0.0)
```

**When Blocked:**
```bash
reviewtask hold <task-id>  # Put on hold (v3.0.0)
reviewtask show            # Find next task to work on
```

### 3. Debugging and Troubleshooting

**Debug Commands for Testing:**
```bash
# Test specific phases independently  
reviewtask debug fetch review 123    # Fetch reviews for PR #123 only
reviewtask debug fetch task 123      # Generate tasks from saved reviews only
reviewtask debug prompt 123 --profile v2   # Render analysis prompt locally (no AI)
```

**Verbose Mode for Detailed Logging:**
```bash
# Enable verbose output in config for debugging
# Set "verbose_mode": true in .pr-review/config.json
```

**Prompt Size and Validation Issues:**
- Large comments (>20KB) are automatically chunked into smaller pieces
- Validation mode optimizes prompt sizes to prevent failures
- Pre-validation size checks avoid wasted API calls

**JSON Recovery and API Resilience:**
- Automatic recovery from truncated or incomplete Claude API responses
- Intelligent retry strategies with prompt size reduction
- Response monitoring and performance analytics
- Pattern detection for common API failure modes

### 3. Team Collaboration Rules

## Golden Tests (Local-Only Snapshot Tests)

- Purpose: lock down prompt outputs and CLI templates as “expected snapshots” and detect regressions.
- Default profile: `v2` (aka `rich`). Use `--profile legacy` to compare with the previous behavior.
- Scope in this repo:
  - Analyzer prompt profiles: legacy, v2/rich, compact, minimal
  - CLI template: `reviewtask prompt stdout pr-review`

Usage (no AI used):
- Run focused tests: `make test-fast`
- Update snapshots intentionally: `UPDATE_GOLDEN=1 go test -v ./internal/ai -run BuildAnalysisPrompt_Golden`
- Update CLI template snapshot: `UPDATE_GOLDEN=1 go test -v ./cmd -run PromptStdout_PRReview_Golden`

Files:
- `internal/ai/testdata/prompts/<profile>/basic.golden`
- `cmd/testdata/pr-review.golden`

Notes:
- Keep snapshots stable; update only when specification changes.
- Normalize or avoid variable data in prompts used for golden tests.

**For PR Authors:**
- Run `reviewtask` immediately after receiving reviews
- Update task statuses as you complete work
- Never manually edit `.pr-review/` files

**For Reviewers:**
- Write actionable, specific feedback
- Follow up reviews add incremental tasks automatically
- Trust that feedback will be systematically addressed

**For Teams:**
- Integrate task status into standup discussions
- Use task completion as PR readiness indicator
- Treat persistent `pending` tasks as team blockers

## Technology Stack and Architecture Decisions

### Core Technology Choices

**Go Programming Language**
- **Rationale**: CLI tools benefit from Go's single-binary distribution and cross-platform support
- **Rule**: All core functionality implemented in Go with minimal external dependencies

**Claude Code CLI Integration**
- **Rationale**: Provides best-in-class AI analysis while maintaining local control
- **Rule**: All AI processing goes through Claude Code CLI, no direct API calls

**JSON-based Local Storage**
- **Rationale**: Human-readable, git-trackable, and easily debuggable
- **Rule**: All data stored as structured JSON with clear schema

**GitHub API Integration**
- **Rationale**: Direct integration provides real-time data and comprehensive access
- **Rule**: Multi-source authentication with fallback strategies

### Project Structure Philosophy

```
cmd/                    # CLI command implementations (Cobra pattern)
internal/              # Private implementation packages
├── ai/               # AI integration and task generation
├── github/           # GitHub API client and authentication
├── storage/          # Data persistence and task management
├── config/           # Configuration management
└── setup/            # Repository initialization
.pr-review/           # Per-repository data storage (gitignored auth)
```

**Architectural Rules:**
- **cmd/** contains only CLI interface logic
- **internal/** packages are single-responsibility focused
- **No circular dependencies between internal packages**
- **Configuration-driven behavior over hard-coded logic**

### Data Management Philosophy

**Local-First Approach:**
- All task data stored locally in repository
- No cloud dependencies for core functionality
- Git integration for sharing configuration (not sensitive data)

**State Preservation Strategy:**
- Task statuses are treated as source of truth
- Tool operations never overwrite user work progress
- Merge conflicts resolved in favor of preserving human work

### Development and Operational Rules

**Code Organization:**
- Follow Go standard project layout
- Each command gets its own file in `cmd/`
- Business logic stays in `internal/` packages
- Configuration changes require documentation updates

**CLI Design Principles:**
- Commands follow `gh` CLI patterns and conventions
- Help text includes practical examples
- Error messages provide actionable guidance
- Progressive disclosure: simple commands first, advanced features discoverable

**v3.0.0 Command Interface Improvements:**
- **Status-Specific Commands**: Intuitive task management with `start`, `done`, `hold` commands
  - `reviewtask start <task-id>` replaces verbose `reviewtask update <task-id> doing`
  - `reviewtask done <task-id>` replaces verbose `reviewtask update <task-id> done`
  - `reviewtask hold <task-id>` replaces verbose `reviewtask update <task-id> pending`
  - Traditional `update` command still supported for backward compatibility
- **Status Command Simplification**: Reduced cognitive load with clearer syntax
  - PR number changed from flag to positional argument: `status 123` instead of `status --pr 123`
  - Removed confusing flags: `--pr`, `--branch`, `--watch` (TUI functionality)
  - Kept only essential flags: `--all`, `--short`
  - Implementation: All new commands delegate to existing `update` logic, ensuring zero behavior changes

**Testing Strategy:**
- Focus on workflow testing over unit testing
- Test real user scenarios end-to-end
- Mock external dependencies (GitHub API, Claude CLI)
- Manual testing of authentication flows

**Security and Privacy:**
- Authentication tokens stored with restricted permissions
- No sensitive data in git history
- Clear separation of local vs shared configuration
- Fail securely when external services unavailable

### Configuration Management Rules

**Priority System:**
- Default rules work for most projects
- Project-specific overrides in `.pr-review/config.json`
- AI settings configurable but with sensible defaults
- User language preferences honored throughout

**AI Processing Configuration:**
- **Verbose Mode**: `"verbose_mode": true` enables detailed logging and debugging output
- **Validation Mode**: `"validation_enabled": true` enables AI-powered task validation with retries
- **Comment Chunking**: Automatic for comments >20KB, configurable chunk size
- **Prompt Size Optimization**: Pre-validation size checks prevent API failures
- **Deduplication**: AI-powered task deduplication with similarity threshold control
- **JSON Recovery**: `"enable_json_recovery": true` recovers tasks from incomplete API responses
- **Intelligent Retry**: Smart retry strategies with exponential backoff and prompt reduction
- **Response Monitoring**: Performance tracking and optimization recommendations

**Advanced Features:**
```json
{
  "ai_settings": {
    "verbose_mode": true,                  // Enable detailed debug logging
    "validation_enabled": true,            // Enable task validation with retries
    "max_retries": 5,                      // Validation retry attempts
    "quality_threshold": 0.8,              // Minimum validation score
    "deduplication_enabled": true,         // AI-powered task deduplication
    "similarity_threshold": 0.8,           // Task similarity detection threshold
    "process_nitpick_comments": false,     // Process CodeRabbit nitpick comments
    "nitpick_priority": "low",             // Priority for nitpick-generated tasks
    "enable_json_recovery": true,          // Enable recovery from incomplete JSON responses
    "max_recovery_attempts": 3,            // Maximum attempts to recover valid tasks
    "partial_response_threshold": 0.7,     // Minimum ratio for accepting partial responses
    "log_truncated_responses": true        // Log truncated responses for debugging
  }
}
```

**Extensibility Strategy:**
- Priority rules easily customizable per project
- AI processing modes configurable (parallel vs validation)
- Authentication sources tried in predictable order
- New features added with feature flags when possible
- Debug commands available for testing individual phases

### Deployment and Distribution

**Single Binary Philosophy:**
- Tool distributed as single binary
- Minimal runtime dependencies
- Cross-platform support (Linux, macOS, Windows)
- Installation via package managers preferred

**Version Management:**
- Semantic versioning for releases
- Breaking changes clearly documented
- Migration guides for configuration changes
- Backward compatibility maintained when possible

## Release Management and Versioning Rules

### Core Versioning Principles

**Semantic Versioning Compliance:**
- **MAJOR.MINOR.PATCH** format strictly enforced
- Version changes must follow semantic meaning
- Git tags use `v` prefix (e.g., `v1.2.3`)
- Version embedded in binary at build time

### Version Decision Matrix

**MAJOR Version Increment (Breaking Changes):**
- CLI command removal or incompatible changes
- Configuration file format breaking changes
- Data storage structure requiring migration
- Authentication or workflow requirement changes
- Minimum Go version or dependency changes

**MINOR Version Increment (New Features):**
- New CLI commands or subcommands
- New configuration options (backwards compatible)
- Performance improvements or new capabilities
- Support for new platforms or GitHub API features
- Enhanced AI analysis features

**PATCH Version Increment (Bug Fixes):**
- Bug fixes without functional changes
- Error message improvements
- Security or dependency updates
- Documentation improvements
- Internal refactoring

### Mandatory Version Management Commands

**Pre-Release Validation:**
```bash
# ALWAYS check current state before any version changes
./scripts/version.sh info

# ALWAYS prepare and validate before releasing
./scripts/release.sh prepare [major|minor|patch]

# ALWAYS test cross-compilation
./scripts/build.sh test
```

**Version Operations:**
```bash
# Check current version
./scripts/version.sh current

# Bump version (creates git tag automatically)
./scripts/version.sh bump [major|minor|patch]

# Create full release (GitHub + binaries)
./scripts/release.sh release [major|minor|patch]
```

### Release Process Enforcement Rules

**Pre-Release Requirements:**
1. **Clean Working Directory**: No uncommitted changes allowed
2. **Main Branch**: Must be on main branch (warnings for others)
3. **Test Validation**: All tests must pass
4. **Cross-Platform Build**: All 6 platforms must compile successfully
5. **Version Consistency**: Binary version must match git tag

**Automated Release Pipeline:**
- GitHub Actions triggered on `v*` tag push
- Cross-platform binary builds (Linux/macOS/Windows on amd64/arm64)
- Automatic release notes generation from git commits
- Checksum generation for security verification
- Draft release creation with manual approval

### Version Source Priority Hierarchy

**Version Detection Order:**
1. **Git Tags**: Exact match for current commit (`git describe --tags --exact-match`)
2. **Latest Git Tag**: Most recent tag (`git describe --tags --abbrev=0`)
3. **VERSION File**: Local version file in repository root
4. **Default**: Fallback to `0.1.0` for new repositories

### Breaking Change Management

**Deprecation Process:**
1. **Announce**: Deprecation warning in MINOR release
2. **Maintain**: Support deprecated features for 1+ major versions
3. **Remove**: Only remove in next MAJOR release
4. **Document**: Clear migration guides required

**Configuration Changes:**
- Backwards compatibility maintained within major versions
- New configuration options default to safe values
- Migration scripts provided for breaking changes
- Clear upgrade instructions in release notes

### Developer Workflow Integration

**During Development:**
```bash
# Check version before starting work
reviewtask version

# Build with version embedding
VERSION=$(./scripts/version.sh current) go build -ldflags="-X main.version=$VERSION" .
```

**Before Committing Version Changes:**
```bash
# Validate all systems
./scripts/test_versioning.sh

# Ensure clean build
./scripts/build.sh clean && ./scripts/build.sh test
```

**Release Preparation Checklist:**
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Breaking changes documented
- [ ] Migration guides written (if needed)
- [ ] Cross-platform build validated
- [ ] Release notes reviewed

### Version Embedding Standards

**Build-Time Variables:**
- `main.version`: Semantic version (e.g., "1.2.3")
- `main.commitHash`: Short git commit hash
- `main.buildDate`: RFC3339 timestamp

**Binary Version Display:**
```bash
$ reviewtask version
reviewtask version 1.2.3
Commit: abc1234
Built: 2023-12-01T10:00:00Z
Go version: go1.21.0
OS/Arch: linux/amd64
```

### Critical Development Rules

**NEVER:**
- Manually edit version numbers in source code
- Create releases without using provided scripts
- Skip cross-platform build testing
- Release with uncommitted changes
- Use lightweight git tags (always annotated)

**ALWAYS:**
- Use semantic versioning decision matrix
- Test version embedding before release
- Generate checksums for binary distributions
- Follow GitHub Actions workflow validation
- Document breaking changes clearly

### Emergency Release Process

**Hotfix Releases:**
1. Create hotfix branch from last stable tag
2. Apply minimal fix
3. Use PATCH version increment
4. Fast-track through release process
5. Merge back to main branch

**Security Releases:**
- Immediate PATCH release for security fixes
- Clear security advisory in release notes
- Coordinate with GitHub security advisories
- Provide upgrade urgency guidance

### Monitoring and Validation

**Automated Checks:**
- GitHub Actions workflow validates all builds
- Cross-compilation verified for all platforms
- Version consistency checked between git tags and binaries
- Release asset integrity validated with checksums

**Manual Verification:**
- Version command output validation
- Installation process testing on multiple platforms
- Backwards compatibility testing with previous versions
- Documentation accuracy verification

For complete versioning guidelines, see [docs/VERSIONING.md](docs/VERSIONING.md).

# important-instruction-reminders
Do what has been asked; nothing more, nothing less.
NEVER create files unless they're absolutely necessary for achieving your goal.
ALWAYS prefer editing an existing file to creating a new one.
NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.

## Version Management Critical Rules
ALWAYS follow semantic versioning rules when making any changes that affect releases.
NEVER manually edit version numbers in source code - use ./scripts/version.sh commands.
ALWAYS test cross-platform builds before any release using ./scripts/build.sh test.
ALWAYS validate version embedding in binaries before release.
NEVER skip release preparation steps - use ./scripts/release.sh prepare before release.
ALWAYS ensure clean working directory before version bumps or releases.
ALWAYS use annotated git tags with v prefix (v1.2.3) for releases.
NEVER create releases without using provided automation scripts.
