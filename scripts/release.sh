#!/bin/bash

# Release automation script for reviewtask
# Handles version bumping, tagging, and release preparation

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION_SCRIPT="${SCRIPT_DIR}/version.sh"
BUILD_SCRIPT="${SCRIPT_DIR}/build.sh"
DETECT_LABEL_SCRIPT="${SCRIPT_DIR}/detect-release-label.sh"
RELEASE_ISSUE_SCRIPT="${SCRIPT_DIR}/create-release-issue.sh"
CHANGELOG_FILE="CHANGELOG.md"
RELEASE_NOTES_FILE="RELEASE_NOTES.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if there are source changes since last release (same as version.sh)
has_source_changes() {
    local last_tag
    last_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    
    if [[ -z "$last_tag" ]]; then
        # No previous tag, so we have changes
        return 0
    fi
    
    # Check for changes in files that affect the binary (Go source code and dependencies only)
    local changes
    changes=$(git diff --name-only "$last_tag"..HEAD | grep -E '^(cmd/.*\.go$|internal/.*\.go$|main\.go$|go\.mod$|go\.sum$)' || true)
    
    if [[ -n "$changes" ]]; then
        return 0  # Has changes
    else
        return 1  # No changes
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a git repository"
        exit 1
    fi
    
    # Check if we're on main branch
    local current_branch
    current_branch=$(git branch --show-current)
    if [ "$current_branch" != "main" ]; then
        log_warning "Not on main branch (current: $current_branch)"
        if [ "$AUTO_CONFIRM" = "true" ]; then
            log_info "Auto-confirming continuation (--yes flag or AUTO_CONFIRM=true)"
        else
            read -p "Continue anyway? (y/N): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                exit 1
            fi
        fi
    fi
    
    # Check if working directory is clean
    if ! git diff --cached --quiet || ! git diff --quiet; then
        log_error "Working directory is not clean. Please commit or stash changes."
        git status --porcelain
        exit 1
    fi
    
    # Check for source changes (unless forced)
    if [[ "$FORCE_RELEASE" != "true" ]] && ! has_source_changes; then
        log_error "No source changes detected since last release"
        log_info "Use --force flag to override this check"
        exit 1
    fi
    
    # Check if scripts exist
    if [ ! -f "$VERSION_SCRIPT" ]; then
        log_error "Version script not found: $VERSION_SCRIPT"
        exit 1
    fi
    
    if [ ! -f "$BUILD_SCRIPT" ]; then
        log_error "Build script not found: $BUILD_SCRIPT"
        exit 1
    fi
    
    # Check if gh CLI is available
    if ! command -v gh > /dev/null; then
        log_error "GitHub CLI (gh) is required but not installed"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Generate release notes from git commits
generate_release_notes() {
    local previous_tag=$1
    local new_version=$2
    
    log_info "Generating release notes..."
    
    # Create release notes file
    cat > "$RELEASE_NOTES_FILE" << EOF
# Release v${new_version}

## Changes

EOF
    
    if [ -n "$previous_tag" ]; then
        # Get commits since last tag
        git log "${previous_tag}..HEAD" --pretty=format:"- %s" --reverse >> "$RELEASE_NOTES_FILE"
    else
        # Get all commits if no previous tag
        git log --pretty=format:"- %s" --reverse >> "$RELEASE_NOTES_FILE"
    fi
    
    # Add additional sections
    cat >> "$RELEASE_NOTES_FILE" << EOF


## Installation

### Download Binary
Download the appropriate binary for your platform from the release assets below.

### Build from Source
\`\`\`bash
git clone https://github.com/biwakonbu/reviewtask.git
cd reviewtask
git checkout v${new_version}
go build -o reviewtask .
\`\`\`

## Checksums
See \`SHA256SUMS\` file in the release assets for binary checksums.
EOF
    
    log_success "Release notes generated: $RELEASE_NOTES_FILE"
}

# Create release
create_release() {
    local release_type=${1:-"patch"}
    local dry_run=${2:-false}
    
    check_prerequisites
    
    # Get current version
    local current_version
    current_version=$("$VERSION_SCRIPT" current)
    
    # Get previous tag for release notes
    local previous_tag
    previous_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    
    # Bump version
    log_info "Bumping version from $current_version ($release_type)"
    local new_version
    new_version=$("$VERSION_SCRIPT" bump "$release_type")
    
    if [ "$dry_run" = true ]; then
        log_info "DRY RUN: Would create release v$new_version"
        log_info "DRY RUN: Current HEAD: $(git rev-parse --short HEAD)"
        log_info "DRY RUN: Changes since $previous_tag:"
        if [ -n "$previous_tag" ]; then
            git log "${previous_tag}..HEAD" --oneline
        else
            git log --oneline
        fi
        return 0
    fi
    
    # Generate release notes
    generate_release_notes "$previous_tag" "$new_version"
    
    # Build release binaries
    log_info "Building release binaries..."
    VERSION="$new_version" "$BUILD_SCRIPT" full
    
    # Commit version file changes
    if git diff --quiet; then
        log_info "No changes to commit"
    else
        git add VERSION
        git commit -m "chore: bump version to v$new_version"
        current_branch=$(git branch --show-current)
        git push origin "HEAD:${current_branch}"
        log_success "Version bump committed and pushed"
    fi
    
    # Create and push tag
    git tag "v$new_version"
    git push origin "v$new_version"
    log_success "Tag v$new_version created and pushed"
    
    # Create GitHub release
    log_info "Creating GitHub release..."
    gh release create "v$new_version" \
        --title "Release v$new_version" \
        --notes-file "$RELEASE_NOTES_FILE" \
        --draft
    
    # Upload release assets
    log_info "Uploading release assets..."
    gh release upload "v$new_version" dist/*.tar.gz dist/*.zip dist/SHA256SUMS
    
    # Publish the release
    gh release edit "v$new_version" --draft=false
    
    # Create GitHub Issue for release notes
    log_info "Creating GitHub Issue for release documentation..."
    if [ -f "$RELEASE_ISSUE_SCRIPT" ]; then
        if "$RELEASE_ISSUE_SCRIPT" --version "v$new_version" --previous-tag "$previous_tag"; then
            log_success "Release issue created successfully"
        else
            log_warning "Failed to create release issue - continuing with release"
        fi
    else
        log_warning "Release issue script not found: $RELEASE_ISSUE_SCRIPT"
    fi
    
    log_success "Release v$new_version created successfully!"
    echo
    echo "View the release at: https://github.com/biwakonbu/reviewtask/releases/tag/v$new_version"
    
    # Clean up
    rm -f "$RELEASE_NOTES_FILE"
}

# Prepare release (without creating)
prepare_release() {
    local release_type=${1:-"patch"}
    
    check_prerequisites
    
    local current_version
    current_version=$("$VERSION_SCRIPT" current)
    
    local new_version
    new_version=$("$VERSION_SCRIPT" next "$release_type")
    
    log_info "Preparing release v$new_version..."
    
    # Test build
    log_info "Testing build process..."
    "$BUILD_SCRIPT" test
    
    # Generate preview of release notes
    local previous_tag
    previous_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    
    echo
    echo "Release Preview:"
    echo "  Current Version: $current_version"
    echo "  New Version: $new_version"
    echo "  Release Type: $release_type"
    echo "  Previous Tag: ${previous_tag:-"none"}"
    echo
    
    if [ -n "$previous_tag" ]; then
        echo "Changes since $previous_tag:"
        git log "${previous_tag}..HEAD" --oneline
    else
        echo "All commits (no previous tag):"
        git log --oneline
    fi
    
    echo
    log_info "Run '$0 release $release_type' to create the actual release"
}

# Global flags
AUTO_CONFIRM=${AUTO_CONFIRM:-false}
FORCE_RELEASE=${FORCE_RELEASE:-false}

# Detect release type from PR label
detect_release_type_from_pr() {
    local pr_number=$1
    
    if [ ! -f "$DETECT_LABEL_SCRIPT" ]; then
        log_error "Label detection script not found: $DETECT_LABEL_SCRIPT"
        exit 1
    fi
    
    log_info "Detecting release type from PR #$pr_number..."
    
    local release_type
    if release_type=$("$DETECT_LABEL_SCRIPT" -q "$pr_number" 2>/dev/null); then
        log_success "Detected release type: $release_type"
        echo "$release_type"
    else
        local exit_code=$?
        case $exit_code in
            1)
                log_error "No release label found on PR #$pr_number"
                log_info "Please add one of: release:major, release:minor, release:patch"
                ;;
            2)
                log_error "Multiple release labels found on PR #$pr_number"
                log_info "Please keep only one release label"
                ;;
            *)
                log_error "Failed to detect release label from PR #$pr_number"
                ;;
        esac
        exit 1
    fi
}

# Main execution
main() {
    # Parse arguments and collect non-flag arguments
    local args=()
    local from_pr=""
    while [[ $# -gt 0 ]]; do
        case $1 in
            --yes|-y)
                AUTO_CONFIRM=true
                shift
                ;;
            --force)
                FORCE_RELEASE=true
                shift
                ;;
            --from-pr)
                from_pr="$2"
                shift 2
                ;;
            *)
                args+=("$1")
                shift
                ;;
        esac
    done
    
    local command=${args[0]:-"prepare"}
    local release_type=${args[1]:-""}
    
    # If --from-pr is specified, detect release type from PR label
    if [ -n "$from_pr" ]; then
        if [ -n "$release_type" ]; then
            log_warning "Release type specified both via argument and --from-pr. Using PR label."
        fi
        release_type=$(detect_release_type_from_pr "$from_pr")
    elif [ -z "$release_type" ]; then
        # Default to patch if no release type specified
        release_type="patch"
    fi
    
    case "$command" in
        "prepare")
            prepare_release "$release_type"
            ;;
        "release")
            create_release "$release_type" false
            ;;
        "dry-run")
            create_release "$release_type" true
            ;;
        *)
            echo "Usage: $0 [OPTIONS] [COMMAND] [RELEASE_TYPE]"
            echo ""
            echo "OPTIONS:"
            echo "  --yes, -y             Auto-confirm prompts (useful for CI/CD)"
            echo "  --force               Force release even without source changes"
            echo "  --from-pr PR_NUMBER   Detect release type from PR/issue label"
            echo ""
            echo "COMMANDS:"
            echo "  prepare   - Prepare and preview release (default)"
            echo "  release   - Create actual release"
            echo "  dry-run   - Simulate release creation"
            echo ""
            echo "RELEASE TYPES:"
            echo "  major     - Breaking changes (x.0.0)"
            echo "  minor     - New features (x.y.0)"
            echo "  patch     - Bug fixes (x.y.z) [default]"
            echo ""
            echo "EXAMPLES:"
            echo "  $0                      # Prepare patch release"
            echo "  $0 prepare minor        # Prepare minor release"
            echo "  $0 release major        # Create major release"
            echo "  $0 --force release patch # Force patch release without source changes"
            echo "  $0 --from-pr 123        # Detect type from PR #123 and its linked issues"
            echo "  $0 --yes release --from-pr 456  # Auto-confirm release from PR"
            echo ""
            echo "ENVIRONMENT VARIABLES:"
            echo "  AUTO_CONFIRM=true  - Same as --yes flag"
            echo "  FORCE_RELEASE=true - Same as --force flag"
            echo ""
            echo "RELEASE PREVENTION:"
            echo "  Releases are blocked if:"
            echo "  - No release labels found on PR or linked issues"
            echo "  - No source changes detected since last release"
            echo "  Use --force flag to override source change check"
            echo ""
            echo "SOURCE CHANGE DETECTION:"
            echo "  Only these files trigger version bumps:"
            echo "    - Go source code: cmd/*.go, internal/*.go, main.go"
            echo "    - Go dependencies: go.mod, go.sum"
            echo "  Excluded (no version bump needed):"
            echo "    - Scripts, documentation, tests, configs, etc."
            echo ""
            echo "RELEASE LABELS (PR or Issue):"
            echo "  release:major - Triggers major version bump"
            echo "  release:minor - Triggers minor version bump"
            echo "  release:patch - Triggers patch version bump"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"