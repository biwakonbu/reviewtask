#!/usr/bin/env bash
set -euo pipefail

# Auto-label PR from Development section linked issues
# This script reads PR's linked issues from GitHub's Development section
# and automatically applies release labels to the PR

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
PR_NUMBER=""
REPO=""
QUIET=false
DRY_RUN=false

# Function to display usage
usage() {
    cat << 'EOF'
Usage: $0 [OPTIONS] PR_NUMBER

Auto-label PR from Development section linked issues.

OPTIONS:
    -r, --repo OWNER/REPO    Specify the GitHub repository (default: current repo)
    -q, --quiet             Only output the result (minimal logging)
    -d, --dry-run           Show what would be done without making changes
    -h, --help              Display this help message

EXAMPLES:
    $0 123                  Auto-label PR #123 from linked issues
    $0 -r owner/repo 45     Auto-label PR #45 in specified repo
    $0 -d -q 123            Dry-run mode with minimal output

PRIORITY:
    1. Development section linked issues (GraphQL API)
    2. Text analysis fallback (PR body/title)
    3. Default to release:minor if no labels found

EXIT CODES:
    0   Success - labels applied or would be applied
    1   Error - no release labels found anywhere
    2   Error - multiple conflicting release labels
    3   Error - invalid arguments or GitHub API error
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--repo)
            REPO="$2"
            shift 2
            ;;
        -q|--quiet)
            QUIET=true
            shift
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            echo "Unknown option: $1" >&2
            usage
            exit 3
            ;;
        *)
            PR_NUMBER="$1"
            shift
            ;;
    esac
done

# Logging functions
log_info() {
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${BLUE}[INFO]${NC} $1" >&2
    fi
}

log_success() {
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
    fi
}

