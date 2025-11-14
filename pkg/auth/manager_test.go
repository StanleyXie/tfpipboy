package auth

import (
	"errors"
	"testing"
	"time"
)

// Mock checker for testing
type mockChecker struct {
	name        string
	status      *Status
	err         error
	checkDelay  time.Duration
	checkCalled bool
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Check() (*Status, error) {
	m.checkCalled = true
	if m.checkDelay > 0 {
		time.Sleep(m.checkDelay)
	}
	return m.status, m.err
}

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	// Should have Azure and GitHub checkers by default
	if len(manager.checkers) != 2 {
		t.Errorf("Expected 2 default checkers, got %d", len(manager.checkers))
	}

	// Verify checker names
	names := make(map[string]bool)
	for _, checker := range manager.checkers {
		names[checker.Name()] = true
	}

	if !names["azure"] {
		t.Error("Expected Azure checker to be registered")
	}

	if !names["github"] {
		t.Error("Expected GitHub checker to be registered")
	}
}

func TestManager_AddChecker(t *testing.T) {
	manager := &Manager{}

	mock := &mockChecker{
		name: "test",
		status: &Status{
			Provider:      "test",
			Authenticated: true,
		},
	}

	manager.AddChecker(mock)

	if len(manager.checkers) != 1 {
		t.Errorf("Expected 1 checker, got %d", len(manager.checkers))
	}

	if manager.checkers[0].Name() != "test" {
		t.Errorf("Expected checker name 'test', got '%s'", manager.checkers[0].Name())
	}
}

func TestManager_CheckAll_Success(t *testing.T) {
	manager := &Manager{}

	// Add mock checkers
	mock1 := &mockChecker{
		name: "provider1",
		status: &Status{
			Provider:      "provider1",
			Authenticated: true,
			User:          "user1",
		},
	}

	mock2 := &mockChecker{
		name: "provider2",
		status: &Status{
			Provider:      "provider2",
			Authenticated: false,
		},
	}

	manager.AddChecker(mock1)
	manager.AddChecker(mock2)

	// Check all
	results := manager.CheckAll()

	// Verify results
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if status := results["provider1"]; status == nil {
		t.Error("Expected status for provider1")
	} else {
		if !status.Authenticated {
			t.Error("Expected provider1 to be authenticated")
		}
		if status.User != "user1" {
			t.Errorf("Expected user 'user1', got '%s'", status.User)
		}
	}

	if status := results["provider2"]; status == nil {
		t.Error("Expected status for provider2")
	} else {
		if status.Authenticated {
			t.Error("Expected provider2 to not be authenticated")
		}
	}

	// Verify both checkers were called
	if !mock1.checkCalled {
		t.Error("Expected mock1.Check() to be called")
	}
	if !mock2.checkCalled {
		t.Error("Expected mock2.Check() to be called")
	}
}

func TestManager_CheckAll_WithErrors(t *testing.T) {
	manager := &Manager{}

	// Add mock checker that returns error
	mockError := &mockChecker{
		name:   "error-provider",
		status: nil,
		err:    errors.New("check failed"),
	}

	// Add successful mock checker
	mockSuccess := &mockChecker{
		name: "success-provider",
		status: &Status{
			Provider:      "success-provider",
			Authenticated: true,
		},
	}

	manager.AddChecker(mockError)
	manager.AddChecker(mockSuccess)

	// Check all
	results := manager.CheckAll()

	// Should only get successful result
	if len(results) != 1 {
		t.Errorf("Expected 1 result (error cases filtered out), got %d", len(results))
	}

	if status := results["success-provider"]; status == nil {
		t.Error("Expected status for success-provider")
	}

	if _, exists := results["error-provider"]; exists {
		t.Error("Expected error-provider to be filtered out")
	}
}

func TestManager_CheckAll_Concurrent(t *testing.T) {
	manager := &Manager{}

	// Add slow mock checkers to verify concurrent execution
	mock1 := &mockChecker{
		name:       "slow1",
		checkDelay: 50 * time.Millisecond,
		status: &Status{
			Provider:      "slow1",
			Authenticated: true,
		},
	}

	mock2 := &mockChecker{
		name:       "slow2",
		checkDelay: 50 * time.Millisecond,
		status: &Status{
			Provider:      "slow2",
			Authenticated: true,
		},
	}

	manager.AddChecker(mock1)
	manager.AddChecker(mock2)

	// Measure execution time
	start := time.Now()
	results := manager.CheckAll()
	duration := time.Since(start)

	// If concurrent, should complete in ~50ms
	// If sequential, would take ~100ms
	if duration > 80*time.Millisecond {
		t.Errorf("Expected concurrent execution (~50ms), took %v", duration)
	}

	// Verify both results exist
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestManager_CheckAll_EmptyManager(t *testing.T) {
	manager := &Manager{}

	results := manager.CheckAll()

	if results == nil {
		t.Error("Expected empty map, got nil")
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}
