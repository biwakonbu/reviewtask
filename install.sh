#!/bin/bash
# reviewtask Installation Script Wrapper
# This is a lightweight wrapper that redirects to the actual installation script

set -e

# Colors for output
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}reviewtask Installation${NC}"
echo "Downloading installation script..."

# Download and execute the actual installation script
if command -v curl >/dev/null 2>&1; then
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- "$@"
elif command -v wget >/dev/null 2>&1; then
    wget -qO- https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- "$@"
else
    echo "Error: Neither curl nor wget is available. Please install one of them."
    exit 1
fi