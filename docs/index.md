# reviewtask - AI-Powered PR Review Management Tool

[![Latest Release](https://img.shields.io/github/v/release/biwakonbu/reviewtask)](https://github.com/biwakonbu/reviewtask/releases/latest)
[![CI](https://github.com/biwakonbu/reviewtask/workflows/CI/badge.svg)](https://github.com/biwakonbu/reviewtask/actions)
[![codecov](https://codecov.io/gh/biwakonbu/reviewtask/branch/main/graph/badge.svg)](https://codecov.io/gh/biwakonbu/reviewtask)
[![Go Report Card](https://goreportcard.com/badge/github.com/biwakonbu/reviewtask)](https://goreportcard.com/report/github.com/biwakonbu/reviewtask)
[![GoDoc](https://godoc.org/github.com/biwakonbu/reviewtask?status.svg)](https://godoc.org/github.com/biwakonbu/reviewtask)

A CLI tool that fetches GitHub Pull Request reviews, analyzes them using AI, and generates actionable tasks for developers to address feedback systematically.

## Features

- **🔍 PR Review Fetching**: Automatically retrieves reviews from GitHub API with nested comment structure
- **🤖 AI Analysis**: Supports multiple AI providers for generating structured, actionable tasks from review content
- **💾 Local Storage**: Stores data in structured JSON format under `.pr-review/` directory
- **📋 Task Management**: Full lifecycle management with status tracking (todo/doing/done/pending/cancel)
- **⚡ Parallel Processing**: Processes multiple comments concurrently for improved performance
- **🔒 Authentication**: Multi-source token detection with interactive setup
- **🎯 Priority-based Analysis**: Customizable priority rules for task generation
- **🔄 Task State Preservation**: Maintains existing task statuses during subsequent runs
- **🆔 UUID-based Task IDs**: Unique task identification to eliminate duplication issues
- **🔌 Extensible AI Provider Support**: Architecture designed for easy integration of multiple AI providers
- **🏷️ Low-Priority Detection**: Automatically identifies and assigns "pending" status to low-priority comments (nits, suggestions)
- **⏱️ Smart Performance**: Automatic optimization based on PR size with no configuration needed
- **💨 API Caching**: Reduces redundant GitHub API calls automatically
- **📊 Auto-Resume**: Seamlessly continues from where it left off if interrupted
- **🔧 Debug Commands**: Test specific phases independently for troubleshooting
- **📏 Prompt Size Optimization**: Automatic chunking for large comments (>20KB) and pre-validation size checks
- **✅ Task Validation**: AI-powered validation with configurable quality thresholds and retry logic
- **🖥️ Verbose Mode**: Detailed logging and debugging output for development and troubleshooting
- **🔄 Smart Deduplication**: AI-powered task deduplication with similarity threshold control
- **🛡️ JSON Recovery**: Automatic recovery from incomplete Claude API responses with partial task extraction
- **🔁 Intelligent Retry**: Smart retry strategies with pattern detection and prompt size adjustment
- **📊 Response Monitoring**: Performance analytics and optimization recommendations for API usage

## Quick Start

Get started with reviewtask in just a few steps:

1. **[Install](installation.md)** the tool using our one-liner installation script
2. **[Initialize](quick-start.md#initialization)** your repository with `reviewtask init`
3. **[Authenticate](authentication.md)** with GitHub using `reviewtask auth login`
4. **[Analyze](quick-start.md#analyzing-pr-reviews)** your PR reviews with `reviewtask`

## Core Workflow

```bash
# Initialize repository
reviewtask init

# Set up authentication
reviewtask auth login

# Analyze current branch's PR
reviewtask

# View and manage tasks
reviewtask status
reviewtask show
reviewtask update <task-id> doing
```

## Why reviewtask?

Transform your GitHub Pull Request reviews into a structured, trackable workflow:

- **Zero Feedback Loss**: Every actionable review comment is captured and tracked
- **State Preservation**: Your work progress is never lost due to tool operations
- **AI-Assisted**: Intelligent task generation and prioritization
- **Developer-Controlled**: You maintain full control over task status and workflow

## Getting Started

Ready to transform your PR review workflow? Start with our [Installation Guide](installation.md) and follow the [Quick Start Guide](quick-start.md) to get up and running in minutes.

For detailed information about commands, configuration, and advanced features, explore the documentation sections using the navigation menu.