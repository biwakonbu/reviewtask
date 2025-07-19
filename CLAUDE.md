## Project Language

- Memories and documents should be written in English
- Conversations should be conducted in the language specified by the user

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
   gh-review-task
   ```

3. **Review What Needs to be Done**
   ```bash
   # See current work or next recommended task
   gh-review-task show
   ```

4. **Work on Tasks Systematically**
   ```bash
   # Start working on a task
   gh-review-task update <task-id> doing
   
   # Complete implementation
   gh-review-task update <task-id> done
   ```

5. **Handle Updated Reviews**
   ```bash
   # Re-run when reviewers add new comments
   gh-review-task
   # Tool automatically preserves your work progress
   ```

### 2. Daily Development Routine

**Morning Startup:**
```bash
gh-review-task show           # What should I work on today?
gh-review-task status         # Overall progress across all PRs
```

**During Implementation:**
```bash
gh-review-task show <task-id> # Full context for current task
# Work on the task...
gh-review-task update <task-id> done
```

**When Blocked:**
```bash
gh-review-task update <task-id> pending  # Mark as blocked
gh-review-task show                      # Find next task to work on
```

### 3. Team Collaboration Rules

**For PR Authors:**
- Run `gh-review-task` immediately after receiving reviews
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

**Extensibility Strategy:**
- Priority rules easily customizable per project
- AI processing modes configurable (parallel vs validation)
- Authentication sources tried in predictable order
- New features added with feature flags when possible

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
