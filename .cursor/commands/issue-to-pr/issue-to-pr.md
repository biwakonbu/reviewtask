# Issue to PR Workflow

## 🎯 Core Principle: Complete Automation of Issues-Driven Development

**Create systematic PRs and implementation from specified GitHub Issues numbers, realizing reliable development with emphasis on specification testing.**

## 🚀 AUTOMATIC RESUME EXECUTION

**Detecting current context and executing resume logic...**

Current branch: `${CURRENT_BRANCH}`
Detected Issue number: **${ISSUE_NUMBER}**
PR Status: **${PR_STATUS}**

**✅ RESUMING ISSUE #${ISSUE_NUMBER} WORKFLOW FROM CURRENT STATE**

Proceeding with Issue #${ISSUE_NUMBER} implementation workflow. Analyzing current progress and determining next steps...

## 🔄 Resume Functionality

**When executed without arguments:**
- Automatically detect current branch and associated Issue/PR
- Resume workflow from the appropriate stage based on current state
- Continue implementation where it was left off
- Maintain full workflow compliance including mandatory recitations

**IMPLEMENTATION NOTE**: This command automatically executes the Resume Detection Phase when no arguments are provided.

## 📊 Development Workflow

```mermaid
graph TD
    A[Issue Specification] --> B[Branch Preparation]
    B --> C[Draft PR Creation]
    C --> D[Command Content Recitation]
    D --> E[Item-by-Item Implementation]
    E --> F[Provisional Implementation Commit]
    F --> G[Push & PR Update]
    G --> H[Unit Test Creation]
    H --> I[Integration Test Creation]
    I --> J[Test Execution & Fix]
    J --> K[Test Commit & Push]
    K --> L[Progress Report & Command Recitation]
    L --> M{Other Items Exist?}
    M -->|Yes| E
    M -->|No| N[Final Confirmation]
    N --> O[Draft PR to Open]
    O --> P[Issue Update & Completion]

    classDef setupNode fill:#e3f2fd,stroke:#1976d2,stroke-width:2px,color:#000
    classDef implNode fill:#f1f8e9,stroke:#388e3c,stroke-width:2px,color:#000
    classDef testNode fill:#fff3e0,stroke:#f57c00,stroke-width:2px,color:#000
    classDef finalNode fill:#fce4ec,stroke:#c2185b,stroke-width:2px,color:#000

    class A,B,C,D setupNode
    class E,F,G implNode
    class H,I,J,K,L testNode
    class M,N,O,P finalNode
```

## 🚀 Execution Steps

### 0. Resume Detection Phase (When No Arguments Provided)

**Automatic State Detection and Execution:**

First, determine current context:
- Current branch: `$(git branch --show-current)`
- Extract Issue number from branch name: `$(git branch --show-current | grep -o 'issue-[0-9]\+' | grep -o '[0-9]\+')`
- Check for existing PR: `$(gh pr view --json number,state --jq '{number: .number, state: .state}' 2>/dev/null)`

**Resume Logic Implementation:**

1. **Extract Issue Number from Current Branch:**
   ```bash
   CURRENT_BRANCH=$(git branch --show-current)
   ISSUE_NUMBER=$(echo "$CURRENT_BRANCH" | grep -o 'issue-[0-9]\+' | grep -o '[0-9]\+')
   ```

2. **Check PR Status:**
   ```bash
   PR_INFO=$(gh pr view --json number,state,title --jq '{number: .number, state: .state, title: .title}' 2>/dev/null)
   ```

3. **Execute Resume Decision:**
   - If Issue number detected AND PR exists: Resume from current PR state
   - If Issue number detected AND no PR: Start fresh workflow for detected Issue
   - If no Issue number detected: Show error and usage information

**Current Context Analysis:**
- Branch: `feature/issue-6-organize-pr-review-commands`
- Detected Issue: #6
- PR Status: #18 (OPEN/DRAFT)
- Action: Continue Issue #6 workflow from current state

**IMMEDIATE EXECUTION:**
When this command is invoked without arguments, the following detection and resume logic will execute automatically:

```bash
# Execute detection logic
CURRENT_BRANCH=$(git branch --show-current)
ISSUE_NUMBER=$(echo "$CURRENT_BRANCH" | grep -o 'issue-[0-9]\+' | grep -o '[0-9]\+')
PR_INFO=$(gh pr view --json number,state,title --jq '{number: .number, state: .state, title: .title}' 2>/dev/null)

# Current context: Issue #6, PR #18 exists
# Resume action: Continue with Issue #6 implementation workflow
```

