# v3.0.0 Concept Document

## Overview

This document outlines the v3.0.0 design for simplifying workflows and adding a modern guidance system.

**Related Issues:**
- [#191](https://github.com/biwakonbu/reviewtask/issues/191) - [Phase 1] Implement unresolved review comment detection
- [#192](https://github.com/biwakonbu/reviewtask/issues/192) - [Phase 2] Implement comprehensive comment analysis with AI impact assessment
- [#193](https://github.com/biwakonbu/reviewtask/issues/193) - [Phase 3] Command integration and flag simplification (v3.0.0)
- [#194](https://github.com/biwakonbu/reviewtask/issues/194) - [Phase 1] Implement modern UI and guidance system
- [#195](https://github.com/biwakonbu/reviewtask/issues/195) - [Phase 4] Implement done command automation routine

### Goals

1. Reduce command count and simplify workflow
2. Remove complex flags
3. Natural guidance through context-aware assistance
4. Modern and clean UI

## Major Changes

### Command Integration and Simplification

| Before (v2.x) | After (v3.0.0) | Effect |
|--------------|----------------|--------|
| `fetch` + `analyze` | `reviewtask` | 2 commands → 1 command |
| `update <id> doing` | `start <id>` | Intuitive |
| `update <id> done` | `done <id>` | Clear |
| `update <id> pending` | `hold <id>` | Clear |
| `complete` | Removed (merged into `done`) | Eliminate duplication |
| `status --pr 123` | `status 123` | Argument-based |

### Flag Reduction

- `analyze --batch-size`, `--max-batches`, `--async` → Removed (auto-optimization)
- `status --pr`, `--branch`, `-w` → Removed
- `status` maintains only `--all`, `--short`

## Phase 1: Command Integration

### `reviewtask` (fetch + analyze integration)

#### Specification
```bash
reviewtask [PR_NUMBER]
```

- Auto-detect from current branch when PR number omitted
- No flags (all auto-optimized)

#### Behavior
1. **Fetch PR reviews**
2. **AI analysis for task generation** (auto batch size)
   - **Analyze all comments** (including minor suggestions)
   - **Impact assessment**: Large changes → PENDING / Small changes → TODO
3. **Task merge** (preserve existing status)
4. **Unresolved comment detection** (new feature)
5. **Guidance output**

#### Task Generation Policy (Important)

**Basic Policy: Generate tasks from all review comments**

Previously, only "actionable comments" were extracted, which led to minor suggestions being overlooked.

**New Approach:**
1. **Analyze all review comments**
   - Include nitpicks (minor suggestions)
   - Include questions
   - Include suggestions and recommendations

2. **AI-based impact assessment and automatic status assignment**
   - **TODO**: Small changes (typo fixes, renaming, adding comments, formatting)
   - **PENDING**: Large changes (design changes, architecture changes, major refactoring)

3. **Handling PENDING tasks**
   - Require explicit user decision
   - Review details with `reviewtask show`, then choose `start` or `cancel`
   - Prompt for PENDING tasks after all TODO tasks are completed

**Impact Assessment Criteria:**

| Impact | Initial Status | Examples |
|--------|---------------|----------|
| Small | TODO | Typo fixes, variable renaming, adding comments, formatting |
| Medium | TODO | Simple logic fixes, adding error handling, adding validation |
| Large | PENDING | Design changes, new features, major refactoring, API changes |

#### Unresolved Comment Detection Integration

```
reviewtask

Fetching reviews...
  ✓ Fetched 8 review comments

Analyzing review comments...
  ✓ Created 6 new tasks from 5 comments
  → 2 tasks set to TODO (small changes)
  → 1 task set to PENDING (requires design change)
  → 3 comments already have tasks

Unresolved Threads Check
────────────────────────
✓ All review comments have been analyzed
! 5 threads still unresolved
  → Complete tasks to resolve threads

Tasks Summary
─────────────
  TODO: 2
  PENDING: 1 (requires your decision)
  DOING: 0
  DONE: 3

Progress [████████░░░░░░░░░░░░] 38% (3/8)

Next Steps
──────────
→ Start working on TODO tasks
  reviewtask show       # See next recommended task

! You have 1 PENDING task requiring decision
  reviewtask show <pending-task-id>
```

## Phase 2: Status-Specific Commands

### New Commands

```bash
reviewtask start <task-id>    # Start task
reviewtask done <task-id>     # Complete task (automation routine)
reviewtask hold <task-id>     # Hold task
reviewtask cancel <task-id>   # Cancel task
```

### `done` Command Automation Routine

#### Complete Flow
1. **Verification**: Build/Test/Lint
2. **Auto-commit**: Structured message
3. **Thread resolution**: When all tasks from same comment are completed
4. **Next task suggestion**: Context-aware guidance

#### Commit Message Format

**Language**: Written in user's configured language (follows `language` setting in config.json)

```
<Task summary>

Review Comment: <URL>

Original Comment:
> <Comment quote>

Changes:
- <Change details>

Implementation Notes:
<Implementation approach>

PR: #<pr-number>
```

**Example (Japanese configuration):**
```
変数名をより明確に変更

Review Comment: https://github.com/user/repo/pull/123#discussion_r456789

Original Comment:
> この変数名は少し分かりにくいです。もっと明確な名前にできますか？

Changes:
- `data` を `userData` に変更
- 関連する型定義も更新

Implementation Notes:
変数のスコープと用途を明確にするため、より具体的な名前に変更しました。

PR: #123
```

#### Configuration Example
```json
{
  "done_workflow": {
    "enable_verification": true,
    "enable_auto_commit": true,
    "enable_auto_resolve": "complete",
    "enable_next_task_suggestion": true
  }
}
```

## Phase 3: `status` Simplification and Unresolved Comment Detection

### Specification
```bash
reviewtask status [PR_NUMBER]
```

#### Flags (only 2)
- `--all, -a`: Show all PRs/detailed view
- `--short, -s`: Brief version

#### Flags to Remove
- `--pr INT` → Replaced with argument
- `--branch STRING` → Handled by branch switching
- `-w, --watch` → TUI removed

### Important Feature Addition: Unresolved Comment Detection

#### Problem Recognition
Current implementation allows AI to overlook new review comments or unresolved comments, leading to incorrect judgment that "nothing needs to be done."

#### Solution
Detect and display the following when running `status` or `fetch`/`analyze`:

1. **Unanalyzed comments**: Exist on GitHub but tasks not yet generated
2. **In-progress comments**: Tasks generated but not completed
3. **Resolved comments**: All tasks completed & GitHub thread resolved

#### Clear Completion Criteria

PR review response is considered "complete" when:
```
✓ All review comments analyzed
✓ All tasks completed
✓ All GitHub threads resolved
```

#### UI Example: With Unresolved Comments

```
reviewtask status

Fetching latest PR state...
  ✓ PR #123 fetched

Review Status
─────────────
Unresolved Comments: 3
  ! 2 comments not yet analyzed
  → 1 comment with pending tasks

Tasks
─────
  TODO: 2
  DOING: 1
  DONE: 4
  HOLD: 0

Progress [██████░░░░░░░░░░░░░░] 30% (3/10)

Next Steps
──────────
! You have unresolved review comments
  reviewtask analyze    # Analyze new comments and create tasks
```

#### UI Example: All Complete

```
reviewtask status

Fetching latest PR state...
  ✓ PR #123 fetched

Review Status
─────────────
✓ No unresolved comments
✓ All tasks completed
✓ All threads resolved

You're all done!

Next Steps
──────────
→ Push your changes
  git push

→ Check for new reviews
  reviewtask
```

#### データ構造拡張

**reviews.json に追加するフィールド:**
```json
{
  "pr_number": 123,
  "comments": [
    {
      "id": 12345,
      "body": "...",
      "github_thread_resolved": false,
      "last_checked_at": "2025-01-05T10:00:00Z",
      "tasks_generated": true,
      "all_tasks_completed": false
    }
  ]
}
```

## Phase 4: Modern UI Design

### Design Principles

1. **Minimal**: No emojis or decorations
2. **Clean**: Simple lines and spacing
3. **Clear**: Hierarchy through typography
4. **Modern**: GitHub CLI-inspired

### Basic Components

#### Section Divider
```
Section Title
─────────────
```

#### Status Symbols
```
✓  Success
✗  Error
→  Next step
!  Warning
```

#### Progress Bar
```
Progress [████████░░░░░░░░░░░░] 40% (4/10)
```

### UI Example: `done` Command Success

```
reviewtask done abc123

Verifying task completion...
  ✓ Build passed
  ✓ Tests passed (14/14)

Creating commit...
  ✓ Created commit a1b2c3d

Resolving review thread...
  ✓ Thread resolved

Task abc123 completed

Progress
────────
4 of 10 tasks complete (40%)

Next Task
─────────
def456  HIGH  Add error handling for edge cases

Next Steps
──────────
→ Continue with next task
  reviewtask show

→ Start immediately
  reviewtask start def456
```

## Phase 5: Context-Aware Guidance

### Design Principle

After each command execution, clearly present context-appropriate next actions.

### Format

```
[Command execution result]

Next Steps
──────────
→ [Highest priority action]
  [Specific command]

→ [Alternative action]
  [Alternative command]
```

### Guidance Patterns

#### After New Task Generation (with TODO)
```
→ Start working on TODO tasks
  reviewtask show       # See next recommended task

→ View all tasks
  reviewtask status
```

#### After New Task Generation (with PENDING)
```
→ Start working on TODO tasks first
  reviewtask show

! You have PENDING tasks requiring decision
  Review them after completing TODO tasks
```

#### After Task Completion (next TODO exists)
```
→ Continue with next task
  reviewtask show

→ Start immediately
  reviewtask start <next-task-id>
```

#### TODO Complete, PENDING Exists
```
! All TODO tasks completed

You have PENDING tasks requiring your decision:

PENDING Tasks
─────────────
abc123  Refactor authentication module
def456  Consider using dependency injection

Next Steps
──────────
→ Review PENDING tasks
  reviewtask show abc123

→ Decide: start or cancel
  reviewtask start abc123   # Start working
  reviewtask cancel abc123  # Skip this task
```

#### All Complete (TODO + PENDING)
```
✓ All tasks completed

→ Push your changes
  git push

→ Check for new reviews
  reviewtask

Your PR is ready for final review
```

## Ideal Workflow (v3.0.0)

```bash
# 1. Fetch and analyze reviews
reviewtask

# 2. Check next task
reviewtask show

# 3. Start working
reviewtask start abc123

# 4. Implementation work
# ... コーディング ...

# 5. Stage changes
git add <files>

# 6. Complete task (auto: verify→commit→resolve→suggest next)
reviewtask done abc123

# 7. Loop or complete
```

**Guidance directs to next action at each step**

## Implementation Package Structure

```
cmd/
  start.go          # start command
  done.go           # done command (automation routine)
  hold.go           # hold command
  pending.go        # pending task dialog
  status.go         # status command (unresolved comment detection added)

internal/
  ui/
    ui.go           # UI components
    colors.go       # Color definitions
    formatting.go   # Formatting functions

  guidance/
    guidance.go     # Guidance system
    detector.go     # Context detection
    formatter.go    # Output formatting

  verification/
    verification.go # Task verification

  git/
    commit.go       # Auto-commit
    templates.go    # Commit message templates

  github/
    client.go       # GitHub API client
    threads.go      # Thread resolution state (new)
    comments.go     # Comment fetch and compare (extended)
```

## Versioning Strategy

### v2.5.0 (Deprecation)
- Add integrated commands
- Add warnings to existing commands
- Update documentation

### v3.0.0 (Major Release)
- Apply breaking changes
- Remove flags
- Integrate commands
- Enable guidance system
- Apply modern UI

### Backward Compatibility

Maintain `fetch`, `analyze`, `update` as aliases (v3.0.0):
```bash
reviewtask fetch 123    # Internally executes 'reviewtask 123'
reviewtask analyze 123  # Does nothing (completed in fetch)
reviewtask update <id> doing  # Delegates to 'start'
```

## Success Metrics

- Commands: 15 → 12 (-20%)
- Flags: 20+ → 10 or less (-50%+)
- Basic workflow: 5 steps → 3 steps (-40%)
- **Review comment oversight: Reduced (by analyzing all comments)**
- **Unaddressed cancellation rate: Reduced (by prompting user decision for PENDING status)**
- Improved first-time user success rate
- Improved AI assistant command suggestion accuracy

## Implementation Roadmap

### Milestone 1: Foundation Implementation (v2.4.0)
- [ ] Create UI component package
- [ ] Guidance system foundation
- [ ] **AI impact assessment** (TODO/PENDING auto-assignment)
- [ ] **Comprehensive comment analysis mode** (including minor suggestions)
- [ ] Unresolved comment detection (GitHub API integration)
- [ ] Data structure extension (reviews.json)
- [ ] Integrated command implementation (including unresolved comment detection)
- [ ] Enhance status command (fetch latest PR state)
- [ ] Implement new commands (start/done/hold)
- [ ] **PENDING task interactive flow**

### Milestone 2: Deprecation (v2.5.0)
- [ ] Add warnings to existing commands
- [ ] Update documentation
- [ ] Create migration guide

### Milestone 3: v3.0.0 Release
- [ ] Remove flags
- [ ] Implement aliases
- [ ] Fully apply modern UI
- [ ] Fully integrate guidance system
- [ ] Create release notes

## References

- [Issues #191-195](https://github.com/biwakonbu/reviewtask/issues) - Implementation tracking
- [CLAUDE.md](../../CLAUDE.md) - Project instructions and philosophy
- [v3 README](./README.md) - v3.0.0 overview and timeline
