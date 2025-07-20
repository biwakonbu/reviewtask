# Cross-platform build system for reviewtask
.PHONY: build build-all clean test version help

# Build variables
BINARY_NAME=reviewtask
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags for optimization and version info
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildDate=$(BUILD_DATE)"

# Default build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Cross-platform build targets
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
DIST_DIR=dist

build-all: clean
	@echo "Building cross-platform binaries..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		OUTPUT_NAME=$(BINARY_NAME)-$(VERSION)-$${GOOS}-$${GOARCH}; \
		if [ "$${GOOS}" = "windows" ]; then \
			OUTPUT_NAME=$${OUTPUT_NAME}.exe; \
		fi; \
		echo "Building for $${GOOS}/$${GOARCH}..."; \
		GOOS=$${GOOS} GOARCH=$${GOARCH} go build $(LDFLAGS) -o $(DIST_DIR)/$${OUTPUT_NAME} .; \
		if [ $$? -ne 0 ]; then \
			echo "Failed to build for $${GOOS}/$${GOARCH}"; \
			exit 1; \
		fi; \
	done
	@echo "Cross-platform build completed successfully!"

# Create distribution archives
package: build-all
	@echo "Creating distribution packages..."
	@cd $(DIST_DIR) && for file in $(BINARY_NAME)-$(VERSION)-*; do \
		case "$$file" in \
			*windows*) \
				zip "$${file%.*}.zip" "$$file"; \
				echo "Created package for $$file"; \
				;; \
			*) \
				tar -czf "$${file}.tar.gz" "$$file"; \
				echo "Created package for $$file"; \
				;; \
		esac; \
	done
	@echo "Distribution packages created successfully!"

# Generate checksums
checksums: package
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && sha256sum *.tar.gz *.zip > SHA256SUMS 2>/dev/null || sha256sum *.tar.gz > SHA256SUMS 2>/dev/null || sha256sum *.zip > SHA256SUMS 2>/dev/null || true
	@echo "Checksums generated in $(DIST_DIR)/SHA256SUMS"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(DIST_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean completed!"

# Run tests
test:
	go test -v ./...

# Test cross-compilation without building
test-cross-compile:
	@echo "Testing cross-compilation capabilities..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		echo "Testing compilation for $${GOOS}/$${GOARCH}..."; \
		GOOS=$${GOOS} GOARCH=$${GOARCH} go build -o /dev/null .; \
		if [ $$? -ne 0 ]; then \
			echo "Failed to compile for $${GOOS}/$${GOARCH}"; \
			exit 1; \
		fi; \
	done
	@echo "Cross-compilation test completed successfully!"

# Display version information
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build Date: $(BUILD_DATE)"

# Display help
help:
	@echo "Available targets:"
	@echo "  build              - Build binary for current platform"
	@echo "  build-all          - Build binaries for all supported platforms"
	@echo "  package            - Create distribution archives"
	@echo "  checksums          - Generate SHA256 checksums"
	@echo "  clean              - Clean build artifacts"
	@echo "  test               - Run tests"
	@echo "  test-cross-compile - Test cross-compilation without building"
	@echo "  version            - Display version information"
	@echo "  help               - Display this help message"
	@echo ""
	@echo "Supported platforms: $(PLATFORMS)"