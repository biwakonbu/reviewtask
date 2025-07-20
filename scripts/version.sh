#!/bin/bash

# Version management script for gh-review-task
# Provides operations for semantic versioning

set -e

# Configuration
VERSION_FILE="VERSION"
DEFAULT_VERSION="0.1.0"

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
    
    if [ -z "$increment_type" ]; then
        log_error "Increment type required (major, minor, or patch)"
        exit 1
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
            git commit -m "chore: bump version to $new_version (auto-commit by script)"
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
    
    echo "Version Information:"
    echo "  Current Version: $current_version"
    echo "  Git Commit: $git_commit$is_dirty"
    echo "  Git Tag: $git_tag"
    echo "  Version File: $([ -f "$VERSION_FILE" ] && echo "exists" || echo "missing")"
}

# Main execution
main() {
    local command=${1:-"current"}
    
    case "$command" in
        "current")
            show_current
            ;;
        "bump")
            bump_version "$2"
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
            echo "Usage: $0 [current|bump|next|set|info] [arguments]"
            echo ""
            echo "Commands:"
            echo "  current           - Show current version"
            echo "  bump <type>       - Bump version (major, minor, patch)"
            echo "  next <type>       - Calculate next version without updating files"
            echo "  set <version>     - Set specific version"
            echo "  info              - Show detailed version information"
            echo ""
            echo "Examples:"
            echo "  $0 current        - Show current version"
            echo "  $0 bump patch     - Increment patch version"
            echo "  $0 next minor     - Show what next minor version would be"
            echo "  $0 set 1.2.3      - Set version to 1.2.3"
            echo "  $0 info           - Show version details"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"