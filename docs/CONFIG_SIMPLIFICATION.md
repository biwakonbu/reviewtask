# Configuration Simplification Plan

This document tracks the implementation progress for Issue #164: Simplify configuration system with smart defaults and progressive disclosure.

## Implementation Phases

### Phase 1: Internal Simplification (No Breaking Changes)
- [ ] Implement smart default values
- [ ] Add automatic project type detection
- [ ] Hide unnecessary settings internally (keep for backward compatibility)
- [ ] Improve error messages with suggested fixes

### Phase 2: New Configuration Format Support
- [ ] Accept simplified configuration format
- [ ] Implement automatic migration from old to new format
- [ ] Add interactive setup wizard (`reviewtask init`)
- [ ] Create configuration validation command (`reviewtask config validate`)

### Phase 3: User Experience Enhancement
- [ ] Implement `reviewtask init` for interactive setup
- [ ] Add `reviewtask config validate` for configuration health check
- [ ] Create `reviewtask config migrate` for explicit migration
- [ ] Update documentation with migration guide

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