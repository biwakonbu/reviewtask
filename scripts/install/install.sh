#!/bin/bash

# reviewtask Installation Script
# Automatically detects platform and installs the appropriate binary

set -euo pipefail

# Default configuration
# Use user's local bin directory as default (no sudo required)
DEFAULT_BIN_DIR="$HOME/.local/bin"
DEFAULT_VERSION="latest"
GITHUB_REPO="biwakonbu/reviewtask"
BINARY_NAME="reviewtask"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables
BIN_DIR="$DEFAULT_BIN_DIR"
VERSION="$DEFAULT_VERSION"
FORCE=false
PRERELEASE=false

# Print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Show usage information
usage() {
    cat << EOF
reviewtask Installation Script

USAGE:
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/install.sh | bash
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/install.sh | bash -s -- [OPTIONS]

OPTIONS:
    --version VERSION    Install specific version (default: latest)
    --bin-dir DIR       Installation directory (default: ~/.local/bin)
    --force             Overwrite existing installation
    --prerelease        Include pre-release versions
    --help              Show this help message

EXAMPLES:
    # Install latest version to user's local directory (no sudo required)
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash

    # Install to system-wide location (requires sudo)
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | sudo bash -s -- --bin-dir /usr/local/bin

    # Install specific version
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --version v1.2.3

    # Install to custom directory
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --bin-dir ~/bin

    # Force overwrite existing installation
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --force
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                [[ -n $2 && $2 != --* ]] || { print_error "Missing value for --version"; usage; exit 1; }
                VERSION="$2"
                shift 2
                ;;
            --bin-dir)
                [[ -n $2 && $2 != --* ]] || { print_error "Missing value for --bin-dir"; usage; exit 1; }
                # Expand a leading ~ to the user's home directory
                case $2 in
                    "~"*) BIN_DIR="${2/#\~/$HOME}" ;;
                    *)    BIN_DIR="$2" ;;
                esac
                shift 2
                ;;
            --force)
                FORCE=true
                shift
                ;;
            --prerelease)
                PRERELEASE=true
                shift
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Detect operating system and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        CYGWIN*|MINGW*|MSYS*)
            print_error "Windows detected. Please use install.ps1 for Windows installation."
            exit 1
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Get latest release version from GitHub API
get_latest_version() {
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases"
    
    if [[ "$PRERELEASE" == "true" ]]; then
        api_url="${api_url}"
    else
        api_url="${api_url}/latest"
    fi

    if command -v curl >/dev/null 2>&1; then
        if [[ "$PRERELEASE" == "true" ]]; then
            curl -s "$api_url" | grep '"tag_name":' | head -1 | sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
        else
            curl -s "$api_url" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
        fi
    elif command -v wget >/dev/null 2>&1; then
        if [[ "$PRERELEASE" == "true" ]]; then
            wget -qO- "$api_url" | grep '"tag_name":' | head -1 | sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
        else
            wget -qO- "$api_url" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
        fi
    else
        print_error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi
}

# Validate version format
validate_version() {
    local version="$1"
    
    if [[ "$version" == "latest" ]]; then
        return 0
    fi
    
    # Check if version starts with 'v' followed by semantic version
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([+-][a-zA-Z0-9.-]*)?$ ]]; then
        print_error "Invalid version format: $version"
        print_info "Version should be in format: v1.2.3 or v1.2.3-beta.1"
        exit 1
    fi
}

# Check if binary already exists
check_existing_installation() {
    local binary_path="$BIN_DIR/$BINARY_NAME"
    
    if [[ -f "$binary_path" ]] && [[ "$FORCE" != "true" ]]; then
        print_warning "reviewtask is already installed at $binary_path"
        local current_version
        current_version=$("$binary_path" version 2>/dev/null | head -1 | awk '{print $3}' || echo "unknown")
        print_info "Current version: $current_version"
        print_info "Use --force to overwrite the existing installation"
        exit 1
    fi
}

