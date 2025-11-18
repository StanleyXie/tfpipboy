package orchestrator

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestValidateBackendAuth_Local(t *testing.T) {
	ctx := context.Background()
	status := ValidateBackendAuth(ctx, "local")

	if !status.Authenticated {
		t.Error("Expected local backend to be authenticated")
	}
	if status.BackendType != "local" {
		t.Errorf("Expected BackendType 'local', got '%s'", status.BackendType)
	}
	if !strings.Contains(status.Message, "Local backend") {
		t.Errorf("Expected message about local backend, got '%s'", status.Message)
	}
}

func TestValidateBackendAuth_Unknown(t *testing.T) {
	ctx := context.Background()
	status := ValidateBackendAuth(ctx, "unknown-backend")

	if !status.Authenticated {
		t.Error("Expected unknown backend to default to authenticated")
	}
	if !strings.Contains(status.Message, "unknown-backend") {
		t.Errorf("Expected message to mention backend type, got '%s'", status.Message)
	}
}

func TestValidateBackendAuth_Timeout(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	// Test with Azure (will timeout due to expired context)
	status := ValidateBackendAuth(ctx, "azurerm")

	// Should handle timeout gracefully
	if status == nil {
		t.Fatal("Expected status, got nil")
	}
	if status.BackendType != "azurerm" {
		t.Errorf("Expected BackendType 'azurerm', got '%s'", status.BackendType)
	}
}

func TestCheckAzureAuth_CLINotInstalled(t *testing.T) {
	// Test when az CLI is not available
	// This test will pass if az is not installed, or we can mock it
	ctx := context.Background()
	status := checkAzureAuth(ctx)

	if status.BackendType != "azurerm" {
		t.Errorf("Expected BackendType 'azurerm', got '%s'", status.BackendType)
	}

	// Status will depend on whether az is actually installed
	// We're just testing that the function doesn't panic
	if status.Authenticated {
		// If authenticated, should have a message
		if status.Message == "" {
			t.Error("Expected message when authenticated")
		}
	} else {
		// If not authenticated, should have an error or message
		if status.Message == "" && status.Error == "" {
			t.Error("Expected message or error when not authenticated")
		}
	}
}

func TestCheckAWSAuth_CLINotInstalled(t *testing.T) {
	// Test when aws CLI is not available
	ctx := context.Background()
	status := checkAWSAuth(ctx)

	if status.BackendType != "s3" {
		t.Errorf("Expected BackendType 's3', got '%s'", status.BackendType)
	}

	// Status will depend on whether aws is actually installed
	// We're just testing that the function doesn't panic
	if status.Authenticated {
		// If authenticated, should have a message
		if status.Message == "" {
			t.Error("Expected message when authenticated")
		}
	} else {
		// If not authenticated, should have an error or message
		if status.Message == "" && status.Error == "" {
			t.Error("Expected message or error when not authenticated")
		}
	}
}

func TestCheckGCPAuth_CLINotInstalled(t *testing.T) {
	// Test when gcloud CLI is not available
	ctx := context.Background()
	status := checkGCPAuth(ctx)

	if status.BackendType != "gcs" {
		t.Errorf("Expected BackendType 'gcs', got '%s'", status.BackendType)
	}

	// Status will depend on whether gcloud is actually installed
	// We're just testing that the function doesn't panic
	if status.Authenticated {
		// If authenticated, should have a message
		if status.Message == "" {
			t.Error("Expected message when authenticated")
		}
	} else {
		// If not authenticated, should have an error or message
		if status.Message == "" && status.Error == "" {
			t.Error("Expected message or error when not authenticated")
		}
	}
}

func TestIsCommandAvailable(t *testing.T) {
	tests := []struct {
		name    string
		command string
		// We don't check expected result as it depends on environment
		// Just testing that the function works
	}{
		{
			name:    "Check common command",
			command: "ls",
		},
		{
			name:    "Check non-existent command",
			command: "definitely-not-a-real-command-xyz123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the function doesn't panic
			_ = isCommandAvailable(tt.command)
		})
	}
}