**State-Based Resume Points:**
- **No PR exists**: Start from Initial Preparation Phase
- **Draft PR exists, no commits**: Resume from Command Recitation
- **Draft PR with commits, no tests**: Resume from Test Implementation Phase  
- **Draft PR with tests**: Resume from Final Completion Phase
- **Open PR exists**: Workflow already complete

### 1. Initial Preparation Phase

**Branch and PR Preparation:**
- Move to default branch (main)
- Update local branch to latest state (`git pull origin main`)
- Fetch and analyze specified Issues content
- Create appropriate branch name based on Issue content
- Create new branch
- Create Draft PR and ensure Issue link is properly set

**Initial Command Content Recitation**
- **MANDATORY**: After Draft PR creation, recite the ENTIRE issue-to-pr command file content word-for-word
- This ensures complete understanding of the workflow before proceeding
- Only proceed to implementation after full recitation is complete

**GitHub Issue-PR Linking Enhancement**
- **AUTOMATIC**: Add GitHub closing keywords to PR description during Draft PR creation
- Use keywords like "Closes #[ISSUE_NUMBER]", "Fixes #[ISSUE_NUMBER]", or "Resolves #[ISSUE_NUMBER]"
- This ensures automatic Issue closure when PR is merged
- Format example: "This PR closes #42 by implementing the requested metrics export functionality."

### 2. Item-by-Item Implementation Phase

**Progressive Implementation:**
- Organize Issues response items in order
- Implement provisional implementation for each item
- Commit immediately after provisional implementation completion
- Execute push after commit
- Update Draft PR description to clearly show progress status

### 3. Comprehensive Test Implementation Phase

**Automated Test Creation with Mock Strategy:**

**Unit Test Implementation:**
- Create comprehensive unit tests for each implemented component
- Use appropriate mocking frameworks for external dependencies
- Test edge cases, error conditions, and boundary values
- Ensure high code coverage (aim for >90% for new code)
- Mock external APIs, file systems, and network calls appropriately
- Validate internal logic independently from external systems

**Integration Test Implementation:**
- Create integration tests where specification requires system interaction
- Test component interactions and data flow
- Mock only external dependencies outside the system boundary
- Verify end-to-end functionality for critical user workflows
- Test error handling and recovery scenarios

**Specification Compliance Testing:**
- **Priority**: Verify if specifications are met through automated tests
- Create test cases that directly validate acceptance criteria
- Execute all tests (unit + integration) and verify results
- Fix implementation if tests reveal oversights or errors
- **Important**: Prioritize achieving correct specification state validated by tests
- Commit & push after all tests pass
- PR progress report: Include test coverage and specification compliance status

### 4. Progress Report Phase

**🚨 CRITICAL MANDATORY: Execution Command Content Recitation**

**ABSOLUTE REQUIREMENT - ZERO TOLERANCE FOR OMISSION:**

**After EVERY implementation item completion (including tests), MUST execute:**

1. **Complete Command Recitation:**
   - **RECITE THE ENTIRE issue-to-pr command file content WORD-FOR-WORD**
   - Include EVERY section from "Core Principle" to "Related Information"
   - NO SUMMARIES, NO ABBREVIATIONS, NO SHORTCUTS ALLOWED
   - **FAILURE TO RECITE = WORKFLOW VIOLATION**

2. **Recitation Verification Checklist:**
   - [ ] Started with "# Issue to PR Workflow"
   - [ ] Included complete Core Principle section
   - [ ] Covered entire Development Workflow diagram
   - [ ] Detailed all 5 Execution Steps
   - [ ] Covered all Usage Examples
   - [ ] Included all Important Notes and Requirements
   - [ ] Ended with Related Information section
   - [ ] **TOTAL RECITATION COMPLETED**

3. **Post-Recitation Confirmation:**
   - State: "✅ COMPLETE COMMAND RECITATION FINISHED"
   - Only then proceed to next implementation item
   - **NO CONTINUATION WITHOUT EXPLICIT RECITATION COMPLETION**

**WORKFLOW ENFORCEMENT:**
- **CANNOT PROCEED** to next implementation without recitation
- **CANNOT ADVANCE** to Final Completion Phase without recitation
- **MUST STOP** if recitation is incomplete or omitted
- This ensures workflow adherence and prevents deviation from specified process

**Repeat this MANDATORY recitation flow after EVERY implementation item until ALL specified Issue requirements are complete**

### 5. Final Completion Phase

