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

# Get PR labels
if ! PR_LABELS=$(gh pr view "${PR_NUMBER}" "${GH_ARGS[@]}" --json labels -q '.labels[].name' 2>/dev/null); then
    echo -e "${RED}Error: Failed to fetch PR #${PR_NUMBER}${NC}" >&2
    exit 3
fi

# Filter release labels
RELEASE_LABELS=$(echo "$PR_LABELS" | grep '^release:' || true)

# Count release labels
LABEL_COUNT=$(echo "$RELEASE_LABELS" | grep -c '^release:' || echo 0)

# Validate label count
if [[ $LABEL_COUNT -eq 0 ]]; then
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${YELLOW}No release label found on PR #${PR_NUMBER}${NC}" >&2
        echo "Available labels: $(echo "$PR_LABELS" | tr '\n' ', ' | sed 's/, $//')" >&2
        echo "" >&2
        echo "To specify a release type, add one of these labels to the PR:" >&2
        echo "  - release:major (for breaking changes)" >&2
        echo "  - release:minor (for new features)" >&2
        echo "  - release:patch (for bug fixes)" >&2
    fi
    exit 1
elif [[ $LABEL_COUNT -gt 1 ]]; then
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${RED}Error: Multiple release labels found on PR #${PR_NUMBER}${NC}" >&2
        echo "Found labels:" >&2
        echo "$RELEASE_LABELS" | sed 's/^/  - /' >&2
        echo "" >&2
        echo "Please remove all but one release label from the PR." >&2
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