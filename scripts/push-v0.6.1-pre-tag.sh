#!/bin/bash

# Script to push v0.6.1-pre release tag
# Run this from your local machine to complete the release

set -e

echo "=== Pushing v0.6.1-pre Release Tag ==="
echo ""

TAG="v0.6.1-pre"
BRANCH="claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG"

echo "Release Configuration:"
echo "  Tag: $TAG"
echo "  Branch: $BRANCH"
echo "  Type: Pre-release"
echo ""

# Check if we're on the right branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "$BRANCH" ]; then
    echo "Switching to branch: $BRANCH"
    git checkout "$BRANCH"
fi

# Pull latest changes
echo "Pulling latest changes..."
git pull origin "$BRANCH"

# Check if tag exists locally
if ! git rev-parse "$TAG" >/dev/null 2>&1; then
    echo "ERROR: Tag $TAG does not exist locally"
    echo "It should have been created. Please check your repository."
    exit 1
fi

echo "Tag $TAG found locally"
echo ""

# Show tag details
echo "Tag details:"
git show "$TAG" --no-patch --format="%h %s" | head -5
echo ""

# Push the tag
echo "Pushing tag to origin..."
if git push origin "$TAG"; then
    echo ""
    echo "✅ SUCCESS: Tag pushed successfully!"
    echo ""
    echo "The v0.6.1-pre pre-release tag is now available."
    echo ""
    echo "Next steps:"
    echo "1. Build release binaries"
    echo "2. Create GitHub release (or let CI handle it)"
    echo "3. Test the pre-release"
    echo ""
else
    echo ""
    echo "❌ ERROR: Failed to push tag"
    echo ""
    echo "Please check:"
    echo "1. Your GitHub credentials and permissions"
    echo "2. Network connectivity"
    echo "3. Try: git push origin $TAG"
    echo ""
    exit 1
fi

echo "=== Release Tag Push Complete ==="
