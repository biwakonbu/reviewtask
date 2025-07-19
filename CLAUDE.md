## Project Language

- Memories and documents should be written in English
- Conversations should be conducted in the language specified by the user

# AI-Powered PR Review Management Tool

## Project Overview

This project implements `gh-review-task`, a CLI tool that fetches GitHub Pull Request reviews, analyzes them using Claude Code integration with advanced two-stage validation, and generates actionable tasks with multi-language support and comment-based management.

## Key Features Implemented

### Enhanced Core Functionality
- **PR Review Fetching**: Retrieves reviews from GitHub API with full metadata and nested comment structure
- **Advanced AI Analysis**: Claude Code integration with proper one-shot mode (`claude -p --output-format json`)
- **Two-Stage Validation**: Mechanical format validation + AI-powered content validation with quality scoring
- **Multi-language Support**: Task descriptions in user's preferred language while preserving original review text
- **Comment-Based Task Management**: Multiple tasks per comment with individual status tracking
- **Enhanced Local Storage**: Structured JSON with origin text preservation and comment-based indexing

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

### Enhanced Command Structure
```
gh-review-task [PR_NUMBER]           # Analyze PR reviews with two-stage validation
gh-review-task status                # Show current task status across all PRs
gh-review-task stats [PR_NUMBER]     # NEW: Comment-level statistics and progress
gh-review-task update <id> <status>  # Update task status (format: comment-{id}-task-{index})
gh-review-task init                  # Initialize repository with AI settings
gh-review-task auth <cmd>            # Authentication management
```

## Technical Architecture

### Enhanced Data Storage Structure
```
.pr-review/
├── config.json              # Priority rules, AI settings, and project configuration
├── auth.json                # Local authentication (gitignored)
└── PR-<number>/
    ├── info.json            # PR metadata
    ├── reviews.json         # Review data with full metadata and nested comments
    └── tasks.json           # AI-generated tasks with validation scores and origin text
```

### Enhanced Key Components
- **GitHub API Client**: Repository detection, PR/review fetching, permission checking
- **Advanced AI Analyzer**: Claude Code integration with two-stage validation system
- **Task Validator**: Mechanical format validation + AI content validation
- **Statistics Manager**: Comment-based analytics and progress tracking
- **Enhanced Storage Manager**: Comment-based queries and extended data structures
- **Enhanced Config System**: AI settings, priority rules, and multi-language configuration
- **Setup Manager**: Repository initialization with AI settings integration

### New Components Added
- **`internal/ai/validator.go`**: Two-stage validation system implementation
- **`internal/ai/statistics.go`**: Comment-based statistical analysis
- **`cmd/stats.go`**: Comment-level statistics command

## Advanced AI Integration

### Two-Stage Validation System
- **Stage 1 - Format Validation**: Mechanical JSON syntax, required fields, and data type validation
- **Stage 2 - Content Validation**: AI-powered assessment of task quality, actionability, and language consistency
- **Quality Scoring**: 0.0-1.0 scoring system with configurable acceptance threshold (default: 0.8)
- **Iterative Improvement**: Up to 5 retry attempts with validation feedback integration
- **Best Result Selection**: Uses highest-scoring result when perfect validation fails

### Enhanced Task Generation
- **Multi-language Support**: Task descriptions generated in user's preferred language
- **Information Preservation**: Original review text preserved alongside AI-generated descriptions
- **Task Splitting Intelligence**: Automatically splits complex comments into multiple actionable tasks
- **Comment-Based Management**: Tasks tracked with `comment-{id}-task-{index}` format
- **Default Rules**: Security (critical) > Performance (high) > Functional (medium) > Style (low)
- **Project Customization**: Enhanced rules via `.pr-review/config.json` with AI settings

### Advanced Comment Chain Processing
- **Nested Structure**: Preserves reply relationships for comprehensive AI context
- **Resolution Detection**: Analyzes entire conversation threads to avoid duplicate tasks
- **Priority Inference**: Uses discussion context and comment metadata for task urgency
- **Language Detection**: Maintains original language context while generating user-preferred descriptions

### Claude Code Integration
- **Proper CLI Usage**: Fixed to use `claude -p --output-format json` instead of deprecated syntax
- **Graceful Degradation**: Fallback mechanisms when Claude Code is unavailable
- **Error Handling**: Comprehensive error detection with specific remediation guidance
- **Configuration**: AI settings in config for max retries, quality thresholds, and language preferences

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

## Enhanced Testing Coverage

All major user flows have been comprehensively tested:
- **Enhanced**: Two-stage validation system with quality scoring and retry mechanisms
- **Enhanced**: Multi-language task generation and validation flows
- **New**: Comment-based task management and statistics generation
- **New**: Claude Code CLI integration with proper one-shot syntax
- First-time setup and initialization with AI settings
- Authentication flows (all sources) with enhanced error handling
- PR review processing with advanced AI integration and validation
- Task lifecycle management with comment-based tracking
- Permission validation and comprehensive error scenarios
- Statistics generation and comment-level progress tracking

## Recent Implementation (Latest PR)

### Major Features Added
1. **Two-Stage Validation System** (`internal/ai/validator.go`)
   - Mechanical format validation
   - AI-powered content validation
   - Quality scoring (0.0-1.0)
   - Iterative improvement with feedback

2. **Enhanced Data Structures**
   - `OriginText` field for information preservation
   - `SourceCommentID` and `TaskIndex` for comment-based management
   - `AISettings` configuration with user language support
   - Enhanced statistics structures (`CommentStats`, `TaskStatistics`)

3. **Comment-Based Task Management**
   - Multiple tasks per comment support
   - Task ID format: `comment-{commentID}-task-{index}`
   - Individual status tracking per task
   - Comment-level progress statistics

4. **Multi-language Support**
   - User-configurable language for task descriptions
   - Original review text preservation
   - Language-aware prompt engineering

5. **Advanced Statistics** (`cmd/stats.go`)
   - Comment-level progress tracking
   - Status summary across all tasks
   - Per-comment completion rates
   - Detailed breakdown with origin text preview

### Technical Improvements
- Fixed Claude Code CLI syntax from `claude code` to `claude -p`
- Implemented proper one-shot mode with `--output-format json`
- Added comprehensive error handling and fallback mechanisms
- Enhanced configuration system with AI settings
- Improved data persistence with extended structures

### Dependencies Added
- **Claude Code CLI**: Required for AI-powered task generation and validation
- Enhanced Go structures for validation and statistics
- No additional runtime dependencies beyond existing Go modules
