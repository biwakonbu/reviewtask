#!/usr/bin/env bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Default values
PR_NUMBER=""
REPO=""
QUIET=false

# Function to display usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS] PR_NUMBER

Detect release label from a GitHub Pull Request.

OPTIONS:
    -r, --repo OWNER/REPO    Specify the GitHub repository (default: current repo)
    -q, --quiet             Only output the release type (major/minor/patch)
    -h, --help              Display this help message

EXAMPLES:
    $0 123                  Detect release label from PR #123
    $0 -r owner/repo 45     Detect from PR #45 in specified repo
    $0 -q 123               Output only the release type

LABELS:
    release:major           Triggers a major version bump (X.0.0)
    release:minor           Triggers a minor version bump (x.Y.0)
    release:patch           Triggers a patch version bump (x.y.Z)

EXIT CODES:
    0   Success - exactly one release label found
    1   Error - no release label found
    2   Error - multiple release labels found
    3   Error - invalid arguments or GitHub CLI error
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

# Validate PR number
if [[ -z "$PR_NUMBER" ]]; then
    echo -e "${RED}Error: PR number is required${NC}" >&2
    usage
    exit 3
fi

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}" >&2
    echo "Please install it from: https://cli.github.com" >&2
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
    if [[ -n "$REPO_INFO" ]]; then
        OWNER=$(echo "$REPO_INFO" | jq -r '.owner.login')
        REPO_NAME=$(echo "$REPO_INFO" | jq -r '.name')
    else
        OWNER=""
        REPO_NAME=""
    fi
fi

# Function to get linked issues from Development section using GraphQL (priority method)
get_development_issues() {
    if [[ -z "$OWNER" || -z "$REPO_NAME" ]]; then
        return 1
    fi
    
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
            .subject.labels.nodes[].name |
            select(startswith("release:"))
        ' 2>/dev/null || echo ""
    fi
}

# Get PR details including linked issues
if ! PR_DATA=$(gh pr view "${PR_NUMBER}" "${GH_ARGS[@]}" --json labels,body,title 2>/dev/null); then
    echo -e "${RED}Error: Failed to fetch PR #${PR_NUMBER}${NC}" >&2
    exit 3
fi

# Get PR labels first
PR_LABELS=$(echo "$PR_DATA" | jq -r '.labels[].name')

# Collect all release labels from PR and linked issues
ALL_RELEASE_LABELS=""

# Add PR release labels
if [[ -n "$PR_LABELS" ]]; then
    PR_RELEASE_LABELS=$(echo "$PR_LABELS" | grep '^release:' || true)
    if [[ -n "$PR_RELEASE_LABELS" ]]; then
        ALL_RELEASE_LABELS="$PR_RELEASE_LABELS"
    fi
fi

# Priority 1: Try Development section first
DEV_RELEASE_LABELS=$(get_development_issues)
if [[ -n "$DEV_RELEASE_LABELS" ]]; then
    if [[ -n "$ALL_RELEASE_LABELS" ]]; then
        ALL_RELEASE_LABELS="$ALL_RELEASE_LABELS"$'\n'"$DEV_RELEASE_LABELS"
    else
        ALL_RELEASE_LABELS="$DEV_RELEASE_LABELS"
    fi
else
    # Priority 2: Fallback to text analysis of PR body and title
    PR_BODY=$(echo "$PR_DATA" | jq -r '.body // ""')
    PR_TITLE=$(echo "$PR_DATA" | jq -r '.title // ""')
    
    # Look for issue references in PR body and title (e.g., "fixes #123", "closes #456")
    LINKED_ISSUES=$(echo -e "$PR_BODY\n$PR_TITLE" | grep -oE '#[0-9]+|[Ff]ix(es|ed)?\s+#[0-9]+|[Cc]lose[sd]?\s+#[0-9]+|[Rr]esolve[sd]?\s+#[0-9]+' | grep -oE '[0-9]+' | sort -u || true)
    
    # Check labels from found issues
    if [[ -n "$LINKED_ISSUES" ]]; then
        while IFS= read -r issue_num; do
            if [[ -n "$issue_num" ]]; then
                if ISSUE_LABELS=$(gh issue view "$issue_num" "${GH_ARGS[@]}" --json labels -q '.labels[].name' 2>/dev/null); then
                    ISSUE_RELEASE_LABELS=$(echo "$ISSUE_LABELS" | grep '^release:' || true)
                    if [[ -n "$ISSUE_RELEASE_LABELS" ]]; then
                        if [[ -n "$ALL_RELEASE_LABELS" ]]; then
                            ALL_RELEASE_LABELS="$ALL_RELEASE_LABELS"$'\n'"$ISSUE_RELEASE_LABELS"
                        else
                            ALL_RELEASE_LABELS="$ISSUE_RELEASE_LABELS"
                        fi
                    fi
                fi
            fi
        done <<< "$LINKED_ISSUES"
    fi
fi

# Use collected labels
RELEASE_LABELS="$ALL_RELEASE_LABELS"

# Count release labels
LABEL_COUNT=$(echo "$RELEASE_LABELS" | grep -c '^release:' || echo 0)

# Validate label count
if [[ $LABEL_COUNT -eq 0 ]]; then
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${YELLOW}No release label found on PR #${PR_NUMBER} or its linked issues${NC}" >&2
        if [[ -n "$LINKED_ISSUES" ]]; then
            echo "Checked linked issues: $(echo "$LINKED_ISSUES" | tr '\n' ', ' | sed 's/, $//')" >&2
        else
            echo "No linked issues found in PR body or title" >&2
        fi
        echo "" >&2
        echo "To specify a release type, add one of these labels to the PR or its linked issue:" >&2
        echo "  - release:major (for breaking changes)" >&2
        echo "  - release:minor (for new features)" >&2
        echo "  - release:patch (for bug fixes)" >&2
    fi
    exit 1
elif [[ $LABEL_COUNT -gt 1 ]]; then
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${RED}Error: Multiple release labels found on PR #${PR_NUMBER} or its linked issues${NC}" >&2
        echo "Found labels:" >&2
        echo "$RELEASE_LABELS" | sed 's/^/  - /' >&2
        echo "" >&2
        echo "Please remove all but one release label from the PR and its linked issues." >&2
    fi
    exit 2
fi

# Extract release type
RELEASE_TYPE=$(echo "$RELEASE_LABELS" | sed 's/^release://')

# Validate release type
case "$RELEASE_TYPE" in
    major|minor|patch)
        if [[ "$QUIET" == "true" ]]; then
            echo "$RELEASE_TYPE"
        else
            echo -e "${GREEN}Found release label: release:${RELEASE_TYPE}${NC}"
            echo "This will trigger a ${RELEASE_TYPE} version bump"
        fi
        exit 0
        ;;
    *)
        echo -e "${RED}Error: Invalid release label 'release:${RELEASE_TYPE}'${NC}" >&2
        echo "Valid labels are: release:major, release:minor, release:patch" >&2
        exit 3
        ;;
esac