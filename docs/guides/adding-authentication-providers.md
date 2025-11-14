# Guide: Adding New Authentication Providers

This guide walks you through adding support for a new authentication provider to tf-pipboy.

## Overview

The authentication system in tf-pipboy is designed to be extensible. Each provider implements the `Checker` interface and is registered with the `Manager` for concurrent execution.

## Architecture

```
pkg/auth/
├── types.go        # Core types and interfaces
├── manager.go      # Orchestrates multiple checkers
├── azure.go        # Azure CLI implementation
├── github.go       # GitHub CLI implementation
└── <new>.go        # Your new provider
```

## Step-by-Step Implementation

### 1. Create Provider File

Create a new file in `pkg/auth/` for your provider (e.g., `aws.go`):

```go
package auth

import (
	"encoding/json"
	"os/exec"
	"time"
)

// AWSChecker checks AWS CLI authentication status
type AWSChecker struct {
	cache *Cache
}

// NewAWSChecker creates a new AWS authentication checker
func NewAWSChecker() *AWSChecker {
	return &AWSChecker{
		cache: NewCache(60 * time.Second), // 60 second TTL
	}
}
```

### 2. Implement the Checker Interface

The `Checker` interface requires three methods:

```go
type Checker interface {
	Check() (*Status, error)
	Name() string
}
```

#### Implement Name()

```go
// Name returns the provider name
func (a *AWSChecker) Name() string {
	return "aws"
}
```

#### Implement Check()

```go
// Check performs AWS CLI authentication check
func (a *AWSChecker) Check() (*Status, error) {
	// 1. Check cache first
	if cached := a.cache.Get(); cached != nil {
		return cached, nil
	}

	// 2. Initialize status
	status := &Status{
		Provider:  "aws",
		CheckedAt: time.Now(),
		Details:   make(map[string]string),
	}

	// 3. Check if CLI is installed
	if !isCommandAvailable("aws") {
		status.Authenticated = false
		status.Details["error"] = "AWS CLI not installed"
		a.cache.Set(status)
		return status, nil
	}

	// 4. Execute CLI command
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--output", "json")
	output, err := cmd.Output()

	if err != nil {
		status.Authenticated = false
		status.Details["error"] = "Not authenticated"
		a.cache.Set(status)
		return status, nil
	}

	// 5. Parse output
	var identity struct {
		UserId  string `json:"UserId"`
		Account string `json:"Account"`
		Arn     string `json:"Arn"`
	}

	if err := json.Unmarshal(output, &identity); err != nil {
		status.Authenticated = false
		status.Details["error"] = "Failed to parse response"
		a.cache.Set(status)
		return status, nil
	}

	// 6. Populate status
	status.Authenticated = true
	status.User = identity.UserId
	status.Details["account"] = identity.Account
	status.Details["arn"] = identity.Arn

	// 7. Cache and return
	a.cache.Set(status)
	return status, nil
}
```

### 3. Register with Manager

Edit `pkg/auth/manager.go` to register your new checker:

```go
func NewManager() *Manager {
	return &Manager{
		checkers: []Checker{
			NewAzureChecker(),
			NewGitHubChecker(),
			NewAWSChecker(),  // Add your checker here
		},
	}
}
```

### 4. Create Tests

Create `pkg/auth/aws_test.go`:

```go
package auth

import (
	"testing"
	"time"
)

func TestAWSChecker_Name(t *testing.T) {
	checker := NewAWSChecker()
	if checker.Name() != "aws" {
		t.Errorf("Expected name 'aws', got '%s'", checker.Name())
	}
}

func TestAWSChecker_Check_NotInstalled(t *testing.T) {
	checker := NewAWSChecker()
	status, err := checker.Check()

	if status == nil {
		t.Error("Expected status to be returned even on error")
	}

	if status != nil {
		if status.Provider != "aws" {
			t.Errorf("Expected provider 'aws', got '%s'", status.Provider)
		}
		if status.CheckedAt.IsZero() {
			t.Error("Expected CheckedAt to be set")
		}
	}
}

func TestAWSChecker_Check_Cache(t *testing.T) {
	checker := NewAWSChecker()

	// First check
	status1, err1 := checker.Check()
	if err1 != nil {
		t.Skip("Skipping cache test: AWS CLI not available or not configured")
	}

	time1 := status1.CheckedAt

	// Immediate second check should return cached result
	status2, _ := checker.Check()
	time2 := status2.CheckedAt

	// Times should be identical (from cache)
	if !time1.Equal(time2) {
		t.Errorf("Expected cached result with same timestamp")
	}
}
```

### 5. Update TUI View

Edit `pkg/tui/view.go` to display your new provider:

```go
func (m Model) renderAuthStatus() string {
	if len(m.authStatus) == 0 {
		return "Checking authentication..."
	}
	
	var parts []string
	
	// Azure
	if azure, ok := m.authStatus["azure"]; ok {
		parts = append(parts, formatAuthStatus("Azure", azure))
	}
	
	// GitHub
	if github, ok := m.authStatus["github"]; ok {
		parts = append(parts, formatAuthStatus("GitHub", github))
	}
	
	// AWS - Add this
	if aws, ok := m.authStatus["aws"]; ok {
		parts = append(parts, formatAuthStatus("AWS", aws))
	}
	
	return strings.Join(parts, " | ")
}

func formatAuthStatus(name string, status *auth.Status) string {
	if status.Authenticated {
		return fmt.Sprintf("✓ %s: %s", name, status.User)
	}
	return fmt.Sprintf("✗ %s: %s", name, status.Details["error"])
}
```

