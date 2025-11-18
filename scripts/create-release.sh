#!/bin/bash

# Script to manually create GitHub release
# Run this script to create the v0.6.0-test-fix1-macos release

set -e

echo "=== Creating macOS Test Release on GitHub ==="
echo ""

# Configuration
TAG="v0.6.0-test-fix1-macos"
TITLE="v0.6.0-test-fix1 - macOS Test Release"
RELEASE_DIR="releases/v0.6.0-test-fix1-macos"

echo "Checking prerequisites..."

# Check if gh CLI is available
if ! command -v gh &> /dev/null; then
    echo "ERROR: GitHub CLI (gh) is not installed"
    echo ""
    echo "Please install it:"
    echo "  macOS:   brew install gh"
    echo "  Linux:   See https://github.com/cli/cli/blob/trunk/docs/install_linux.md"
    echo ""
    echo "Then authenticate:"
    echo "  gh auth login"
    echo ""
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo "ERROR: Not authenticated with GitHub"
    echo ""
    echo "Please run: gh auth login"
    echo ""
    exit 1
fi

echo "✓ GitHub CLI is installed and authenticated"
echo ""

# Ensure we're on the right branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG" ]; then
    echo "Switching to branch: claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG"
    git checkout claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
    git pull
fi

echo "Current branch: $(git branch --show-current)"
echo ""

# Check if release files exist
if [ ! -d "$RELEASE_DIR" ]; then
    echo "ERROR: Release directory not found: $RELEASE_DIR"
    echo ""
    echo "Please run: git pull origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG"
    echo ""
    exit 1
fi

echo "✓ Release files found in: $RELEASE_DIR"
echo ""

# List files to be uploaded
echo "Files to upload:"
ls -lh "$RELEASE_DIR"/*.tar.gz "$RELEASE_DIR"/checksums*.txt "$RELEASE_DIR"/BUILD_INFO_MACOS.md 2>/dev/null || true
echo ""

# Check if tag exists locally
if ! git rev-parse "$TAG" >/dev/null 2>&1; then
    echo "Creating tag: $TAG at commit 1c98bc3"
    git tag -a "$TAG" 1c98bc3 -m "macOS Test Release v0.6.0-test-fix1"
    echo "✓ Tag created"
else
    echo "✓ Tag already exists: $TAG"
fi
echo ""

# Try to push the tag (may fail due to permissions, but that's ok)
echo "Attempting to push tag..."
if git push origin "$TAG" 2>/dev/null; then
    echo "✓ Tag pushed to remote"
else
    echo "⚠ Could not push tag (will create release anyway)"
fi
echo ""

# Create release using gh CLI
echo "Creating GitHub release..."
echo ""

gh release create "$TAG" \
    --title "$TITLE" \
    --notes-file "$RELEASE_DIR/RELEASE_NOTES_v0.6.0-test-fix1.md" \
    --target claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG \
    --prerelease \
    "$RELEASE_DIR/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz#macOS Intel (x86_64) Binary" \
    "$RELEASE_DIR/tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz#macOS Apple Silicon (arm64) Binary" \
    "$RELEASE_DIR/checksums_0.6.0-test-fix1.txt#SHA256 Checksums" \
    "$RELEASE_DIR/BUILD_INFO_MACOS.md#Build Information"

echo ""
echo "✅ SUCCESS! Release created"
echo ""
echo "View the release at:"
echo "  https://github.com/StanleyXie/tfpipboy/releases/tag/$TAG"
echo ""
echo "Download links:"
echo "  Intel:   https://github.com/StanleyXie/tfpipboy/releases/download/$TAG/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz"
echo "  ARM64:   https://github.com/StanleyXie/tfpipboy/releases/download/$TAG/tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz"
echo ""
