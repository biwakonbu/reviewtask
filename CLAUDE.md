## Project Language

- Memories and documents should be written in English
- Conversations should be conducted in the language specified by the user

# AI-Powered PR Review Management Tool

## Project Overview

This project implements `gh-review-task`, a CLI tool that fetches GitHub Pull Request reviews, analyzes them using AI, and generates actionable tasks for developers.

## Key Features Implemented

### Core Functionality
- **PR Review Fetching**: Retrieves reviews from GitHub API with nested comment structure
- **AI Analysis**: Uses Claude Code's one-shot functionality to generate structured tasks from review content
- **Local Storage**: Stores data in structured JSON format under `.pr-review/` directory
- **Task Management**: Full lifecycle management with status tracking (todo/doing/done/pending/cancel)

### Authentication System
- **Multi-source Token Detection**: Environment variables → Local config → gh CLI config
- **Interactive Setup**: `auth login/logout/status/check` commands
- **Permission Validation**: Comprehensive checking of GitHub API permissions
- **Error Guidance**: Detailed troubleshooting with specific remediation steps

### Project Initialization
- **Auto-init**: Prompts for initialization on first use
- **Git Integration**: Automatic `.gitignore` management
- **Configuration Generation**: Creates default priority rules and project settings
- **Validation**: Checks repository access and required permissions

### Command Structure
```
gh-review-task [PR_NUMBER]     # Analyze PR reviews and generate tasks
gh-review-task status          # Show current task status and statistics  
gh-review-task update <id> <status>  # Update task status
gh-review-task init            # Initialize repository
gh-review-task auth <cmd>      # Authentication management
```

## Technical Architecture

### Data Storage Structure
```
.pr-review/
├── config.json              # Priority rules and project settings
├── auth.json                # Local authentication (gitignored)
└── PR-<number>/
    ├── info.json            # PR metadata
    ├── reviews.json         # Review data with nested comments
    └── tasks.json           # AI-generated tasks
```

### Key Components
- **GitHub API Client**: Repository detection, PR/review fetching, permission checking
- **AI Analyzer**: Claude Code integration for contextual task generation  
- **Storage Manager**: JSON-based local data persistence
- **Config System**: Priority rules with project-specific overrides
- **Setup Manager**: Repository initialization and git integration

## AI Integration

### Priority-based Task Generation
- **Default Rules**: Security (critical) > Performance (high) > Functional (medium) > Style (low)
- **Project Customization**: Override rules via `.pr-review/config.json`
- **Contextual Analysis**: Considers comment chains and reply context for task relevance

### Comment Chain Processing
- **Nested Structure**: Preserves reply relationships for AI context
- **Resolution Detection**: Analyzes conversation threads to avoid duplicate tasks
- **Priority Inference**: Uses discussion context to determine task urgency

## Development Patterns

### Error Handling
- Graceful degradation with clear user guidance
- Specific error messages with actionable remediation steps
- Progressive authentication checking with fallback options

### User Experience
- Self-guided setup with minimal manual configuration
- Consistent CLI patterns following `gh` conventions  
- Progressive disclosure of advanced features

### Security Considerations
- Token stored with restricted file permissions (600)
- Sensitive data excluded from git tracking
- Clear separation of local vs global authentication

## Testing Coverage

All major user flows have been tested:
- First-time setup and initialization
- Authentication flows (all sources)
- PR review processing and task generation
- Status management and updates
- Permission validation and error scenarios
