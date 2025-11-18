package orchestrator

import (
	"context"
	"testing"
	"time"
)

func TestCheckAzureAuth(t *testing.T) {
	ctx := context.Background()
	status := CheckAzureAuth(ctx)

	// Basic validation - function should not panic
	if status.Provider != "Azure" {
		t.Errorf("Expected Provider 'Azure', got '%s'", status.Provider)
	}

	// Regardless of authentication status, we should have a valid response
	if status.Authenticated {
		// If authenticated, identity should be set
		if status.Identity == "" {
			t.Error("Expected non-empty identity when authenticated")
		}
	} else {
		// If not authenticated, error should be set
		if status.Error == "" {
			t.Error("Expected error message when not authenticated")
		}
	}
}

func TestCheckAWSAuth(t *testing.T) {
	ctx := context.Background()
	status := CheckAWSAuth(ctx)

	if status.Provider != "AWS" {
		t.Errorf("Expected Provider 'AWS', got '%s'", status.Provider)
	}

	// Regardless of authentication status, we should have a valid response
	if status.Authenticated {
		// If authenticated, identity should be set
		if status.Identity == "" {
			t.Error("Expected non-empty identity when authenticated")
		}
	} else {
		// If not authenticated, error should be set
		if status.Error == "" {
			t.Error("Expected error message when not authenticated")
		}
	}
}

func TestCheckGCPAuth(t *testing.T) {
	ctx := context.Background()
	status := CheckGCPAuth(ctx)

	if status.Provider != "GCP" {
		t.Errorf("Expected Provider 'GCP', got '%s'", status.Provider)
	}

	// Regardless of authentication status, we should have a valid response
	if status.Authenticated {
		// If authenticated, identity should be set
		if status.Identity == "" {
			t.Error("Expected non-empty identity when authenticated")
		}
	} else {
		// If not authenticated, error should be set
		if status.Error == "" {
			t.Error("Expected error message when not authenticated")
		}
	}
}

func TestCheckGitHubAuth(t *testing.T) {
	ctx := context.Background()
	status := CheckGitHubAuth(ctx)

	if status.Provider != "GitHub" {
		t.Errorf("Expected Provider 'GitHub', got '%s'", status.Provider)
	}

	// Accounts slice should always be initialized
	if status.Accounts == nil {
		t.Error("Expected Accounts slice to be initialized")
	}

	// Regardless of authentication status, we should have a valid response
	if status.Authenticated {
		// If authenticated, should have at least one account
		if len(status.Accounts) == 0 {
			t.Error("Expected at least one account when authenticated")
		}
		if status.Identity == "" {
			t.Error("Expected non-empty identity when authenticated")
		}
	} else {
		// If not authenticated, error should be set
		if status.Error == "" {
			t.Error("Expected error message when not authenticated")
		}
	}
}

func TestCheckRequiredAuth_AzureBackend(t *testing.T) {
	ctx := context.Background()
	backends := []string{"azurerm"}

	results := CheckRequiredAuth(ctx, backends)

	// Should check Azure
	if azureStatus, exists := results["azure"]; !exists {
		t.Error("Expected Azure auth to be checked")
	} else {
		if azureStatus.Provider != "Azure" {
			t.Errorf("Expected Provider 'Azure', got '%s'", azureStatus.Provider)
		}
	}

	// Should also check GitHub (always checked)
	if _, exists := results["github"]; !exists {
		t.Error("Expected GitHub auth to be checked")
	}
}

func TestCheckRequiredAuth_AWSBackend(t *testing.T) {
	ctx := context.Background()
	backends := []string{"s3"}

	results := CheckRequiredAuth(ctx, backends)

	// Should check AWS
	if awsStatus, exists := results["aws"]; !exists {
		t.Error("Expected AWS auth to be checked")
	} else {
		if awsStatus.Provider != "AWS" {
			t.Errorf("Expected Provider 'AWS', got '%s'", awsStatus.Provider)
		}
	}

	// Should also check GitHub (always checked)
	if _, exists := results["github"]; !exists {
		t.Error("Expected GitHub auth to be checked")
	}

	// Should NOT check Azure or GCP
	if _, exists := results["azure"]; exists {
		t.Error("Did not expect Azure auth to be checked")
	}
	if _, exists := results["gcp"]; exists {
		t.Error("Did not expect GCP auth to be checked")
	}
}

func TestCheckRequiredAuth_GCSBackend(t *testing.T) {
	ctx := context.Background()
	backends := []string{"gcs"}

	results := CheckRequiredAuth(ctx, backends)

	// Should check GCP
	if gcpStatus, exists := results["gcp"]; !exists {
		t.Error("Expected GCP auth to be checked")
	} else {
		if gcpStatus.Provider != "GCP" {
			t.Errorf("Expected Provider 'GCP', got '%s'", gcpStatus.Provider)
		}
	}

	// Should also check GitHub (always checked)
	if _, exists := results["github"]; !exists {
		t.Error("Expected GitHub auth to be checked")
	}
}

