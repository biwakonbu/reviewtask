# Product Requirements Document (PRD)

## gh-review-task: AI-Powered PR Review Management Tool

### 1. Overview

A command-line tool that fetches GitHub Pull Request reviews, saves them in a structured format locally, and uses AI to analyze review content for task generation.

### 2. Core Features

#### 2.1 Command Interface
- **Command Name**: `gh-review-task`
- **Usage**:
  - `gh-review-task` - Check reviews for the current branch's PR
  - `gh-review-task <PR_NUMBER>` - Check reviews for the specified PR number
  - `gh-review-task status` - Show current task status and statistics
  - `gh-review-task update <task-id> <new-status>` - Update task status
  - `gh-review-task init` - Initialize repository for gh-review-task
  - `gh-review-task auth <login|logout|status|check>` - Authentication management

#### 2.2 Data Collection
- Fetch PR information from GitHub API
- Collect all reviews associated with the target PR
- Extract reviewer details, comments, approval status, and timestamps
- Support nested comment chains and reply threading
- Parse conversation context for AI analysis

#### 2.3 Local Storage
- **Format**: JSON
- **Structure**: PR number-based directory organization
- **Location**: `.pr-review/` directory in the project root
- **Configuration**: `.pr-review/config.json` for priority rules and project-specific settings
- **Authentication**: `.pr-review/auth.json` for local token storage (gitignored)
- **Auto-initialization**: Automatic setup with git integration

```
.pr-review/
├── config.json        # Priority rules and project settings
├── auth.json          # Local authentication (gitignored)
├── PR-123/
│   ├── info.json      # PR basic information
│   ├── reviews.json   # Review data with nested comments
│   └── tasks.json     # AI-generated tasks
└── PR-124/
    ├── info.json
    ├── reviews.json
    └── tasks.json
```

#### 2.4 AI-Powered Task Generation
- Utilize Claude Code's one-shot functionality for review analysis
- AI analyzes review content contextually (not keyword-based)
- Generate structured tasks based on review comments and suggestions
- Consider review intent and priority in task creation
- Analyze comment chains and replies to determine task relevance and resolution status
- Use configurable priority rules for project-specific task prioritization
- Support task status management workflow (todo → doing → done → pending → cancel)

#### 2.5 Authentication System
- **Multi-source Token Detection**: Environment variables → Local config → gh CLI config
- **Interactive Setup**: Comprehensive auth command suite (login/logout/status/check)
- **Permission Validation**: Repository and PR access verification
- **Error Guidance**: Detailed troubleshooting with remediation steps
- **Security**: Local token storage with restricted permissions

#### 2.6 Project Initialization
- **Auto-detection**: Prompts for setup on first use
- **Git Integration**: Automatic `.gitignore` management  
- **Configuration**: Default priority rules and settings generation
- **Validation**: Comprehensive authentication and permission checks

### 3. Technical Specifications

#### 3.1 Implementation
- **Language**: Go
- **Architecture**: CLI application
- **Dependencies**: GitHub REST API client

#### 3.2 Authentication
- **Methods**: Multiple token sources with priority hierarchy
  1. Environment variable `GITHUB_TOKEN` (highest priority)
  2. Local config `.pr-review/auth.json` (project-specific)
  3. gh CLI config `~/.config/gh/hosts.yml` (global)
- **Required Scopes**: `repo`, `pull_requests`
- **Validation**: Comprehensive permission checking with error guidance

#### 3.3 Repository Detection
- Automatic detection from local `.git` configuration
- Support for current working directory Git repository
- Validation of GitHub remote and access permissions

### 4. Data Schema

#### 4.1 info.json
```json
{
  "pr_number": 123,
  "title": "Feature: Add new functionality",
  "author": "username",
  "created_at": "2025-01-19T10:00:00Z",
  "updated_at": "2025-01-19T15:30:00Z",
  "state": "open",
  "repository": "owner/repo-name"
}
```

#### 4.2 reviews.json
```json
{
  "reviews": [
    {
      "id": 12345,
      "reviewer": "reviewer-username",
      "state": "APPROVED|CHANGES_REQUESTED|COMMENTED",
      "body": "Review comment body",
      "submitted_at": "2025-01-19T12:00:00Z",
      "comments": [
        {
          "id": 67890,
          "file": "src/main.go",
          "line": 42,
          "body": "Consider using a more descriptive variable name",
          "author": "reviewer-username",
          "created_at": "2025-01-19T12:30:00Z",
          "replies": [
            {
              "id": 67891,
              "body": "Good point, will fix in next commit",
              "author": "pr-author",
              "created_at": "2025-01-19T13:00:00Z"
            },
            {
              "id": 67892,
              "body": "Actually, let me suggest 'userAccountManager' as the name",
              "author": "reviewer-username", 
              "created_at": "2025-01-19T13:15:00Z"
            }
          ]
        },
        {
          "id": 67893,
          "file": "src/utils.go",
          "line": 15,
          "body": "This function looks good to me",
          "author": "reviewer-username",
          "created_at": "2025-01-19T14:00:00Z",
          "replies": []
        }
      ]
    }
  ]
}
```

