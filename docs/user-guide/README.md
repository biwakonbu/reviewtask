# User Guide

Welcome to the reviewtask User Guide. This documentation is for developers who want to use reviewtask to manage their PR reviews more effectively.

## Supported AI Providers & Review Tools

### AI Providers for Task Generation
- **🤖 Cursor CLI** - Cursor's AI with automatic model selection (recommended)
- **🤖 Claude Code** - Anthropic's Claude via command-line interface
- **🤖 Auto-detection** - Automatically finds and uses available providers

### Review Source Integration
reviewtask automatically detects and processes reviews from multiple sources:

- **✅ Standard GitHub Reviews** - Direct comment processing with thread auto-resolution
- **✅ CodeRabbit** (`coderabbitai[bot]`) - Automatic nitpick comment detection and filtering
- **✅ Codex** (`chatgpt-codex-connector`) - Parses embedded comments with P1/P2/P3 priority badges

**No configuration required** - all review sources work automatically!

## Getting Started

- [Installation](installation.md) - How to install reviewtask on your system
- [Quick Start](quick-start.md) - Get up and running in 5 minutes
- [Authentication Setup](authentication.md) - Configure GitHub access

## Core Features

- [Commands Reference](commands.md) - All available CLI commands
- [Workflow Guide](workflow.md) - Best practices for daily use
- [Configuration](configuration.md) - Customize reviewtask for your needs

## Advanced Usage

- [Troubleshooting](troubleshooting.md) - Common issues and solutions
- [Prompt Templates](../prompts/README.md) - Customize AI task generation

## Quick Links

- [GitHub Repository](https://github.com/biwakonbu/reviewtask)
- [Latest Release](https://github.com/biwakonbu/reviewtask/releases/latest)
- [Report Issues](https://github.com/biwakonbu/reviewtask/issues)