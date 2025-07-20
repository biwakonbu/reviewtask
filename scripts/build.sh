#!/bin/bash

# Cross-platform build script for reviewtask
# This script builds binaries for multiple platforms and architectures

set -e

# Configuration
BINARY_NAME="reviewtask"
DIST_DIR="dist"
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

# Get version information
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE}"

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

# Print build information
print_build_info() {
    log_info "Build Information:"
    echo "  Binary Name: ${BINARY_NAME}"
    echo "  Version: ${VERSION}"
    echo "  Commit Hash: ${COMMIT_HASH}"
    echo "  Build Date: ${BUILD_DATE}"
    echo "  Platforms: ${#PLATFORMS[@]} targets"
    echo ""
}

# Clean previous builds
clean_dist() {
    if [ -d "${DIST_DIR}" ]; then
        log_info "Cleaning previous build artifacts..."
        rm -rf "${DIST_DIR}"
    fi
    mkdir -p "${DIST_DIR}"
}

# Build for all platforms
build_all() {
    log_info "Starting cross-platform build..."
    
    local success_count=0
    local total_count=${#PLATFORMS[@]}
    
    for platform in "${PLATFORMS[@]}"; do
        local goos=${platform%/*}
        local goarch=${platform#*/}
        local output_name="${BINARY_NAME}-${VERSION}-${goos}-${goarch}"
        
        if [ "${goos}" = "windows" ]; then
            output_name="${output_name}.exe"
        fi
        
        local output_path="${DIST_DIR}/${output_name}"
        
        log_info "Building for ${goos}/${goarch}..."
        
        if GOOS=${goos} GOARCH=${goarch} go build -ldflags="${LDFLAGS}" -o "${output_path}" .; then
            local file_size=$(du -h "${output_path}" | cut -f1)
            log_success "Built ${output_name} (${file_size})"
            ((success_count++))
        else
            log_error "Failed to build for ${goos}/${goarch}"
            return 1
        fi
    done
    
    log_success "Cross-platform build completed: ${success_count}/${total_count} platforms"
}

# Create distribution packages
create_packages() {
    log_info "Creating distribution packages..."
    
    cd "${DIST_DIR}"
    
    for file in ${BINARY_NAME}-${VERSION}-*; do
        if [[ "${file}" == *"windows"* ]]; then
            local zip_name="${file%.*}.zip"
            zip -q "${zip_name}" "${file}"
            log_success "Created ${zip_name}"
        else
            local tar_name="${file}.tar.gz"
            tar -czf "${tar_name}" "${file}"
            log_success "Created ${tar_name}"
        fi
    done
    
    cd ..
}

# Generate checksums
generate_checksums() {
    log_info "Generating checksums..."
    
    cd "${DIST_DIR}"
    sha256sum *.tar.gz *.zip > SHA256SUMS 2>/dev/null || true
    
    if [ -f "SHA256SUMS" ]; then
        log_success "Checksums generated in ${DIST_DIR}/SHA256SUMS"
        log_info "Checksum preview:"
        head -3 SHA256SUMS | sed 's/^/  /'
        if [ $(wc -l < SHA256SUMS) -gt 3 ]; then
            echo "  ... and $(($(wc -l < SHA256SUMS) - 3)) more files"
        fi
    else
        log_warning "No package files found for checksum generation"
    fi
    
    cd ..
}

# Test cross-compilation
test_cross_compile() {
    log_info "Testing cross-compilation capabilities..."
    
    for platform in "${PLATFORMS[@]}"; do
        local goos=${platform%/*}
        local goarch=${platform#*/}
        
        log_info "Testing compilation for ${goos}/${goarch}..."
        
        if GOOS=${goos} GOARCH=${goarch} go build -o /dev/null . 2>/dev/null; then
            log_success "Cross-compilation test passed for ${goos}/${goarch}"
        else
            log_error "Cross-compilation test failed for ${goos}/${goarch}"
            return 1
        fi
    done
    
    log_success "All cross-compilation tests passed!"
}

# Main execution
main() {
    local command=${1:-"build"}
    
    print_build_info
    
    case "${command}" in
        "build")
            clean_dist
            build_all
            ;;
        "package")
            clean_dist
            build_all
            create_packages
            ;;
        "full")
            clean_dist
            build_all
            create_packages
            generate_checksums
            ;;
        "test")
            test_cross_compile
            ;;
        "clean")
            if [ -d "${DIST_DIR}" ]; then
                rm -rf "${DIST_DIR}"
                log_success "Cleaned build artifacts"
            else
                log_info "No build artifacts to clean"
            fi
            ;;
        *)
            echo "Usage: $0 [build|package|full|test|clean]"
            echo ""
            echo "Commands:"
            echo "  build   - Build binaries for all platforms"
            echo "  package - Build and create distribution packages"
            echo "  full    - Build, package, and generate checksums"
            echo "  test    - Test cross-compilation without building"
            echo "  clean   - Clean build artifacts"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"