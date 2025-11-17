package state

import (
	"testing"
	"time"
)

func TestChangeTracker_TrackChanges(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ct := NewChangeTracker(storage)

	// Create previous snapshot
	previous := &StateSnapshot{
		ID:          "snapshot-1",
		WorkspaceID: "test-workspace",
		Timestamp:   time.Now(),
		Version:     1,
		Resources: []ResourceState{
			{
				Address: "null_resource.test1",
				Type:    "null_resource",
				Name:    "test1",
				Instances: []ResourceInstance{
					{
						Attributes: map[string]interface{}{
							"id":       "123",
							"triggers": nil,
						},
					},
				},
			},
			{
				Address: "null_resource.test2",
				Type:    "null_resource",
				Name:    "test2",
				Instances: []ResourceInstance{
					{
						Attributes: map[string]interface{}{
							"id": "456",
						},
					},
				},
			},
		},
	}

	// Save previous snapshot for storage lookup
	if err := storage.SaveSnapshot(previous); err != nil {
		t.Fatalf("Failed to save previous snapshot: %v", err)
	}

	// Create current snapshot with changes
	current := &StateSnapshot{
		ID:          "snapshot-2",
		WorkspaceID: "test-workspace",
		Timestamp:   time.Now(),
		Version:     2,
		Resources: []ResourceState{
			{
				Address: "null_resource.test1",
				Type:    "null_resource",
				Name:    "test1",
				Instances: []ResourceInstance{
					{
						Attributes: map[string]interface{}{
							"id":       "123",
							"triggers": map[string]interface{}{"key": "value"}, // Changed
						},
					},
				},
			},
			// test2 deleted
			{
				Address: "null_resource.test3", // New resource
				Type:    "null_resource",
				Name:    "test3",
				Instances: []ResourceInstance{
					{
						Attributes: map[string]interface{}{
							"id": "789",
						},
					},
				},
			},
		},
	}

	// Save current snapshot
	if err := storage.SaveSnapshot(current); err != nil {
		t.Fatalf("Failed to save current snapshot: %v", err)
	}

	// Track changes
	changes, err := ct.TrackChanges(previous, current)
	if err != nil {
		t.Fatalf("Failed to track changes: %v", err)
	}

	// We expect 3 changes: 1 update, 1 delete, 1 create
	if len(changes) != 3 {
		t.Fatalf("Expected 3 changes, got %d", len(changes))
	}

	// Count operation types
	createCount := 0
	updateCount := 0
	deleteCount := 0

	for _, change := range changes {
		switch change.Operation {
		case OperationCreate:
			createCount++
			if change.ResourceAddress != "null_resource.test3" {
				t.Errorf("Expected create for test3, got %s", change.ResourceAddress)
			}
		case OperationUpdate:
			updateCount++
			if change.ResourceAddress != "null_resource.test1" {
				t.Errorf("Expected update for test1, got %s", change.ResourceAddress)
			}
		case OperationDelete:
			deleteCount++
			if change.ResourceAddress != "null_resource.test2" {
				t.Errorf("Expected delete for test2, got %s", change.ResourceAddress)
			}
		}
	}

	if createCount != 1 {
		t.Errorf("Expected 1 create, got %d", createCount)
	}
	if updateCount != 1 {
		t.Errorf("Expected 1 update, got %d", updateCount)
	}
	if deleteCount != 1 {
		t.Errorf("Expected 1 delete, got %d", deleteCount)
	}
}

func TestChangeTracker_GetChangeHistory(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ct := NewChangeTracker(storage)

	// Create and save snapshots
	snapshot1 := &StateSnapshot{
		ID:          "snapshot-1",
		WorkspaceID: "test-workspace",
		Timestamp:   time.Now(),
		Version:     1,
		Resources:   []ResourceState{},
	}

	snapshot2 := &StateSnapshot{
		ID:          "snapshot-2",
		WorkspaceID: "test-workspace",
		Timestamp:   time.Now(),
		Version:     2,
		Resources: []ResourceState{
			{
				Address: "null_resource.test",
				Type:    "null_resource",
				Name:    "test",
				Instances: []ResourceInstance{
					{
						Attributes: map[string]interface{}{"id": "123"},
					},
				},
			},
		},
	}

	if err := storage.SaveSnapshot(snapshot1); err != nil {
		t.Fatalf("Failed to save snapshot1: %v", err)
	}
	if err := storage.SaveSnapshot(snapshot2); err != nil {
		t.Fatalf("Failed to save snapshot2: %v", err)
	}

	// Track changes
	_, err = ct.TrackChanges(snapshot1, snapshot2)
	if err != nil {
		t.Fatalf("Failed to track changes: %v", err)
	}

	// Get change history
	history, err := ct.GetChangeHistory("test-workspace")
	if err != nil {
		t.Fatalf("Failed to get change history: %v", err)
	}

	if len(history.Events) != 1 {
		t.Errorf("Expected 1 change event, got %d", len(history.Events))
	}

	if history.WorkspaceID != "test-workspace" {
		t.Errorf("Expected workspace ID 'test-workspace', got '%s'", history.WorkspaceID)
	}
}
