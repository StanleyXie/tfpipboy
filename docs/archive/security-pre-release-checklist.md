# Security Pre-Release Checklist

This checklist must be completed before making the tf-pipboy repository public.

## ✅ Completed Items

### 1. Examples Directory
- [x] Examples moved to `../tf-pipboy-examples` (outside repo)
- [x] Symlink created for local development access
- [x] `.gitignore` configured to exclude `examples/`
- [x] All example files removed from git history in current branch
- [x] Commit: `a39d04f` - "chore: remove examples directory from repository"

## 🔍 Required Security Audit

### 2. Source Code Review

#### Hardcoded Credentials
- [ ] Scan for hardcoded passwords, API keys, tokens
  ```bash
  grep -ri "password\|secret\|api[_-]key\|token" pkg/ cmd/ internal/ --exclude-dir=.git
  ```
- [ ] Check for AWS access keys (AKIA prefix)
- [ ] Check for Azure credentials or subscription IDs
- [ ] Check for GCP service account keys
- [ ] Check for GitHub tokens

#### Sensitive Configuration
- [ ] Review `.tfpipboy/` test configuration directories
- [ ] Check for environment-specific configurations in code
- [ ] Verify no production URLs or endpoints hardcoded
- [ ] Check test files for sensitive data

#### Personal Information
- [ ] Scan for email addresses (except public contacts)
- [ ] Check for internal usernames
- [ ] Review git commit history for author info
- [ ] Check for internal hostnames or IP addresses

### 3. Documentation Review

- [ ] README.md - Remove company-specific references
- [ ] CHANGELOG.md - Review for sensitive information
- [ ] Design documents - Check for architecture details that shouldn't be public
- [ ] Check all `*.md` files for:
  - Internal URLs
  - Company names (if confidential)
  - Infrastructure details
  - Deployment procedures

### 4. Git History Audit

- [ ] Run git-secrets or gitleaks to scan history
  ```bash
  # Install gitleaks
  brew install gitleaks
  
  # Scan repository
  gitleaks detect --source . --verbose
  ```
- [ ] Check for commits with sensitive messages
- [ ] Review all branches for leaked secrets
- [ ] Consider using BFG Repo-Cleaner if secrets found in history

### 5. Configuration Files

- [ ] `.goreleaser.yml` - Check for sensitive build configurations
- [ ] `.github/workflows/` - Review CI/CD secrets usage
- [ ] Check for `.env` files (should be in .gitignore)
- [ ] Review `go.mod` for internal package dependencies

### 6. Test Data

- [ ] `tests/` directory - Remove real credentials
- [ ] Check test fixtures for sensitive data
- [ ] Verify mock data is truly mocked
- [ ] Review test configuration files

### 7. Build Artifacts

- [ ] Verify `dist/` is in .gitignore
- [ ] Check no binaries in repo
- [ ] Ensure build logs not committed
- [ ] Verify `.DS_Store` files are excluded

### 8. Legal & Compliance

- [ ] LICENSE file is correct and appropriate
- [ ] No proprietary code included
- [ ] Third-party licenses acknowledged
- [ ] Copyright notices appropriate
- [ ] Check CONTRIBUTING.md for company-specific policies

## 🛡️ Security Best Practices

### 9. Code Security

- [ ] No SQL injection vulnerabilities
- [ ] No command injection risks
- [ ] File path handling is secure (no path traversal)
- [ ] Input validation on all user inputs
- [ ] No eval() or similar dangerous functions

### 10. Dependency Security

- [ ] Run `go list -m all` to review dependencies
- [ ] Check for known vulnerabilities:
  ```bash
  go install golang.org/x/vuln/cmd/govulncheck@latest
  govulncheck ./...
  ```
- [ ] Review direct dependencies for security issues
- [ ] Check indirect dependencies

### 11. Authentication & Authorization

- [ ] Authentication checks only use official CLIs
- [ ] No credential storage in code
- [ ] No authentication bypass paths
- [ ] Proper error messages (don't leak auth details)

## 🔒 Pre-Release Actions

### 12. Final Verification

- [ ] Push changes to feature branch
- [ ] Review entire diff before merging to main
- [ ] Run full test suite
- [ ] Build and test binary
- [ ] Code review by second person (if available)

### 13. GitHub Repository Settings

- [ ] Remove all secrets from GitHub Actions (if not needed)
- [ ] Configure branch protection rules
- [ ] Set up security advisories
- [ ] Enable Dependabot alerts
- [ ] Configure security policy (SECURITY.md)

### 14. Documentation Updates

- [ ] Update README with public installation instructions
- [ ] Add SECURITY.md with vulnerability reporting process
- [ ] Update CONTRIBUTING.md for public contributors
- [ ] Add CODE_OF_CONDUCT.md if desired
- [ ] Update issue templates

## 📋 Tools to Use

### Recommended Security Scanning Tools

1. **gitleaks** - Scan for secrets in git history
   ```bash
   brew install gitleaks
   gitleaks detect --source . --verbose
   ```

2. **git-secrets** - Prevent secrets from being committed
   ```bash
   brew install git-secrets
   git secrets --scan-history
   ```

3. **govulncheck** - Scan Go code for vulnerabilities
   ```bash
   go install golang.org/x/vuln/cmd/govulncheck@latest
   govulncheck ./...
   ```

4. **trivy** - Comprehensive security scanner
   ```bash
   brew install trivy
   trivy fs --security-checks vuln,config .
   ```

5. **semgrep** - Static analysis for security issues
   ```bash
   brew install semgrep
   semgrep --config=auto .
   ```

## ✅ Final Checklist

Before making repository public:

- [ ] All items in this checklist completed
- [ ] Security scan tools run with no critical findings
- [ ] Code review completed
- [ ] Documentation reviewed and cleaned
- [ ] Examples removed from repository
- [ ] Git history reviewed
- [ ] No sensitive data in any branch
- [ ] GitHub repository settings configured
- [ ] Ready for public release

## 📝 Sign-Off

- [ ] Security review completed by: ________________
- [ ] Date: ________________
- [ ] Any issues found: ________________
- [ ] All issues resolved: ________________
- [ ] Approved for public release: ________________

## 📖 References

- [GitHub Security Best Practices](https://docs.github.com/en/code-security)
- [OWASP Secure Coding Practices](https://owasp.org/www-project-secure-coding-practices-quick-reference-guide/)
- [Go Security Checklists](https://go.dev/security/)

---

**Next Steps After Completion:**
1. Merge feature branch to main
2. Create release tag
3. Change repository visibility to public
4. Announce release (if desired)
