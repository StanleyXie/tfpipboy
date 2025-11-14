# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.6.x   | :white_check_mark: |
| < 0.6   | :x:                |

## Reporting a Vulnerability

We take the security of tf-pipboy seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to: [Your security contact email - TO BE ADDED]

You should receive a response within 48 hours. If for some reason you do not, please follow up via email to ensure we received your original message.

### What to Include

Please include the following information in your report:

- Type of vulnerability
- Full paths of source file(s) related to the vulnerability
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

This information will help us triage your report more quickly.

## Security Considerations

### Credentials and Secrets

tf-pipboy **does not**:
- Store credentials
- Transmit credentials over the network
- Log credentials or sensitive data

tf-pipboy **does**:
- Check authentication status via official CLI tools (aws, az, gcloud, gh)
- Read existing authentication state from CLI tools
- Respect environment variables for authentication

**Important:** tf-pipboy relies on existing authentication mechanisms. Ensure your:
- AWS credentials are properly secured
- Azure CLI tokens are managed safely
- GCP service accounts follow least-privilege principles
- GitHub tokens have appropriate scopes

### Terraform State Files

tf-pipboy interacts with Terraform state files which may contain sensitive information:

- **Current behavior**: tf-pipboy executes Terraform commands that may read and write state files
- **State location**: Supports both local and remote backends (S3, Azure Blob, GCS, etc.)
- **State management**: tf-pipboy does not directly manipulate state files; all state operations go through Terraform
- **No state data logging**: State content is never logged or transmitted by tf-pipboy
- **Future capability**: Planned support for multi-state file management and visualization

**Security recommendations:**
- Use remote backends with encryption at rest
- Enable state locking to prevent concurrent modifications
- Follow Terraform security best practices for state management
- Implement proper access controls on state storage
- Consider using encrypted backends (S3 with KMS, Azure with encryption, etc.)

### Command Execution

tf-pipboy executes Terraform commands:

- All Terraform commands run with user permissions
- No privilege escalation
- Command arguments are sanitized
- Output is captured and filtered

### Configuration Files

Project configuration files (`.tfpipboy/tfproject.yaml`) may contain:

- Backend configurations
- Module paths
- Variable values

**Best Practices:**
- Do not commit secrets to configuration files
- Use environment variables for sensitive values
- Restrict file permissions on configuration directories
- Use `.gitignore` to exclude sensitive configurations

## Security Features

### Input Validation

- Configuration file validation with error reporting
- Path validation to prevent directory traversal
- Workspace name sanitization
- Variable value validation

### Authentication Checks

- Automatic detection of expired credentials
- Blocking execution on missing required authentication
- Clear error messages for authentication failures

### Safe Defaults

- Read-only access to Terraform state
- No automatic approval of destructive operations
- Explicit confirmation required for apply/destroy
- Dry-run mode available for all operations

## Known Security Considerations

### Local Execution Only

tf-pipboy currently operates only on the local filesystem:
- No network communication (except Terraform itself)
- No cloud API calls (except via Terraform and auth tools)
- No telemetry or analytics

### Dependencies

tf-pipboy uses the following external dependencies:
- See `go.mod` for complete list
- Dependencies are managed via Go modules
- Regular security updates applied

### Filesystem Access

tf-pipboy requires filesystem access to:
- Read Terraform configuration files
- Execute Terraform binary
- Read/write workspace directories
- Create log files

Ensure appropriate filesystem permissions are set.

## Security Updates

Security updates will be released as soon as possible after a vulnerability is confirmed.

Updates will be announced via:
- GitHub Security Advisories
- Release notes
- CHANGELOG.md

## Disclosure Policy

When we receive a security report, we will:

1. Confirm receipt within 48 hours
2. Provide an initial assessment within 7 days
3. Work with the reporter to understand and reproduce the issue
4. Develop and test a fix
5. Release a security update
6. Publicly disclose the vulnerability after a fix is available

We ask that you:
- Give us reasonable time to fix the issue before public disclosure
- Make a good faith effort to avoid privacy violations and data destruction
- Do not exploit the vulnerability beyond what is necessary to demonstrate it

## Security Best Practices for Users

1. **Keep tf-pipboy updated** to the latest version
2. **Run with minimal required permissions**
3. **Audit configuration files** before sharing
4. **Use encrypted backend storage** for Terraform state
5. **Enable MFA** on cloud provider accounts
6. **Rotate credentials regularly**
7. **Review logs** for unexpected behavior
8. **Use version control** for configuration files
9. **Implement least-privilege access** for service accounts
10. **Monitor authentication status** regularly

## Compliance

tf-pipboy is designed to support:
- SOC 2 compliance (when using appropriate backend and access controls)
- GDPR compliance (no personal data collection)
- HIPAA compliance (with appropriate Terraform backend configuration)

Note: Compliance depends on how tf-pipboy is configured and used. Users are responsible for ensuring their specific configuration meets compliance requirements.

## Contact

For security concerns, contact: [TO BE ADDED]

For general questions: See [CONTRIBUTING.md](CONTRIBUTING.md)

---

**Last Updated:** November 2024
