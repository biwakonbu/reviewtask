#!/bin/bash

# Version management script for reviewtask
# Provides operations for semantic versioning

set -e

# Configuration
VERSION_FILE="VERSION"
DEFAULT_VERSION="0.1.0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DETECT_LABEL_SCRIPT="${SCRIPT_DIR}/detect-release-label.sh"

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

# Get current version from various sources
get_current_version() {
    # Try git tags first
    if git describe --tags --exact-match HEAD 2>/dev/null; then
        return 0
    fi
    
    # Try latest git tag
    if git describe --tags --abbrev=0 2>/dev/null; then
        return 0
    fi
    
    # Try VERSION file
    if [ -f "${VERSION_FILE}" ]; then
        cat "${VERSION_FILE}"
        return 0
    fi
    
    # Default version
    echo "${DEFAULT_VERSION}"
}

# Parse semantic version
parse_version() {
    local version=$1
    # Remove 'v' prefix if present
    version=${version#v}
    
    if [[ $version =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
        echo "${BASH_REMATCH[1]} ${BASH_REMATCH[2]} ${BASH_REMATCH[3]}"
    else
        log_error "Invalid version format: $version (expected: MAJOR.MINOR.PATCH)"
        exit 1
    fi
}

# Increment version
increment_version() {
    local current_version=$1
    local increment_type=$2
    
    local major minor patch
    read -r major minor patch < <(parse_version "$current_version")
    
    case $increment_type in
        "major")
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        "minor")
            minor=$((minor + 1))
            patch=0
            ;;
        "patch")
            patch=$((patch + 1))
            ;;
        *)
            log_error "Invalid increment type: $increment_type (expected: major, minor, or patch)"
            exit 1
            ;;
    esac
    
    echo "${major}.${minor}.${patch}"
}

# Show current version
show_current() {
    local current_version
    current_version=$(get_current_version)
    echo "$current_version"
}

# Bump version
bump_version() {
    local increment_type=$1
    local force=${2:-false}
    
    if [ -z "$increment_type" ]; then
        log_error "Increment type required (major, minor, or patch)"
        exit 1
    fi
    
    # Check for source changes unless forced
    if [[ "$force" != "true" ]] && git rev-parse --git-dir > /dev/null 2>&1; then
        if ! has_source_changes; then
            log_error "No source changes detected since last release"
            log_info "Use 'bump --force $increment_type' to override this check"
            exit 1
        fi
    fi
    
    local current_version
    current_version=$(get_current_version)
    
    local new_version
    new_version=$(increment_version "$current_version" "$increment_type")
    
    log_info "Current version: $current_version"
    log_info "New version: $new_version"
    
    # Save to VERSION file
    echo "$new_version" > "$VERSION_FILE"
    log_success "Version file updated: $VERSION_FILE"
    
    # Create git tag if in git repository
    if git rev-parse --git-dir > /dev/null 2>&1; then
        if ! git diff --cached --quiet || ! git diff --quiet; then
            log_warning "Uncommitted changes detected – committing VERSION bump only"
            git add "$VERSION_FILE"
            git commit -m "chore: bump version to $new_version"
            git tag "v$new_version"
            log_success "Git tag created: v$new_version"
        else
            git tag "v$new_version"
            log_success "Git tag created: v$new_version"
        fi
    fi
    
    echo "$new_version"
}

# Set specific version
set_version() {
    local new_version=$1
    
    if [ -z "$new_version" ]; then
        log_error "Version required"
        exit 1
    fi
    
    # Validate version format
    parse_version "$new_version" > /dev/null
    
    echo "$new_version" > "$VERSION_FILE"
    log_success "Version set to: $new_version"
    
    echo "$new_version"
}

# Calculate next version without updating files
calculate_next_version() {
    local increment_type=$1
    
    if [ -z "$increment_type" ]; then
        log_error "Increment type required (major, minor, or patch)"
        exit 1
    fi
    
    local current_version
    current_version=$(get_current_version)
    
    local new_version
    new_version=$(increment_version "$current_version" "$increment_type")
    
    echo "$new_version"
}

# Check if there are source changes since last release
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

