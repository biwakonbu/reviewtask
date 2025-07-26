# Fetch Command Performance Optimization

This document describes the performance optimizations available in the `reviewtask fetch` command to handle large PRs and prevent timeouts.

## Overview

The fetch command now supports incremental processing, allowing it to:
- Process review comments in configurable batches
- Resume from the last checkpoint if interrupted
- Cache GitHub API responses to reduce redundant calls
- Provide progress tracking during processing
- Skip non-essential processing in fast mode

## Command Options

### Batch Processing

Process comments in smaller batches to avoid timeouts:

```bash
reviewtask fetch --batch-size=10
```

- Default: 5 comments per batch
- Recommended for large PRs: 10-20 comments per batch
- Each batch is processed independently with progress saved

### Resume from Checkpoint

If processing is interrupted, resume from where it left off:

```bash
reviewtask fetch --resume
```

- Automatically detects and loads the last checkpoint
- Skips already processed comments
- Preserves partial results from previous runs

### Fast Mode

Skip validation and use simplified processing for speed:

```bash
reviewtask fetch --fast-mode
```

- Skips AI validation steps
- Uses simplified prompts
- Ideal for quick initial processing
- May produce less accurate task prioritization

### Timeout Configuration

Set a custom timeout for the operation:

```bash
reviewtask fetch --timeout=600  # 10 minutes
```

- Default: 300 seconds (5 minutes)
- Maximum: 3600 seconds (1 hour)
- Processing saves checkpoint before timeout

### Progress Display

Control progress display during processing:

```bash
reviewtask fetch --no-progress  # Disable progress display
```

- Default: Progress is shown
- Shows percentage completion and current batch

## Combined Usage

For optimal performance on large PRs, combine options:

```bash
# Process large PR with resume support
reviewtask fetch 123 --batch-size=20 --resume --timeout=900

# Quick processing without validation
reviewtask fetch --fast-mode --batch-size=50

# CI/CD friendly (no progress output)
reviewtask fetch --batch-size=10 --no-progress
```

## Performance Tips

1. **For PRs with 50+ comments**: Use `--batch-size=20` or higher
2. **For CI/CD environments**: Use `--no-progress` and appropriate `--timeout`
3. **For initial processing**: Use `--fast-mode` for quick results
4. **For interrupted processing**: Always use `--resume` to continue

## Technical Details

### Checkpointing

- Checkpoints are saved after each batch in `.pr-review/PR-{number}/checkpoint.json`
- Contains processed comment IDs, partial results, and progress information
- Automatically cleaned up after successful completion
- Stale checkpoints (>24 hours) are ignored

### API Caching

- GitHub API responses are cached for 5 minutes
- Cache is stored in `~/.cache/reviewtask/github-api/`
- Reduces API calls for repeated operations
- Automatically expires old cache entries

### Parallel Processing

- Comments within a batch are processed in parallel
- Utilizes available CPU cores efficiently
- Maintains order of results despite parallel execution

## Troubleshooting

### Timeout Errors

If you encounter timeout errors:

1. Reduce batch size: `--batch-size=5`
2. Increase timeout: `--timeout=600`
3. Use resume to continue: `--resume`
4. Try fast mode: `--fast-mode`

### Memory Issues

For very large PRs:

1. Use smaller batch sizes
2. Process in multiple runs with `--resume`
3. Clear cache if needed: `rm -rf ~/.cache/reviewtask`

### Checkpoint Issues

If checkpoint is corrupted:

1. Delete checkpoint: `rm .pr-review/PR-{number}/checkpoint.json`
2. Restart processing from beginning

## Example Workflows

### Large PR Review (100+ comments)

```bash
# Initial processing with resume support
reviewtask fetch 456 --batch-size=25 --resume

# If timeout occurs, simply re-run the same command
reviewtask fetch 456 --batch-size=25 --resume
```

### CI/CD Integration

```bash
# Non-interactive with strict timeout
reviewtask fetch $PR_NUMBER \
  --batch-size=10 \
  --timeout=300 \
  --no-progress \
  --fast-mode
```

### Development Workflow

```bash
# Quick initial scan
reviewtask fetch --fast-mode --batch-size=50

# Detailed analysis later
reviewtask fetch --batch-size=10 --resume
```