**Completion and Quality Assurance:**
- Final commit & push of all implementation items and tests
- Verify branch implementation content with `git log`
- Run complete test suite to ensure all tests pass
- Update PR description to latest status with test coverage report
- Record remaining issues in PR comments if any

**🚨 CRITICAL MANDATORY: Draft PR to Open Conversion**

**AUTOMATED EXECUTION REQUIRED - NO EXCEPTIONS:**

1. **Immediate Draft Removal:**
   ```bash
   # MUST EXECUTE THIS COMMAND AUTOMATICALLY
   gh pr ready <PR_NUMBER>
   ```

2. **Conversion Verification:**
   ```bash
   # MUST VERIFY PR STATUS CHANGED TO OPEN
   gh pr view <PR_NUMBER> --json state --jq '.state'
   # Expected output: "OPEN"
   ```

3. **Failure Handling:**
   - If `gh pr ready` fails: STOP workflow and report error
   - If PR state is not "OPEN": Re-attempt conversion
   - If still fails: Manual intervention required with clear error message

**CONVERSION CHECKLIST - ALL MUST BE COMPLETED:**
- [ ] Execute `gh pr ready <PR_NUMBER>` command
- [ ] Verify PR state changed from "DRAFT" to "OPEN"
- [ ] Confirm PR title and description accurately reflect completed work
- [ ] Add appropriate labels and reviewers as needed
- [ ] Document successful conversion in workflow log

**WORKFLOW CANNOT PROCEED TO ISSUE UPDATE WITHOUT SUCCESSFUL PR CONVERSION**

**Final Issue Update:**
- Update Issue status to completed/closed
- Link the merged PR in Issue comments
- Document any additional notes or follow-up items discovered during implementation

## 📝 Usage Examples

### Fresh Start with Issue Number
```bash
# Start implementing functionality for specified Issue
/issue-to-pr 42

# Execution result example (for Issue #42):
# 1. Move to main branch and update
# 2. Create feature/issue-42-add-metrics-export branch
# 3. Create Draft PR (linked with Issue #42, includes "Closes #42" in description)
# 4. **RECITE ENTIRE COMMAND CONTENT**
# 5. Provisional implementation of metrics collection functionality
# 6. Unit tests for metrics collection with mocks
# 7. Integration tests for metrics collection
# 8. Commit, push, and progress report
# 9. **RECITE ENTIRE COMMAND CONTENT**
# 10. Provisional implementation of export functionality  
# 11. Unit tests for export functionality with mocks
# 12. Integration tests for export functionality
# 13. Commit, push, and progress report
# 14. Final test suite execution and coverage verification
# 15. **Convert Draft PR to Open status**
# 16. Final confirmation and Issue completion
```

### Resume Existing Work (No Arguments)
```bash
# Resume work on current branch automatically
/issue-to-pr

# Automatic detection and resume examples:

# Scenario 1: On feature/issue-42-add-metrics-export branch with Draft PR
# Result: Auto-detects Issue #42, resumes from current stage

# Scenario 2: On feature/issue-42-add-metrics-export branch, implementation started
# Result: Auto-detects Issue #42, resumes from Test Implementation Phase

# Scenario 3: On main branch
# Result: ❌ Cannot detect Issue number from current context

# Scenario 4: On feature/issue-42-add-metrics-export branch with Open PR
# Result: ✅ Workflow already complete for Issue #42
```

## 📋 Workflow Compliance Checklist

**🚨 CRITICAL CHECKPOINTS - MUST BE VERIFIED AT EACH STAGE**

### Phase 1: Initial Preparation Compliance
- [ ] Moved to main branch successfully
- [ ] Pulled latest changes (`git pull origin main`)
- [ ] Created appropriate branch name (format: `feature/issue-N-description`)
- [ ] Created Draft PR with correct Issue link
- [ ] **COMPLETED INITIAL COMMAND RECITATION** ✅

### Phase 2: Implementation Compliance (PER ITEM)
- [ ] Implemented specific functionality item
- [ ] Created comprehensive unit tests with mocks
- [ ] Created integration tests where required
- [ ] All tests passing (>90% coverage target)
- [ ] Committed changes with descriptive message
- [ ] Pushed changes to remote branch
- [ ] Updated PR description with progress
- [ ] **COMPLETED POST-IMPLEMENTATION COMMAND RECITATION** ✅