# Show version information
show_info() {
    local current_version
    current_version=$(get_current_version)
    
    local git_commit
    git_commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    local git_tag
    git_tag=$(git describe --tags --exact-match HEAD 2>/dev/null || echo "none")
    
    local is_dirty=""
    if git rev-parse --git-dir > /dev/null 2>&1; then
        if ! git diff --cached --quiet || ! git diff --quiet; then
            is_dirty=" (dirty)"
        fi
    fi
    
    local last_tag
    last_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "none")
    
    local has_changes="unknown"
    if git rev-parse --git-dir > /dev/null 2>&1; then
        if has_source_changes; then
            has_changes="yes"
        else
            has_changes="no"
        fi
    fi
    
    echo "Version Information:"
    echo "  Current Version: $current_version"
    echo "  Git Commit: $git_commit$is_dirty"
    echo "  Git Tag: $git_tag"
    echo "  Last Release Tag: $last_tag"
    echo "  Source Changes Since Last Release: $has_changes"
    echo "  Version File: $([ -f "$VERSION_FILE" ] && echo "exists" || echo "missing")"
}

# Bump version based on PR label
bump_from_pr() {
    local pr_number=$1
    
    if [ -z "$pr_number" ]; then
        log_error "PR number is required"
        echo "Usage: $0 bump-from-pr PR_NUMBER"
        exit 1
    fi
    
    if [ ! -f "$DETECT_LABEL_SCRIPT" ]; then
        log_error "Label detection script not found: $DETECT_LABEL_SCRIPT"
        exit 1
    fi
    
    log_info "Detecting release type from PR #$pr_number..."
    
    local release_type
    if release_type=$("$DETECT_LABEL_SCRIPT" -q "$pr_number" 2>/dev/null); then
        log_success "Detected release type: $release_type"
        bump_version "$release_type"
    else
        local exit_code=$?
        case $exit_code in
            1)
                log_error "No release label found on PR #$pr_number"
                echo "Please add one of: release:major, release:minor, release:patch"
                ;;
            2)
                log_error "Multiple release labels found on PR #$pr_number"
                echo "Please keep only one release label"
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
    local command=${1:-"current"}
    local force_flag=false
    
    # Check for --force flag
    if [[ "$command" == "--force" ]]; then
        force_flag=true
        command=${2:-"current"}
        shift # Remove --force from arguments
        shift # Remove command, so $2 becomes $1, etc.
        set -- "$command" "$@" # Rebuild arguments without --force
    fi
    
    case "$command" in
        "current")
            show_current
            ;;
        "bump")
            bump_version "$2" "$force_flag"
            ;;
        "bump-from-pr")
            bump_from_pr "$2"
            ;;
        "next")
            calculate_next_version "$2"
            ;;
        "set")
            set_version "$2"
            ;;
        "info")
            show_info
            ;;
        *)
            echo "Usage: $0 [--force] [COMMAND] [ARGUMENTS]"
            echo ""
            echo "OPTIONS:"
            echo "  --force              - Force version bump even without source changes"
            echo ""
            echo "COMMANDS:"
            echo "  current              - Show current version"
            echo "  bump <type>          - Bump version (major, minor, patch)"
            echo "  bump-from-pr <PR#>   - Bump version based on PR label"
            echo "  next <type>          - Calculate next version without updating"
            echo "  set <version>        - Set specific version"
            echo "  info                 - Show detailed version information"
            echo ""
            echo "EXAMPLES:"
            echo "  $0 current           - Show current version"
            echo "  $0 bump patch        - Increment patch version"
            echo "  $0 --force bump patch - Force increment patch version"
            echo "  $0 bump-from-pr 123  - Bump based on PR #123 label"
            echo "  $0 next minor        - Show what next minor version would be"
            echo "  $0 set 1.2.3         - Set version to 1.2.3"
            echo "  $0 info              - Show version details"
            echo ""
            echo "PR LABELS:"
            echo "  release:major - Major version bump"
            echo "  release:minor - Minor version bump"
            echo "  release:patch - Patch version bump"
            echo ""
            echo "SOURCE CHANGE DETECTION:"
            echo "  Version bumps are prevented if no binary changes are detected"
            echo "  since the last release. Use --force to override this check."
            echo "  Only these files trigger version bumps:"
            echo "    - Go source code: cmd/*.go, internal/*.go, main.go"
            echo "    - Go dependencies: go.mod, go.sum"
            echo "  Excluded (no version bump needed):"
            echo "    - Scripts, documentation, tests, configs, etc."
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"