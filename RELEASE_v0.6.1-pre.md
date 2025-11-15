# Release v0.6.1-pre - Terminal Rendering Fix

**Release Date:** 2025-11-15
**Type:** Pre-release
**Branch:** claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
**Tag:** v0.6.1-pre
**Commit:** d1bec28

---

## 🐛 Critical Bug Fix

### Issue: Repeated Frame Rendering

**Problem:**
The parallel execution progress box was being printed repeatedly instead of updating in-place, causing terminal spam:

```
╭──────────────────────────────────────────────────────────────────────────────╮
│ PARALLEL EXECUTION IN PROGRESS                               Elapsed:     0s │
╰──────────────────────────────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────╮
│ PARALLEL EXECUTION IN PROGRESS                               Elapsed:     0s │
╰──────────────────────────────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────╮
│ PARALLEL EXECUTION IN PROGRESS                               Elapsed:     0s │
╰──────────────────────────────────────────────────────────────────────────────╯
(repeated many times...)
```

**Root Cause:**
- No TTY (terminal) detection
- ANSI cursor movement codes used inappropriately in non-terminal contexts
- Same refresh logic for interactive terminals and redirected output

---

## ✅ Solution Implemented

### TTY Detection and Adaptive Rendering

**Changes in `pkg/orchestrator/liveboard.go`:**

1. **Added TTY Detection**
   ```go
   isTTY := term.IsTerminal(int(os.Stdout.Fd()))
   ```

2. **Dual Rendering Strategy**

   **TTY Mode (Interactive Terminal):**
   - Uses ANSI cursor movement for in-place updates
   - Refresh rate: 200ms
   - Moves cursor up using `\033[%dA`
   - Clears and rewrites content
   - Smooth progress animation

   **Non-TTY Mode (Redirected/Piped):**
   - Only renders at START and COMPLETION
   - Refresh rate: 2 seconds
   - No ANSI cursor codes
   - Clean, non-repetitive output
   - Perfect for logs and files

3. **Line Tracking**
   ```go
   lastRenderLines int  // Tracks lines for proper cursor positioning
   ```

---

## 📊 Technical Details

### Files Modified

**pkg/orchestrator/liveboard.go**
- Added `isTTY` field for terminal detection
- Added `lastRenderLines` for cursor management
- Modified `NewLiveBoard()` to detect terminal capability
- Modified `render()` function with dual rendering logic
- Optimized refresh rates based on output type

**pkg/version/version.go**
- Updated version from "0.6.0" to "0.6.1-pre"

**go.mod**
- Added `golang.org/x/term v0.37.0` dependency

**go.sum**
- Updated with golang.org/x/term and golang.org/x/sys checksums

---

## 🔍 Behavior Changes

### Before Fix

**Interactive Terminal:**
- ❌ Box printed repeatedly (hundreds of times)
- ❌ Scroll spam
- ❌ Unreadable output

**Redirected Output:**
- ❌ Log files filled with repeated frames
- ❌ Thousands of duplicate lines
- ❌ Unusable for automation

### After Fix

**Interactive Terminal:**
- ✅ Progress box updates smoothly in-place
- ✅ No scrolling
- ✅ Clean, animated display
- ✅ Spinner animations work correctly

**Redirected Output:**
- ✅ Shows initial state (once)
- ✅ Shows final state (once)
- ✅ No intermediate spam
- ✅ Clean, parseable logs

---

## 🧪 Testing

### Test Scenarios

1. **Interactive Terminal:**
   ```bash
   ./tfpipboy --config . --targets-all --operation plan
   ```
   Expected: Progress box updates in-place, no scrolling

2. **Redirected to File:**
   ```bash
   ./tfpipboy --config . --targets-all --operation plan > output.log
   ```
   Expected: Only start and end frames in log file

3. **Piped to Another Command:**
   ```bash
   ./tfpipboy --config . --targets-all --operation plan | tee output.log
   ```
   Expected: Clean output, no spam

4. **Background Job:**
   ```bash
   ./tfpipboy --config . --targets-all --operation plan &
   ```
   Expected: No TTY interference

---

## 📦 Release Artifacts

### Git Information

**Commits:**
- `d1bec28` - release: bump version to 0.6.1-pre
- `bc32185` - fix: prevent repeated frame rendering in non-TTY environments

**Tag:** v0.6.1-pre (created locally, needs push)

**Branch:** claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG ✅ Pushed

---

## 🚀 Deployment Instructions

### For Repository Owner (You)

The code changes are pushed to the repository, but the tag needs to be pushed from your local machine:

**Option 1: Quick Script**
```bash
# Pull latest changes
git checkout claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
git pull origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG

# Run the push script
./push-v0.6.1-pre-tag.sh
```

**Option 2: Manual Push**
```bash
# Pull latest changes
git checkout claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
git pull origin claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG

# Push the tag
git push origin v0.6.1-pre
```

**Option 3: Create GitHub Release**
Go to: https://github.com/StanleyXie/tfpipboy/releases/new
- Tag: `v0.6.1-pre`
- Target: `claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG`
- Mark as: ☑️ Pre-release

---

## 📝 Changelog

### v0.6.1-pre (2025-11-15)

**Bug Fixes:**
- Fixed repeated frame rendering in non-TTY environments
- Progress box now updates in-place in interactive terminals
- Clean output when piped or redirected (no spam)
- Proper handling of background jobs and automation

**Technical Changes:**
- Added TTY detection using golang.org/x/term
- Optimized refresh rates: 200ms (TTY) vs 2s (non-TTY)
- Proper ANSI cursor management for in-place updates
- Skip intermediate renders in non-TTY mode
- Added line counting for accurate cursor positioning

**Dependencies:**
- Added golang.org/x/term v0.37.0

**Files Changed:**
- pkg/orchestrator/liveboard.go (rendering logic)
- pkg/version/version.go (version bump)
- go.mod (new dependency)
- go.sum (dependency checksums)

---

## ⚠️ Breaking Changes

**None** - This is a bug fix release with no API changes.

---

## 🔜 Next Steps

1. **Test the Pre-release**
   - Build and test on your local machine
   - Verify TTY detection works correctly
   - Test in various output scenarios

2. **Create Release Binaries** (if needed)
   - Linux x86_64
   - macOS Intel (x86_64)
   - macOS Apple Silicon (arm64)

3. **Promote to Stable**
   - If testing passes, create v0.6.1 stable release
   - Remove "-pre" suffix
   - Update documentation

---

## 📞 Support

**Repository:** https://github.com/StanleyXie/tfpipboy
**Branch:** claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
**Issues:** https://github.com/StanleyXie/tfpipboy/issues

---

## ✨ Summary

This pre-release fixes a critical rendering bug that made the parallel execution display unusable in many scenarios. The fix uses proper TTY detection to provide:

- **Interactive terminals:** Smooth, animated, in-place updates
- **Logs and files:** Clean, non-repetitive output
- **Automation:** Predictable, parseable results

**Status:** ✅ Ready for Testing
**Recommendation:** Test before promoting to stable release

---

**Generated:** 2025-11-15
**Type:** Pre-release Documentation
**Version:** 0.6.1-pre
