# Product Requirements Document (PRD)

## reviewtask: AI-Powered PR Review Management Tool

### 1. Overview

A command-line tool that fetches GitHub Pull Request reviews, saves them in a structured format locally, and uses AI to analyze review content for task generation.

### 2. Core Features

#### 2.1 Command Interface
- **Command Name**: `reviewtask`
- **Usage**:
  - `reviewtask [PR_NUMBER]` - Check reviews for current branch or specified PR number
  - `reviewtask --refresh-cache` - Clear cache and reprocess all comments
  - `reviewtask status [options]` - Show current task status and statistics
  - `reviewtask show [task-id]` - Show current/next task or specific task details
  - `reviewtask stats [PR_NUMBER] [options]` - Show detailed task statistics with comment breakdown
  - `reviewtask update <task-id> <new-status>` - Update task status
  - `reviewtask version [VERSION]` - Show version information or switch to specific version
  - `reviewtask versions` - List available versions from GitHub releases
  - `reviewtask claude <target>` - Generate Claude Code integration templates
  - `reviewtask init` - Initialize repository for reviewtask
  - `reviewtask auth <login|logout|status|check>` - Authentication management

##### Command Options
- **Global Options**: `--refresh-cache` for cache management
- **Filtering Options**: `--all`, `--pr <number>`, `--branch <name>` for status/stats commands
- **Version Options**: `--check` for update checking

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

#### 2.7 Advanced Analytics and Statistics
- **Detailed Statistics**: Comment-level task breakdown with `stats` command
- **Multi-dimensional Filtering**: Support for `--all`, `--pr <number>`, and `--branch <name>` options
- **Progress Tracking**: Task completion rates and priority distribution analysis
- **File-level Summary**: Task grouping by affected files for targeted development focus
- **Performance Metrics**: Processing time and cache efficiency monitoring

#### 2.8 Version Management and Self-Update
- **Automatic Update Detection**: Checks for newer versions on startup
- **GitHub Releases Integration**: Direct binary downloads from GitHub releases
- **Version Switching**: Easy switching between versions with `version <VERSION>` command
- **Release Information**: Display of recent versions with `versions` command
- **Rollback Capability**: Return to previous versions if needed
- **Update Notifications**: Proactive notifications of available updates

#### 2.9 Performance Optimization and Cache Management
- **Intelligent Caching**: Avoids re-processing unchanged comments for improved performance
- **Selective Refresh**: Only processes changed or new content by default
- **Manual Cache Override**: `--refresh-cache` flag for complete data reprocessing
- **Cache Consistency**: Maintains task state preservation across cache operations
- **Performance Monitoring**: Cache hit rates and processing time optimization

#### 2.10 Claude Code Integration
- **Template Generation**: Creates optimized Claude Code command templates
- **Workflow Integration**: PR review analysis workflow templates for `.claude/commands/`
- **Structured Analysis**: Integration with existing reviewtask data structures
- **Quality Consistency**: Standardized review format and approach
- **Development Efficiency**: Streamlined AI-assisted development workflows

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

#### 3.4 Statistics and Analytics Implementation
- **Data Aggregation**: Comment-level task counting and categorization
- **Filtering Engine**: Multi-dimensional data filtering by PR, branch, and status
- **Performance Metrics**: Cache hit rates, processing times, and efficiency monitoring
- **Output Formatting**: Structured display with priority and status breakdowns
- **Real-time Calculation**: Dynamic statistics computation from current task data

#### 3.5 Version Management System
- **GitHub API Integration**: Releases API for version information retrieval
- **Binary Management**: Automatic download and installation of specific versions
- **Version Detection**: Current version embedding and comparison logic
- **Update Mechanism**: Automatic replacement of current binary with new version
- **Rollback Support**: Preservation of previous versions for fallback scenarios

#### 3.6 Cache Management Architecture
- **Content Hashing**: Comment content change detection using hash comparison
- **State Preservation**: Task status maintenance across cache operations
- **Selective Processing**: Intelligent determination of changed content requiring reprocessing
- **Performance Optimization**: Reduced AI processing load through effective caching
- **Cache Invalidation**: Manual and automatic cache clearing mechanisms

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
- **Individual Updates**: `reviewtask update <task-id> <new-status>`
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
- **Manual Init**: `reviewtask init` for explicit setup
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

#### 6.1 Implemented Features (Now Core Functionality)
- ✅ **Advanced Statistics and Analytics**: Detailed task breakdown with multi-dimensional filtering
- ✅ **Version Management System**: GitHub releases integration with self-update capability  
- ✅ **Performance Optimization**: Intelligent caching with selective refresh capabilities
- ✅ **Claude Code Integration**: Template generation for AI-assisted development workflows

#### 6.2 Potential Future Enhancements
- Integration with external task management tools (Jira, Asana, GitHub Issues)
- Support for multiple repository monitoring and cross-repository analytics
- Automated task completion detection based on commit analysis
- Team collaboration features with shared task states
- Custom AI analyzer plugins for domain-specific review patterns
- Web dashboard for visual analytics and progress tracking

#### 6.3 Extensibility Framework
- Plugin architecture for custom AI analyzers and task generators
- Configurable output formats (JSON, XML, CSV export)
- Custom priority rules and task generation logic
- Integration APIs for third-party development tools
- Webhook support for real-time notifications and integrations

### 7. Success Criteria

- Successfully fetch and store PR review data
- Generate meaningful tasks from review content using AI
- Maintain structured local storage for easy access
- Provide clear command-line interface for users
- Handle edge cases and errors gracefully