func TestFormatBackendAuthStatus_Authenticated(t *testing.T) {
	status := &BackendAuthStatus{
		BackendType:   "azurerm",
		Authenticated: true,
		Message:       "Authenticated to Azure",
		Details:       "Subscription: test-sub | User: test@example.com",
	}

	result := FormatBackendAuthStatus(status)

	if !strings.Contains(result, "Authenticated to Azure") {
		t.Errorf("Expected message in output, got '%s'", result)
	}
	if !strings.Contains(result, "test-sub") {
		t.Errorf("Expected details in output, got '%s'", result)
	}
	if !strings.Contains(result, "✓") {
		t.Errorf("Expected checkmark in output, got '%s'", result)
	}
}

func TestFormatBackendAuthStatus_NotAuthenticated(t *testing.T) {
	status := &BackendAuthStatus{
		BackendType:   "s3",
		Authenticated: false,
		Message:       "No AWS credentials found",
		Error:         "AWS credentials not configured",
	}

	result := FormatBackendAuthStatus(status)

	if !strings.Contains(result, "No AWS credentials found") {
		t.Errorf("Expected message in output, got '%s'", result)
	}
	if !strings.Contains(result, "AWS credentials not configured") {
		t.Errorf("Expected error in output, got '%s'", result)
	}
	if !strings.Contains(result, "✗") {
		t.Errorf("Expected X mark in output, got '%s'", result)
	}
}

func TestGetAuthenticationCommand(t *testing.T) {
	tests := []struct {
		name        string
		backendType string
		expected    string
	}{
		{
			name:        "Azure backend",
			backendType: "azurerm",
			expected:    "az login",
		},
		{
			name:        "AWS backend",
			backendType: "s3",
			expected:    "aws",
		},
		{
			name:        "GCP backend",
			backendType: "gcs",
			expected:    "gcloud",
		},
		{
			name:        "Local backend",
			backendType: "local",
			expected:    "",
		},
		{
			name:        "Unknown backend",
			backendType: "unknown",
			expected:    "",
		},
		{
			name:        "Case insensitive - AZURERM",
			backendType: "AZURERM",
			expected:    "az login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAuthenticationCommand(tt.backendType)

			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty command, got '%s'", result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected command to contain '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

func TestBackendAuthStatus_AllBackendTypes(t *testing.T) {
	ctx := context.Background()
	backends := []string{"local", "azurerm", "s3", "gcs", "unknown"}

	for _, backend := range backends {
		t.Run(backend, func(t *testing.T) {
			status := ValidateBackendAuth(ctx, backend)

			if status == nil {
				t.Fatal("Expected status, got nil")
			}

			if status.BackendType != backend {
				t.Errorf("Expected BackendType '%s', got '%s'", backend, status.BackendType)
			}

			if status.Message == "" {
				t.Error("Expected non-empty message")
			}
		})
	}
}

func TestCheckAzureAuth_WithContext(t *testing.T) {
	// Test with a normal context that won't timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := checkAzureAuth(ctx)

	// Basic validation - function should not panic
	if status == nil {
		t.Fatal("Expected status, got nil")
	}

	if status.BackendType != "azurerm" {
		t.Errorf("Expected BackendType 'azurerm', got '%s'", status.BackendType)
	}
}

func TestCheckAWSAuth_WithContext(t *testing.T) {
	// Test with a normal context that won't timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := checkAWSAuth(ctx)

	// Basic validation - function should not panic
	if status == nil {
		t.Fatal("Expected status, got nil")
	}

	if status.BackendType != "s3" {
		t.Errorf("Expected BackendType 's3', got '%s'", status.BackendType)
	}
}

func TestCheckGCPAuth_WithContext(t *testing.T) {
	// Test with a normal context that won't timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := checkGCPAuth(ctx)

	// Basic validation - function should not panic
	if status == nil {
		t.Fatal("Expected status, got nil")
	}

	if status.BackendType != "gcs" {
		t.Errorf("Expected BackendType 'gcs', got '%s'", status.BackendType)
	}
}
