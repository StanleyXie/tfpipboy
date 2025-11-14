package auth

import (
	"os/exec"
	"testing"
	"time"
)

func TestAzureChecker_Name(t *testing.T) {
	checker := NewAzureChecker()
	if checker.Name() != "azure" {
		t.Errorf("Expected name 'azure', got '%s'", checker.Name())
	}
}

func TestAzureChecker_Check_NotInstalled(t *testing.T) {
	// This test assumes 'az' command might not be available
	checker := NewAzureChecker()
	status, err := checker.Check()

	if err != nil {
		// If az CLI is not installed, we should get an error
		if _, ok := err.(*exec.Error); !ok {
			t.Errorf("Expected exec.Error when az CLI not available, got %T", err)
		}
	}

	// Status should be returned even on error
	if status == nil {
		t.Error("Expected status to be returned even on error")
	}

	if status != nil {
		if status.Provider != "azure" {
			t.Errorf("Expected provider 'azure', got '%s'", status.Provider)
		}
		if status.CheckedAt.IsZero() {
			t.Error("Expected CheckedAt to be set")
		}
	}
}

func TestAzureChecker_Check_Cache(t *testing.T) {
	checker := NewAzureChecker()

	// First check
	status1, err1 := checker.Check()
	if err1 != nil {
		t.Skip("Skipping cache test: az CLI not available or not logged in")
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

func TestAzureChecker_Check_CacheExpiration(t *testing.T) {
	// Create checker with short TTL for testing
	checker := &AzureChecker{
		cache: NewCache(100 * time.Millisecond), // 100ms TTL
	}

	// First check
	status1, err1 := checker.Check()
	if err1 != nil {
		t.Skip("Skipping cache expiration test: az CLI not available or not logged in")
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

func TestCache_GetSet(t *testing.T) {
	cache := NewCache(time.Second)

	// Get from empty cache
	if cached := cache.Get(); cached != nil {
		t.Error("Expected nil from empty cache")
	}

	// Set and get
	status := &Status{
		Provider:      "test",
		Authenticated: true,
		CheckedAt:     time.Now(),
	}

	cache.Set(status)

	cached := cache.Get()
	if cached == nil {
		t.Fatal("Expected cached status")
	}

	if cached.Provider != "test" {
		t.Errorf("Expected provider 'test', got '%s'", cached.Provider)
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)

	status := &Status{
		Provider:      "test",
		Authenticated: true,
		CheckedAt:     time.Now(),
	}

	cache.Set(status)

	// Should be cached immediately
	if cached := cache.Get(); cached == nil {
		t.Error("Expected cached status immediately after set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	if cached := cache.Get(); cached != nil {
		t.Error("Expected cache to be expired")
	}
}