## Best Practices

### 1. Use Caching

Always implement caching with appropriate TTL (typically 60 seconds):

```go
// Check cache first
if cached := a.cache.Get(); cached != nil {
	return cached, nil
}

// ... perform check ...

// Cache result
a.cache.Set(status)
```

### 2. Handle CLI Not Installed

Check if the CLI command is available:

```go
if !isCommandAvailable("aws") {
	status.Authenticated = false
	status.Details["error"] = "AWS CLI not installed"
	a.cache.Set(status)
	return status, nil
}
```

### 3. Provide Detailed Error Messages

Set meaningful error messages in `Details`:

```go
status.Details["error"] = "Not authenticated"
status.Details["error"] = "Invalid credentials"
status.Details["error"] = "Token expired"
```

### 4. Parse Structured Output

Prefer JSON output when available:

```go
cmd := exec.Command("aws", "sts", "get-caller-identity", "--output", "json")
```

Use regex for text parsing:

```go
usernameRegex := regexp.MustCompile(`User: ([\w\-]+)`)
if matches := usernameRegex.FindStringSubmatch(output); len(matches) > 1 {
	status.User = matches[1]
}
```

### 5. Set CheckedAt Timestamp

Always set the timestamp:

```go
status := &Status{
	Provider:  "aws",
	CheckedAt: time.Now(),
	Details:   make(map[string]string),
}
```

### 6. Initialize Details Map

Always initialize the Details map:

```go
Details: make(map[string]string)
```

### 7. Handle Both Success and Failure

Cache both authenticated and unauthenticated states:

```go
if err != nil {
	status.Authenticated = false
	status.Details["error"] = "Check failed"
	a.cache.Set(status)  // Still cache the failure
	return status, nil
}
```

## Testing Your Implementation

### Run Tests

```bash
# Run all auth tests
go test ./pkg/auth/... -v

# Run specific provider tests
go test ./pkg/auth/ -run TestAWS -v
```

### Manual Testing

```bash
# Build and run
make build
./bin/tfpipboy

# The status bar should show your provider's auth status
```

### Test Different States

1. **Not installed**: Uninstall the CLI temporarily
2. **Not authenticated**: Log out of the CLI
3. **Authenticated**: Log in with valid credentials
4. **Caching**: Check that status doesn't flicker

## Common Patterns

### JSON Parsing

```go
var response struct {
	User    string `json:"user"`
	Account string `json:"account"`
}
if err := json.Unmarshal(output, &response); err != nil {
	// Handle error
}
```

### Regex Parsing

```go
import "regexp"

pattern := regexp.MustCompile(`User: ([\w\-]+)`)
if matches := pattern.FindStringSubmatch(output); len(matches) > 1 {
	username := matches[1]
}
```

### Multiple Output Sources

Some CLIs output to stderr instead of stdout:

```go
output, err := cmd.CombinedOutput()  // Gets both stdout and stderr
```

## Troubleshooting

### Provider Not Showing Up

- Check if registered in `manager.go`
- Verify `Name()` returns correct string
- Check TUI view includes your provider

### Tests Failing

- Ensure CLI is installed for integration tests
- Use `t.Skip()` when CLI not available
- Mock external dependencies for unit tests

### Cache Issues

- Verify TTL is reasonable (60s recommended)
- Check cache is initialized in constructor
- Ensure cache.Set() is called in all code paths

## Example: GCP Provider

Here's a complete example for Google Cloud:

```go
package auth

import (
	"encoding/json"
	"os/exec"
	"time"
)

type GCPChecker struct {
	cache *Cache
}

func NewGCPChecker() *GCPChecker {
	return &GCPChecker{
		cache: NewCache(60 * time.Second),
	}
}

func (g *GCPChecker) Name() string {
	return "gcp"
}

func (g *GCPChecker) Check() (*Status, error) {
	if cached := g.cache.Get(); cached != nil {
		return cached, nil
	}

	status := &Status{
		Provider:  "gcp",
		CheckedAt: time.Now(),
		Details:   make(map[string]string),
	}

	if !isCommandAvailable("gcloud") {
		status.Authenticated = false
		status.Details["error"] = "gcloud CLI not installed"
		g.cache.Set(status)
		return status, nil
	}

	cmd := exec.Command("gcloud", "auth", "list", "--format=json")
	output, err := cmd.Output()

	if err != nil {
		status.Authenticated = false
		status.Details["error"] = "Check failed"
		g.cache.Set(status)
		return status, nil
	}

	var accounts []struct {
		Account string `json:"account"`
		Status  string `json:"status"`
	}

	if err := json.Unmarshal(output, &accounts); err != nil {
		status.Authenticated = false
		status.Details["error"] = "Parse failed"
		g.cache.Set(status)
		return status, nil
	}

	// Find active account
	for _, account := range accounts {
		if account.Status == "ACTIVE" {
			status.Authenticated = true
			status.User = account.Account
			break
		}
	}

	if !status.Authenticated {
		status.Details["error"] = "No active account"
	}

	g.cache.Set(status)
	return status, nil
}
```

## Next Steps

- Add more context information (subscription, project, region)
- Implement credential expiration checking
- Add support for multiple simultaneous accounts
- Create integration tests with mocked CLIs

## References

- [Authentication Types Documentation](../auth-types.md)
- [Manager Implementation](../../pkg/auth/manager.go)
- [Azure Implementation Example](../../pkg/auth/azure.go)
- [GitHub Implementation Example](../../pkg/auth/github.go)
