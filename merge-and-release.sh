#!/bin/bash
#
# Script to merge security fixes and trigger v0.6.2 production release
# Run this from your local machine to complete the release process
#

set -e  # Exit on error

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Merge and Release v0.6.2${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Step 1: Fetch latest changes
echo -e "${YELLOW}Step 1/4: Fetching latest changes...${NC}"
git fetch origin
echo -e "${GREEN}✓ Fetch complete${NC}"
echo ""

# Step 2: Checkout and update main
echo -e "${YELLOW}Step 2/4: Checking out main branch...${NC}"
git checkout main
git pull origin main
echo -e "${GREEN}✓ Main branch updated${NC}"
echo ""

# Step 3: Merge feature branch
echo -e "${YELLOW}Step 3/4: Merging security fixes...${NC}"
if git merge origin/claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG --no-ff -m "Merge security fixes and documentation for v0.6.2 production release

This merge includes:
- Terminal rendering fixes for non-TTY environments (bc32185)
- File and directory permission hardening (3eba01f)
- Security audit documentation (828e156, 6c2ca26)
- Production release documentation for v0.6.2

Security improvements:
- Fixed all 25 MEDIUM file permission vulnerabilities
- Documented accepted risks for subprocess execution and file inclusion
- Added comprehensive security scan analysis

All remaining MEDIUM findings are intentional design choices required
for Terraform orchestration functionality.

This is a production release (not pre-release)."; then
    echo -e "${GREEN}✓ Merge successful${NC}"
else
    echo -e "${YELLOW}Note: Merge may have already been completed${NC}"
fi
echo ""

# Step 4: Push to main to trigger release
echo -e "${YELLOW}Step 4/4: Pushing to main (this will trigger the release workflow)...${NC}"
git push origin main
echo -e "${GREEN}✓ Push complete${NC}"
echo ""

# Optional: Create and push tag
echo -e "${YELLOW}Optional: Create git tag v0.6.2?${NC}"
echo "This step is optional. Your release workflow may create the tag automatically."
read -p "Create and push tag? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Creating tag v0.6.2...${NC}"

    if git tag -a v0.6.2 -m "Release v0.6.2: Production release with security fixes and terminal rendering improvements

Release Highlights:
- Production release (not pre-release)
- Fixed repeated frame rendering in non-TTY environments
- Hardened file/directory permissions (25 security fixes)
- Comprehensive security audit documentation

Security Improvements:
- File permissions: 0644 → 0600 (prevents unauthorized reads)
- Directory permissions: 0755 → 0750 (restricts directory listing)
- All security findings documented in SECURITY.md
- Accepted risks clearly explained with mitigations

Bug Fixes:
- Terminal rendering now properly detects TTY vs non-TTY environments
- Progress display updates in-place on terminals
- Non-TTY output shows only start/end frames (prevents spam)

Technical Changes:
- Added golang.org/x/term v0.37.0 for TTY detection
- Dual rendering strategy for different output contexts
- Enhanced cursor position tracking for clean updates

See RELEASE_v0.6.2.md for full details."; then
        echo -e "${GREEN}✓ Tag created${NC}"

        echo -e "${YELLOW}Pushing tag to remote...${NC}"
        git push origin v0.6.2
        echo -e "${GREEN}✓ Tag pushed${NC}"
    else
        echo -e "${YELLOW}Note: Tag may already exist${NC}"
    fi
else
    echo -e "${BLUE}Skipping tag creation${NC}"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Release process complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Next steps:"
echo "1. Monitor the GitHub Actions workflow at:"
echo "   https://github.com/StanleyXie/tfpipboy/actions"
echo ""
echo "2. Once the workflow completes, check the release at:"
echo "   https://github.com/StanleyXie/tfpipboy/releases/tag/v0.6.2"
echo ""
echo "3. Verify the release includes:"
echo "   - Linux binaries (x86_64, arm64)"
echo "   - macOS binaries (x86_64, arm64)"
echo "   - SHA256 checksums"
echo "   - SBOM (Software Bill of Materials)"
echo ""
echo "4. This is a PRODUCTION release (not pre-release)"
echo "   Ensure it's marked as the latest release on GitHub"
echo ""
