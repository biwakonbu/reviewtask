#!/bin/bash

# Release automation script for gh-review-task
# Handles version bumping, tagging, and release preparation

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION_SCRIPT="${SCRIPT_DIR}/version.sh"
BUILD_SCRIPT="${SCRIPT_DIR}/build.sh"
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
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # Check if working directory is clean
    if ! git diff --cached --quiet || ! git diff --quiet; then
        log_error "Working directory is not clean. Please commit or stash changes."
        git status --porcelain
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
git clone https://github.com/biwakonbu/ai-pr-review-checker.git
cd ai-pr-review-checker
git checkout v${new_version}
go build -o gh-review-task .
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
        log_success "Version bump committed"
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
    
    log_success "Release v$new_version created successfully!"
    echo
    echo "View the release at: https://github.com/biwakonbu/ai-pr-review-checker/releases/tag/v$new_version"
    
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
    new_version=$("$VERSION_SCRIPT" current)
    # Calculate what the new version would be
    case "$release_type" in
        "major") new_version=$(echo "$current_version" | awk -F. '{print ($1+1)".0.0"}') ;;
        "minor") new_version=$(echo "$current_version" | awk -F. '{print $1"."($2+1)".0"}') ;;
        "patch") new_version=$(echo "$current_version" | awk -F. '{print $1"."$2"."($3+1)}') ;;
    esac
    
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

# Main execution
main() {
    local command=${1:-"prepare"}
    local release_type=${2:-"patch"}
    
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
            echo "Usage: $0 [prepare|release|dry-run] [major|minor|patch]"
            echo ""
            echo "Commands:"
            echo "  prepare   - Prepare and preview release (default)"
            echo "  release   - Create actual release"
            echo "  dry-run   - Simulate release creation"
            echo ""
            echo "Release Types:"
            echo "  major     - Breaking changes (x.0.0)"
            echo "  minor     - New features (x.y.0)"
            echo "  patch     - Bug fixes (x.y.z)"
            echo ""
            echo "Examples:"
            echo "  $0 prepare patch    - Prepare patch release"
            echo "  $0 release minor    - Create minor release"
            echo "  $0 dry-run major    - Simulate major release"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"