# Create installation directory if it doesn't exist
create_install_dir() {
    if [[ ! -d "$BIN_DIR" ]]; then
        print_info "Creating installation directory: $BIN_DIR"
        if ! mkdir -p "$BIN_DIR" 2>/dev/null; then
            print_error "Failed to create directory $BIN_DIR"
            print_info "You may need to run with sudo or choose a different directory with --bin-dir"
            exit 1
        fi
    fi
    
    # Check if directory is writable
    if [[ ! -w "$BIN_DIR" ]]; then
        print_error "Directory $BIN_DIR is not writable"
        print_info "You may need to run with sudo or choose a different directory with --bin-dir"
        exit 1
    fi
}

# Download file with checksum verification
download_with_verification() {
    local url="$1"
    local output_file="$2"
    local checksum_url="$3"
    
    print_info "Downloading $url"
    
    # Download the binary
    if command -v curl >/dev/null 2>&1; then
        if ! curl -fsSL "$url" -o "$output_file"; then
            print_error "Failed to download $url"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "$url" -O "$output_file"; then
            print_error "Failed to download $url"
            exit 1
        fi
    else
        print_error "Neither curl nor wget is available"
        exit 1
    fi
    
    # Verify checksum if available
    if [[ -n "$checksum_url" ]]; then
        local hasher
        if command -v sha256sum >/dev/null 2>&1; then
            hasher=(sha256sum)
        elif command -v shasum >/dev/null 2>&1; then
            hasher=(shasum -a 256)
        else
            print_warning "No SHA-256 tool found – skipping checksum verification"
            return
        fi

        print_info "Verifying checksum..."
        local expected_checksum
        if command -v curl >/dev/null 2>&1; then
            expected_checksum=$(curl -fsSL "$checksum_url" | grep "$(basename "$output_file")" | awk '{print $1}')
        elif command -v wget >/dev/null 2>&1; then
            expected_checksum=$(wget -qO- "$checksum_url" | grep "$(basename "$output_file")" | awk '{print $1}')
        fi

        if [[ -n "$expected_checksum" ]]; then
            local actual_checksum
            actual_checksum=$("${hasher[@]}" "$output_file" | awk '{print $1}')

            if [[ "$actual_checksum" != "$expected_checksum" ]]; then
                print_error "Checksum verification failed"
                print_error "Expected: $expected_checksum"
                print_error "Actual: $actual_checksum"
                rm -f "$output_file"
                exit 1
            fi
            print_success "Checksum verification passed"
        else
            print_error "Checksum not found for $(basename "$output_file"); aborting"
            rm -f "$output_file"
            exit 1
        fi
    fi
}

