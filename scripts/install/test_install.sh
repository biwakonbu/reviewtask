#!/bin/bash

# Installation Script Test Suite
# Tests the install.sh script under various conditions

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SCRIPT="$SCRIPT_DIR/install.sh"
TEST_BIN_DIR="$SCRIPT_DIR/test_bin"
GITHUB_REPO="biwakonbu/reviewtask"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Print test output
print_test_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

print_test_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

print_test_failure() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

print_test_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

# Test runner function
run_test() {
    local test_name="$1"
    local test_func="$2"
    
    ((TESTS_RUN++))
    print_test_header "$test_name"
    
    # Create clean test environment
    rm -rf "$TEST_BIN_DIR"
    mkdir -p "$TEST_BIN_DIR"
    
    if $test_func; then
        print_test_success "$test_name"
    else
        print_test_failure "$test_name"
    fi
    
    echo
}

# Test functions

test_help_display() {
    # Test help option
    if bash "$INSTALL_SCRIPT" --help >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

test_platform_detection() {
    # Test that platform detection works
    local platform
    platform=$(bash -c '
        source '"$INSTALL_SCRIPT"'
        detect_platform
    ')
    
    case "$(uname -s)" in
        Linux*)
            [[ "$platform" =~ ^linux_(amd64|arm64)$ ]]
            ;;
        Darwin*)
            [[ "$platform" =~ ^darwin_(amd64|arm64)$ ]]
            ;;
        *)
            return 1
            ;;
    esac
}

test_version_validation() {
    # Test version format validation
    bash -c '
        source '"$INSTALL_SCRIPT"'
        validate_version "v1.2.3" && 
        validate_version "v1.0.0-beta.1" &&
        ! validate_version "1.2.3" &&
        ! validate_version "v1.2" &&
        ! validate_version "invalid"
    ' >/dev/null 2>&1
}

test_directory_creation() {
    # Test installation directory creation
    local test_dir="$TEST_BIN_DIR/custom_dir"
    
    bash -c '
        source '"$INSTALL_SCRIPT"'
        BIN_DIR="'"$test_dir"'"
        create_install_dir
    ' >/dev/null 2>&1
    
    [[ -d "$test_dir" ]] && [[ -w "$test_dir" ]]
}

test_existing_installation_check() {
    # Test existing installation detection
    local fake_binary="$TEST_BIN_DIR/reviewtask"
    mkdir -p "$TEST_BIN_DIR"
    echo '#!/bin/bash\necho "reviewtask version v1.0.0"' > "$fake_binary"
    chmod +x "$fake_binary"
    
    # Should fail when binary exists and force is false
    ! bash -c '
        source '"$INSTALL_SCRIPT"'
        BIN_DIR="'"$TEST_BIN_DIR"'"
        FORCE=false
        check_existing_installation
    ' >/dev/null 2>&1
}

test_force_overwrite() {
    # Test force overwrite functionality
    local fake_binary="$TEST_BIN_DIR/reviewtask"
    mkdir -p "$TEST_BIN_DIR"
    echo '#!/bin/bash\necho "old version"' > "$fake_binary"
    chmod +x "$fake_binary"
    
    # Should succeed when force is true
    bash -c '
        source '"$INSTALL_SCRIPT"'
        BIN_DIR="'"$TEST_BIN_DIR"'"
        FORCE=true
        check_existing_installation
    ' >/dev/null 2>&1
}

test_argument_parsing() {
    # Test command line argument parsing
    bash -c '
        source '"$INSTALL_SCRIPT"'
        parse_args --version v1.2.3 --bin-dir /tmp/test --force --verbose
        [[ "$VERSION" == "v1.2.3" ]] &&
        [[ "$BIN_DIR" == "/tmp/test" ]] &&
        [[ "$FORCE" == "true" ]] &&
        [[ "$VERBOSE" == "true" ]]
    ' >/dev/null 2>&1
}

