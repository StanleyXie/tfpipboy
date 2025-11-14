package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BackendAuthStatus represents the authentication status for a backend
type BackendAuthStatus struct {
	BackendType   string // azurerm, s3, gcs, etc.
	Authenticated bool
	Message       string // User-friendly status message
	Details       string // Additional details (account info, etc.)
	Error         string // Error message if check failed
}

// ValidateBackendAuth checks if authentication is valid for a given backend type
func ValidateBackendAuth(ctx context.Context, backendType string) *BackendAuthStatus {
	status := &BackendAuthStatus{
		BackendType: backendType,
	}

	// Set timeout for auth checks
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	switch strings.ToLower(backendType) {
	case "azurerm":
		return checkAzureAuth(checkCtx)
	case "s3":
		return checkAWSAuth(checkCtx)
	case "gcs":
		return checkGCPAuth(checkCtx)
	case "local":
		// Local backend doesn't require authentication
		status.Authenticated = true
		status.Message = "Local backend (no authentication required)"
		return status
	default:
		// Unknown backend type - assume no auth check needed
		status.Authenticated = true
		status.Message = fmt.Sprintf("Backend type '%s' (no authentication check)", backendType)
		return status
	}
}

// checkAzureAuth checks Azure CLI authentication status
func checkAzureAuth(ctx context.Context) *BackendAuthStatus {
	status := &BackendAuthStatus{
		BackendType: "azurerm",
	}

	// Check if az CLI is installed
	if !isCommandAvailable("az") {
		status.Authenticated = false
		status.Message = "Azure CLI not installed"
		status.Error = "Azure CLI (az) is not installed or not in PATH"
		return status
	}

	// Run: az account show --output json
	cmd := exec.CommandContext(ctx, "az", "account", "show", "--output", "json")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if it's a timeout
		if ctx.Err() == context.DeadlineExceeded {
			status.Authenticated = false
			status.Message = "Azure CLI check timed out"
			status.Error = "Authentication check timed out - Azure CLI may be unresponsive"
			return status
		}

		// Parse error output for specific messages
		errorMsg := string(output)

		if strings.Contains(errorMsg, "Please run 'az login'") ||
			strings.Contains(errorMsg, "not logged in") ||
			strings.Contains(errorMsg, "No subscriptions found") {
			status.Authenticated = false
			status.Message = "Not logged in to Azure"
			status.Error = "Azure CLI is not authenticated. Run 'az login' to authenticate."
			return status
		}

		if strings.Contains(errorMsg, "token has expired") ||
			strings.Contains(errorMsg, "token expired") ||
			strings.Contains(errorMsg, "AADSTS") { // Azure AD error codes
			status.Authenticated = false
			status.Message = "Azure token expired"
			status.Error = "Azure CLI session has expired. Run 'az login' to refresh authentication."
			return status
		}

		// Generic error
		status.Authenticated = false
		status.Message = "Azure CLI error"
		status.Error = fmt.Sprintf("Failed to check Azure authentication: %s", strings.TrimSpace(errorMsg))
		return status
	}

	// Parse successful output
	var accountInfo map[string]interface{}
	if err := json.Unmarshal(output, &accountInfo); err == nil {
		// Extract account details
		if name, ok := accountInfo["name"].(string); ok {
			status.Details = fmt.Sprintf("Subscription: %s", name)
		}
		if user, ok := accountInfo["user"].(map[string]interface{}); ok {
			if userName, ok := user["name"].(string); ok {
				if status.Details != "" {
					status.Details += " | "
				}
				status.Details += fmt.Sprintf("User: %s", userName)
			}
		}
	}

	status.Authenticated = true
	status.Message = "Authenticated to Azure"
	return status
}