#### 4.3 tasks.json
```json
{
  "generated_at": "2025-01-19T16:00:00Z",
  "tasks": [
    {
      "id": "task-1",
      "description": "Update variable naming in src/main.go line 42",
      "priority": "medium",
      "source_review_id": 12345,
      "file": "src/main.go",
      "line": 42,
      "status": "todo",
      "created_at": "2025-01-19T16:00:00Z",
      "updated_at": "2025-01-19T16:00:00Z"
    }
  ]
}
```

#### 4.4 config.json
```json
{
  "priority_rules": {
    "critical": "Security vulnerabilities, authentication bypasses, data exposure risks",
    "high": "Performance bottlenecks, memory leaks, database optimization issues", 
    "medium": "Functional bugs, logic improvements, error handling",
    "low": "Code style, naming conventions, comment improvements"
  },
  "project_specific": {
    "critical": "Additionally consider: payment processing errors, user data validation",
    "high": "API response time over 500ms, concurrent user handling issues"
  },
  "task_settings": {
    "default_status": "todo",
    "auto_prioritize": true
  }
}
```

### 5. Task Management

#### 5.1 Task Status Workflow
- **todo**: Ready to start
- **doing**: Currently in progress
- **done**: Completed
- **pending**: Needs evaluation (whether to address or not)
- **cancel**: Decided not to address

#### 5.2 Status Command Output
- **Current Tasks**: List of tasks with `doing` status
- **Next Tasks**: Top priority `todo` tasks (sorted by priority)
- **Statistics**: 
  - Status breakdown (todo/doing/done/pending/cancel counts)
  - Priority breakdown (critical/high/medium/low counts)
  - PR breakdown (task counts per PR)
  - Completion rate (done / total tasks)

#### 5.3 Task Update Commands
- **Individual Updates**: `gh-review-task update <task-id> <new-status>`
- **Status Validation**: Only valid status transitions allowed
- **Timestamp Tracking**: Automatic `updated_at` field management

### 6. Authentication Commands

#### 6.1 Authentication Management
- **login**: Interactive token setup with local storage
- **logout**: Remove local authentication (preserves gh CLI config)
- **status**: Show current authentication source and user
- **check**: Comprehensive validation of token and permissions

#### 6.2 Permission Validation
- **Repository Access**: Test read access to target repository
- **Pull Request Access**: Verify PR listing and review capabilities
- **Scope Detection**: Identify available token permissions
- **Error Reporting**: Specific guidance for permission issues

### 7. Project Initialization

#### 7.1 Setup Workflow
- **Detection**: Automatic check for existing `.pr-review/` setup
- **Prompting**: Interactive confirmation for first-time users
- **Directory Creation**: `.pr-review/` structure generation
- **Git Integration**: Automatic `.gitignore` entry addition
- **Configuration**: Default priority rules and settings
- **Validation**: Authentication and permission verification

#### 7.2 Initialization Commands
- **Manual Init**: `gh-review-task init` for explicit setup
- **Auto-prompt**: Triggered on first use of main commands
- **Reinitialize**: Option to recreate configuration files

### 8. User Experience

#### 8.1 Success Scenarios
- **First-time Setup**: Guided initialization with clear progress indicators
- **Routine Usage**: Seamless execution with informative output
- **Task Management**: Intuitive status display and update workflows
- **Authentication**: Self-guided troubleshooting and resolution

#### 8.2 Error Handling
- **Repository Issues**: PR not found, invalid PR numbers, repository access
- **Authentication Issues**: Missing tokens, invalid credentials, insufficient permissions
- **Network Issues**: API connectivity, rate limiting, timeout handling
- **Configuration Issues**: Invalid settings, corrupted files, permission problems

#### 8.3 User Guidance
- **Progressive Disclosure**: Basic → Advanced features as needed
- **Context-aware Help**: Specific error messages with actionable solutions
- **Consistent Patterns**: Following `gh` CLI conventions and terminology
- **Self-recovery**: Automatic detection and resolution of common issues

### 6. Future Considerations

#### 6.1 Potential Enhancements
- Integration with existing task management tools
- Support for multiple repository monitoring
- Review status tracking and updates
- Automated task completion detection

#### 6.2 Extensibility
- Plugin architecture for custom AI analyzers
- Configurable output formats
- Custom task generation rules

### 7. Success Criteria

- Successfully fetch and store PR review data
- Generate meaningful tasks from review content using AI
- Maintain structured local storage for easy access
- Provide clear command-line interface for users
- Handle edge cases and errors gracefully