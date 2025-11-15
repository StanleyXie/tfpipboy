# CLI Wrapper Code Review and Security Scan Plan

**Review Date:** 2025-11-15
**Branch:** claude/review-cli-wrapper-01SGi2ciK13nqvJi1HweKKFG
**Component:** pkg/cli/wrapper.go
**Reviewer:** Claude Code

---

## Executive Summary

This document outlines the findings from a comprehensive code review and security scan of the CLI wrapper feature in tfpipboy. The CLI wrapper provides an interactive shell interface for Terraform operations with authentication monitoring and context awareness.

**Overall Assessment:** 🟡 MODERATE RISK
The implementation is functionally sound but contains several security concerns that should be addressed before production release.

**Key Findings:**
- 3 High-priority security issues
- 4 Medium-priority security issues
- 5 Code quality improvements needed
- Test coverage insufficient for security-critical code

---

## 1. Code Review Findings

### 1.1 High-Priority Issues

#### HS-001: Command Injection Surface (HIGH)
**Location:** `pkg/cli/wrapper.go:272`
**Severity:** HIGH (by design, but needs documentation)
**Finding:**
```go
cmd := exec.Command(shell, "-c", command)
```

**Analysis:**
- User input from readline is passed directly to shell execution
- This is intentional functionality for a CLI wrapper
- However, it creates a significant attack surface if the tool is ever used programmatically or with untrusted input sources

**Risk:**
- If history file or readline library has vulnerabilities, arbitrary command execution possible
- No input sanitization or validation
- All environment variables exposed to child processes

**Recommendation:**
- ✅ Add security warning in documentation
- ✅ Add comment explaining security model in code
- ✅ Consider adding optional "restricted mode" for automated use
- ✅ Implement command logging for audit trail

---

#### HS-002: Unsafe Package Usage
**Location:** `pkg/cli/wrapper.go:12, 359-380`
**Severity:** MEDIUM
**Finding:**
```go
import "unsafe"
// ...
retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
    uintptr(syscall.Stdout),
    uintptr(syscall.TIOCGWINSZ),
    uintptr(unsafe.Pointer(ws)))
```

**Analysis:**
- Uses unsafe pointer for terminal size detection
- Syscall directly accesses OS-level IOCTL
- Proper bounds checking exists (line 374-377)
- Falls back to safe defaults on error

**Risk:**
- Potential for memory corruption if winsize struct changes
- Platform-specific code (Linux-only)
- Unsafe pointer manipulation

**Recommendation:**
- ✅ Document platform requirements
- ✅ Add build tags for Linux-only compilation
- ✅ Consider using established terminal library (e.g., golang.org/x/term)
- ⚠️ Add unit tests for error conditions

---

#### HS-003: History File Security
**Location:** `pkg/cli/wrapper.go:68`
**Severity:** MEDIUM
**Finding:**
```go
HistoryFile: os.ExpandEnv("$HOME/.tfpipboy_history"),
```

**Analysis:**
- History file created without explicit permission setting
- Could contain sensitive commands (e.g., with credentials in arguments)
- No cleanup or rotation policy
- Readline library handles file creation

**Risk:**
- History file may have overly permissive permissions
- Sensitive data persisted to disk
- No size limits (could grow unbounded)

**Recommendation:**
- ✅ Set file permissions to 0600 (owner read/write only)
- ✅ Add history file size limit
- ✅ Implement history cleanup on exit
- ✅ Add option to disable history
- ✅ Document sensitive data handling

---

### 1.2 Medium-Priority Issues

#### MS-001: Environment Variable Exposure
**Location:** `pkg/cli/wrapper.go:274`
**Severity:** MEDIUM
**Finding:**
```go
cmd.Env = os.Environ()
```

**Analysis:**
- All environment variables passed to child processes
- Includes potentially sensitive data (API keys, tokens, etc.)
- Standard practice but increases attack surface