func TestCheckRequiredAuth_MultipleBackends(t *testing.T) {
	ctx := context.Background()
	backends := []string{"azurerm", "s3", "gcs"}

	results := CheckRequiredAuth(ctx, backends)

	// Should check all three cloud providers
	if _, exists := results["azure"]; !exists {
		t.Error("Expected Azure auth to be checked")
	}
	if _, exists := results["aws"]; !exists {
		t.Error("Expected AWS auth to be checked")
	}
	if _, exists := results["gcp"]; !exists {
		t.Error("Expected GCP auth to be checked")
	}
	if _, exists := results["github"]; !exists {
		t.Error("Expected GitHub auth to be checked")
	}
}

func TestCheckRequiredAuth_LocalBackend(t *testing.T) {
	ctx := context.Background()
	backends := []string{"local"}

	results := CheckRequiredAuth(ctx, backends)

	// Should only check GitHub (always checked)
	if _, exists := results["github"]; !exists {
		t.Error("Expected GitHub auth to be checked")
	}

	// Should NOT check cloud providers
	if _, exists := results["azure"]; exists {
		t.Error("Did not expect Azure auth to be checked for local backend")
	}
	if _, exists := results["aws"]; exists {
		t.Error("Did not expect AWS auth to be checked for local backend")
	}
	if _, exists := results["gcp"]; exists {
		t.Error("Did not expect GCP auth to be checked for local backend")
	}
}

func TestCheckRequiredAuth_EmptyBackends(t *testing.T) {
	ctx := context.Background()
	backends := []string{}

	results := CheckRequiredAuth(ctx, backends)

	// Should only check GitHub (always checked)
	if _, exists := results["github"]; !exists {
		t.Error("Expected GitHub auth to be checked even with no backends")
	}

	// Should NOT check cloud providers
	if _, exists := results["azure"]; exists {
		t.Error("Did not expect Azure auth to be checked with no backends")
	}
	if _, exists := results["aws"]; exists {
		t.Error("Did not expect AWS auth to be checked with no backends")
	}
	if _, exists := results["gcp"]; exists {
		t.Error("Did not expect GCP auth to be checked with no backends")
	}
}

func TestCheckAzureAuth_WithTimeout(t *testing.T) {
	// Test with a context that has already expired
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	status := CheckAzureAuth(ctx)

	// Should handle timeout gracefully
	if status.Provider != "Azure" {
		t.Errorf("Expected Provider 'Azure', got '%s'", status.Provider)
	}
}

func TestCheckAWSAuth_WithTimeout(t *testing.T) {
	// Test with a context that has already expired
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	status := CheckAWSAuth(ctx)

	// Should handle timeout gracefully
	if status.Provider != "AWS" {
		t.Errorf("Expected Provider 'AWS', got '%s'", status.Provider)
	}
}

func TestCheckGCPAuth_WithTimeout(t *testing.T) {
	// Test with a context that has already expired
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	status := CheckGCPAuth(ctx)

	// Should handle timeout gracefully
	if status.Provider != "GCP" {
		t.Errorf("Expected Provider 'GCP', got '%s'", status.Provider)
	}
}

func TestCheckGitHubAuth_WithTimeout(t *testing.T) {
	// Test with a context that has already expired
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	status := CheckGitHubAuth(ctx)

	// Should handle timeout gracefully
	if status.Provider != "GitHub" {
		t.Errorf("Expected Provider 'GitHub', got '%s'", status.Provider)
	}

	// Accounts should still be initialized
	if status.Accounts == nil {
		t.Error("Expected Accounts slice to be initialized even with timeout")
	}
}

func TestAuthStatus_Fields(t *testing.T) {
	// Test that AuthStatus struct can be created with all fields
	status := AuthStatus{
		Provider:      "TestProvider",
		Authenticated: true,
		Identity:      "test-user",
		Token:         "test-token",
		Accounts:      []string{"account1", "account2"},
		ActiveAccount: "account1",
		Error:         "",
	}

	if status.Provider != "TestProvider" {
		t.Errorf("Expected Provider 'TestProvider', got '%s'", status.Provider)
	}
	if !status.Authenticated {
		t.Error("Expected Authenticated to be true")
	}
	if status.Identity != "test-user" {
		t.Errorf("Expected Identity 'test-user', got '%s'", status.Identity)
	}
	if status.Token != "test-token" {
		t.Errorf("Expected Token 'test-token', got '%s'", status.Token)
	}
	if len(status.Accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(status.Accounts))
	}
	if status.ActiveAccount != "account1" {
		t.Errorf("Expected ActiveAccount 'account1', got '%s'", status.ActiveAccount)
	}
	if status.Error != "" {
		t.Errorf("Expected no error, got '%s'", status.Error)
	}
}
