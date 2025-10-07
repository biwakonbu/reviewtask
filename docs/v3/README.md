# v3.0.0 Concept Documentation

This directory contains design documents and specifications for reviewtask v3.0.0.

## Documents

### [concept.md](./concept.md)
Main v3.0.0 design document covering:
- Command integration and simplification
- Modern UI and guidance system
- Comprehensive comment analysis with AI impact assessment
- Unresolved comment detection
- Done command automation routine

## Implementation Status

v3.0.0 is tracked through the following issues:

### Phase 1: Foundation (Week 1-2) ✅ COMPLETED
- [#191](https://github.com/biwakonbu/reviewtask/issues/191) - ✅ Implement unresolved review comment detection
- [#194](https://github.com/biwakonbu/reviewtask/issues/194) - ✅ Implement modern UI and guidance system

### Phase 2: Core Features (Week 3-5) ✅ COMPLETED
- [#192](https://github.com/biwakonbu/reviewtask/issues/192) - ✅ Implement comprehensive comment analysis with AI impact assessment

### Phase 3: Integration (Week 6-8) ✅ COMPLETED
- [#193](https://github.com/biwakonbu/reviewtask/issues/193) - ✅ Command integration and flag simplification (v3.0.0)

### Phase 4: Automation (Week 9-11) ✅ COMPLETED
- [#195](https://github.com/biwakonbu/reviewtask/issues/195) - ✅ Implement done command automation routine

### Phase 5: Guidance Enhancement ✅ COMPLETED
- [#206](https://github.com/biwakonbu/reviewtask/issues/206) - ✅ Implement context-aware guidance system for all commands

### Phase 6: Command Simplification ✅ COMPLETED
- [#215](https://github.com/biwakonbu/reviewtask/issues/215) - ✅ Implement integrated reviewtask command (fetch + analyze)
- [#216](https://github.com/biwakonbu/reviewtask/issues/216) - ✅ Remove or deprecate analyze command flags
- [#217](https://github.com/biwakonbu/reviewtask/issues/217) - ✅ Deprecate complete command in favor of done

## Release Timeline

```
Week 1-2:  Phase 1 (#191 + #194) - Foundation
Week 3-5:  Phase 2 (#192) - Core Features
Week 6-8:  Phase 3 (#193) - Integration → v2.5.0 deprecation release
Week 9-11: Phase 4 (#195) - Automation → v3.0.0 final release
```

## Key Changes in v3.0.0

### Command Simplification
- `fetch` + `analyze` → `reviewtask` (single command)
- `update <id> <status>` → status-specific commands (`start`, `done`, `hold`)
- Remove complex flags, use smart defaults

### New Features
- Detect and display unresolved GitHub review comments
- Analyze ALL review comments (including nitpicks)
- AI-based impact assessment (TODO/PENDING auto-assignment)
- Modern, clean UI inspired by GitHub CLI
- Context-aware guidance after every command
- Done command automation (verify → commit → resolve → next task)

### Breaking Changes
- v2.5.0: Deprecation warnings
- v3.0.0: Remove deprecated commands, maintain aliases for backward compatibility

## References

- [CLAUDE.md](../../CLAUDE.md) - Project instructions and philosophy
- [Issues #191-195](https://github.com/biwakonbu/reviewtask/issues) - Implementation tracking
