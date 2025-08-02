# Troubleshooting Guide

This comprehensive guide helps you diagnose and resolve common issues with reviewtask.

## Quick Diagnosis Commands

Start troubleshooting with these diagnostic commands:

```bash
# Check version and system info
reviewtask version

# Verify authentication
reviewtask auth status
reviewtask auth check

# Test basic functionality
reviewtask show
reviewtask status
```

## Authentication Issues

### Token Validation Failed

**Symptoms:**
- "Authentication failed" errors
- "Invalid token" messages
- 401 Unauthorized errors

**Solutions:**

1. **Check token validity:**
   ```bash
   reviewtask auth check
   ```

2. **Verify token hasn't expired:**
   - Check GitHub Settings > Developer settings > Personal access tokens
   - Look for expiration date and token status

3. **Ensure correct scopes:**
   - **Private repos**: Requires `repo` scope
   - **Public repos**: Requires `public_repo` scope  
   - **Organization repos**: Requires `read:org` scope

4. **Re-authenticate:**
   ```bash
   reviewtask auth logout
   reviewtask auth login
   ```

### Repository Access Denied

**Symptoms:**
- "Repository not found" errors
- "Access denied" when fetching PR data
- 403 Forbidden errors

**Solutions:**

1. **For private repositories:**
   ```bash
   # Ensure token has 'repo' scope
   reviewtask auth check
   ```

2. **For organization repositories:**
   ```bash
   # Ensure token has 'read:org' scope
   # Check organization's third-party access settings
   ```

3. **Verify repository URL:**
   ```bash
   git remote -v
   # Ensure you're in the correct repository
   ```

### Multiple Authentication Sources

**Symptoms:**
- Confusion about which token is being used
- Inconsistent authentication behavior

**Diagnosis:**
```bash
reviewtask auth status
```

**Priority order:**
1. `GITHUB_TOKEN` environment variable (highest)
2. Local config file (`.pr-review/auth.json`)
3. GitHub CLI (`gh auth token`) (lowest)

**Solutions:**
```bash
# Use specific source
export GITHUB_TOKEN="your_token"  # Environment variable

# Or remove conflicting sources
reviewtask auth logout            # Remove local config
unset GITHUB_TOKEN               # Remove environment variable
```

## Rate Limit Issues

### API Rate Limiting

**Symptoms:**
- "Rate limit exceeded" errors
- Slow or failing API requests
- 429 Too Many Requests errors

**Check current status:**
```bash
reviewtask auth check
```

**Rate limits:**
- Authenticated: 5,000 requests/hour
- Unauthenticated: 60 requests/hour

**Solutions:**
1. **Ensure authentication:**
   ```bash
   reviewtask auth status
   ```

2. **Use caching:**
   ```bash
   # Avoid --refresh-cache unless necessary
   reviewtask  # Uses cached data when possible
   ```

3. **Wait for reset:**
   - Rate limits reset every hour
   - Check `X-RateLimit-Reset` time in auth check output

## Version and Update Issues

### Binary Not Found

**Symptoms:**
- "command not found: reviewtask"
- "reviewtask: command not found"

**Solutions:**

1. **Check installation:**
   ```bash
   which reviewtask
   # Should show path to binary
   ```

2. **Verify PATH:**
   ```bash
   echo $PATH
   # Should include installation directory
   ```

3. **Add to PATH:**
   ```bash
   # For bash/zsh
   export PATH="$HOME/.local/bin:$PATH"
   
   # For fish
   set -gx PATH $HOME/.local/bin $PATH
   ```

4. **Reinstall:**
   ```bash
   curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --force
   ```

### Version Issues

**Symptoms:**
- Old version behavior
- Missing features
- Update failures

**Solutions:**

1. **Check current version:**
   ```bash
   reviewtask version
   ```

2. **Update to latest:**
   ```bash
   reviewtask version latest
   ```

3. **Manual reinstall:**
   ```bash
   curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --force
   ```