// checkAWSAuth checks AWS CLI authentication status
func checkAWSAuth(ctx context.Context) *BackendAuthStatus {
	status := &BackendAuthStatus{
		BackendType: "s3",
	}

	// Check if aws CLI is installed
	if !isCommandAvailable("aws") {
		status.Authenticated = false
		status.Message = "AWS CLI not installed"
		status.Error = "AWS CLI is not installed or not in PATH"
		return status
	}

	// Run: aws sts get-caller-identity --output json
	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if it's a timeout
		if ctx.Err() == context.DeadlineExceeded {
			status.Authenticated = false
			status.Message = "AWS CLI check timed out"
			status.Error = "Authentication check timed out - AWS CLI may be unresponsive"
			return status
		}

		// Parse error output for specific messages
		errorMsg := string(output)

		if strings.Contains(errorMsg, "Unable to locate credentials") ||
			strings.Contains(errorMsg, "No credentials found") {
			status.Authenticated = false
			status.Message = "No AWS credentials found"
			status.Error = "AWS credentials not configured. Run 'aws configure' or 'aws sso login'."
			return status
		}

		if strings.Contains(errorMsg, "ExpiredToken") ||
			strings.Contains(errorMsg, "token has expired") ||
			strings.Contains(errorMsg, "security token included in the request is expired") {
			status.Authenticated = false
			status.Message = "AWS token expired"
			status.Error = "AWS session token has expired. Run 'aws sso login' to refresh authentication."
			return status
		}

		if strings.Contains(errorMsg, "InvalidClientTokenId") {
			status.Authenticated = false
			status.Message = "Invalid AWS credentials"
			status.Error = "AWS credentials are invalid. Check your AWS access key configuration."
			return status
		}

		// Generic error
		status.Authenticated = false
		status.Message = "AWS CLI error"
		status.Error = fmt.Sprintf("Failed to check AWS authentication: %s", strings.TrimSpace(errorMsg))
		return status
	}

	// Parse successful output
	var identity map[string]interface{}
	if err := json.Unmarshal(output, &identity); err == nil {
		// Extract identity details
		if account, ok := identity["Account"].(string); ok {
			status.Details = fmt.Sprintf("Account: %s", account)
		}
		if arn, ok := identity["Arn"].(string); ok {
			if status.Details != "" {
				status.Details += " | "
			}
			status.Details += fmt.Sprintf("ARN: %s", arn)
		}
	}

	status.Authenticated = true
	status.Message = "Authenticated to AWS"
	return status
}

// checkGCPAuth checks GCP CLI authentication status
func checkGCPAuth(ctx context.Context) *BackendAuthStatus {
	status := &BackendAuthStatus{
		BackendType: "gcs",
	}

	// Check if gcloud CLI is installed
	if !isCommandAvailable("gcloud") {
		status.Authenticated = false
		status.Message = "Google Cloud SDK not installed"
		status.Error = "gcloud CLI is not installed or not in PATH"
		return status
	}

	// Run: gcloud auth application-default print-access-token
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "application-default", "print-access-token")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if it's a timeout
		if ctx.Err() == context.DeadlineExceeded {
			status.Authenticated = false
			status.Message = "GCP CLI check timed out"
			status.Error = "Authentication check timed out - gcloud CLI may be unresponsive"
			return status
		}

		// Parse error output for specific messages
		errorMsg := string(output)

		if strings.Contains(errorMsg, "application default credentials are not available") ||
			strings.Contains(errorMsg, "Could not automatically determine credentials") {
			status.Authenticated = false
			status.Message = "No GCP credentials found"
			status.Error = "GCP application default credentials not configured. Run 'gcloud auth application-default login'."
			return status
		}

		// Generic error
		status.Authenticated = false
		status.Message = "GCP CLI error"
		status.Error = fmt.Sprintf("Failed to check GCP authentication: %s", strings.TrimSpace(errorMsg))
		return status
	}

	// If we got a token, authentication is valid
	token := strings.TrimSpace(string(output))
	if len(token) > 0 {
		// Get active account info
		accountCmd := exec.CommandContext(ctx, "gcloud", "config", "get-value", "account")
		if accountOutput, err := accountCmd.Output(); err == nil {
			account := strings.TrimSpace(string(accountOutput))
			if account != "" {
				status.Details = fmt.Sprintf("Account: %s", account)
			}
		}

		status.Authenticated = true
		status.Message = "Authenticated to GCP"
		return status
	}

	status.Authenticated = false
	status.Message = "GCP authentication check failed"
	status.Error = "Failed to retrieve GCP access token"
	return status
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// FormatBackendAuthStatus formats authentication status for display
func FormatBackendAuthStatus(status *BackendAuthStatus) string {
	// ANSI color codes
	green := "\033[32m"
	red := "\033[31m"
	yellow := "\033[33m"
	reset := "\033[0m"

	var sb strings.Builder

	if status.Authenticated {
		sb.WriteString(fmt.Sprintf("%s✓%s %s", green, reset, status.Message))
		if status.Details != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", status.Details))
		}
	} else {
		sb.WriteString(fmt.Sprintf("%s✗%s %s", red, reset, status.Message))
		if status.Error != "" {
			sb.WriteString(fmt.Sprintf("\n  %sError:%s %s", yellow, reset, status.Error))
		}
	}

	return sb.String()
}

// GetAuthenticationCommand returns the command to authenticate for a given backend
func GetAuthenticationCommand(backendType string) string {
	switch strings.ToLower(backendType) {
	case "azurerm":
		return "az login"
	case "s3":
		return "aws sso login --profile <profile>  # or: aws configure"
	case "gcs":
		return "gcloud auth application-default login"
	default:
		return ""
	}
}