# Install the binary
install_binary() {
    local platform="$1"
    local version="$2"
    
    # Resolve latest version if needed
    if [[ "$version" == "latest" ]]; then
        print_info "Resolving latest version..."
        version=$(get_latest_version)
        if [[ -z "$version" ]]; then
            print_error "Failed to determine latest version"
            exit 1
        fi
        print_info "Latest version: $version"
    fi
    
    validate_version "$version"
    
    # Construct download URLs
    # Align with build artefact naming scheme: reviewtask-<version>-<os>-<arch>
    local platform_dash=$(echo "$platform" | tr '_' '-')
    local archive_ext="tar.gz"
    if [[ "$platform" == windows* ]]; then
        archive_ext="zip"
    fi
    local archive_filename="${BINARY_NAME}-${version}-${platform_dash}.${archive_ext}"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${archive_filename}"
    local checksum_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/SHA256SUMS"
    
    # Create temporary directory
    local temp_dir
    temp_dir=$(mktemp -d)
    local temp_archive="$temp_dir/$archive_filename"
    local temp_binary="$temp_dir/$BINARY_NAME"
    
    # Cleanup function with proper variable handling
    trap "rm -rf '$temp_dir'" EXIT
    
    # Download and verify the archive
    download_with_verification "$download_url" "$temp_archive" "$checksum_url"
    
    # Extract the binary from archive
    print_info "Extracting binary from archive..."
    case "$archive_ext" in
        "tar.gz")
            if ! tar -xzf "$temp_archive" -C "$temp_dir"; then
                print_error "Failed to extract tar.gz archive"
                exit 1
            fi
            ;;
        "zip")
            if ! unzip -q "$temp_archive" -d "$temp_dir"; then
                print_error "Failed to extract zip archive"
                exit 1
            fi
            ;;
        *)
            print_error "Unknown archive format: $archive_ext"
            exit 1
            ;;
    esac
    
    # Find the binary (it might be in a subdirectory or have version in name)
    if [[ ! -f "$temp_binary" ]]; then
        # Try to find the binary - it might have the version in its name
        temp_binary=$(find "$temp_dir" -name "${BINARY_NAME}*" -type f -executable | head -1)
        if [[ -z "$temp_binary" ]]; then
            print_error "Binary not found in archive"
            exit 1
        fi
    fi
    
    # Make it executable
    chmod +x "$temp_binary"
    
    # Ensure installation directory exists
    if [ ! -d "$BIN_DIR" ]; then
        print_info "Creating installation directory: $BIN_DIR"
        if ! mkdir -p "$BIN_DIR" 2>/dev/null; then
            print_error "Failed to create directory: $BIN_DIR"
            print_info "Try one of the following:"
            print_info "  1. Run: mkdir -p $BIN_DIR (then re-run this installer)"
            print_info "  2. Use --bin-dir to specify a different directory"
            print_info "  3. For system-wide install: curl ... | sudo bash -s -- --bin-dir /usr/local/bin"
            exit 1
        fi
    fi
    
    # Check write permissions
    if [ ! -w "$BIN_DIR" ]; then
        print_error "No write permission to: $BIN_DIR"
        print_info "Try one of the following:"
        print_info "  1. Use --bin-dir to specify a writable directory (e.g., --bin-dir ~/.local/bin)"
        print_info "  2. For system-wide install: curl ... | sudo bash -s -- --bin-dir /usr/local/bin"
        exit 1
    fi
    
    # Move to final location
    local final_path="$BIN_DIR/$BINARY_NAME"
    print_info "Installing to $final_path"
    
    if ! install -m 0755 "$temp_binary" "${final_path}.tmp" || ! mv -f "${final_path}.tmp" "$final_path"; then
        print_error "Failed to install binary to $final_path"
        rm -f "${final_path}.tmp"
        exit 1
    fi
    
    print_success "Successfully installed reviewtask $version to $final_path"
}

# Detect user's shell
detect_shell() {
    local shell_name=""
    
    # Try to detect from SHELL environment variable
    if [[ -n "$SHELL" ]]; then
        shell_name=$(basename "$SHELL")
    fi
    
    # Fallback to checking running processes
    if [[ -z "$shell_name" ]] && command -v ps >/dev/null 2>&1; then
        shell_name=$(ps -p $$ -o comm= 2>/dev/null | xargs basename 2>/dev/null)
    fi
    
    # Normalize shell name
    case "$shell_name" in
        bash|sh)
            echo "bash"
            ;;
        zsh)
            echo "zsh"
            ;;
        fish)
            echo "fish"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# Get shell configuration file
get_shell_config() {
    local shell_type="$1"
    
    case "$shell_type" in
        bash)
            if [[ -f "$HOME/.bashrc" ]]; then
                echo "$HOME/.bashrc"
            elif [[ -f "$HOME/.bash_profile" ]]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.bashrc"
            fi
            ;;
        zsh)
            if [[ -f "$HOME/.zshrc" ]]; then
                echo "$HOME/.zshrc"
            else
                echo "$HOME/.zshrc"
            fi
            ;;
        fish)
            echo "$HOME/.config/fish/config.fish"
            ;;
        *)
            echo ""
            ;;
    esac
}

