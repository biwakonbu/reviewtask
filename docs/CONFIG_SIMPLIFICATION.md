# Configuration Simplification (v1.11.0)

This document describes the simplified configuration system introduced in reviewtask v1.11.0, featuring smart defaults and progressive disclosure.

## Overview

The configuration system has been redesigned to minimize setup complexity while maintaining flexibility for advanced users. Most users can now get started with just 2 configuration settings.

## Available Tools

### Interactive Setup
```bash
reviewtask init
```
Creates a minimal configuration file through an interactive wizard that:
- Detects available AI providers (Cursor CLI, Claude Code)
- Sets your preferred language for task descriptions
- Automatically configures smart defaults

### Configuration Validation
```bash
reviewtask config validate
```
Checks your current configuration for:
- Syntax errors
- Missing required fields
- Compatibility issues
- Provides helpful suggestions for fixes

### Configuration Migration
```bash
reviewtask config migrate
```
Automatically migrates older configuration formats to the new simplified format:
- Creates a backup of your existing configuration
- Converts to the minimal format
- Preserves all custom settings

## Configuration Levels

### Level 1: Minimal Configuration (90% of users)
```json
{
  "language": "English",
  "ai_provider": "auto"
}
```

### Level 2: Basic Configuration
```json
{
  "language": "English",
  "ai_provider": "cursor",
  "model": "grok",
  "priorities": {
    "project_specific": {
      "critical": "Authentication vulnerabilities",
      "high": "Payment processing errors"
    }
  }
}
```

### Level 3: Advanced Configuration
Full control with advanced settings for power users.