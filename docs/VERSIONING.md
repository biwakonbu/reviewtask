# Versioning Guide

## Overview

`gh-review-task` follows [Semantic Versioning 2.0.0](https://semver.org/) (SemVer) for all releases. This document describes our versioning rules, release process, and usage guidelines.

## Semantic Versioning Rules

### Version Format: `MAJOR.MINOR.PATCH`

Given a version number `MAJOR.MINOR.PATCH`, increment the:

1. **MAJOR** version when you make incompatible API changes
2. **MINOR** version when you add functionality in a backwards compatible manner  
3. **PATCH** version when you make backwards compatible bug fixes

### Version Examples

- `1.0.0` - Initial stable release
- `1.0.1` - Bug fix release
- `1.1.0` - New feature release (backwards compatible)
- `2.0.0` - Breaking change release

## When to Bump Each Version Component

### MAJOR Version (Breaking Changes)

**Increment when:**
- Removing or changing existing CLI commands or flags
- Changing configuration file format in incompatible ways
- Modifying data storage structure requiring migration
- Changing behavior that could break existing workflows
- Requiring different minimum Go version or dependencies

**Examples:**
- Removing the `--old-flag` option
- Changing `.pr-review/config.json` schema
- Requiring authentication where none was needed before
- Changing default behavior of existing commands

### MINOR Version (New Features)

**Increment when:**
- Adding new CLI commands or subcommands
- Adding new configuration options (with backwards compatibility)
- Adding new features that don't affect existing functionality
- Improving performance significantly
- Adding support for new platforms or environments

**Examples:**
- Adding `gh-review-task export` command
- Adding new options to existing commands
- Supporting new GitHub API features
- Adding new AI analysis capabilities
- Enhancing output formats

### PATCH Version (Bug Fixes)

**Increment when:**
- Fixing bugs without changing functionality
- Improving error messages or documentation
- Updating dependencies for security or stability
- Making internal refactoring that doesn't affect behavior
- Fixing typos or improving help text

**Examples:**
- Fixing authentication error handling
- Correcting task status updates
- Improving GitHub API error recovery
- Fixing edge cases in task generation

## Pre-release and Development Versions

### Development Versions
- Use `dev` for local development builds
- Format: `1.2.3-dev` or just `dev`

### Pre-release Versions (Future Use)
- Alpha: `1.2.3-alpha.1` (early testing)
- Beta: `1.2.3-beta.1` (feature complete, testing)
- Release Candidate: `1.2.3-rc.1` (potential final release)

## Version Management Commands

### Check Current Version

```bash
# Show version in version command output
./gh-review-task version

# Get just the version number
./scripts/version.sh current

# Show detailed version information
./scripts/version.sh info
```

### Bump Version

```bash
# Increment patch version (1.0.0 → 1.0.1)
./scripts/version.sh bump patch

# Increment minor version (1.0.1 → 1.1.0)
./scripts/version.sh bump minor

# Increment major version (1.1.0 → 2.0.0)
./scripts/version.sh bump major
```

### Set Specific Version

```bash
# Set exact version
./scripts/version.sh set 1.2.3
```

## Release Process

### 1. Preparation

```bash
# Check current state
./scripts/version.sh info

# Prepare release (dry-run and validation)
./scripts/release.sh prepare patch  # or minor/major
```

### 2. Create Release

```bash
# Create actual release
./scripts/release.sh release patch  # or minor/major
```

### 3. Automated Process

The release script automatically:
1. Validates working directory is clean
2. Bumps version using semantic versioning rules
3. Creates git tag (`v1.2.3`)
4. Builds cross-platform binaries
5. Generates release notes from git commits
6. Creates GitHub release
7. Uploads distribution packages and checksums

## Version Sources Priority

The version is determined in this order:

1. **Git tags** - Exact match for current commit (`git describe --tags --exact-match`)
2. **Latest git tag** - Most recent tag (`git describe --tags --abbrev=0`)
3. **VERSION file** - Local version file in repository root
4. **Default** - Falls back to `0.1.0`

## Git Tag Format

- **Format:** `vMAJOR.MINOR.PATCH` (e.g., `v1.2.3`)
- **Prefix:** Always use `v` prefix for git tags
- **Signing:** Tags should be annotated, not lightweight

```bash
# Correct tag creation (done automatically by scripts)
git tag -a v1.2.3 -m "Release v1.2.3"

# Incorrect (don't do this manually)
git tag 1.2.3  # missing 'v' prefix
git tag v1.2.3  # lightweight tag
```

## Binary Version Embedding

### Build-time Variables

Versions are embedded in binaries using Go's `-ldflags`:

```bash
go build -ldflags="
  -X main.version=1.2.3
  -X main.commitHash=abc1234
  -X main.buildDate=2023-12-01T10:00:00Z
" -o gh-review-task .
```

### Version Information Display

```bash
$ ./gh-review-task version
gh-review-task version 1.2.3
Commit: abc1234
Built: 2023-12-01T10:00:00Z
Go version: go1.21.0
OS/Arch: linux/amd64
```

## Release Naming and Documentation

### Release Names
- Use simple descriptive names: `Release v1.2.3`
- Avoid creative codenames for consistency

### Release Notes Structure

```markdown
# Release v1.2.3

## Changes
- Feature: Add new task export functionality
- Fix: Resolve authentication timeout issues
- Improvement: Enhance error message clarity

## Installation
[Installation instructions]

## Verification
[Checksum verification instructions]
```

## Compatibility Policy

### Backwards Compatibility
- **MINOR and PATCH versions**: Always backwards compatible
- **MAJOR versions**: May introduce breaking changes
- **Configuration**: Provide migration guides for breaking changes
- **Data**: Maintain data format compatibility within major versions

### Deprecation Policy
1. **Deprecation Notice**: Announce in MINOR release
2. **Deprecation Period**: Minimum 1 major version
3. **Removal**: Only in next MAJOR release

Example:
- v1.5.0: Deprecate `--old-flag` (show warning)
- v1.6.0-v1.x.x: Continue supporting with warnings
- v2.0.0: Remove `--old-flag` completely

## Version Validation

### Automated Checks
- Version format validation in scripts
- Build verification for all target platforms
- GitHub Actions workflow validation
- Cross-compilation testing

### Manual Verification
```bash
# Test version operations
./scripts/test_versioning.sh

# Verify cross-platform builds
./scripts/build.sh test

# Check GitHub Actions workflow
gh workflow run release.yml --ref v1.2.3
```

## Best Practices

### For Developers
1. **Always** use scripts for version management
2. **Never** manually edit version numbers in code
3. **Test** cross-compilation before releases
4. **Document** breaking changes clearly
5. **Follow** commit message conventions for release notes

### For Contributors
1. **Consider** version impact of changes
2. **Document** breaking changes in PR descriptions
3. **Test** version embedding in builds
4. **Verify** backwards compatibility

### For Users
1. **Pin** to specific versions in production
2. **Test** new versions in development first
3. **Read** release notes for breaking changes
4. **Verify** checksums for downloaded binaries

## Troubleshooting

### Common Issues

**Version shows as "dev"**
```bash
# Cause: No git tags or VERSION file
# Solution: Create initial tag or VERSION file
echo "1.0.0" > VERSION
git tag v1.0.0
```

**Version bump fails**
```bash
# Cause: Uncommitted changes
# Solution: Commit or stash changes first
git status
git add .
git commit -m "Prepare for release"
```

**Cross-compilation fails**
```bash
# Cause: Missing dependencies or invalid GOOS/GOARCH
# Solution: Test individual platforms
GOOS=linux GOARCH=amd64 go build .
```

### Getting Help

- Check version info: `./scripts/version.sh info`
- Test build process: `./scripts/build.sh test`
- Prepare release: `./scripts/release.sh prepare`
- Report issues: [GitHub Issues](https://github.com/biwakonbu/ai-pr-review-checker/issues)