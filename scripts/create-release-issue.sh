#!/bin/bash

# GitHub Issue creation script for release notes
# Creates a standardized GitHub Issue for each release with comprehensive changelog

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISSUE_TEMPLATE_FILE="/tmp/release_issue_template.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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
    log_info "Checking prerequisites for issue creation..."
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a git repository"
        exit 1
    fi
    
    # Check if gh CLI is available
    if ! command -v gh > /dev/null; then
        log_error "GitHub CLI (gh) is required but not installed"
        exit 1
    fi
    
    # Check if we can access the repository
    if ! gh repo view > /dev/null 2>&1; then
        log_error "Cannot access repository via GitHub CLI. Please check authentication."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Determine release type based on semantic version changes
determine_release_type() {
    local current_version=$1
    local previous_version=$2
    
    if [[ -z "$previous_version" ]]; then
        echo "initial"
        return
    fi
    
    # Extract version numbers (remove 'v' prefix if present)
    current_clean=${current_version#v}
    previous_clean=${previous_version#v}
    
    # Parse version components
    IFS='.' read -ra current_parts <<< "$current_clean"
    IFS='.' read -ra previous_parts <<< "$previous_clean"
    
    local current_major=${current_parts[0]}
    local current_minor=${current_parts[1]}
    local current_patch=${current_parts[2]}
    
    local previous_major=${previous_parts[0]}
    local previous_minor=${previous_parts[1]}
    local previous_patch=${previous_parts[2]}
    
    # Determine release type
    if [[ "$current_major" -gt "$previous_major" ]]; then
        echo "major"
    elif [[ "$current_minor" -gt "$previous_minor" ]]; then
        echo "minor"
    elif [[ "$current_patch" -gt "$previous_patch" ]]; then
        echo "patch"
    else
        echo "unknown"
    fi
}

# Generate categorized changelog
generate_changelog() {
    local previous_tag=$1
    local current_version=$2
    local changelog_file=$3
    
    log_info "Generating categorized changelog..."
    
    if [[ -n "$previous_tag" ]]; then
        # Get all commits since last tag
        local commit_range="${previous_tag}..HEAD"
        
        # Features
        echo "### ✨ Features" >> "$changelog_file"
        if git log --pretty=format:"- %s (%h)" --reverse "$commit_range" --grep="feat" --grep="feature" | head -20 > /tmp/features.txt && [[ -s /tmp/features.txt ]]; then
            cat /tmp/features.txt >> "$changelog_file"
        else
            echo "*No new features in this release*" >> "$changelog_file"
        fi
        echo "" >> "$changelog_file"
        
        # Bug Fixes
        echo "### 🐛 Bug Fixes" >> "$changelog_file"
        if git log --pretty=format:"- %s (%h)" --reverse "$commit_range" --grep="fix" --grep="bug" | head -20 > /tmp/fixes.txt && [[ -s /tmp/fixes.txt ]]; then
            cat /tmp/fixes.txt >> "$changelog_file"
        else
            echo "*No bug fixes in this release*" >> "$changelog_file"
        fi
        echo "" >> "$changelog_file"
        
        # Documentation
        echo "### 📚 Documentation" >> "$changelog_file"
        if git log --pretty=format:"- %s (%h)" --reverse "$commit_range" --grep="docs" --grep="doc" | head -10 > /tmp/docs.txt && [[ -s /tmp/docs.txt ]]; then
            cat /tmp/docs.txt >> "$changelog_file"
        else
            echo "*No documentation changes in this release*" >> "$changelog_file"
        fi
        echo "" >> "$changelog_file"
        
        # Other Changes
        echo "### 🔧 Other Changes" >> "$changelog_file"
        if git log --pretty=format:"- %s (%h)" --reverse "$commit_range" --invert-grep --grep="feat" --grep="fix" --grep="docs" | head -15 > /tmp/others.txt && [[ -s /tmp/others.txt ]]; then
            cat /tmp/others.txt >> "$changelog_file"
        else
            echo "*No other changes in this release*" >> "$changelog_file"
        fi
        echo "" >> "$changelog_file"
    else
        # Initial release
        echo "### 🎉 Initial Release" >> "$changelog_file"
        echo "This is the first stable release of reviewtask featuring:" >> "$changelog_file"
        echo "- AI-powered PR review task generation and management" >> "$changelog_file"
        echo "- Smart review caching and state preservation system" >> "$changelog_file"
        echo "- Cross-platform binary distribution" >> "$changelog_file"
        echo "- Comprehensive CLI interface with subcommands" >> "$changelog_file"
        echo "" >> "$changelog_file"
    fi
    
    # Clean up temporary files
    rm -f /tmp/features.txt /tmp/fixes.txt /tmp/docs.txt /tmp/others.txt
}

# Create GitHub Issue template
create_issue_template() {
    local version=$1
    local release_type=$2
    local previous_tag=$3
    local template_file=$4
    
    log_info "Creating issue template for release $version..."
    
    # Determine release type emoji and description
    local release_emoji
    local release_desc
    case "$release_type" in
        "major")
            release_emoji="💥"
            release_desc="Major Release (Breaking Changes)"
            ;;
        "minor")
            release_emoji="✨"
            release_desc="Minor Release (New Features)"
            ;;
        "patch")
            release_emoji="🐛"
            release_desc="Patch Release (Bug Fixes)"
            ;;
        "initial")
            release_emoji="🎉"
            release_desc="Initial Release"
            ;;
        *)
            release_emoji="📦"
            release_desc="Release"
            ;;
    esac
    
    # Create issue template
    cat > "$template_file" << EOF