log_warning() {
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${YELLOW}[WARNING]${NC} $1" >&2
    fi
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Validate PR number
if [[ -z "$PR_NUMBER" ]]; then
    log_error "PR number is required"
    usage
    exit 3
fi

# Check prerequisites
if ! command -v gh &> /dev/null; then
    log_error "GitHub CLI (gh) is not installed"
    echo "Please install it from: https://cli.github.com" >&2
    exit 3
fi

if ! command -v jq &> /dev/null; then
    log_error "jq is not installed"
    echo "Please install it from your package manager (e.g., apt install jq, brew install jq)" >&2
    exit 3
fi

# Construct gh command arguments
GH_ARGS=()
if [[ -n "$REPO" ]]; then
    GH_ARGS+=(-R "$REPO")
fi

# Get repository info for GraphQL
if [[ -n "$REPO" ]]; then
    OWNER="${REPO%/*}"
    REPO_NAME="${REPO#*/}"
else
    # Get current repo info
    REPO_INFO=$(gh repo view "${GH_ARGS[@]}" --json owner,name 2>/dev/null || echo "")
    if [[ -z "$REPO_INFO" ]]; then
        log_error "Could not determine repository information"
        exit 3
    fi
    OWNER=$(echo "$REPO_INFO" | jq -r '.owner.login')
    REPO_NAME=$(echo "$REPO_INFO" | jq -r '.name')
fi

log_info "Processing PR #${PR_NUMBER} in ${OWNER}/${REPO_NAME}"

# Function to get linked issues from Development section using GraphQL
get_development_issues() {
    log_info "Checking Development section for linked issues..."
    
    local graphql_query='
    query($owner: String!, $repo: String!, $pr_number: Int!) {
      repository(owner: $owner, name: $repo) {
        pullRequest(number: $pr_number) {
          timelineItems(first: 50, itemTypes: [CONNECTED_EVENT]) {
            nodes {
              ... on ConnectedEvent {
                subject {
                  ... on Issue {
                    number
                    labels(first: 20) {
                      nodes {
                        name
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }'
    
    local result
    result=$(gh api graphql \
        --field query="$graphql_query" \
        --field owner="$OWNER" \
        --field repo="$REPO_NAME" \
        --field pr_number="$PR_NUMBER" 2>/dev/null || echo "")
    
    if [[ -n "$result" ]]; then
        echo "$result" | jq -r '
            .data.repository.pullRequest.timelineItems.nodes[]? |
            select(.subject) |
            {
                issue: .subject.number,
                labels: [.subject.labels.nodes[].name | select(startswith("release:"))]
            } |
            select(.labels | length > 0) |
            "\(.issue):\(.labels | join(","))"
        ' 2>/dev/null || echo ""
    fi
}

# Function to get linked issues from text analysis (fallback)
get_text_analysis_issues() {
    log_info "Falling back to text analysis for linked issues..."
    
    local pr_data
    pr_data=$(gh pr view "${PR_NUMBER}" "${GH_ARGS[@]}" --json body,title 2>/dev/null || echo "")
    
    if [[ -z "$pr_data" ]]; then
        return 1
    fi
    
    local pr_body pr_title
    pr_body=$(echo "$pr_data" | jq -r '.body // ""')
    pr_title=$(echo "$pr_data" | jq -r '.title // ""')
    
    # Look for issue references in PR body and title
    local linked_issues
    linked_issues=$(echo -e "$pr_body\n$pr_title" | 
        grep -oE '#[0-9]+|[Ff]ix(es|ed)?\s+#[0-9]+|[Cc]lose[sd]?\s+#[0-9]+|[Rr]esolve[sd]?\s+#[0-9]+' | 
        grep -oE '[0-9]+' | 
        sort -u || true)
    
    if [[ -z "$linked_issues" ]]; then
        return 1
    fi
    
    # Check labels from found issues
    while IFS= read -r issue_num; do
        if [[ -n "$issue_num" ]]; then
            local issue_labels
            issue_labels=$(gh issue view "$issue_num" "${GH_ARGS[@]}" --json labels -q '.labels[].name' 2>/dev/null | 
                grep '^release:' || true)
            
            if [[ -n "$issue_labels" ]]; then
                echo "${issue_num}:$(echo "$issue_labels" | tr '\n' ',')"
            fi
        fi
    done <<< "$linked_issues"
}

# Function to determine the highest priority release type
get_highest_priority_release_type() {
    local labels="$1"
    
    # Priority: major > minor > patch
    if echo "$labels" | grep -q "release:major"; then
        echo "major"
    elif echo "$labels" | grep -q "release:minor"; then
        echo "minor"
    elif echo "$labels" | grep -q "release:patch"; then
        echo "patch"
    else
        echo ""
    fi
}

# Function to apply label to PR
apply_label_to_pr() {
    local label="$1"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "DRY-RUN: Would apply label '${label}' to PR #${PR_NUMBER}"
        return 0
    fi
    
    log_info "Applying label '${label}' to PR #${PR_NUMBER}..."
    
    if gh pr edit "${PR_NUMBER}" "${GH_ARGS[@]}" --add-label "$label" 2>/dev/null; then
        log_success "Label '${label}' applied successfully"
        return 0
    else
        log_error "Failed to apply label '${label}'"
        return 1
    fi
}

# Main logic
main() {
    local all_release_labels=""
    local found_method=""
    
    # Try Development section first
    local dev_issues
    dev_issues=$(get_development_issues)
    
    if [[ -n "$dev_issues" ]]; then
        found_method="Development section"
        log_info "Found linked issues in Development section"
        
        while IFS= read -r line; do
            if [[ -n "$line" ]]; then
                local issue_labels="${line#*:}"
                all_release_labels="${all_release_labels} ${issue_labels//,/ }"
                log_info "Issue #${line%:*}: ${issue_labels//,/, }"
            fi
        done <<< "$dev_issues"
    else
        # Fallback to text analysis
        local text_issues
        text_issues=$(get_text_analysis_issues)
        
        if [[ -n "$text_issues" ]]; then
            found_method="Text analysis"
            log_info "Found linked issues via text analysis"
            
            while IFS= read -r line; do
                if [[ -n "$line" ]]; then
                    local issue_labels="${line#*:}"
                    all_release_labels="${all_release_labels} ${issue_labels//,/ }"
                    log_info "Issue #${line%:*}: ${issue_labels//,/, }"
                fi
            done <<< "$text_issues"
        fi
    fi
    
    # Clean up labels and remove duplicates
    all_release_labels=$(echo "$all_release_labels" | tr ' ' '\n' | grep '^release:' | sort -u | tr '\n' ' ')
    
    if [[ -z "$all_release_labels" ]]; then
        log_warning "No release labels found in linked issues"
        log_info "Applying default label: release:minor"
        
        if apply_label_to_pr "release:minor"; then
            if [[ "$QUIET" == "true" ]]; then
                echo "minor"
            else
                log_success "Default release:minor label applied (method: default)"
            fi
            exit 0
        else
            exit 3
        fi
    fi
    
    # Determine the highest priority release type
    local release_type
    release_type=$(get_highest_priority_release_type "$all_release_labels")
    
    if [[ -z "$release_type" ]]; then
        log_error "Invalid release labels found: $all_release_labels"
        exit 3
    fi
    
    local final_label="release:${release_type}"
    
    # Check if PR already has this label
    local current_labels
    current_labels=$(gh pr view "${PR_NUMBER}" "${GH_ARGS[@]}" --json labels -q '.labels[].name' 2>/dev/null | 
        grep '^release:' || true)
    
    if echo "$current_labels" | grep -q "^${final_label}$"; then
        log_info "PR already has the correct label: $final_label"
        if [[ "$QUIET" == "true" ]]; then
            echo "$release_type"
        else
            log_success "Label $final_label already present (method: $found_method)"
        fi
        exit 0
    fi
    
    # Remove existing release labels first
    if [[ -n "$current_labels" ]]; then
        log_info "Removing existing release labels..."
        while IFS= read -r existing_label; do
            if [[ -n "$existing_label" ]]; then
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_info "DRY-RUN: Would remove label '${existing_label}'"
                else
                    gh pr edit "${PR_NUMBER}" "${GH_ARGS[@]}" --remove-label "$existing_label" 2>/dev/null || true
                fi
            fi
        done <<< "$current_labels"
    fi
    
    # Apply the determined label
    if apply_label_to_pr "$final_label"; then
        if [[ "$QUIET" == "true" ]]; then
            echo "$release_type"
        else
            log_success "Applied label: $final_label (method: $found_method)"
        fi
        exit 0
    else
        exit 3
    fi
}

# Execute main function
main "$@"