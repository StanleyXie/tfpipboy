# Release Notes: tfpipboy v0.6.2

**Release Date:** November 2024
**Release Type:** Production Release
**Previous Version:** v0.6.1-pre

## Overview

tfpipboy v0.6.2 is a production release that includes important security hardening, bug fixes, and improved terminal rendering. This release addresses all actionable security findings from comprehensive Gosec audits and fixes a critical issue with terminal output in non-TTY environments.

## What's New

### Security Hardening

**File and Directory Permission Fixes (25 MEDIUM vulnerabilities)**

All file and directory creation operations have been hardened with restrictive permissions:

- **File Permissions**: Changed from `0644` (world-readable) to `0600` (owner-only)
  - Prevents unauthorized users from reading sensitive Terraform files
  - Affects: Debug logs, job metadata, plan outputs, state files, configuration files
  - Modified files: `display.go`, `executor.go`, `workspace.go`, `terraform_output_filter.go`, `message_pipeline.go`, `logger.go`, `config.go`

- **Directory Permissions**: Changed from `0755` (world-executable) to `0750` (group-restricted)
  - Prevents unauthorized users from listing workspace contents
  - Affects: Workspace directories, log output directories
  - Modified files: `workspace.go`, `message_pipeline.go`

**Impact**: Significantly reduces information disclosure risk for sensitive Terraform state and configuration data.

### Bug Fixes

**Fixed Repeated Frame Rendering in Non-TTY Environments**

**Issue**: The parallel execution progress box was being printed repeatedly (hundreds of times) instead of updating in-place when running in non-TTY environments (CI/CD pipelines, redirected output, etc.).

**Root Cause**: No TTY detection - ANSI cursor movement codes were being used inappropriately in non-terminal contexts.

**Fix**:
- Added TTY detection using `golang.org/x/term.IsTerminal()`
- Implemented dual rendering strategy:
  - **TTY mode**: In-place updates with ANSI cursor codes (existing behavior)
  - **Non-TTY mode**: Show only start and end frames (prevents spam)
- Enhanced cursor position tracking for clean updates
- Reduced refresh rate for non-TTY from 200ms to 2s

**Impact**: Clean, readable output in all execution contexts (terminals, CI/CD, logs, files).

**Files Modified**:
- `pkg/orchestrator/liveboard.go` - Added TTY detection and dual rendering
- `go.mod` - Added `golang.org/x/term v0.37.0` dependency

### Documentation

**Comprehensive Security Audit Documentation**

Added detailed security documentation in `SECURITY.md`:

- Complete analysis of all Gosec security scan findings
- Clear explanation of fixed vulnerabilities (file permissions)
- Documentation of accepted risks (subprocess execution, file inclusion)
  - These are **by design** and required for Terraform orchestration
  - Detailed security model and mitigations provided
- Risk assessment and justification for all findings

**Release Documentation**:
- `RELEASE_v0.6.2.md` - This document
- `RELEASE_v0.6.1-pre.md` - Pre-release documentation
- Updated `SECURITY.md` with current scan results

## Security Assessment

### Fixed Vulnerabilities

✅ **25 MEDIUM**: File and directory permissions (all fixed)

### Accepted Risks (By Design)

The following Gosec findings remain but are **intentional design choices** required for tfpipboy's Terraform orchestration functionality:

**22 MEDIUM**: Subprocess execution and file inclusion
- 4 subprocess execution (running Terraform CLI commands)
- 18 file inclusion (reading user Terraform modules)
- Comprehensive mitigations in place
- Operates in user security context (no privilege escalation)
- Fully documented in SECURITY.md

**46 LOW**: Error handling and unsafe calls
- 45+ unhandled errors in cleanup/diagnostic code (standard Go practice)
- 1 unsafe syscall for terminal size detection (platform-specific, required)

All remaining findings have been reviewed and accepted with documented justifications.

## Upgrade Instructions

### From v0.6.1-pre or Earlier

No configuration changes required. Simply replace the binary:

```bash
# Download the new version
VERSION=v0.6.2
OS=Darwin  # or Linux
ARCH=arm64  # or x86_64

wget https://github.com/StanleyXie/tfpipboy/releases/download/${VERSION}/tfpipboy_${VERSION}_${OS}_${ARCH}.tar.gz

# Verify checksum
wget https://github.com/StanleyXie/tfpipboy/releases/download/${VERSION}/checksums.txt
shasum -a 256 -c checksums.txt --ignore-missing

# Extract and install
tar -xzf tfpipboy_${VERSION}_${OS}_${ARCH}.tar.gz
sudo mv tfpipboy /usr/local/bin/

# Verify installation
tfpipboy --version
```

### For Homebrew Users

```bash
# Update tap (when available)
brew update
brew upgrade tfpipboy
```

## Breaking Changes

None. This release is fully backward compatible with v0.6.x configurations.

## Known Issues

None identified at release time.

## Testing

### Tested Platforms

- ✅ macOS 14+ (Intel and Apple Silicon)
- ✅ Linux (x86_64, arm64)
- ✅ Ubuntu 22.04, 24.04
- ✅ Debian 11, 12

### Test Coverage

- ✅ All unit tests passing
- ✅ Race condition detection (no races found)
- ✅ Integration tests with Terraform 1.5+, 1.6+, 1.7+
- ✅ CI/CD pipeline testing (non-TTY rendering)
- ✅ Security scans (Gosec, Trivy)

## Dependencies

### New Dependencies

- `golang.org/x/term v0.37.0` - TTY detection for improved terminal rendering

### Updated Dependencies

None in this release.

## Migration Guide

No migration needed. All existing configurations, workspaces, and workflows continue to work without modification.

## Performance Impact

- Minimal performance impact from TTY detection (one-time check at startup)
- Improved performance in non-TTY environments (slower refresh rate reduces overhead)

## Security Considerations

This release significantly improves security posture:

1. **File Permissions**: All workspace files now use restrictive permissions (0600/0750)
2. **Information Disclosure**: Reduced risk of unauthorized access to Terraform state/config
3. **Documentation**: Full transparency on security model and accepted risks

### Recommended Actions After Upgrade

1. **Review existing workspaces**: Old workspace files retain their original permissions. Consider:
   ```bash
   # Optional: Restrict permissions on existing workspaces
   chmod 750 ~/.tfpipboy/workspaces/*
   chmod 600 ~/.tfpipboy/workspaces/*/*.tf*
   ```

2. **Review SECURITY.md**: Understand the security model and accepted risks

3. **Update CI/CD**: Verify clean output in your CI/CD pipelines (non-TTY fix)

## Credits

Thanks to all contributors and security researchers who helped identify and validate these improvements.

## Support

- **Issues**: https://github.com/StanleyXie/tfpipboy/issues
- **Security**: See SECURITY.md for vulnerability reporting
- **Discussions**: https://github.com/StanleyXie/tfpipboy/discussions

## Full Changelog

### Security
- **[3eba01f]** security: fix file and directory permissions per Gosec scan
- **[828e156]** docs: document security audit findings and fixes for v0.6.1-pre
- **[6c2ca26]** docs: update SECURITY.md with current Gosec scan results

### Bug Fixes
- **[bc32185]** fix: prevent repeated frame rendering in non-TTY environments

### Documentation
- **[ff14ceb]** docs: add v0.6.1-pre release documentation and push script
- This release: v0.6.2 production release documentation

### Tooling
- **[cc9ef27]** feat: add automated merge and release script
- **[d1bec28]** release: bump version to 0.6.1-pre
- This release: bump version to 0.6.2

## Verification

To verify the integrity of this release:

```bash
# Download release and checksums
wget https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.2/tfpipboy_v0.6.2_Darwin_arm64.tar.gz
wget https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.2/checksums.txt

# Verify SHA256 checksum
shasum -a 256 -c checksums.txt --ignore-missing

# View Software Bill of Materials (SBOM)
wget https://github.com/StanleyXie/tfpipboy/releases/download/v0.6.2/sbom.spdx.json
cat sbom.spdx.json | jq
```

## Next Steps

After installation:

1. Test with your existing Terraform configurations
2. Verify output in both terminal and CI/CD environments
3. Review workspace file permissions
4. Report any issues on GitHub

---

**Full Release**: https://github.com/StanleyXie/tfpipboy/releases/tag/v0.6.2
**SBOM**: Available in release assets as `sbom.spdx.json`
**Checksums**: Available in release assets as `checksums.txt`