## Task Generation Issues

### Missing Tasks

**Symptoms:**
- No tasks generated from PR reviews
- Empty task lists despite review comments

**Diagnosis:**
```bash
# Enable verbose mode
# Edit .pr-review/config.json:
{
  "ai_settings": {
    "verbose_mode": true
  }
}

# Test specific phases
reviewtask debug fetch review 123
reviewtask debug fetch task 123
```

**Solutions:**

1. **Force refresh:**
   ```bash
   reviewtask --refresh-cache
   ```

2. **Check AI provider:**
   ```bash
   claude --version  # For Claude Code
   ```

3. **Verify PR has reviews:**
   ```bash
   # Check on GitHub that PR actually has review comments
   ```

### Inconsistent Task Generation

**Symptoms:**
- Tasks appear and disappear
- Duplicate tasks
- Incorrect task content

**Solutions:**

1. **Clear cache:**
   ```bash
   reviewtask --refresh-cache
   ```

2. **Check comment changes:**
   - Tasks are cancelled when comments change significantly
   - New tasks created for new/updated comments

3. **Adjust deduplication:**
   ```json
   {
     "ai_settings": {
       "deduplication_enabled": false,
       "similarity_threshold": 0.7
     }
   }
   ```

### Task Status Issues

**Symptoms:**
- Task statuses reset unexpectedly
- Cannot update task status
- Status changes not persisting

**Solutions:**

1. **Check task ID:**
   ```bash
   reviewtask show  # Get correct task UUID
   ```

2. **Verify file permissions:**
   ```bash
   ls -la .pr-review/
   # Ensure write permissions on task files
   ```

3. **Manual status check:**
   ```bash
   cat .pr-review/PR-*/tasks.json
   # Inspect task file directly
   ```

## AI Provider Integration Issues

### Claude Code Integration

**Symptoms:**
- "claude command not found"
- AI analysis failures
- Empty or malformed task generation

**Solutions:**

1. **Verify installation:**
   ```bash
   claude --version
   which claude
   ```

2. **Check PATH:**
   ```bash
   echo $PATH | grep -o '[^:]*claude[^:]*'
   ```

