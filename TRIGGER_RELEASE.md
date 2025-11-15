# How to Trigger the macOS Test Release

## ✅ Everything is Ready!

All files have been committed and pushed to the repository:

**Latest Commits:**
- `4c6d6d5` - Release trigger script
- `4789acc` - GitHub Actions workflow for macOS releases
- `5573b17` - macOS test release artifacts (binaries + docs)
- `1c98bc3` - Bug fix (backend type optional)

**Branch:** `claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG`

---

## 🚀 Trigger the Release (Choose One Method)

### Method 1: Run the Script (Recommended)

```bash
# Pull latest changes
git pull origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG

# Run the trigger script
./trigger-macos-release.sh
```

This script will:
1. Verify you're on the correct branch
2. Create the tag `v0.6.0-test-fix1-macos` if needed
3. Push the tag to GitHub
4. Trigger the GitHub Actions workflow
5. Show you the URLs to check progress

### Method 2: Manual Tag Push

```bash
# Ensure you're on the right branch
git checkout claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
git pull

# The tag should already exist, just push it
git push origin v0.6.0-test-fix1-macos
```

### Method 3: Create Tag Manually (if it doesn't exist)

```bash
# Create the tag at the bug fix commit
git tag -a v0.6.0-test-fix1-macos 1c98bc3 -m "macOS Test Release v0.6.0-test-fix1"

# Push it
git push origin v0.6.0-test-fix1-macos
```

---

## 📋 What Happens Next

When you push the tag, GitHub Actions will:

1. **Detect the tag** `v0.6.0-test-fix1-macos`
2. **Run the workflow** `.github/workflows/release-macos.yml`
3. **Locate release files** in `releases/v0.6.0-test-fix1-macos/`
4. **Create GitHub Release** with title "v0.6.0-test-fix1 - macOS Test Release"
5. **Upload release assets:**
   - `tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz` (Intel)
   - `tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz` (Apple Silicon)
   - `checksums_0.6.0-test-fix1.txt`
   - `BUILD_INFO_MACOS.md`
6. **Mark as pre-release** (test release flag)

---

## 🔍 Monitor Progress

### Check Workflow Status
```bash
# View in terminal (if gh CLI installed)
gh run list --workflow=release-macos.yml

# Or visit in browser:
https://github.com/StanleyXie/tfpipboy/actions/workflows/release-macos.yml
```

### View the Release (After Workflow Completes)
```bash
# View in terminal
gh release view v0.6.0-test-fix1-macos

# Or visit in browser:
https://github.com/StanleyXie/tfpipboy/releases/tag/v0.6.0-test-fix1-macos
```

---

## 📦 Release Contents

The workflow will create a GitHub release with:

**Binaries:**
- ✅ Intel (x86_64): 3.3 MB - SHA256: `ce3ef473...`
- ✅ Apple Silicon (arm64): 3.2 MB - SHA256: `41e9e9da...`

**Documentation:**
- ✅ Release notes with installation instructions
- ✅ Build information
- ✅ Checksums for verification

**Download URLs (after release):**
- Intel: `https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz`
- Apple Silicon: `https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz`

---

## ⚠️ If Tag Push Fails

If you get a 403 or permission error:

1. **Check your GitHub token:**
   ```bash
   gh auth status
   ```

2. **Re-authenticate if needed:**
   ```bash
   gh auth login
   ```

3. **Verify repository permissions:**
   - You need push access to the repository
   - Tokens need `repo` scope for pushing tags

4. **Alternative: Create release via Web UI**
   - Go to https://github.com/StanleyXie/tfpipboy/releases/new
   - Use tag: `v0.6.0-test-fix1-macos`
   - Upload files from `releases/v0.6.0-test-fix1-macos/`
   - Mark as pre-release

---

## ✅ Success Checklist

After the workflow completes, verify:

- [ ] Release appears at: https://github.com/StanleyXie/tfpipboy/releases
- [ ] Release is marked as "Pre-release"
- [ ] Both Darwin tar.gz files are attached
- [ ] Checksums file is attached
- [ ] BUILD_INFO_MACOS.md is attached
- [ ] Release notes are displayed
- [ ] Download links work

---

## 🎯 Quick Start for Users (After Release)

Once the release is published, users can install with:

**Intel Macs:**
```bash
curl -LO https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
tar xzf tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
sudo mv tfpipboy /usr/local/bin/
tfpipboy --version
```

**Apple Silicon Macs:**
```bash
curl -LO https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
tar xzf tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
sudo mv tfpipboy /usr/local/bin/
tfpipboy --version
```

---

## 📞 Need Help?

If you encounter issues:

1. Check workflow logs: https://github.com/StanleyXie/tfpipboy/actions
2. Verify tag exists: `git tag -l | grep v0.6.0`
3. Check release files: `ls -lh releases/v0.6.0-test-fix1-macos/`
4. Review workflow file: `.github/workflows/release-macos.yml`

---

**Current Status:** ✅ Ready to trigger
**Action Required:** Run `./trigger-macos-release.sh` or push the tag manually
