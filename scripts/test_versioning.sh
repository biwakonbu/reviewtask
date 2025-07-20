#!/bin/bash

# Specification tests for release versioning system
# Tests compliance with Issue #13 requirements

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION_SCRIPT="${SCRIPT_DIR}/version.sh"
BUILD_SCRIPT="${SCRIPT_DIR}/build.sh"
RELEASE_SCRIPT="${SCRIPT_DIR}/release.sh"
TEST_DIR=$(mktemp -d)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

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

test_pass() {
    ((TESTS_RUN++))
    ((TESTS_PASSED++))
    log_success "✓ $1"
}

test_fail() {
    ((TESTS_RUN++))
    ((TESTS_FAILED++))
    log_error "✗ $1"
}

# Test helper functions
run_test() {
    local test_name="$1"
    shift
    
    log_info "Running test: $test_name"
    
    if "$@"; then
        test_pass "$test_name"
    else
        test_fail "$test_name"
    fi
}

# Test 1: Version command shows current version
test_version_command() {
    local output
    output=$(./gh-review-task version 2>/dev/null || echo "FAILED")
    
    if [[ "$output" == *"gh-review-task version"* ]]; then
        return 0
    else
        log_error "Version command output: $output"
        return 1
    fi
}

# Test 2: Version script operations
test_version_script_operations() {
    cd "$TEST_DIR"
    
    # Test current version
    local current_version
    current_version=$("$VERSION_SCRIPT" current)
    
    if [[ "$current_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        log_info "Current version format valid: $current_version"
    else
        log_error "Invalid version format: $current_version"
        return 1
    fi
    
    # Test version info
    local info_output
    info_output=$("$VERSION_SCRIPT" info)
    
    if [[ "$info_output" == *"Version Information:"* ]]; then
        log_info "Version info command works"
    else
        log_error "Version info failed"
        return 1
    fi
    
    return 0
}

# Test 3: Semantic versioning compliance
test_semantic_versioning() {
    cd "$TEST_DIR"
    
    # Initialize git repo for testing
    git init > /dev/null 2>&1
    git config user.email "test@example.com"
    git config user.name "Test User"
    
    # Set initial version
    local initial_version="1.0.0"
    "$VERSION_SCRIPT" set "$initial_version" > /dev/null
    
    # Test patch bump
    local patch_version
    patch_version=$("$VERSION_SCRIPT" bump patch)
    if [[ "$patch_version" != "1.0.1" ]]; then
        log_error "Patch bump failed: expected 1.0.1, got $patch_version"
        return 1
    fi
    
    # Test minor bump
    local minor_version
    minor_version=$("$VERSION_SCRIPT" bump minor)
    if [[ "$minor_version" != "1.1.0" ]]; then
        log_error "Minor bump failed: expected 1.1.0, got $minor_version"
        return 1
    fi
    
    # Test major bump
    local major_version
    major_version=$("$VERSION_SCRIPT" bump major)
    if [[ "$major_version" != "2.0.0" ]]; then
        log_error "Major bump failed: expected 2.0.0, got $major_version"
        return 1
    fi
    
    return 0
}

# Test 4: Build script functionality
test_build_script() {
    # Test cross-compilation capability
    if "$BUILD_SCRIPT" test > /dev/null 2>&1; then
        log_info "Cross-compilation test passed"
    else
        log_error "Cross-compilation test failed"
        return 1
    fi
    
    return 0
}

# Test 5: Version embedding in binary
test_version_embedding() {
    # Build binary with specific version
    local test_version="9.9.9"
    local test_commit="test123"
    local test_date="2023-01-01T00:00:00Z"
    
    VERSION="$test_version" COMMIT_HASH="$test_commit" BUILD_DATE="$test_date" \
        go build -ldflags "-X main.version=$test_version -X main.commitHash=$test_commit -X main.buildDate=$test_date" \
        -o "${TEST_DIR}/test-binary" . > /dev/null 2>&1
    
    if [ ! -f "${TEST_DIR}/test-binary" ]; then
        log_error "Failed to build test binary"
        return 1
    fi
    
    # Test version output
    local version_output
    version_output=$("${TEST_DIR}/test-binary" version 2>/dev/null || echo "FAILED")
    
    if [[ "$version_output" == *"$test_version"* ]] && \
       [[ "$version_output" == *"$test_commit"* ]] && \
       [[ "$version_output" == *"$test_date"* ]]; then
        log_info "Version embedding test passed"
        return 0
    else
        log_error "Version embedding test failed. Output: $version_output"
        return 1
    fi
}

# Test 6: Release script validation
test_release_script() {
    # Test dry-run functionality
    cd "$TEST_DIR"
    
    # Initialize git repo
    git init > /dev/null 2>&1
    git config user.email "test@example.com"
    git config user.name "Test User"
    git commit --allow-empty -m "Initial commit" > /dev/null 2>&1
    
    # Copy scripts to test directory
    cp "$VERSION_SCRIPT" ./version.sh
    cp "$BUILD_SCRIPT" ./build.sh
    cp "$RELEASE_SCRIPT" ./release.sh
    
    # Set initial version
    echo "1.0.0" > VERSION
    git add VERSION
    git commit -m "Add VERSION file" > /dev/null 2>&1
    
    # Test prepare command
    if ./release.sh prepare patch > /dev/null 2>&1; then
        log_info "Release prepare command works"
        return 0
    else
        log_error "Release prepare command failed"
        return 1
    fi
}

# Test 7: GitHub Actions workflow validation
test_github_actions_workflow() {
    local workflow_file=".github/workflows/release.yml"
    
    if [ ! -f "$workflow_file" ]; then
        log_error "GitHub Actions workflow file not found"
        return 1
    fi
    
    # Check for required elements
    local workflow_content
    workflow_content=$(cat "$workflow_file")
    
    local required_elements=(
        "on:"
        "push:"
        "tags:"
        "v*"
        "jobs:"
        "release:"
        "ubuntu-latest"
        "actions/checkout"
        "actions/setup-go"
        "go build"
        "cross-platform"
    )
    
    for element in "${required_elements[@]}"; do
        if [[ "$workflow_content" != *"$element"* ]]; then
            log_error "Workflow missing required element: $element"
            return 1
        fi
    done
    
    log_info "GitHub Actions workflow validation passed"
    return 0
}

# Main test execution
main() {
    log_info "Starting Release Versioning Specification Tests"
    log_info "Test directory: $TEST_DIR"
    echo
    
    # Store original directory
    ORIGINAL_DIR=$(pwd)
    
    # Run tests
    run_test "Version command shows current version" test_version_command
    run_test "Version script operations" test_version_script_operations  
    run_test "Semantic versioning compliance" test_semantic_versioning
    run_test "Build script functionality" test_build_script
    run_test "Version embedding in binary" test_version_embedding
    run_test "Release script validation" test_release_script
    run_test "GitHub Actions workflow validation" test_github_actions_workflow
    
    # Return to original directory
    cd "$ORIGINAL_DIR"
    
    # Clean up test directory
    rm -rf "$TEST_DIR"
    
    # Print results
    echo
    log_info "Test Results:"
    echo "  Total Tests: $TESTS_RUN"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log_success "All tests passed! Release versioning specification is compliant."
        exit 0
    else
        log_error "Some tests failed. Please fix the issues before proceeding."
        exit 1
    fi
}

# Execute main function
main "$@"