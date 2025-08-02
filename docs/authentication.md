# Authentication Guide

reviewtask requires GitHub authentication to access your repositories and pull request data. This guide covers all authentication methods and troubleshooting.

## Quick Setup

The easiest way to set up authentication:

```bash
reviewtask auth login
```

This interactive command will guide you through creating and configuring a GitHub token.

## Authentication Sources

reviewtask checks for authentication in this priority order:

### 1. Environment Variable (Highest Priority)

```bash
export GITHUB_TOKEN="your_token_here"
```

This method is recommended for:
- CI/CD environments
- Docker containers
- Temporary authentication

### 2. Local Configuration File

Stored in `.pr-review/auth.json` (automatically gitignored):

```json
{
  "github_token": "your_token_here"
}
```

This file is created automatically when you run `reviewtask auth login`.

### 3. GitHub CLI Integration (Fallback)

If you have the [GitHub CLI](https://cli.github.com/) installed and authenticated:

```bash
gh auth login
```

reviewtask will automatically use the GitHub CLI token as a fallback.

## Creating a GitHub Token

### Personal Access Token (Classic)

1. Go to [GitHub Settings > Developer settings > Personal access tokens](https://github.com/settings/tokens)
2. Click "Generate new token (classic)"
3. Set expiration and select scopes:

#### Required Scopes

For **private repositories**:
- `repo` (Full control of private repositories)

For **public repositories**:
- `public_repo` (Access public repositories)

For **organization repositories**:
- `read:org` (Read org and team membership)

### Fine-grained Personal Access Token

1. Go to [GitHub Settings > Developer settings > Personal access tokens > Fine-grained tokens](https://github.com/settings/personal-access-tokens/new)
2. Select repository access
3. Set permissions:
   - **Repository permissions**:
     - Pull requests: Read
     - Contents: Read
     - Metadata: Read
   - **Account permissions**:
     - Organization permissions: Read (if applicable)

## Authentication Commands

### Login

```bash
reviewtask auth login
```

Interactive setup that:
- Prompts for GitHub token
- Tests token permissions
- Saves to local configuration
- Verifies repository access

### Status Check

```bash
reviewtask auth status
```

Shows:
- Current authentication source
- Authenticated user information
- Token expiration (if available)

### Comprehensive Check

```bash
reviewtask auth check
```

Performs detailed validation:
- Token validity
- Required permissions
- Repository access
- Rate limit status

### Logout

```bash
reviewtask auth logout
```

Removes local authentication configuration.

## Repository Access Requirements

reviewtask needs access to:

- **Pull requests**: Read pull request data and reviews
- **Issues**: Access to issue comments (if reviewing issue-linked PRs)
- **Repository contents**: Basic repository information
- **Organization membership**: For organization repositories

## Troubleshooting Authentication

### Token Validation Failed

```bash
# Check token permissions
reviewtask auth check

# Common solutions:
# 1. Verify token hasn't expired
# 2. Check required scopes are selected
# 3. Ensure token has repository access
```

### Repository Access Denied

```bash
# For private repositories
# Ensure token has 'repo' scope

# For organization repositories  
# Ensure token has 'read:org' scope
# Check organization's third-party access settings
```

### Rate Limit Issues

```bash
# Check current rate limit status
reviewtask auth check

# GitHub API rate limits:
# - Authenticated: 5,000 requests/hour
# - Unauthenticated: 60 requests/hour
```

### Multiple Authentication Sources

If you have multiple authentication methods configured, reviewtask uses the highest priority source. To debug:

```bash
# Check which source is being used
reviewtask auth status

# Remove local config to use environment variable
reviewtask auth logout

# Unset environment variable to use GitHub CLI
unset GITHUB_TOKEN
```

## Security Best Practices

### Token Management

1. **Use minimal required scopes**: Only grant necessary permissions
2. **Set expiration dates**: Use tokens with reasonable expiration periods
3. **Rotate regularly**: Update tokens periodically
4. **Monitor usage**: Check GitHub's token usage in settings

### Environment Security

1. **Don't commit tokens**: Never commit `.pr-review/auth.json` or tokens to git
2. **Use secrets in CI**: Store tokens in CI/CD secret management
3. **Limit token exposure**: Avoid logging or displaying tokens
4. **Revoke unused tokens**: Clean up old or unused tokens

### Organization Settings

For organization repositories:

1. **Third-party access**: Ensure your organization allows personal access tokens
2. **SSO requirements**: Enable SSO for tokens if required
3. **Repository permissions**: Verify token has access to specific repositories

## Integration Examples

### CI/CD Environment

```yaml
# GitHub Actions
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

# GitLab CI
variables:
  GITHUB_TOKEN: $CI_GITHUB_TOKEN
```

### Docker Container

```bash
docker run -e GITHUB_TOKEN="$GITHUB_TOKEN" reviewtask
```

### Development Environment

```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export GITHUB_TOKEN="your_token_here"
```

## Advanced Authentication

### Multiple Organizations

For working with multiple GitHub organizations, you may need different tokens:

```bash
# Project-specific token
cd /path/to/org1/project
export GITHUB_TOKEN="org1_token"
reviewtask

# Different organization
cd /path/to/org2/project  
export GITHUB_TOKEN="org2_token"
reviewtask
```

### GitHub Enterprise

For GitHub Enterprise instances:

```bash
# Set enterprise API endpoint
export GITHUB_API_URL="https://github.company.com/api/v3"
export GITHUB_TOKEN="enterprise_token"
```

Note: reviewtask currently supports GitHub.com. Enterprise support may require additional configuration.

## Getting Help

If you're still having authentication issues:

1. **Check the logs**: Run with verbose mode in `.pr-review/config.json`
2. **Verify permissions**: Use `reviewtask auth check` for detailed validation
3. **Test manually**: Try accessing the GitHub API directly with your token
4. **Check GitHub status**: Verify GitHub API is operational

For additional help, see the [Troubleshooting Guide](troubleshooting.md) or open an issue on the [GitHub repository](https://github.com/biwakonbu/reviewtask/issues).