3. **Install Claude Code:**
   Follow [official installation instructions](https://docs.anthropic.com/en/docs/claude-code)

4. **Test Claude integration:**
   ```bash
   reviewtask prompt claude pr-review
   ```

### JSON Recovery Issues

**Symptoms:**
- "unexpected end of JSON input" errors
- Truncated responses
- Malformed task data

**Solutions:**

1. **Enable recovery:**
   ```json
   {
     "ai_settings": {
       "enable_json_recovery": true,
       "max_recovery_attempts": 3,
       "log_truncated_responses": true
     }
   }
   ```

2. **Reduce prompt size:**
   ```json
   {
     "ai_settings": {
       "validation_enabled": true,
       "max_retries": 5
     }
   }
   ```

3. **Check response analytics:**
   ```bash
   cat .pr-review/response_analytics.json
   ```

## Cache and Performance Issues

### Cache Problems

**Symptoms:**
- Outdated task content
- Missing new comments
- Inconsistent behavior

**Solutions:**

1. **Force refresh:**
   ```bash
   reviewtask --refresh-cache
   ```

2. **Clear cache manually:**
   ```bash
   rm -rf .pr-review/cache/
   reviewtask
   ```

3. **Check cache permissions:**
   ```bash
   ls -la .pr-review/cache/
   ```

### Performance Issues

**Symptoms:**
- Slow processing
- Timeouts
- High memory usage

**Solutions:**

1. **Check PR size:**
   ```bash
   reviewtask stats --pr 123
   ```

2. **Enable optimization:**
   - Tool automatically optimizes for large PRs
   - Uses parallel processing and chunking

3. **Reduce processing load:**
   ```json
   {
     "ai_settings": {
       "validation_enabled": false,
       "deduplication_enabled": false
     }
   }
   ```

## Configuration Issues

### Invalid Configuration

**Symptoms:**
- Tool ignores configuration changes
- Default behavior instead of custom settings
- JSON parsing errors

**Solutions:**

1. **Validate JSON syntax:**
   ```bash
   cat .pr-review/config.json | jq .
   ```

2. **Reset to defaults:**
   ```bash
   reviewtask init  # Recreates default config
   ```

3. **Check file permissions:**
   ```bash
   ls -la .pr-review/config.json
   ```

### Priority Assignment Issues

**Symptoms:**
- All tasks get same priority
- Priority rules not working
- Unexpected priority assignments

**Solutions:**

1. **Enable verbose mode:**
   ```json
   {
     "ai_settings": {
       "verbose_mode": true
     }
   }
   ```

2. **Review priority rules:**
   ```json
   {
     "priority_rules": {
       "critical": "Security vulnerabilities, authentication bypasses",
       "high": "Performance bottlenecks, memory leaks",
       "medium": "Functional bugs, logic improvements",
       "low": "Code style, naming conventions"
     }
   }
   ```

3. **Test with simple rules:**
   - Start with clear, distinct priority descriptions
   - Add complexity gradually

## Network and Connectivity Issues

### GitHub API Issues

**Symptoms:**
- Connection timeouts
- Intermittent failures
- "Network unreachable" errors

**Solutions:**

1. **Check GitHub status:**
   - Visit [GitHub Status](https://www.githubstatus.com/)
   - Verify GitHub API is operational

2. **Test connectivity:**
   ```bash
   curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
   ```

3. **Use retry logic:**
   - Tool includes automatic retry for transient failures
   - Enable verbose mode to see retry attempts

### Proxy and Firewall Issues

**Symptoms:**
- Connection failures in corporate environments
- SSL certificate errors
- Blocked requests

**Solutions:**

1. **Configure proxy:**
   ```bash
   export HTTPS_PROXY=https://proxy.company.com:8080
   export HTTP_PROXY=http://proxy.company.com:8080
   ```

2. **Check firewall rules:**
   - Ensure access to api.github.com
   - Verify HTTPS (443) is allowed

## Debug Mode and Logging

### Enable Verbose Output

For detailed troubleshooting, enable verbose mode:

```json
{
  "ai_settings": {
    "verbose_mode": true,
    "log_truncated_responses": true
  }
}
```

### Debug Commands

Test specific functionality:

```bash
# Test review fetching only
reviewtask debug fetch review 123

# Test task generation only  
reviewtask debug fetch task 123

# Both commands enable verbose mode automatically
```

### Log Analysis

Check log output for:
- API request/response details
- Task generation process
- Error messages and stack traces
- Performance metrics

## Getting Additional Help

### Self-Diagnosis Checklist

1. ✅ **Authentication working?** `reviewtask auth check`
2. ✅ **Latest version?** `reviewtask version`
3. ✅ **Valid configuration?** `cat .pr-review/config.json | jq .`
4. ✅ **AI provider available?** `claude --version`
5. ✅ **Repository access?** Can you see PR on GitHub?
6. ✅ **Cache cleared?** Try `reviewtask --refresh-cache`

### When to Seek Help

Open a GitHub issue if:
- Self-diagnosis doesn't resolve the issue
- You encounter unexpected behavior
- You have feature requests or suggestions
- Documentation is unclear or incomplete

### Issue Report Template

When reporting issues, include:

```
**Environment:**
- reviewtask version: `reviewtask version`
- Operating system: 
- AI provider: Claude Code version X.X.X

**Problem:**
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior

**Logs:**
- Enable verbose mode and include relevant log output
- Sanitize any sensitive information (tokens, private repo names)

**Configuration:**
- Include .pr-review/config.json (sanitized)
- Mention any custom settings
```

For more help:
- [GitHub Issues](https://github.com/biwakonbu/reviewtask/issues)
- [Command Reference](commands.md)
- [Configuration Guide](configuration.md)