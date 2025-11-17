// Package auth provides cloud provider authentication status checking
// for AWS, Azure, GCP, and GitHub services.
package auth

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// AzureChecker checks Azure CLI authentication status
type AzureChecker struct {
	cache *Cache
}

// NewAzureChecker creates a new Azure authentication checker
func NewAzureChecker() *AzureChecker {
	return &AzureChecker{
		cache: NewCache(60 * time.Second), // 60 second TTL
	}
}

// Name returns the provider name
func (a *AzureChecker) Name() string {
	return "azure"
}

// Check performs Azure CLI authentication check
func (a *AzureChecker) Check() (*Status, error) {
	// Try to get from cache first
	if cached := a.cache.Get(); cached != nil {
		return cached, nil
	}

	// Perform actual check
	status := &Status{
		Provider:  "azure",
		CheckedAt: time.Now(),
		Details:   make(map[string]string),
	}

	// Check if az CLI is installed
	if !isCommandAvailable("az") {
		status.Authenticated = false
		status.Details["error"] = "Azure CLI not installed"
		a.cache.Set(status)
		return status, nil
	}

	// Execute: az account show --output json
	cmd := exec.Command("az", "account", "show", "--output", "json")
	output, err := cmd.Output()

	if err != nil {
		// Not authenticated or error
		status.Authenticated = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Please run 'az login'") ||
				strings.Contains(stderr, "not logged in") {
				status.Details["error"] = "Not logged in"
			} else {
				status.Details["error"] = "Check failed"
			}
		}
		a.cache.Set(status)
		return status, nil
	}

	// Parse JSON output
	var account AzureAccount
	if err := json.Unmarshal(output, &account); err != nil {
		status.Authenticated = false
		status.Details["error"] = "Failed to parse response"
		a.cache.Set(status)
		return status, fmt.Errorf("failed to parse az output: %w", err)
	}

	// Successfully authenticated
	status.Authenticated = true
	status.User = account.User.Name
	status.Details["subscription"] = account.Name
	status.Details["subscription_id"] = account.ID
	status.Details["tenant_id"] = account.TenantID
	status.Details["state"] = account.State

	a.cache.Set(status)
	return status, nil
}

// AzureAccount represents the JSON response from 'az account show'
type AzureAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	State    string `json:"state"`
	TenantID string `json:"tenantId"`
	User     struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"user"`
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
