// Package orchestrator coordinates Terraform module execution with dependency
// management, parallel execution, and real-time progress monitoring.
package orchestrator

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// AuthStatus represents authentication status for a cloud provider
type AuthStatus struct {
	Provider      string // azure, aws, gcp, github
	Authenticated bool
	Identity      string   // User/account identifier
	Token         string   // Authentication token (for GitHub)
	Accounts      []string // Multiple accounts (for providers supporting multiple logins)
	ActiveAccount string   // Currently active account
	Error         string
}

// CheckAzureAuth checks if user is authenticated with Azure CLI
func CheckAzureAuth(ctx context.Context) AuthStatus {
	status := AuthStatus{
		Provider:      "Azure",
		Authenticated: false,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// First check if there's an active account
	accountCmd := exec.CommandContext(ctx, "az", "account", "show", "--output", "json")
	accountOutput, err := accountCmd.Output()

	if err != nil {
		status.Error = "Not authenticated"
		return status
	}

	// Extract subscription info for display
	outputStr := string(accountOutput)
	if strings.Contains(outputStr, "\"name\"") {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "\"name\"") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					name := strings.TrimSpace(parts[1])
					name = strings.Trim(name, "\",")
					if name != "" {
						status.Identity = name
						break
					}
				}
			}
		}
	}

	// CRITICAL: Verify the token is actually valid by making an authenticated API call
	// This catches expired tokens that 'az account show' doesn't detect
	tokenCmd := exec.CommandContext(ctx, "az", "account", "get-access-token", "--output", "json")
	_, err = tokenCmd.CombinedOutput()

	if err != nil {
		// Token is expired or invalid
		status.Error = "Authentication expired"
		status.Authenticated = false
		return status
	}

	// Token is valid
	status.Authenticated = true
	if status.Identity == "" {
		status.Identity = "Authenticated"
	}

	return status
}

// CheckAWSAuth checks if user is authenticated with AWS CLI
func CheckAWSAuth(ctx context.Context) AuthStatus {
	status := AuthStatus{
		Provider:      "AWS",
		Authenticated: false,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
	output, err := cmd.Output()

	if err != nil {
		status.Error = "Not authenticated"
		return status
	}

	status.Authenticated = true

	// Extract account ID if available
	outputStr := string(output)
	if strings.Contains(outputStr, "\"Account\"") {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "\"Account\"") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					account := strings.TrimSpace(parts[1])
					account = strings.Trim(account, "\",")
					if account != "" {
						status.Identity = account
						break
					}
				}
			}
		}
	}

	if status.Identity == "" {
		status.Identity = "Authenticated"
	}

	return status
}

// CheckGCPAuth checks if user is authenticated with gcloud CLI
func CheckGCPAuth(ctx context.Context) AuthStatus {
	status := AuthStatus{
		Provider:      "GCP",
		Authenticated: false,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	output, err := cmd.Output()

	if err != nil {
		status.Error = "Not authenticated"
		return status
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr != "" {
		status.Authenticated = true
		status.Identity = outputStr
	} else {
		status.Error = "No active account"
	}

	return status
}

// CheckGitHubAuth checks if user is authenticated with GitHub CLI
func CheckGitHubAuth(ctx context.Context) AuthStatus {
	status := AuthStatus{
		Provider:      "GitHub",
		Authenticated: false,
		Accounts:      []string{},
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// First check if gh is installed
	cmd := exec.CommandContext(ctx, "gh", "auth", "status")
	output, err := cmd.CombinedOutput()

	if err != nil {
		status.Error = "Not authenticated or gh CLI not installed"
		return status
	}

	outputStr := string(output)

	// Parse the output to extract account information
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for "Logged in to github.com account USERNAME"
		if strings.Contains(line, "Logged in to") && strings.Contains(line, "account") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "account" && i+1 < len(parts) {
					account := parts[i+1]
					status.Accounts = append(status.Accounts, account)
					status.Identity = account
				}
			}
		}

		// Check if this is the active account
		if strings.Contains(line, "Active account: true") && status.Identity != "" {
			status.ActiveAccount = status.Identity
		}
	}

	if len(status.Accounts) > 0 {
		status.Authenticated = true

		// Get the token for the active account
		tokenCmd := exec.CommandContext(ctx, "gh", "auth", "token")
		tokenOutput, err := tokenCmd.Output()
		if err == nil {
			status.Token = strings.TrimSpace(string(tokenOutput))
		}

		// If no active account was detected, use the first one
		if status.ActiveAccount == "" && len(status.Accounts) > 0 {
			status.ActiveAccount = status.Accounts[0]
		}
	} else {
		status.Error = "No GitHub accounts found"
	}

	return status
}

// CheckRequiredAuth checks authentication for required providers based on backend types
func CheckRequiredAuth(ctx context.Context, backendTypes []string) map[string]AuthStatus {
	results := make(map[string]AuthStatus)

	// Determine which providers to check based on backend types
	needsAzure := false
	needsAWS := false
	needsGCP := false
	needsGitHub := false

	for _, backend := range backendTypes {
		switch backend {
		case "azurerm":
			needsAzure = true
		case "s3":
			needsAWS = true
		case "gcs":
			needsGCP = true
		}
	}

	// Always check GitHub as it might be needed for provider authentication
	needsGitHub = true

	// Check only required providers
	if needsAzure {
		results["azure"] = CheckAzureAuth(ctx)
	}
	if needsAWS {
		results["aws"] = CheckAWSAuth(ctx)
	}
	if needsGCP {
		results["gcp"] = CheckGCPAuth(ctx)
	}
	if needsGitHub {
		results["github"] = CheckGitHubAuth(ctx)
	}

	return results
}
