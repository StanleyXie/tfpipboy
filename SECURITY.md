# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.6.x   | :white_check_mark: |
| < 0.6   | :x:                |

## Security Measures

### Automated Security Scanning

Every release undergoes comprehensive security validation:

1. **Dependency Vulnerability Scanning** (Trivy)
   - Scans all Go dependencies for known vulnerabilities
   - Fails build on CRITICAL or HIGH severity issues
   - Results uploaded to GitHub Security tab

2. **Static Code Analysis** (Gosec)
   - Detects common security issues in Go code
   - Checks for hardcoded credentials, unsafe practices
   - SARIF reports available in Security tab

3. **Race Condition Detection**
   - All tests run with `-race` flag
   - Ensures thread-safe concurrent execution
   - Critical for tfpipboy's parallel job execution

4. **Software Bill of Materials (SBOM)**
   - SPDX-format SBOM generated for every release
   - Lists all dependencies and their versions
   - Available in release assets as `sbom.spdx.json`

5. **Binary Verification**
   - SHA256 checksums for all release artifacts
   - Available in `checksums.txt` in release assets
   - Post-release binary scanning with Trivy

### Manual Security Review

Before each release, we perform:

- Code review of security-sensitive changes
- Verification of authentication handling
- Review of Terraform state file access patterns
- Validation of environment variable handling

## Recent Security Audit (v0.6.1-pre)

A comprehensive Gosec security audit was conducted in November 2024. The following issues were identified and addressed:

### Fixed Issues

**File and Directory Permissions (25 MEDIUM)**

All file and directory creation operations were hardened to use restrictive permissions:

- **File Permissions**: Changed from `0644` (rw-r--r--) to `0600` (rw-------)
  - Prevents unauthorized users from reading sensitive Terraform files
  - Affects: Debug logs, job metadata, plan outputs, state files, configuration files
  - Modified files: `display.go`, `executor.go`, `workspace.go`, `terraform_output_filter.go`, `message_pipeline.go`, `logger.go`, `config.go`

- **Directory Permissions**: Changed from `0755` (rwxr-xr-x) to `0750` (rwxr-x---)
  - Prevents unauthorized users from listing workspace contents
  - Affects: Workspace directories, log output directories
  - Modified files: `workspace.go`, `message_pipeline.go`

**Commit**: 3eba01f - "security: restrict file and directory permissions"

### Accepted Risk Items

The following Gosec findings are **by design** and represent intended functionality:

**Subprocess Execution with Variables (4 MEDIUM)**

tfpipboy's core purpose is to orchestrate Terraform CLI executions. Subprocess execution is essential:

- **Purpose**: Execute `terraform init`, `terraform plan`, `terraform apply` commands
- **Security Model**:
  - Commands are constructed from validated configuration
  - User provides modules and variables explicitly in config
  - Runs in user context with user's Terraform authentication
  - No privilege escalation occurs
- **Mitigations**:
  - Commands use Go's `exec.Command()` which properly handles argument separation
  - Working directories are isolated per module
  - Environment variables are inherited, not modified
  - All execution is synchronous and monitored

**File Inclusion via Variables (18 MEDIUM)**

tfpipboy must read user-provided Terraform modules and configuration files:

- **Purpose**: Load and validate Terraform modules, variables, and backend configurations
- **Security Model**:
  - User explicitly defines module paths in configuration
  - All file access is within user-specified workspace
  - No arbitrary file system traversal
  - Read-only operations (never modifies source modules)
- **Mitigations**:
  - Workspace isolation prevents cross-contamination
  - File paths are validated and normalized
  - symlink resolution prevents directory traversal
  - All operations are logged

**Risk Assessment**: These operations are **intentional and required** for tfpipboy's orchestration functionality. The tool operates in the user's security context and cannot perform actions the user couldn't perform directly with Terraform CLI.

## Security Best Practices

tfpipboy is designed with security in mind:

### Read-Only Operations
- **Never modifies Terraform state** - Only reads context information
- **No credential storage** - Uses existing CLI tool authentication
- **No configuration changes** - Never alters Terraform configs

### Authentication
- Relies on official CLI tools (aws, az, gcloud, gh)
- Does not cache or store credentials
- Authentication checks timeout after 2 seconds

### Data Privacy
- All context data stays local
- No telemetry or data collection
- Cache files stored with restricted permissions (0700)

### Subprocess Execution
- Uses Go's `os/exec` for secure subprocess handling
- Properly sanitizes arguments passed to Terraform
- Inherits environment variables safely

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please follow these steps:

### Where to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security issues via:

1. **GitHub Security Advisories** (preferred)
   - Go to: https://github.com/StanleyXie/tfpipboy/security/advisories/new
   - Click "Report a vulnerability"

2. **Email** (alternative)
   - Send to: [stnaley.xie@outlook.com]
   - Subject: "[SECURITY] tfpipboy vulnerability report"

### What to Include

Please provide as much information as possible:

- **Vulnerability Description**: What is the security issue?
- **Impact**: What could an attacker accomplish?
- **Affected Versions**: Which versions are affected?
- **Steps to Reproduce**: Detailed steps to reproduce the issue
- **Proof of Concept**: Code or commands demonstrating the issue
- **Suggested Fix**: If you have ideas for fixing the issue

### What to Expect

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 5 business days
- **Regular Updates**: At least every 7 days
- **Disclosure Timeline**: 90 days from report (or earlier if fixed)

### Our Process

1. **Triage**: We'll confirm the vulnerability and assess severity
2. **Fix Development**: We'll develop and test a fix
3. **Private Testing**: We may ask you to verify the fix
4. **Release**: We'll release a patched version
5. **Public Disclosure**: We'll publish a security advisory
6. **Credit**: We'll credit you (unless you prefer to remain anonymous)

## Security Advisories

Published security advisories are available at:
https://github.com/StanleyXie/tfpipboy/security/advisories

Subscribe to releases to be notified of security updates.

## Dependency Security

### Keeping Dependencies Updated

We use Dependabot to:
- Monitor dependencies for known vulnerabilities
- Automatically create PRs for security updates
- Keep dependencies up-to-date

### Verifying Releases

To verify the integrity of a release:

```bash
# Download release and checksums
VERSION=v0.6.0
wget https://github.com/StanleyXie/tfpipboy/releases/download/${VERSION}/tfpipboy_Darwin_x86_64.tar.gz
wget https://github.com/StanleyXie/tfpipboy/releases/download/${VERSION}/checksums.txt

# Verify checksum
shasum -a 256 -c checksums.txt --ignore-missing

# View SBOM
wget https://github.com/StanleyXie/tfpipboy/releases/download/${VERSION}/sbom.spdx.json
cat sbom.spdx.json
```

## Security Testing

### Running Security Scans Locally

```bash
# Install security tools
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run Gosec
gosec ./...

# Run tests with race detector
go test -race ./...

# Check for known vulnerabilities in dependencies
go list -json -m all | docker run --rm -i sonatypecommunity/nancy:latest sleuth
```

### CI/CD Security Checks

Every PR and commit to main runs:
- Gosec static analysis
- Race condition detection
- Dependency vulnerability scanning
- SARIF report upload to GitHub Security

## License

This security policy is licensed under CC-BY-4.0.
