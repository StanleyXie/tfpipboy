# Create Release from Your Local Machine

The tag cannot be pushed from this environment due to proxy restrictions. Follow these steps on **your local machine** instead:

---

## Step 1: Pull the Repository on Your Local Machine

```bash
# Clone or navigate to your local copy
cd ~/path/to/tfpipboy  # or clone if you don't have it

# Fetch the latest changes
git fetch origin

# Checkout the feature branch
git checkout claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
git pull origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
```

---

## Step 2: Verify Release Files Exist

```bash
ls -lh releases/v0.6.0-test-fix1-macos/
```

You should see:
- ✅ tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz (1.5 MB)
- ✅ tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz (1.4 MB)
- ✅ checksums_0.6.0-test-fix1.txt
- ✅ BUILD_INFO_MACOS.md
- ✅ RELEASE_NOTES_v0.6.0-test-fix1.md

---

## Step 3: Create and Push Tag (Choose Method A or B)

### Method A: Using GitHub CLI (Recommended)

```bash
# Install gh CLI if needed
# macOS: brew install gh
# Linux: see https://github.com/cli/cli#installation

# Authenticate
gh auth login

# Run the automated script
./create-release.sh
```

This will:
- Create the tag
- Push it to GitHub
- Upload all release files
- Publish the release

**Done!** The release will be at:
https://github.com/StanleyXie/tfpipboy/releases/tag/v0.6.0-test-fix1-macos

---

### Method B: Manual Tag Push + GitHub Actions

```bash
# Create the tag (if it doesn't exist locally)
git tag -a v0.6.0-test-fix1-macos 1c98bc3 -m "macOS Test Release v0.6.0-test-fix1"

# Push the tag with YOUR credentials
git push origin v0.6.0-test-fix1-macos
```

This will trigger the GitHub Actions workflow which will:
1. Detect the tag
2. Find release files in releases/v0.6.0-test-fix1-macos/
3. Create the GitHub release automatically
4. Upload all files

**Monitor progress:**
https://github.com/StanleyXie/tfpipboy/actions/workflows/release-macos.yml

---

## Step 4: Verify the Release

Once created, verify:

```bash
# Using gh CLI
gh release view v0.6.0-test-fix1-macos

# Or visit in browser
open https://github.com/StanleyXie/tfpipboy/releases/tag/v0.6.0-test-fix1-macos
```

Test download:

```bash
# Intel Mac
curl -LO https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz

# Verify checksum
echo "ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546  tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz" | shasum -a 256 -c

# Extract and test
tar xzf tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
./tfpipboy --version
```

---

## Alternative: Use GitHub Web UI

If you prefer not to use command line:

1. **Go to:** https://github.com/StanleyXie/tfpipboy/releases/new

2. **Fill in:**
   - Tag: `v0.6.0-test-fix1-macos`
   - Target: `claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG`
   - Title: `v0.6.0-test-fix1 - macOS Test Release`

3. **Description:** Copy from `releases/v0.6.0-test-fix1-macos/RELEASE_NOTES_v0.6.0-test-fix1.md`

4. **Upload files:**
   - tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
   - tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
   - checksums_0.6.0-test-fix1.txt
   - BUILD_INFO_MACOS.md

5. **Check:** ☑️ This is a pre-release

6. **Click:** Publish release

---

## Why This Happened

The server environment uses a local proxy that restricts tag pushes. All release files are in the repository, but the final tag push needs to be done from your local machine with proper GitHub credentials.

---

## Summary

**What's Ready:**
- ✅ All release binaries committed to git
- ✅ Complete documentation
- ✅ GitHub Actions workflow configured
- ✅ Automated release script ready

**What You Need to Do:**
1. Pull the branch on your local machine
2. Run `./create-release.sh` OR push the tag manually
3. Verify the release is published

**Estimated Time:** 2-3 minutes

---

**Questions?** See MANUAL_RELEASE_STEPS.md for detailed instructions.