# Show PATH configuration instructions
show_path_instructions() {
    local shell_type
    shell_type=$(detect_shell)
    local config_file
    config_file=$(get_shell_config "$shell_type")
    
    print_warning "$BIN_DIR is not in your PATH"
    print_info ""
    print_info "To add reviewtask to your PATH, follow these instructions:"
    print_info ""
    
    case "$shell_type" in
        bash)
            print_info "For Bash users:"
            print_info "  1. Add this line to your $config_file:"
            print_info "     ${GREEN}export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
            print_info ""
            print_info "  2. Then reload your shell configuration:"
            print_info "     ${GREEN}source $config_file${NC}"
            ;;
        zsh)
            print_info "For Zsh users:"
            print_info "  1. Add this line to your $config_file:"
            print_info "     ${GREEN}export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
            print_info ""
            print_info "  2. Then reload your shell configuration:"
            print_info "     ${GREEN}source $config_file${NC}"
            ;;
        fish)
            print_info "For Fish users:"
            print_info "  1. Add this to your $config_file:"
            print_info "     ${GREEN}set -gx PATH \$HOME/.local/bin \$PATH${NC}"
            print_info ""
            print_info "  2. Then reload your shell configuration:"
            print_info "     ${GREEN}source $config_file${NC}"
            ;;
        *)
            print_info "For other shells:"
            print_info "  Add $HOME/.local/bin to your PATH environment variable"
            print_info "  The exact method depends on your shell"
            ;;
    esac
    
    print_info ""
    print_info "Alternatively, you can run reviewtask with the full path:"
    print_info "  ${GREEN}$BIN_DIR/$BINARY_NAME${NC}"
    print_info ""
    print_info "After updating your PATH, you can run:"
    print_info "  ${GREEN}reviewtask --help${NC}"
}

# Verify installation
verify_installation() {
    local binary_path="$BIN_DIR/$BINARY_NAME"
    
    print_info "Verifying installation..."
    
    # Check if binary exists and is executable
    if [[ ! -x "$binary_path" ]]; then
        print_error "Binary is not executable: $binary_path"
        exit 1
    fi
    
    # Check if binary works
    if ! "$binary_path" version >/dev/null 2>&1; then
        print_error "Binary verification failed: $binary_path version"
        exit 1
    fi
    
    local installed_version
    installed_version=$("$binary_path" version 2>/dev/null | head -1 | awk '{print $3}' || echo "unknown")
    print_success "Installation verified successfully"
    print_info "Installed version: $installed_version"
    
    # Check if binary is in PATH
    if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
        show_path_instructions
    else
        print_success "reviewtask is available in your PATH"
        print_info "You can now run: reviewtask --help"
    fi
}

# Main installation function
main() {
    print_info "reviewtask Installation Script"
    print_info "Repository: https://github.com/$GITHUB_REPO"
    
    # Parse arguments
    parse_args "$@"
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    print_info "Detected platform: $platform"
    
    # Show configuration
    print_info "Configuration:"
    print_info "  Version: $VERSION"
    print_info "  Install directory: $BIN_DIR"
    print_info "  Force overwrite: $FORCE"
    print_info "  Include prereleases: $PRERELEASE"
    
    # Check existing installation
    check_existing_installation
    
    # Create installation directory
    create_install_dir
    
    # Install binary
    install_binary "$platform" "$VERSION"
    
    # Verify installation
    verify_installation
    
    print_success "Installation completed successfully!"
}

# Run main function with all arguments only if script is executed directly
# When piped from curl, BASH_SOURCE might not be set, so handle that case
if [[ "${BASH_SOURCE[0]:-}" == "${0}" ]] || [[ -z "${BASH_SOURCE[0]:-}" ]]; then
    main "$@"
fi