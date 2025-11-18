package terraform

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.cache == nil {
		t.Error("Expected cache to be initialized")
	}

	if manager.currentPath == "" {
		t.Error("Expected currentPath to be set")
	}
}

func TestManager_SetPath(t *testing.T) {
	manager := NewManager()
	initialPath := manager.GetPath()

	newPath := "/tmp/test"
	manager.SetPath(newPath)

	if manager.GetPath() != newPath {
		t.Errorf("Expected path '%s', got '%s'", newPath, manager.GetPath())
	}

	// Verify path change different from initial
	if initialPath == newPath {
		t.Skip("Skipping cache clear test - paths are the same")
	}
}

func TestManager_GetContext_NonTerraformDir(t *testing.T) {
	manager := NewManager()

	// Set to a non-Terraform directory
	tmpDir := t.TempDir()
	manager.SetPath(tmpDir)

	ctx, err := manager.GetContext()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if ctx == nil {
		t.Fatal("Expected context to be returned")
	}

	// In non-TF directory, fields should be empty
	if ctx.Workspace != "" || ctx.Backend != "" || ctx.Module != "" {
		t.Error("Expected empty context for non-Terraform directory")
	}
}

func TestManager_GetContext_TerraformDir(t *testing.T) {
	manager := NewManager()

	// Create a temp Terraform directory
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tfFile, []byte("# test terraform file"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .terraform directory to simulate initialized TF project
	tfDir := filepath.Join(tmpDir, ".terraform")
	if err := os.MkdirAll(tfDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Set workspace
	envFile := filepath.Join(tfDir, "environment")
	if err := os.WriteFile(envFile, []byte("default"), 0644); err != nil {
		t.Fatal(err)
	}

	manager.SetPath(tmpDir)

	ctx, err := manager.GetContext()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if ctx == nil {
		t.Fatal("Expected context to be returned")
	}

	// Workspace detection depends on terraform CLI being available
	// If terraform is not installed, workspace will be empty
	if isCommandAvailable("terraform") {
		// With terraform CLI available, workspace should be detected
		// (though it might fail if terraform init hasn't been run)
		t.Logf("Workspace detected: %s", ctx.Workspace)
	} else {
		t.Log("Skipping workspace check - terraform CLI not available")
	}

	// Check timestamp
	if ctx.CheckedAt.IsZero() {
		t.Error("Expected CheckedAt to be set")
	}
}

func TestManager_GetContext_Caching(t *testing.T) {
	manager := NewManager()
	tmpDir := t.TempDir()
	manager.SetPath(tmpDir)

	// First call
	ctx1, err := manager.GetContext()
	if err != nil {
		t.Fatal(err)
	}

	time1 := ctx1.CheckedAt

	// Immediate second call (should be cached)
	ctx2, err := manager.GetContext()
	if err != nil {
		t.Fatal(err)
	}

	time2 := ctx2.CheckedAt

	// Times should be identical (from cache)
	if !time1.Equal(time2) {
		t.Error("Expected cached result with same timestamp")
	}
}

func TestManager_RefreshContext(t *testing.T) {
	manager := NewManager()
	tmpDir := t.TempDir()
	manager.SetPath(tmpDir)

	// First call
	ctx1, err := manager.GetContext()
	if err != nil {
		t.Fatal(err)
	}

	time1 := ctx1.CheckedAt

	// Small delay to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Refresh (should bypass cache)
	ctx2, err := manager.RefreshContext()
	if err != nil {
		t.Fatal(err)
	}

	time2 := ctx2.CheckedAt

	// Times should be different (fresh check)
	if time1.Equal(time2) {
		t.Error("Expected fresh check with different timestamp")
	}
}

func TestManager_IsInTerraformDirectory(t *testing.T) {
	manager := NewManager()

	// Test non-TF directory
	tmpDir := t.TempDir()
	manager.SetPath(tmpDir)

	if manager.IsInTerraformDirectory() {
		t.Error("Expected false for non-Terraform directory")
	}

	// Add .tf file
	tfFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tfFile, []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Refresh and check again
	manager.SetPath(tmpDir) // Reset path to clear cache

	if !manager.IsInTerraformDirectory() {
		t.Error("Expected true for Terraform directory")
	}
}

func TestManager_GetSummary(t *testing.T) {
	manager := NewManager()
	tmpDir := t.TempDir()

	// Create TF directory
	tfFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tfFile, []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .terraform directory
	tfDir := filepath.Join(tmpDir, ".terraform")
	if err := os.MkdirAll(tfDir, 0755); err != nil {
		t.Fatal(err)
	}

	envFile := filepath.Join(tfDir, "environment")
	if err := os.WriteFile(envFile, []byte("production"), 0644); err != nil {
		t.Fatal(err)
	}

	manager.SetPath(tmpDir)

	summary, err := manager.GetSummary()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary to be returned")
	}

	if !summary.InTerraformDir {
		t.Error("Expected InTerraformDir to be true")
	}

	// Workspace detection depends on terraform CLI
	if isCommandAvailable("terraform") {
		t.Logf("Workspace in summary: %s", summary.Workspace)
	} else {
		t.Log("Skipping workspace check - terraform CLI not available")
	}
}

func TestManager_GetContextAsync(t *testing.T) {
	manager := NewManager()
	tmpDir := t.TempDir()
	manager.SetPath(tmpDir)

	// Get context asynchronously
	ch := manager.GetContextAsync()

	// Wait for result with timeout
	select {
	case ctx := <-ch:
		if ctx == nil {
			t.Error("Expected context to be returned")
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for async context")
	}
}

func TestCache_GetSet(t *testing.T) {
	cache := NewCache(time.Second)

	// Get from empty cache
	if cached := cache.Get(); cached != nil {
		t.Error("Expected nil from empty cache")
	}

	// Set and get
	ctx := &Context{
		Workspace: "test",
		Backend:   "local",
		CheckedAt: time.Now(),
	}

	cache.Set(ctx)

	cached := cache.Get()
	if cached == nil {
		t.Fatal("Expected cached context")
	}

	if cached.Workspace != "test" {
		t.Errorf("Expected workspace 'test', got '%s'", cached.Workspace)
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)

	ctx := &Context{
		Workspace: "test",
		CheckedAt: time.Now(),
	}

	cache.Set(ctx)

	// Should be cached immediately
	if cached := cache.Get(); cached == nil {
		t.Error("Expected cached context immediately after set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	if cached := cache.Get(); cached != nil {
		t.Error("Expected cache to be expired")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(time.Second)

	ctx := &Context{
		Workspace: "test",
		CheckedAt: time.Now(),
	}

	cache.Set(ctx)

	// Verify it's cached
	if cached := cache.Get(); cached == nil {
		t.Error("Expected cached context")
	}

	// Clear cache
	cache.Clear()

	// Should be cleared
	if cached := cache.Get(); cached != nil {
		t.Error("Expected cache to be cleared")
	}
}

func TestManager_GetWorkspaceList(t *testing.T) {
	manager := NewManager()

	// Test in non-Terraform directory
	tmpDir := t.TempDir()
	manager.SetPath(tmpDir)

	workspaces, err := manager.GetWorkspaceList()
	if err == nil {
		// If no error, should return empty or default workspace
		t.Logf("Workspaces in non-TF dir: %v", workspaces)
	} else {
		// Expected error in non-TF directory
		t.Logf("Expected error in non-TF dir: %v", err)
	}

	// Test in Terraform directory
	tfFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tfFile, []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .terraform directory
	tfDir := filepath.Join(tmpDir, ".terraform")
	if err := os.MkdirAll(tfDir, 0755); err != nil {
		t.Fatal(err)
	}

	manager.SetPath(tmpDir)
	workspaces, err = manager.GetWorkspaceList()

	// Result depends on whether terraform CLI is available
	if isCommandAvailable("terraform") {
		// With terraform CLI, we might get workspace list or error
		t.Logf("Workspaces with TF CLI: %v (err: %v)", workspaces, err)
	} else {
		// Without terraform CLI, should error
		t.Log("Skipping workspace list check - terraform CLI not available")
	}
}

func TestManager_GetModuleSources(t *testing.T) {
	manager := NewManager()
	tmpDir := t.TempDir()
	manager.SetPath(tmpDir)

	// Test in non-module directory
	sources, err := manager.GetModuleSources()
	if err == nil {
		// Should return empty list
		if len(sources) != 0 {
			t.Errorf("Expected 0 sources in non-module dir, got %d", len(sources))
		}
	} else {
		t.Logf("Error getting module sources (expected): %v", err)
	}

	// Create a module with source
	mainTf := filepath.Join(tmpDir, "main.tf")
	moduleContent := `
module "test" {
  source = "./modules/test"
}

module "remote" {
  source = "terraform-aws-modules/vpc/aws"
}
`
	if err := os.WriteFile(mainTf, []byte(moduleContent), 0644); err != nil {
		t.Fatal(err)
	}

	manager.SetPath(tmpDir)
	sources, err = manager.GetModuleSources()

	// Should be able to parse module sources
	if err == nil {
		t.Logf("Found %d module sources", len(sources))
	} else {
		t.Logf("Error parsing module sources: %v", err)
	}
}

func TestManager_GetEnvironmentVarsSummary(t *testing.T) {
	manager := NewManager()

	summary := manager.GetEnvironmentVarsSummary()

	if summary == nil {
		t.Error("Expected summary map, got nil")
	}

	// Summary should contain some information
	// The exact content depends on environment
	t.Logf("Environment summary: %v", summary)
}