# ${release_emoji} Release ${version} - ${release_desc}

> **Release Information**
> - **Version**: ${version}
> - **Type**: ${release_desc}
> - **Date**: $(date -u +"%Y-%m-%d")
> - **Commit**: $(git rev-parse --short HEAD)

## 📋 Release Summary

This release includes various improvements and updates to reviewtask.

EOF
    
    # Add changelog section
    generate_changelog "$previous_tag" "$version" "$template_file"
    
    # Get repository information dynamically
    local repo_url=$(git remote get-url origin | sed 's/\.git$//')
    local repo_name=$(basename "$repo_url")
    local repo_owner=$(basename "$(dirname "$repo_url")")
    
    # Add installation and download information
    cat >> "$template_file" << EOF

## 📦 Installation & Downloads

### Download Binary
Download the appropriate binary for your platform from the [release assets](${repo_url}/releases/tag/${version}).

### Install with Go
\`\`\`bash
go install ${repo_url#https://}@${version}
\`\`\`

### Build from Source
\`\`\`bash
git clone ${repo_url}.git
cd ${repo_name}
git checkout ${version}
go build -o ${repo_name} .
\`\`\`

## 🔒 Security & Verification

Binary checksums are provided in the \`SHA256SUMS\` file attached to the release assets.

### Verify Download Integrity
\`\`\`bash
# Download checksum file and binary
curl -sL ${repo_url}/releases/download/${version}/SHA256SUMS -o SHA256SUMS
curl -sL ${repo_url}/releases/download/${version}/${repo_name}-\${version#v}-\${OS}-\${ARCH}.tar.gz -o ${repo_name}.tar.gz

# Verify checksum
sha256sum -c SHA256SUMS --ignore-missing
\`\`\`

## 🔗 Links

- **GitHub Release**: ${repo_url}/releases/tag/${version}
- **Full Changelog**: ${repo_url}/compare/${previous_tag}...${version}
- **Documentation**: ${repo_url}#readme

## 📞 Support

If you encounter any issues with this release, please:
1. Check the [troubleshooting guide](${repo_url}#troubleshooting)
2. Search [existing issues](${repo_url}/issues)
3. [Create a new issue](${repo_url}/issues/new) with details

---

*This issue was automatically created by the release automation system.*

EOF
    
    log_success "Issue template created: $template_file"
}

# Create GitHub Issue
create_release_issue() {
    local version=$1
    local template_file=$2
    local release_type=$3
    
    log_info "Creating GitHub Issue for release $version..."
    
    # Determine labels based on release type (only use release:type labels)
    local labels=""
    case "$release_type" in
        "major")
            labels="release:major"
            ;;
        "minor")
            labels="release:minor"
            ;;
        "patch")
            labels="release:patch"
            ;;
        "initial")
            labels="release:initial"
            ;;
    esac
    
    # Create the issue with or without labels
    local issue_url
    local issue_title="Release ${version} - ${release_type^} Release"
    
    if [[ -n "$labels" ]]; then
        # Try to create with labels first
        local create_error
        create_error=$(gh issue create \
            --title "$issue_title" \
            --body-file "$template_file" \
            --label "$labels" 2>&1) && issue_url="$create_error" || issue_url=""
        
        # If label creation failed, try without labels
        if [[ -z "$issue_url" ]]; then
            log_warning "Failed to create issue with labels: $create_error"
            log_warning "Trying without labels..."
            issue_url=$(gh issue create \
                --title "$issue_title" \
                --body-file "$template_file" 2>/dev/null || echo "")
        fi
    else
        # Create without labels
        issue_url=$(gh issue create \
            --title "$issue_title" \
            --body-file "$template_file" 2>/dev/null || echo "")
    fi
    
    if [[ -n "$issue_url" ]]; then
        log_success "Release issue created: $issue_url"
        echo "$issue_url"
        return 0
    else
        log_warning "Failed to create GitHub Issue - this won't prevent the release"
        return 1
    fi
}

# Main execution function
main() {
    local version=${1:-""}
    local previous_tag=""
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                version="$2"
                shift 2
                ;;
            --previous-tag)
                previous_tag="$2"
                shift 2
                ;;
            --help|-h)
                cat << EOF
Usage: $0 [OPTIONS] [VERSION]

Create a GitHub Issue for release notes.

OPTIONS:
  --version VERSION       Specify the release version (required)
  --previous-tag TAG      Previous release tag (auto-detected if not specified)
  --help, -h             Show this help message

EXAMPLES:
  $0 --version v1.2.3
  $0 --version v2.0.0 --previous-tag v1.9.5
  $0 v1.0.1

ENVIRONMENT:
  The script requires 'gh' (GitHub CLI) to be installed and authenticated.

EOF
                exit 0
                ;;
            *)
                if [[ -z "$version" ]]; then
                    version="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Validation
    if [[ -z "$version" ]]; then
        log_error "Version is required. Use --version or provide as first argument."
        log_info "Run '$0 --help' for usage information."
        exit 1
    fi
    
    # Ensure version starts with 'v'
    if [[ ! "$version" =~ ^v ]]; then
        version="v$version"
    fi
    
    # Validate version format
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
        log_error "Invalid version format: $version"
        log_info "Expected format: v<major>.<minor>.<patch>[-<prerelease>]"
        exit 1
    fi
    
    check_prerequisites
    
    # Auto-detect previous tag if not provided
    if [[ -z "$previous_tag" ]]; then
        previous_tag=$(git tag --sort=-version:refname | grep -v "^${version}$" | head -n1 || echo "")
        if [[ -n "$previous_tag" ]]; then
            log_info "Auto-detected previous tag: $previous_tag"
        else
            log_info "No previous tag found - treating as initial release"
        fi
    fi
    
    # Determine release type
    local release_type
    release_type=$(determine_release_type "$version" "$previous_tag")
    log_info "Release type: $release_type"
    
    # Create issue template
    create_issue_template "$version" "$release_type" "$previous_tag" "$ISSUE_TEMPLATE_FILE"
    
    # Create GitHub Issue
    local issue_url
    issue_url=$(create_release_issue "$version" "$ISSUE_TEMPLATE_FILE" "$release_type")
    
    # Clean up
    rm -f "$ISSUE_TEMPLATE_FILE"
    
    echo
    log_success "Release issue creation completed!"
    echo
    echo "📋 Issue URL: $issue_url"
    echo "🏷️  Labels: release:$release_type"
    echo "📝 Title: Release $version - ${release_type^} Release"
}

# Execute main function with all arguments
main "$@"