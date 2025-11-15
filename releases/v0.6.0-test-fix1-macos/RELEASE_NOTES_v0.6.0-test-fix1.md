# Release Notes - tfpipboy v0.6.0-test-fix1 (macOS)

**Release Date:** 2025-11-15
**Release Type:** Test Release
**Platforms:** macOS Intel (x86_64), macOS Apple Silicon (arm64)
**Branch:** claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
**Commit:** 1c98bc3

---

## 📦 Download

### macOS Intel (x86_64)
```bash
# Download
curl -LO https://[release-url]/tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz

# Verify checksum
echo "ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546  tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz" | shasum -a 256 -c

# Extract and install
tar xzf tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
chmod +x tfpipboy
sudo mv tfpipboy /usr/local/bin/
```

**Binary Size:** 3.3 MB
**Archive Size:** 1.5 MB
**Compatible with:** Intel-based Macs (macOS 10.13+)

### macOS Apple Silicon (arm64)
```bash
# Download
curl -LO https://[release-url]/tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz

# Verify checksum
echo "41e9e9daafbe88791ac5cf4e42cd583d57f8e9df692cee7d0dd142577754121b  tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz" | shasum -a 256 -c

# Extract and install
tar xzf tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
chmod +x tfpipboy
sudo mv tfpipboy /usr/local/bin/
```

**Binary Size:** 3.2 MB
**Archive Size:** 1.4 MB
**Compatible with:** M1, M2, M3 Macs (macOS 11.0+)

---

## 🔐 Checksums (SHA256)

### Archives
```
41e9e9daafbe88791ac5cf4e42cd583d57f8e9df692cee7d0dd142577754121b  tfpipboy_0.6.0-test-fix1_Darwin_arm64.tar.gz
ce3ef473db57c89f2e052d12475e74b50f0e67fe1a2f82db9a2ff9c07ef16546  tfpipboy_0.6.0-test-fix1_Darwin_x86_64.tar.gz
```

### Binaries
```
4c6080674b2fc968c0f484ea04d24add954d2ab8d56d24ccefae6814a670cb59  tfpipboy (arm64)
f7affdb495359de173f130f0b1d8326581b38df8f09e929ede97189eea2ef468  tfpipboy (x86_64)
```

---

## 🐛 Bug Fix (Fix1)

This release fixes a critical validation error reported by users:

### Problem
```
ERROR: Failed to load configuration from tfpipboy.yaml:
  - Configuration validation failed:
    - instances[instance-name] backend: backend type is required
```

### Solution
Made the backend `type` field **optional** in configuration validation. This allows:
- ✅ Partial backend configurations (without explicit type)
- ✅ Backend inheritance patterns
- ✅ Backwards compatibility with existing configurations

**File Modified:** pkg/orchestrator/config.go:479-482
**Commit:** 1c98bc3

---

## ✨ Features Included

### Phase 1 Security Fixes

1. **History File Security (HS-003)**
   - Permissions: 0600 (user read/write only)
   - Size limit: 10,000 lines maximum
   - Automatic trimming implemented

2. **Version Management (CQ-001)**
   - Centralized version package
   - Consistent versioning across application

3. **Command Timeout (MS-004)**
   - 30-minute default timeout
   - Context-based cancellation
   - Timeout error detection

4. **Input Validation (MS-002)**
   - Path validation for cd command
   - Symlink resolution
   - Canonical path enforcement

5. **Security Documentation**
   - Security model documented
   - Trust boundaries defined
   - Usage guidelines provided

6. **Configuration Validation (Fix1)**
   - Backend type field optional
   - Partial configuration support
   - Validation error fixed

---

## 🧪 Testing

### Build Tests
- ✅ Cross-compilation successful (darwin/amd64)
- ✅ Cross-compilation successful (darwin/arm64)
- ✅ Binary format verified (Mach-O)
- ✅ Archives created and verified
- ✅ Checksums generated and validated

### Unit Tests (Linux)
- ✅ 14 test cases - 100% pass rate
- ✅ Race detector - PASSED
- ✅ Coverage - 33.3% overall, 80%+ security functions

---

## ⚠️ macOS Gatekeeper

**First Run:** You will see a security warning because this binary is not code-signed.

### Bypass Methods

**Method 1: Right-click**
1. Right-click on `tfpipboy`
2. Select "Open"
3. Click "Open" in dialog

**Method 2: Command line**
```bash
xattr -d com.apple.quarantine tfpipboy
./tfpipboy --version
```

**Method 3: System Preferences**
1. Go to System Preferences > Security & Privacy
2. Click "Open Anyway"

---

## ⚠️ Test Release Limitations

This is a **TEST RELEASE** with limitations:

1. ❌ **Not Code-Signed** - Triggers Gatekeeper warnings
2. ❌ **Not Notarized** - Not verified by Apple
3. ❌ **Cross-Compiled** - Built on Linux, not macOS
4. ❌ **No SBOM** - Software Bill of Materials not included

**For Production:** Code signing, notarization, and native builds will be added.

---

## 📋 What's Included

Each archive contains:
- `tfpipboy` - Binary executable
- `README.md` - Project documentation
- `LICENSE` - MIT License
- `CHANGELOG.md` - Version history
- `checksums.txt` - SHA256 checksums

---

## 🔍 Verification

### After Download
```bash
# Verify checksum (shown above)
shasum -a 256 -c <<< "checksum  filename"

# Extract
tar xzf tfpipboy_0.6.0-test-fix1_Darwin_*.tar.gz

# Test
./tfpipboy --version
# Output: tfpipboy version 0.6.0

./tfpipboy --help
# Should display help text
```

---

## 🎯 Platform Notes

### Intel Macs (x86_64)
- All Intel-based Macs supported
- Minimum: macOS 10.13 (High Sierra)
- Also runs on Apple Silicon via Rosetta 2

### Apple Silicon (arm64)
- Native M1, M2, M3 support
- Minimum: macOS 11.0 (Big Sur)
- Better performance than Rosetta
- Slightly smaller binary (3.2MB vs 3.3MB)

---

## 🔗 Related Documentation

- **BUILD_INFO_MACOS.md** - Complete build information
- **BUG_FIX_NOTES.md** - Detailed bug fix documentation
- **REVIEW_PLAN_CLI_WRAPPER.md** - Security review plan
- **IMPLEMENTATION_SUMMARY.md** - Phase 1 implementation
- **TEST_REPORT.md** - Comprehensive test results

---

## 📞 Support

**Check Your Architecture:**
```bash
uname -m
# x86_64 = Intel
# arm64 = Apple Silicon
```

**Check macOS Version:**
```bash
sw_vers
```

**Issues:** Report at https://github.com/StanleyXie/tfpipboy/issues

---

## ✅ Release Checklist

- [x] Binaries compiled for both architectures
- [x] Archives created and compressed
- [x] Checksums generated and verified
- [x] Documentation included in archives
- [x] Build information documented
- [x] Bug fix notes documented
- [x] Installation instructions provided
- [x] Gatekeeper workarounds documented

**Status:** ✅ READY FOR TESTING

---

## 🎉 Summary

**Version:** 0.6.0-test-fix1
**Platforms:** macOS Intel + Apple Silicon
**Build Type:** Cross-compiled test release
**Bug Fix:** Backend type validation made optional
**Security:** Phase 1 fixes included
**Status:** Ready for user testing

**Download Size:** ~1.4-1.5 MB per platform
**Binary Size:** ~3.2-3.3 MB per platform

---

**Release Generated:** 2025-11-15
**Branch:** claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
**Tag:** v0.6.0-test-fix1-macos
**Verification:** ✅ PASSED