### Phase 3: Final Completion Compliance
- [ ] All implementation items completed
- [ ] Complete test suite passing
- [ ] Final commit and push executed
- [ ] PR description fully updated
- [ ] **EXECUTED `gh pr ready <PR_NUMBER>` COMMAND** ✅
- [ ] **VERIFIED PR STATE CHANGED TO "OPEN"** ✅
- [ ] Issue status updated to completed/closed
- [ ] **COMPLETED FINAL COMMAND RECITATION** ✅

### Workflow Violation Responses
**If ANY checklist item is incomplete:**
1. **STOP WORKFLOW IMMEDIATELY**
2. **IDENTIFY MISSING ITEM(S)**
3. **COMPLETE MISSING REQUIREMENTS**
4. **VERIFY COMPLETION WITH CHECKLIST**
5. **ONLY THEN CONTINUE WORKFLOW**

**Common Violations and Fixes:**
- **Missing Recitation**: Execute complete command file recitation immediately
- **Draft PR Not Converted**: Execute `gh pr ready <PR_NUMBER>` and verify
- **Incomplete Tests**: Create missing tests and achieve coverage requirements
- **Uncommitted Changes**: Commit and push all changes before proceeding

**ZERO TOLERANCE POLICY: No workflow advancement without checklist completion**

## ⚠️ Important Notes

**Resume Functionality Requirements:**
- **AUTO-DETECTION**: When no arguments provided, automatically detect Issue from current context
- **STATE AWARENESS**: Resume from appropriate workflow stage based on current progress
- **BRANCH PATTERN**: Supports branch names like `feature/issue-N-description` or `issue-N-description`
- **PR STATE**: Analyzes Draft/Open PR status to determine resume point

**🤖 AUTOMATED EXECUTION REQUIREMENTS:**

**Command Content Recitation - AUTOMATED:**
- **AUTO-TRIGGER**: After Draft PR creation (fresh start or resume)
- **AUTO-TRIGGER**: After each implementation item completion
- **AUTO-EXECUTION**: Recite ENTIRE command file content word-for-word
- **AUTO-VERIFICATION**: Confirm recitation includes all sections
- **FAILURE HANDLING**: STOP workflow if recitation incomplete

**Test Implementation - AUTOMATED:**
- **AUTO-CREATE**: Comprehensive unit tests with appropriate mocking
- **AUTO-CREATE**: Integration tests where specification requires
- **AUTO-EXECUTE**: Run all tests and verify >90% coverage
- **AUTO-VALIDATE**: Ensure all tests pass before proceeding
- **FAILURE HANDLING**: Fix implementation until tests pass

**Draft PR Conversion - AUTOMATED:**
- **AUTO-EXECUTE**: `gh pr ready <PR_NUMBER>` command
- **AUTO-VERIFY**: PR state changed from "DRAFT" to "OPEN"
- **AUTO-RETRY**: Re-attempt if initial conversion fails
- **FAILURE HANDLING**: STOP workflow and report error if conversion fails

**👁️ MANUAL VERIFICATION CHECKPOINTS:**

**Recitation Verification:**
- [ ] Verify complete command content was recited
- [ ] Confirm all sections included (Core Principle → Related Information)
- [ ] Validate recitation before proceeding to next step

**Test Coverage Verification:**
- [ ] Review test files created for each component
- [ ] Confirm mocking strategy appropriate for external dependencies
- [ ] Validate test coverage reports show >90% for new code

**PR Conversion Verification:**
- [ ] Confirm PR shows "OPEN" status in GitHub UI
- [ ] Verify PR title and description accurately reflect work
- [ ] Validate all commits and tests included in PR

**Specification Test Priority Principle:**
- Prioritize correct specification implementation validated by automated tests
- Accurately report test coverage and specification compliance status
- Ensure all tests pass before final completion

**Thorough Issue・PR Integration:**
- Always set Issue link when creating Draft PR
- **CRITICAL**: Include GitHub closing keywords in PR description (e.g., "Closes #[ISSUE_NUMBER]")
- This enables automatic Issue closure when PR is merged
- Maintain accuracy and transparency of progress reports with test results
- Clear identification and proper recording of remaining issues

**Continuous Quality Assurance:**
- Emphasize gradual completion of each implementation item with tests
- Regular verification of implementation content through git log and test execution
- Continuous updating of PR description with test coverage information

## 📚 Related Information

- Automation using GitHub Issues API
- Progressive development utilizing Draft PR functionality
- Comprehensive automated testing with mocking strategies
- Test-Driven Development and Specification-Driven Testing methodology
- Git workflow optimization patterns
- Draft PR to Open conversion best practices