**Risk:**
- Credential leakage through child processes
- Environment variable injection attacks
- Debugging output could expose secrets

**Recommendation:**
- ✅ Document environment variable handling
- ✅ Consider filtering sensitive env vars (or allowlist approach)
- ✅ Add warning if sensitive env vars detected

---

#### MS-002: Missing Input Validation
**Location:** `pkg/cli/wrapper.go:239-263`
**Severity:** MEDIUM
**Finding:**
```go
func (w *Wrapper) handleCD(parts []string) {
    // ...
    if strings.HasPrefix(dir, "~") {
        dir = strings.Replace(dir, "~", os.Getenv("HOME"), 1)
    }
    if err := os.Chdir(dir); err != nil {
        fmt.Printf("\033[31mcd: %v\033[0m\n", err)
        return
    }
}
```

**Analysis:**
- No path validation or sanitization
- Relies on os.Chdir for validation
- No check for directory traversal attempts
- No canonicalization of paths

**Risk:**
- Directory traversal possible (though limited by user's own permissions)
- Symlink following could lead to unexpected locations
- No prevention of "cd ../../sensitive/path"

**Recommendation:**
- ✅ Add path validation
- ✅ Use filepath.Abs() and filepath.Clean()
- ✅ Optional: Implement working directory restrictions
- ✅ Add logging of directory changes

---

#### MS-003: Signal Handler Race Condition
**Location:** `pkg/cli/wrapper.go:78-86`
**Severity:** LOW
**Finding:**
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
go func() {
    <-sigChan
    rl.Close()
    fmt.Println("\nGoodbye!")
    os.Exit(0)
}()
```

**Analysis:**
- Signal handler goroutine started without context
- No cleanup of resources beyond readline
- Abrupt termination with os.Exit(0)
- No synchronization with main loop

**Risk:**
- Resources might not be properly cleaned up
- Running commands could be orphaned
- Race condition between signal handler and main loop

**Recommendation:**
- ✅ Use context.Context for graceful shutdown
- ✅ Add cleanup routine for running processes
- ✅ Synchronize with main loop before exit
- ✅ Use sync.WaitGroup for goroutine coordination

---

#### MS-004: External Command Timeouts
**Location:** `pkg/cli/wrapper.go:265-288`
**Severity:** LOW
**Finding:**
```go
func (w *Wrapper) runExternalCommand(command string) {
    cmd := exec.Command(shell, "-c", command)
    // ...
    if err := cmd.Run(); err != nil {
        fmt.Printf("\n\033[31m[Error: %v]\033[0m\n", err)
    }
}
```

**Analysis:**
- No timeout on external command execution
- Long-running commands could hang indefinitely
- No progress indication for long operations
- No way to cancel running commands

**Risk:**
- UI can hang on slow/blocked commands
- No ability to interrupt hung processes
- Resource exhaustion possible

**Recommendation:**
- ✅ Add context with timeout for commands
- ✅ Implement command cancellation (Ctrl+C forwarding)
- ✅ Add progress indication for long operations
- ✅ Consider command execution time tracking

---

### 1.3 Code Quality Issues

#### CQ-001: Version Mismatch
**Location:** `pkg/cli/wrapper.go:141`
**Finding:**
```go
fmt.Printf("%s│%s                        %stfpipboy v0.2.0%s
```
**Issue:** Version hardcoded as v0.2.0, but main.go shows v0.6.0

**Recommendation:**
- ✅ Create version constant in shared package
- ✅ Import version from main package
- ✅ Use build-time version injection

---

#### CQ-002: Dead Code
**Location:** `pkg/cli/wrapper.go:302-315`
**Finding:**
```go
// statusUpdateLoop runs in background to update status
// NOTE: This function is currently disabled because background updates
// interfere with liner's display. Status is updated after each command instead.
func (w *Wrapper) statusUpdateLoop() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            w.updateStatus()
        }
    }
}
```

**Recommendation:**
- ✅ Remove unused function
- ✅ Or implement properly with readline compatibility
- ✅ Document why synchronous updates are preferred

---

#### CQ-003: Error Handling
**Location:** Multiple locations
**Finding:**
- Some errors silently ignored (e.g., `os.Getwd()` at line 195, 261, 273)
- Inconsistent error reporting format
- No structured logging

**Recommendation:**
- ✅ Add consistent error handling
- ✅ Implement structured logging (e.g., slog)
- ✅ Return errors to caller where appropriate

---

#### CQ-004: Magic Numbers
**Location:** `pkg/cli/wrapper.go:19-37`
**Finding:**
```go
const (
    colorReset   = "\033[0m"
    colorRed     = "\033[31m"
    // ...
)
```

**Recommendation:**
- ✅ Good use of constants
- ✅ Consider extracting to shared color package
- ✅ Add ANSI capability detection

---

#### CQ-005: Function Complexity
**Location:** `pkg/cli/wrapper.go:59-135 (Run method)`
**Finding:** Run() method is 76 lines with multiple responsibilities

**Recommendation:**
- ✅ Extract setup logic to separate function
- ✅ Extract command loop to separate function
- ✅ Improve testability through decomposition

---

## 2. Security Scan Findings

### 2.1 Dependency Analysis

**Go Version:** 1.25.0
**Direct Dependencies:**
- github.com/ergochat/readline v0.1.3
- github.com/charmbracelet/bubbletea v1.3.10
- github.com/charmbracelet/lipgloss v1.1.0
- gopkg.in/yaml.v3 v3.0.1

**Security Notes:**
- ✅ All dependencies from reputable sources
- ✅ No known critical vulnerabilities in current versions
- ⚠️ Recommend running `go mod audit` regularly
- ⚠️ Consider using Dependabot for automated updates

---

### 2.2 Static Analysis Recommendations

Based on the security workflow (`.github/workflows/release.yml`), the following tools are already configured:

**Existing Security Tools:**
- ✅ Gosec (static analysis)
- ✅ Trivy (vulnerability scanning)
- ✅ CodeQL (code analysis)
- ✅ Race detector
- ✅ SBOM generation

**Additional Recommendations:**
- Add staticcheck for linting
- Add golangci-lint with security-focused linters
- Add govulncheck for Go vulnerability scanning

---

### 2.3 Authentication System Review

**Files Reviewed:**
- `pkg/auth/manager.go`
- `pkg/auth/azure.go`
- `pkg/auth/github.go`

**Findings:**
- ✅ Read-only authentication checks (no credential storage)
- ✅ Uses official CLI tools (az, gh)
- ✅ Proper caching with TTL (60 seconds)
- ✅ Concurrent checking with proper synchronization
- ⚠️ No rate limiting on auth checks
- ⚠️ Regex patterns could be more robust (github.go:79-96)

**Recommendations:**
- ✅ Add rate limiting to prevent CLI tool abuse
- ✅ Add metrics for auth check performance
- ✅ Improve regex patterns with better error handling
- ✅ Add tests for regex pattern matching

---

## 3. Risk Assessment

### 3.1 Risk Matrix

| Risk ID | Description | Likelihood | Impact | Overall Risk | Priority |
|---------|-------------|------------|--------|--------------|----------|
| HS-001 | Command Injection Surface | Medium | High | High | P0 |
| HS-002 | Unsafe Package Usage | Low | Medium | Low-Med | P2 |
| HS-003 | History File Security | Medium | Medium | Medium | P1 |
| MS-001 | Environment Variable Exposure | Medium | Low | Low-Med | P2 |
| MS-002 | Missing Input Validation | Low | Low | Low | P3 |
| MS-003 | Signal Handler Race | Low | Low | Low | P3 |
| MS-004 | External Command Timeouts | Medium | Low | Low-Med | P2 |

---

### 3.2 Attack Vectors

**1. Malicious History File**
- Attacker modifies `.tfpipboy_history`
- User loads history containing malicious commands
- Commands executed when user scrolls through history
- **Mitigation:** Validate history file permissions, sanitize loaded commands

**2. Environment Variable Injection**
- Malicious process sets environment variables
- Variables inherited by wrapper and passed to commands
- Credential theft or privilege escalation
- **Mitigation:** Filter sensitive env vars, document security model

**3. Terminal Escape Sequence Injection**
- Malicious command output contains ANSI escape sequences
- Terminal emulator vulnerabilities exploited
- **Mitigation:** Use readline's built-in escaping, sanitize output

---

## 4. Test Coverage Analysis

**Current State:**
- Total test files: 7
- CLI wrapper tests: ❌ NONE
- Auth package tests: ✅ 3 files (azure_test.go, github_test.go, manager_test.go)
- Terraform package tests: ✅ 3 files

**Test Coverage Gaps:**
1. ❌ No tests for pkg/cli/wrapper.go
2. ❌ No integration tests for authentication flow
3. ❌ No security-focused test cases
4. ❌ No fuzz testing for input validation
5. ❌ No tests for signal handling
6. ❌ No tests for error conditions

**Recommendations:**

### Unit Tests Needed:
```go
// pkg/cli/wrapper_test.go
- TestNewWrapper()
- TestBuildPrompt()
- TestHandleCD()
- TestHandleBuiltinCommand()
- TestAddToHistory()
- TestSignalHandling()
- TestGetTerminalSize()
```

### Security Tests Needed:
```go
// pkg/cli/wrapper_security_test.go
- TestCommandInjectionProtection()
- TestPathTraversal()
- TestHistoryFilePermissions()
- TestEnvironmentVariableFiltering()
- TestANSIEscapeHandling()
```

### Integration Tests Needed:
```go
// pkg/cli/wrapper_integration_test.go
- TestFullCommandExecution()
- TestAuthenticationFlow()
- TestWorkspaceContextSwitch()
```

---

## 5. Remediation Plan

### Phase 1: Critical Security Fixes (Week 1)

**Priority P0 - Must Fix Before Release:**

1. **HS-001: Document Security Model**
   - Add security.md section for CLI wrapper
   - Document command execution model
   - Add warning about untrusted input
   - Estimate: 2 hours

2. **HS-003: Fix History File Security**
   - Set history file permissions to 0600
   - Add history size limit (10,000 lines)
   - Implement cleanup on exit
   - Estimate: 4 hours

---

### Phase 2: Medium Priority Fixes (Week 2)

**Priority P1:**

3. **MS-001: Environment Variable Security**
   - Document env var handling
   - Add optional filtering for sensitive vars
   - Implement warning system
   - Estimate: 6 hours

4. **MS-004: Command Timeout Implementation**
   - Add context with configurable timeout
   - Implement Ctrl+C forwarding
   - Add progress indication
   - Estimate: 8 hours

5. **CQ-001: Version Management**
   - Create shared version package
   - Update all version references
   - Add build-time injection
   - Estimate: 2 hours

---

### Phase 3: Code Quality & Testing (Week 3)

**Priority P2:**

6. **Test Coverage**
   - Write unit tests for wrapper.go (target: 80% coverage)
   - Add security test cases
   - Add integration tests
   - Estimate: 16 hours

7. **MS-002: Input Validation**
   - Add path validation for cd command
   - Implement canonicalization
   - Add logging
   - Estimate: 4 hours

8. **CQ-002: Code Cleanup**
   - Remove dead code
   - Refactor Run() method
   - Improve error handling
   - Estimate: 6 hours

---

### Phase 4: Enhancements (Week 4)

**Priority P3:**

9. **HS-002: Terminal Library Migration**
   - Evaluate golang.org/x/term
   - Migrate away from unsafe package
   - Add platform build tags
   - Estimate: 8 hours

10. **MS-003: Graceful Shutdown**
    - Implement context-based shutdown
    - Add resource cleanup
    - Improve signal handling
    - Estimate: 6 hours

11. **Additional Security Tools**
    - Add staticcheck to CI
    - Configure golangci-lint
    - Add govulncheck
    - Estimate: 4 hours

---

## 6. Implementation Checklist

### Immediate Actions (Before Production Release)

- [ ] Add security documentation for CLI wrapper
- [ ] Fix history file permissions (0600)
- [ ] Implement history size limit
- [ ] Add version constant management
- [ ] Document environment variable handling
- [ ] Add security warning comments in code

### Short-Term (v0.7.0)

- [ ] Write comprehensive unit tests (target: 80% coverage)
- [ ] Add security-focused test cases
- [ ] Implement command timeouts
- [ ] Add input validation for cd command
- [ ] Remove dead code
- [ ] Refactor Run() method for testability
- [ ] Add structured logging

### Medium-Term (v0.8.0)

- [ ] Migrate to golang.org/x/term
- [ ] Implement graceful shutdown
- [ ] Add command execution audit logging
- [ ] Implement restricted mode for automation
- [ ] Add environment variable filtering
- [ ] Add staticcheck and golangci-lint to CI
- [ ] Add fuzz testing for input handlers

### Long-Term (v1.0.0+)

- [ ] Consider adding command allowlist/denylist
- [ ] Implement session recording capability
- [ ] Add plugin system for custom commands
- [ ] Add remote execution capability (with proper auth)
- [ ] Implement RBAC for command execution
- [ ] Add telemetry and metrics

---

## 7. Testing Strategy

### 7.1 Unit Testing

**Coverage Target:** 80% for pkg/cli/wrapper.go

**Test Categories:**
1. Positive path testing
2. Error condition testing
3. Edge case testing
4. Security testing

**Example Test Structure:**
```go
func TestHandleCD_PathTraversal(t *testing.T) {
    w := NewWrapper()

    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"normal path", "testdir", false},
        {"home directory", "~", false},
        {"parent directory", "..", false},
        {"absolute path", "/tmp", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

---

### 7.2 Integration Testing

**Scenarios:**
1. Full command execution flow
2. Authentication status updates
3. Terraform context detection
4. Signal handling and cleanup

**Tools:**
- Test containers for isolated environments
- Mock external CLI tools (az, gh, terraform)
- Table-driven test approach

---

### 7.3 Security Testing

**Fuzz Testing:**
```go
func FuzzCommandInput(f *testing.F) {
    // Seed corpus
    f.Add("terraform plan")
    f.Add("cd ../../../")
    f.Add("\x1b[31mmalicious\x1b[0m")

    f.Fuzz(func(t *testing.T, input string) {
        w := NewWrapper()
        // Ensure no crashes or security violations
        w.executeCommand(input)
    })
}
```

**Penetration Testing Checklist:**
- [ ] Command injection attempts
- [ ] Path traversal attempts
- [ ] ANSI escape sequence injection
- [ ] Environment variable manipulation
- [ ] History file tampering
- [ ] Signal bombing
- [ ] Resource exhaustion attacks

---

## 8. Documentation Requirements

### 8.1 Security Documentation

**Required Sections:**
1. **Security Model**
   - Command execution model
   - Trust boundaries
   - Threat model

2. **Safe Usage Guidelines**
   - Do's and don'ts
   - Best practices
   - Configuration recommendations

3. **Incident Response**
   - Reporting vulnerabilities
   - Security update process
   - Known limitations

---

### 8.2 User Documentation

**Required Additions:**
1. CLI wrapper feature overview
2. Authentication monitoring explanation
3. Security considerations
4. Troubleshooting guide
5. FAQ for common issues

---

## 9. Compliance & Standards

### 9.1 Security Standards Compliance

**Recommended Standards:**
- [ ] OWASP Secure Coding Practices
- [ ] CWE Top 25 Most Dangerous Software Weaknesses
- [ ] NIST SP 800-218 (Secure Software Development Framework)

### 9.2 Code Quality Standards

- [ ] Go Code Review Comments compliance
- [ ] Effective Go best practices
- [ ] Project-specific coding standards

---

## 10. Monitoring & Maintenance

### 10.1 Security Monitoring

**Metrics to Track:**
1. Authentication check failure rates
2. Command execution errors
3. Unusual command patterns
4. Performance degradation

**Alerting:**
- Failed authentication attempts
- Suspicious command patterns
- Resource exhaustion
- Crash/panic events

---

### 10.2 Dependency Management

**Process:**
1. Weekly dependency update check
2. Monthly security audit
3. Quarterly dependency cleanup
4. Automated Dependabot PRs

**Tools:**
- `go mod audit` (Go 1.25+)
- Trivy for vulnerability scanning
- Dependabot for automated updates

---

## 11. Release Criteria

### 11.1 Security Release Checklist

Before releasing CLI wrapper feature to production:

**Must Have (Blocking):**
- [x] Security documentation complete
- [ ] History file permissions fixed
- [ ] Version management implemented
- [ ] P0 security issues resolved
- [ ] Security-focused tests written
- [ ] Code review by security team
- [ ] Penetration testing completed

**Should Have (Non-Blocking):**
- [ ] 80% test coverage achieved
- [ ] All P1 issues resolved
- [ ] Integration tests passing
- [ ] Performance benchmarks established

**Nice to Have:**
- [ ] All P2 issues resolved
- [ ] Fuzz testing implemented
- [ ] Command timeout implemented
- [ ] Graceful shutdown implemented

---

## 12. Continuous Improvement

### 12.1 Feedback Loop

**Process:**
1. Collect user feedback on security concerns
2. Monitor security advisories for dependencies
3. Review incident reports
4. Update threat model quarterly
5. Perform annual security audit

### 12.2 Security Training

**Recommendations:**
1. Team training on secure Go development
2. Security champion designation
3. Regular security review meetings
4. Incident response drills

---

## 13. Conclusion

The CLI wrapper feature provides valuable functionality for interactive Terraform operations. While the current implementation is functionally sound, several security improvements are recommended before production release.

**Priority Actions:**
1. ✅ Fix history file permissions (P0)
2. ✅ Add comprehensive security documentation (P0)
3. ✅ Implement test coverage (P1)
4. ✅ Add command timeouts (P1)
5. ✅ Improve error handling and logging (P2)

**Timeline to Production:**
- **Phase 1 (Week 1):** Critical security fixes
- **Phase 2 (Week 2):** Medium priority fixes
- **Phase 3 (Week 3):** Testing and code quality
- **Phase 4 (Week 4):** Final enhancements and release prep

**Estimated Total Effort:** 66 hours (~2 weeks with 1 developer)

---

## 14. Appendix

### A. References

**Security Resources:**
- [OWASP Go Secure Coding Practices](https://owasp.org/www-project-go-secure-coding-practices-guide/)
- [CWE-78: OS Command Injection](https://cwe.mitre.org/data/definitions/78.html)
- [Go Security](https://go.dev/security/)

**Code Quality Resources:**
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### B. Tools & Libraries

**Security Scanning:**
- Gosec: https://github.com/securego/gosec
- Trivy: https://github.com/aquasecurity/trivy
- Govulncheck: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck

**Testing:**
- Testify: https://github.com/stretchr/testify
- Go-fuzz: https://github.com/dvyukov/go-fuzz

**Alternative Libraries:**
- golang.org/x/term: Better terminal handling
- github.com/urfave/cli: Alternative CLI framework
- github.com/spf13/cobra: Command structure

---

## Document Information

**Version:** 1.0
**Last Updated:** 2025-11-15
**Next Review:** 2025-12-15
**Owner:** Development Team
**Approvers:** Security Team, Tech Lead

**Change Log:**
- 2025-11-15: Initial review and plan creation