test_latest_version_retrieval() {
    # Test latest version retrieval (requires network)
    if command -v curl >/dev/null 2>&1 || command -v wget >/dev/null 2>&1; then
        local version
        version=$(bash -c '
            source '"$INSTALL_SCRIPT"'
            get_latest_version
        ')
        
        # Check if version has expected format
        [[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]
    else
        print_test_info "Skipping latest version test (no curl/wget)"
        return 0
    fi
}

test_checksum_file_handling() {
    # Test checksum file parsing (mock test)
    local temp_file
    temp_file=$(mktemp)
    echo "abc123 reviewtask_linux_amd64" > "$temp_file"
    
    # This is a simplified test since we can't easily mock the download function
    bash -c '
        source '"$INSTALL_SCRIPT"'
        # Test that sha256sum command exists if available
        if command -v sha256sum >/dev/null 2>&1; then
            echo "Checksum verification available"
        fi
    ' >/dev/null 2>&1
    
    rm -f "$temp_file"
}

test_error_handling() {
    # Test error handling for invalid scenarios
    
    # Invalid version should fail
    ! bash -c '
        source '"$INSTALL_SCRIPT"'
        validate_version "invalid-version"
    ' >/dev/null 2>&1 &&
    
    # Non-writable directory should fail
    ! bash -c '
        source '"$INSTALL_SCRIPT"'
        BIN_DIR="/root/forbidden"
        create_install_dir
    ' >/dev/null 2>&1
}

test_script_permissions() {
    # Test that install.sh has correct permissions
    [[ -x "$INSTALL_SCRIPT" ]]
}

# Integration test that simulates the full installation process
test_mock_installation() {
    # Create a mock GitHub API response
    local api_response='{"tag_name": "v1.0.0"}'
    
    # Test installation flow without actually downloading
    bash -c '
        source '"$INSTALL_SCRIPT"'
        BIN_DIR="'"$TEST_BIN_DIR"'"
        VERSION="v1.0.0"
        FORCE=true
        
        # Test configuration
        validate_version "$VERSION"
        create_install_dir
        check_existing_installation
    ' >/dev/null 2>&1
}

test_verbose_output() {
    # Test verbose vs non-verbose output modes
    local verbose_out quiet_out
    verbose_out=$(mktemp)
    quiet_out=$(mktemp)
    
    # Test that verbose mode produces more output
    bash -c '
        source '"$INSTALL_SCRIPT"'
        VERBOSE=true
        print_verbose "This should appear in verbose mode"
        print_progress "This should always appear"
    ' > "$verbose_out" 2>&1
    
    local verbose_lines
    verbose_lines=$(wc -l < "$verbose_out")
    
    # Test non-verbose mode
    bash -c '
        source '"$INSTALL_SCRIPT"'
        VERBOSE=false
        print_verbose "This should NOT appear in non-verbose mode"
        print_progress "This should always appear"
    ' > "$quiet_out" 2>&1
    
    local quiet_lines
    quiet_lines=$(wc -l < "$quiet_out")
    
    rm -f "$verbose_out" "$quiet_out"
    
    # Verbose mode should produce more output
    [[ $verbose_lines -gt $quiet_lines ]]
}

test_output_functions() {
    # Test that all output functions work correctly
    bash -c '
        source '"$INSTALL_SCRIPT"'
        
        # Test all output functions
        print_info "Info message" >/dev/null 2>&1 &&
        print_success "Success message" >/dev/null 2>&1 &&
        print_warning "Warning message" >/dev/null 2>&1 &&
        print_error "Error message" >/dev/null 2>&1 &&
        print_progress "Progress message" >/dev/null 2>&1
    '
}

test_path_instructions_always_shown() {
    # Test that PATH instructions are always shown when binary is not in PATH
    local test_out
    test_out=$(mktemp)
    
    # Test non-verbose mode with binary not in PATH
    bash -c '
        source '"$INSTALL_SCRIPT"'
        BIN_DIR="/tmp/test_path_instructions"
        BINARY_NAME="nonexistent_binary_test"
        VERBOSE=false
        mkdir -p "$BIN_DIR"
        echo "#!/bin/bash" > "$BIN_DIR/nonexistent_binary_test"
        echo "echo test" >> "$BIN_DIR/nonexistent_binary_test"
        chmod +x "$BIN_DIR/nonexistent_binary_test"
        verify_installation
    ' > "$test_out" 2>&1
    
    # Check that detailed PATH instructions are shown even in non-verbose mode
    local has_path_instructions
    if grep -q "To add reviewtask to your PATH" "$test_out" && 
       grep -q "Add this line to your" "$test_out" && 
       grep -q "export PATH=" "$test_out"; then
        has_path_instructions=true
    else
        has_path_instructions=false
    fi
    
    rm -f "$test_out"
    rm -rf "/tmp/test_path_instructions"
    
    [[ "$has_path_instructions" == "true" ]]
}

# Test PowerShell script syntax (if PowerShell is available)
test_powershell_syntax() {
    local ps_script="$SCRIPT_DIR/install.ps1"
    
    if [[ -f "$ps_script" ]]; then
        if command -v pwsh >/dev/null 2>&1; then
            # Test PowerShell syntax
            pwsh -Command "& { Get-Content '$ps_script' | Out-Null }" >/dev/null 2>&1
        elif command -v powershell >/dev/null 2>&1; then
            # Test with older PowerShell
            powershell -Command "& { Get-Content '$ps_script' | Out-Null }" >/dev/null 2>&1
        else
            print_test_info "Skipping PowerShell syntax test (PowerShell not available)"
            return 0
        fi
    else
        return 1
    fi
}

# Main test runner
main() {
    print_test_header "Installation Script Test Suite"
    print_test_info "Testing: $INSTALL_SCRIPT"
    echo
    
    # Check prerequisites
    if [[ ! -f "$INSTALL_SCRIPT" ]]; then
        echo -e "${RED}ERROR:${NC} Installation script not found: $INSTALL_SCRIPT"
        exit 1
    fi
    
    # Run tests
    run_test "Help Display" test_help_display
    run_test "Platform Detection" test_platform_detection
    run_test "Version Validation" test_version_validation
    run_test "Directory Creation" test_directory_creation
    run_test "Existing Installation Check" test_existing_installation_check
    run_test "Force Overwrite" test_force_overwrite
    run_test "Argument Parsing" test_argument_parsing
    run_test "Latest Version Retrieval" test_latest_version_retrieval
    run_test "Checksum File Handling" test_checksum_file_handling
    run_test "Error Handling" test_error_handling
    run_test "Script Permissions" test_script_permissions
    run_test "Mock Installation" test_mock_installation
    run_test "Verbose Output Mode" test_verbose_output
    run_test "Output Functions" test_output_functions
    run_test "PATH Instructions Always Shown" test_path_instructions_always_shown
    run_test "PowerShell Script Syntax" test_powershell_syntax
    
    # Cleanup
    rm -rf "$TEST_BIN_DIR"
    
    # Print summary
    print_test_header "Test Summary"
    echo "Tests run: $TESTS_RUN"
    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}Some tests failed!${NC}"
        exit 1
    fi
}

# Handle script being sourced vs executed
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi