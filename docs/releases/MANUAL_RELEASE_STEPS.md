# Manual Steps to Create macOS GitHub Release

I encountered permission restrictions when pushing tags from this environment. Please follow one of these methods to create the release:

---

## ⚡ Method 1: Using GitHub CLI (Recommended - Fastest)

### Step 1: Install GitHub CLI (if not already installed)

**macOS:**
```bash
brew install gh
```

**Linux (Debian/Ubuntu):**
```bash
sudo apt update
sudo apt install gh
```

**Other Linux:**
```bash
# See: https://github.com/cli/cli/blob/trunk/docs/install_linux.md
```

### Step 2: Authenticate

```bash
gh auth login
# Follow the prompts to authenticate with your GitHub account
```

### Step 3: Pull the latest code

```bash
cd /path/to/tfpipboy
git checkout claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
git pull origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
```

### Step 4: Run the release script

```bash
./create-release.sh
```

This script will:
- Create the tag `v0.6.0-test-fix1-macos`
- Upload all release files
- Create the GitHub release
- Mark it as pre-release

**Done!** The release will be available at:
https://github.com/StanleyXie/tfpipboy/releases/tag/v0.6.0-test-fix1-macos

---

## 🌐 Method 2: Using GitHub Web UI (No CLI Required)

### Step 1: Pull the latest code

```bash
cd /path/to/tfpipboy
git checkout claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
git pull origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
```

### Step 2: Go to GitHub Releases Page

Open in your browser:
```
https://github.com/StanleyXie/tfpipboy/releases/new
```

### Step 3: Fill in Release Details

**Tag:**
```
v0.6.0-test-fix1-macos
```

**Target:**
```
claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
```

**Release Title:**
```
v0.6.0-test-fix1 - macOS Test Release
```

**Description:**
Copy the content from: `releases/v0.6.0-test-fix1-macos/RELEASE_NOTES_v0.6.0-test-fix1.md`

Or use this summary:
```markdown
# macOS Test Release v0.6.0-test-fix1

**Platforms:** macOS Intel (x86_64) + Apple Silicon (arm64)
**Build Date:** 2025-11-15

## 🐛 Bug Fix
- Fixed backend type validation - made `type` field optional
- Allows partial backend configurations

## 📦 Downloads

**Intel (x86_64):** 1.5 MB
SHA256: `ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546`

**Apple Silicon (arm64):** 1.4 MB
SHA256: `41e9e9daafbe88791ac5cf4e42cd583d57f8e9df692cee7d0dd142577754121b`

## 🚀 Installation

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

## ⚠️ Security Warning
Not code-signed. First run: Right-click → Open or `xattr -d com.apple.quarantine tfpipboy`

## ✨ Features
- Phase 1 security fixes
- Backend type validation fix
- Comprehensive testing (100% pass rate)
```

### Step 4: Upload Release Files

Click "Attach binaries" and upload these files from `releases/v0.6.0-test-fix1-macos/`:

1. ✅ `tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz`
2. ✅ `tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz`
3. ✅ `checksums_0.6.0-test-fix1.txt`
4. ✅ `BUILD_INFO_MACOS.md`

### Step 5: Mark as Pre-release

☑️ Check "This is a pre-release"

### Step 6: Publish

Click **"Publish release"**

**Done!** Your release is now live.

---

## 🔧 Method 3: Using Git with Your Credentials

If you have push access to the repository:

```bash
# Pull latest changes
git checkout claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
git pull origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG

# Create and push tag with YOUR credentials
git tag -a v0.6.0-test-fix1-macos 1c98bc3 -m "macOS Test Release v0.6.0-test-fix1"
git push origin v0.6.0-test-fix1-macos
```

This will trigger the GitHub Actions workflow automatically.

Then monitor:
```
https://github.com/StanleyXie/tfpipboy/actions/workflows/release-macos.yml
```

---

## 📋 Quick Reference

**All release files are located in:**
```
releases/v0.6.0-test-fix1-macos/
```

**Files to upload:**
- `tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz` (1.5 MB)
- `tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz` (1.4 MB)
- `checksums_0.6.0-test-fix1.txt` (223 bytes)
- `BUILD_INFO_MACOS.md` (8.7 KB)

**Checksums:**
```
ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546  Darwin_x86_64
41e9e9daafbe88791ac5cf4e42cd583d57f8e9df692cee7d0dd142577754121b  Darwin_arm64
```

---

## ✅ After Release is Created

Verify the release:

```bash
# Using gh CLI
gh release view v0.6.0-test-fix1-macos

# Or visit in browser
https://github.com/StanleyXie/tfpipboy/releases/tag/v0.6.0-test-fix1-macos
```

Test download:

```bash
# Intel
curl -LO https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
echo "ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546  tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz" | shasum -a 256 -c

# Apple Silicon
curl -LO https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.0-test-fix1-macos/tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
echo "41e9e9daafbe88791ac5cf4e42cd583d57f8e9df692cee7d0dd142577754121b  tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz" | shasum -a 256 -c
```

---

## 🆘 Need Help?

**Issue:** Can't push tag
- **Solution:** Use Method 1 (gh CLI) or Method 2 (Web UI) - they don't require tag push permissions

**Issue:** gh CLI not authenticated
- **Solution:** Run `gh auth login` and follow prompts

**Issue:** Release files not found
- **Solution:** Run `git pull origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG`

---

**Recommendation:** Use Method 1 (GitHub CLI) for the fastest and most reliable release creation.
