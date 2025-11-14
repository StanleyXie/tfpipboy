package auth

import (
	"os/exec"
	"testing"
	"time"
)

func TestGitHubChecker_Name(t *testing.T) {
	checker := NewGitHubChecker()
	if checker.Name() != "github" {
		t.Errorf("Expected name 'github', got '%s'", checker.Name())
	}
}

func TestGitHubChecker_Check_NotInstalled(t *testing.T) {
	// This test assumes 'gh' command might not be available
	checker := NewGitHubChecker()
	status, err := checker.Check()

	if err != nil {
		// If gh CLI is not installed, we should get an error
		if _, ok := err.(*exec.Error); !ok {
			t.Errorf("Expected exec.Error when gh CLI not available, got %T", err)
		}
	}

	// Status should be returned even on error
	if status == nil {
		t.Error("Expected status to be returned even on error")
	}

	if status != nil {
		if status.Provider != "github" {
			t.Errorf("Expected provider 'github', got '%s'", status.Provider)
		}
		if status.CheckedAt.IsZero() {
			t.Error("Expected CheckedAt to be set")
		}
	}
}

func TestGitHubChecker_Check_Cache(t *testing.T) {
	checker := NewGitHubChecker()

	// First check
	status1, err1 := checker.Check()
	if err1 != nil {
		t.Skip("Skipping cache test: gh CLI not available or not logged in")
	}

	if status1 == nil {
		t.Fatal("Expected status from first check")
	}

	time1 := status1.CheckedAt

	// Immediate second check should return cached result
	status2, err2 := checker.Check()
	if err2 != nil {
		t.Fatalf("Expected successful cached check, got error: %v", err2)
	}

	if status2 == nil {
		t.Fatal("Expected status from cached check")
	}

	time2 := status2.CheckedAt

	// Times should be identical (from cache)
	if !time1.Equal(time2) {
		t.Errorf("Expected cached result with same timestamp. First: %v, Second: %v", time1, time2)
	}
}

func TestGitHubChecker_Check_CacheExpiration(t *testing.T) {
	// Create checker with short TTL for testing
	checker := &GitHubChecker{
		cache: NewCache(100 * time.Millisecond), // 100ms TTL
	}

	// First check
	status1, err1 := checker.Check()
	if err1 != nil {
		t.Skip("Skipping cache expiration test: gh CLI not available or not logged in")
	}

	if status1 == nil {
		t.Fatal("Expected status from first check")
	}

	time1 := status1.CheckedAt

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Second check should fetch fresh data
	status2, err2 := checker.Check()
	if err2 != nil {
		t.Fatalf("Expected successful check after cache expiration, got error: %v", err2)
	}

	if status2 == nil {
		t.Fatal("Expected status from second check")
	}

	time2 := status2.CheckedAt

	// Times should be different (cache expired, new check performed)
	if time1.Equal(time2) {
		t.Error("Expected fresh check with different timestamp after cache expiration")
	}
}

func TestGitHubChecker_ParseOutput_Authenticated(t *testing.T) {
	// Test parsing of successful authentication output
	// Note: This is an example output format, actual testing happens with real gh CLI
	_ = `✓ Logged in to github.com as testuser (keyring)
✓ Git operations for github.com configured to use https protocol.
✓ Token: *******************`

	checker := NewGitHubChecker()
	status, _ := checker.Check()

	// We can't test the actual parsing without mocking exec.Command,
	// but we can verify the structure is correct
	if status == nil {
		t.Skip("Skipping parse test: gh CLI not available")
	}

	// Verify status has required fields
	if status.Provider != "github" {
		t.Errorf("Expected provider 'github', got '%s'", status.Provider)
	}

	if status.Details == nil {
		t.Error("Expected Details map to be initialized")
	}

	// Only check user if authenticated (may not be logged in during test)
	if status.Authenticated {
		if status.User == "" {
			t.Error("Expected User to be set when authenticated")
		}
	}
}

func TestGitHubChecker_ParseOutput_NotAuthenticated(t *testing.T) {
	// Test that not-logged-in status is handled correctly
	checker := NewGitHubChecker()
	status, _ := checker.Check()

	if status == nil {
		t.Skip("Skipping test: gh CLI not available")
	}

	// Verify error details are captured when not authenticated
	if !status.Authenticated {
		if status.Details["error"] == "" {
			t.Error("Expected error details when not authenticated")
		}
	}
}
