#!/bin/bash

# Script to trigger macOS test release workflow
# This creates and pushes the release tag to trigger GitHub Actions

set -e

echo "=== Triggering macOS Test Release ==="
echo ""

# Configuration
TAG="v0.6.0-test-fix1-macos"
COMMIT="1c98bc3"
BRANCH="claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG"

echo "Release Configuration:"
echo "  Tag: $TAG"
echo "  Commit: $COMMIT"
echo "  Branch: $BRANCH"
echo ""

# Check if we're on the right branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "$BRANCH" ]; then
    echo "Switching to branch: $BRANCH"
    git checkout "$BRANCH"
    git pull origin "$BRANCH"
fi

# Check if tag already exists
if git rev-parse "$TAG" >/dev/null 2>&1; then
    echo "Tag $TAG already exists locally"

    # Check if it points to the right commit
    TAG_COMMIT=$(git rev-parse "$TAG^{commit}")
    if [ "$TAG_COMMIT" != "$(git rev-parse $COMMIT)" ]; then
        echo "Warning: Tag points to different commit. Deleting and recreating..."
        git tag -d "$TAG"
    else
        echo "Tag points to correct commit: $COMMIT"
    fi
fi

# Create tag if it doesn't exist
if ! git rev-parse "$TAG" >/dev/null 2>&1; then
    echo "Creating annotated tag: $TAG"
    git tag -a "$TAG" "$COMMIT" -m "$(cat <<'EOF'
Release v0.6.0-test-fix1 (macOS Test Release)

macOS test release for Intel (x86_64) and Apple Silicon (arm64)

Bug Fix:
- Fixed backend type validation - made type field optional for instances
- Allows partial backend configurations and inheritance patterns

Features:
- Phase 1 security fixes (history file security, command timeout, input validation)
- Centralized version management
- Comprehensive security documentation

Platforms:
- macOS Intel (x86_64): 3.3 MB binary, 1.5 MB archive
- macOS Apple Silicon (arm64): 3.2 MB binary, 1.4 MB archive

Checksums (SHA256):
- Darwin x86_64: ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546
- Darwin arm64: 41e9e9daafbe88791ac5cf4e42cd583d57f8e9df692cee7d0dd142577754121b

Build:
- Cross-compiled on Linux
- Go 1.24.7
- Static linking (CGO_ENABLED=0)
- Stripped binaries (-s -w)

Limitations:
- NOT code-signed (will trigger macOS Gatekeeper)
- NOT notarized by Apple
- Test release only

Status: Ready for Testing
EOF
)"
    echo "Tag created successfully"
fi

echo ""
echo "Pushing tag to origin..."
echo "This will trigger the macOS release workflow on GitHub Actions"
echo ""

# Push the tag
if git push origin "$TAG"; then
    echo ""
    echo "✅ SUCCESS: Tag pushed successfully!"
    echo ""
    echo "GitHub Actions workflow should now be running."
    echo "Check the progress at:"
    echo "  https://github.com/StanleyXie/tfpipboy/actions"
    echo ""
    echo "Once complete, the release will be available at:"
    echo "  https://github.com/StanleyXie/tfpipboy/releases/tag/$TAG"
    echo ""
else
    echo ""
    echo "❌ ERROR: Failed to push tag"
    echo ""
    echo "If you see permission errors, you may need to:"
    echo "1. Ensure you have push access to the repository"
    echo "2. Check GitHub token permissions"
    echo "3. Try pushing manually: git push origin $TAG"
    echo ""
    exit 1
fi

echo "=== Release Trigger Complete ==="
