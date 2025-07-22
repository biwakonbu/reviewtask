#!/bin/bash

# Setup Git Hooks for reviewtask
# This script installs the pre-push hook to ensure code quality before pushing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "🪝 Setting up Git hooks for reviewtask..."

# Check if we're in a git repository
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo "❌ Error: Not in a git repository"
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "$HOOKS_DIR"

# Install pre-push hook
echo "📝 Installing pre-push hook..."
cp "$SCRIPT_DIR/hooks/pre-push" "$HOOKS_DIR/pre-push"
chmod +x "$HOOKS_DIR/pre-push"

# Copy README for reference
cp "$SCRIPT_DIR/hooks/README.md" "$HOOKS_DIR/README.md"

echo "✅ Git hooks installed successfully!"
echo ""
echo "📚 The pre-push hook will now:"
echo "   🎨 Check code formatting (gofmt)"
echo "   🔍 Run linting (golangci-lint, if available)"
echo "   🧪 Execute all tests"
echo "   🔨 Verify build success"
echo ""
echo "💡 To skip the hook: git push --no-verify"
echo "📖 For more details: cat .git/hooks/README.md"