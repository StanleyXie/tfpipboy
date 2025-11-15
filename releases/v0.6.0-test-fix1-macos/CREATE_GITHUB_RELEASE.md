# Instructions to Create GitHub Release

## Quick Access to Files

The release files are now in the repository. To access them:

```bash
# Pull the latest changes
git fetch origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
git checkout claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
git pull

# Navigate to release directory
cd releases/v0.6.0-test-fix1-macos

# List all files
ls -lh
```

## Option 1: Create GitHub Release via Web UI

1. **Go to GitHub Repository:**
   - Navigate to https://github.com/StanleyXie/tfpipboy
   - Click on "Releases" (right sidebar)

2. **Create New Release:**
   - Click "Draft a new release"
   - **Tag:** `v0.6.0-test-fix1-macos`
   - **Target:** `claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG`
   - **Release title:** `v0.6.0-test-fix1 - macOS Test Release`

3. **Add Description:**
   Copy the content from `RELEASE_NOTES_v0.6.0-test-fix1.md`

4. **Upload Assets:**
   Drag and drop these files:
   - `tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz`
   - `tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz`
   - `checksums_0.6.0-test-fix1.txt`
   - `BUILD_INFO_MACOS.md`

5. **Mark as Pre-release:**
   - ✅ Check "This is a pre-release"
   - Click "Publish release"

## Option 2: Create GitHub Release via CLI (if gh is installed)

```bash
# Navigate to repository root
cd /home/user/tfpipboy

# Create release
gh release create v0.6.0-test-fix1-macos \
  --title "v0.6.0-test-fix1 - macOS Test Release" \
  --notes-file releases/v0.6.0-test-fix1-macos/RELEASE_NOTES_v0.6.0-test-fix1.md \
  --target claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG \
  --prerelease \
  releases/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz \
  releases/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz \
  releases/v0.6.0-test-fix1-macos/checksums_0.6.0-test-fix1.txt \
  releases/v0.6.0-test-fix1-macos/BUILD_INFO_MACOS.md
```

## Option 3: Direct Download Links (After Creating Release)

Once the release is created, the download URLs will be:

**Intel (x86_64):**
```
https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
```

**Apple Silicon (arm64):**
```
https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
```

**Checksums:**
```
https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/checksums_0.6.0-test-fix1.txt
```

## Release Notes Template

Use this for the GitHub release description:

---

# macOS Test Release v0.6.0-test-fix1

**Platforms:** macOS Intel (x86_64) + Apple Silicon (arm64)
**Release Type:** Test Release (Pre-release)
**Build Date:** 2025-11-15

## 🐛 Bug Fix

Fixed critical validation error:
- **Issue:** Backend type was incorrectly required for all instances
- **Fix:** Made backend `type` field optional
- **Impact:** Allows partial backend configurations and inheritance patterns

## 📦 Downloads

### macOS Intel (x86_64)
- **File:** `tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz`
- **Size:** 1.5 MB
- **SHA256:** `ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546`
- **Compatible:** Intel Macs (macOS 10.13+)

### macOS Apple Silicon (arm64)
- **File:** `tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz`
- **Size:** 1.4 MB
- **SHA256:** `41e9e9daafbe88791ac5cf4e42cd583d57f8e9df692cee7d0dd142577754121b`
- **Compatible:** M1, M2, M3 Macs (macOS 11.0+)

## 🚀 Installation

### Intel Macs
```bash
curl -LO https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
echo "ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546  tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz" | shasum -a 256 -c
tar xzf tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
chmod +x tfpipboy
sudo mv tfpipboy /usr/local/bin/
tfpipboy --version
```

### Apple Silicon Macs
```bash
curl -LO https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
echo "41e9e9daafbe88791ac5cf4e42cd583d57f8e9df692cee7d0dd142577754121b  tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz" | shasum -a 256 -c
tar xzf tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
chmod +x tfpipboy
sudo mv tfpipboy /usr/local/bin/
tfpipboy --version
```

## ⚠️ macOS Security Warning

**First Run:** This binary is not code-signed and will trigger Gatekeeper.

**Bypass Methods:**
1. Right-click → "Open" → Confirm
2. Command line: `xattr -d com.apple.quarantine tfpipboy`
3. System Preferences → Security & Privacy → "Open Anyway"

## ✨ Features

**Phase 1 Security Fixes:**
- History file security (0600 permissions, 10K line limit)
- Command timeout (30 minutes default)
- Input validation (path validation, symlink resolution)
- Centralized version management
- Security documentation

**Bug Fix (Fix1):**
- Backend type field made optional
- Partial backend configurations supported
- Fixes: "backend type is required" error

## 🧪 Testing

- ✅ 14 unit tests - 100% pass rate
- ✅ Race detector - PASSED
- ✅ Coverage - 33.3% overall, 80%+ security

## ⚠️ Limitations

This is a **TEST RELEASE**:
- ❌ NOT code-signed (Gatekeeper warnings expected)
- ❌ NOT notarized by Apple
- ❌ Cross-compiled on Linux (not native macOS build)

**For production:** Code signing and notarization will be added.

## 📚 Documentation

- [BUILD_INFO_MACOS.md](BUILD_INFO_MACOS.md) - Complete build information
- [BUG_FIX_NOTES.md](BUG_FIX_NOTES.md) - Bug fix details
- [RELEASE_READY.txt](RELEASE_READY.txt) - Distribution guide

## 📋 Checksums (SHA256)

```
ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546  tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
41e9e9daafbe88791ac5cf4e42cd583d57f8e9df692cee7d0dd142577754121b  tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
```

---

**Status:** ✅ Ready for Testing
**Branch:** `claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG`
**Commit:** 1c98bc3

---

## Verification

After creating the release, verify it:

```bash
# Check release exists
gh release view v0.6.0-test-fix1-macos

# List assets
gh release view v0.6.0-test-fix1-macos --json assets

# Download and test (Intel example)
curl -LO https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
tar xzf tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
./tfpipboy --version
```

## Files to Upload

From `releases/v0.6.0-test-fix1-macos/`:
1. ✅ tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz (1.5 MB)
2. ✅ tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz (1.4 MB)
3. ✅ checksums_0.6.0-test-fix1.txt (223 bytes)
4. ✅ BUILD_INFO_MACOS.md (8.7 KB)

**All files are ready in the git repository!**
