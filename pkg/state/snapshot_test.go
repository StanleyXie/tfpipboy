package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotManager_Capture(t *testing.T) {
	// Create temporary directory for test storage
	tmpDir := t.TempDir()

	// Create test storage
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	sm := NewSnapshotManager(storage)

	// Create a mock tfstate file
	tfstateContent := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "test-lineage-123",
  "outputs": {
    "example_output": {
      "value": "test-value"
    }
  },
  "resources": [
    {
      "mode": "managed",
      "type": "null_resource",
      "name": "test",
      "provider": "provider[\"registry.terraform.io/hashicorp/null\"]",
      "instances": [
        {
          "attributes": {
            "id": "123456",
            "triggers": null
          }
        }
      ]
    }
  ]
}`

	tfstatePath := filepath.Join(tmpDir, "terraform.tfstate")
	if err := os.WriteFile(tfstatePath, []byte(tfstateContent), 0644); err != nil {
		t.Fatalf("Failed to write test tfstate: %v", err)
	}

	// Capture snapshot
	snapshot, err := sm.Capture("test-workspace", tfstatePath)
	if err != nil {
		t.Fatalf("Failed to capture snapshot: %v", err)
	}

	// Verify snapshot
	if snapshot.WorkspaceID != "test-workspace" {
		t.Errorf("Expected workspace ID 'test-workspace', got '%s'", snapshot.WorkspaceID)
	}

	if snapshot.Version != 1 {
		t.Errorf("Expected version 1, got %d", snapshot.Version)
	}

	if snapshot.TerraformVersion != "1.5.0" {
		t.Errorf("Expected terraform version '1.5.0', got '%s'", snapshot.TerraformVersion)
	}

	if len(snapshot.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(snapshot.Resources))
	}

	if len(snapshot.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(snapshot.Outputs))
	}

	// Verify resource details
	resource := snapshot.Resources[0]
	if resource.Type != "null_resource" {
		t.Errorf("Expected resource type 'null_resource', got '%s'", resource.Type)
	}

	if resource.Name != "test" {
		t.Errorf("Expected resource name 'test', got '%s'", resource.Name)
	}

	if resource.Address != "null_resource.test" {
		t.Errorf("Expected resource address 'null_resource.test', got '%s'", resource.Address)
	}
}

func TestSnapshotManager_GetLatest(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	sm := NewSnapshotManager(storage)

	// Create mock tfstate
	tfstateContent := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "test-lineage",
  "outputs": {},
  "resources": []
}`

	tfstatePath := filepath.Join(tmpDir, "terraform.tfstate")
	if err := os.WriteFile(tfstatePath, []byte(tfstateContent), 0644); err != nil {
		t.Fatalf("Failed to write test tfstate: %v", err)
	}

	// Capture multiple snapshots
	snapshot1, err := sm.Capture("test-workspace", tfstatePath)
	if err != nil {
		t.Fatalf("Failed to capture snapshot 1: %v", err)
	}

	snapshot2, err := sm.Capture("test-workspace", tfstatePath)
	if err != nil {
		t.Fatalf("Failed to capture snapshot 2: %v", err)
	}

	// Get latest should return snapshot 2
	latest, err := sm.GetLatest("test-workspace")
	if err != nil {
		t.Fatalf("Failed to get latest snapshot: %v", err)
	}

	if latest.ID != snapshot2.ID {
		t.Errorf("Expected latest snapshot ID '%s', got '%s'", snapshot2.ID, latest.ID)
	}

	if latest.Version != 2 {
		t.Errorf("Expected version 2, got %d", latest.Version)
	}

	// Verify snapshot 1 still exists
	snap1, err := sm.GetByVersion("test-workspace", 1)
	if err != nil {
		t.Fatalf("Failed to get snapshot version 1: %v", err)
	}

	if snap1.ID != snapshot1.ID {
		t.Errorf("Expected snapshot 1 ID '%s', got '%s'", snapshot1.ID, snap1.ID)
	}
}

func TestSnapshotManager_List(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	sm := NewSnapshotManager(storage)

	// Create mock tfstate
	tfstateContent := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "test-lineage",
  "outputs": {},
  "resources": []
}`

	tfstatePath := filepath.Join(tmpDir, "terraform.tfstate")
	if err := os.WriteFile(tfstatePath, []byte(tfstateContent), 0644); err != nil {
		t.Fatalf("Failed to write test tfstate: %v", err)
	}

	// Capture 3 snapshots
	for i := 0; i < 3; i++ {
		_, err := sm.Capture("test-workspace", tfstatePath)
		if err != nil {
			t.Fatalf("Failed to capture snapshot %d: %v", i+1, err)
		}
	}

	// List all snapshots
	snapshots, err := sm.List("test-workspace")
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}

	// Verify versions are in order
	for i, snapshot := range snapshots {
		expectedVersion := i + 1
		if snapshot.Version != expectedVersion {
			t.Errorf("Expected snapshot %d to have version %d, got %d", i, expectedVersion, snapshot.Version)
		}
	}
}
