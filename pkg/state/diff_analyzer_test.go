package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiffAnalyzer_DiffVersions(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	sm := NewSnapshotManager(storage)
	ct := NewChangeTracker(storage)
	da := NewDiffAnalyzer(storage, sm, ct)

	// Create test snapshots
	snapshot1 := &StateSnapshot{
		ID:          "snap-1",
		WorkspaceID: "test-workspace",
		Version:     1,
		Timestamp:   time.Now(),
		Resources: []ResourceState{
			{
				Address: "aws_instance.web",
				Type:    "aws_instance",
				Name:    "web",
				Instances: []ResourceInstance{
					{
						Attributes: map[string]interface{}{
							"id":            "i-123",
							"instance_type": "t2.micro",
						},
					},
				},
			},
		},
		Outputs: map[string]interface{}{
			"instance_ip": "1.2.3.4",
		},
	}

	snapshot2 := &StateSnapshot{
		ID:          "snap-2",
		WorkspaceID: "test-workspace",
		Version:     2,
		Timestamp:   time.Now().Add(5 * time.Minute),
		Resources: []ResourceState{
			{
				Address: "aws_instance.web",
				Type:    "aws_instance",
				Name:    "web",
				Instances: []ResourceInstance{
					{
						Attributes: map[string]interface{}{
							"id":            "i-123",
							"instance_type": "t2.small", // Changed
						},
					},
				},
			},
			{
				Address: "aws_s3_bucket.data", // Added
				Type:    "aws_s3_bucket",
				Name:    "data",
				Instances: []ResourceInstance{
					{
						Attributes: map[string]interface{}{
							"id":     "my-bucket",
							"bucket": "my-data-bucket",
						},
					},
				},
			},
		},
		Outputs: map[string]interface{}{
			"instance_ip": "1.2.3.5", // Changed
			"bucket_name": "my-data-bucket", // Added
		},
	}

	// Save snapshots
	if err := storage.SaveSnapshot(snapshot1); err != nil {
		t.Fatalf("Failed to save snapshot1: %v", err)
	}
	if err := storage.SaveSnapshot(snapshot2); err != nil {
		t.Fatalf("Failed to save snapshot2: %v", err)
	}

	// Diff versions
	diff, err := da.DiffVersions("test-workspace", 1, 2)
	if err != nil {
		t.Fatalf("Failed to diff versions: %v", err)
	}

	// Verify diff
	if diff.FromVersion != 1 || diff.ToVersion != 2 {
		t.Errorf("Expected versions 1→2, got %d→%d", diff.FromVersion, diff.ToVersion)
	}

	// Check summary
	if diff.Summary.TotalResourcesAdded != 1 {
		t.Errorf("Expected 1 resource added, got %d", diff.Summary.TotalResourcesAdded)
	}

	if diff.Summary.TotalResourcesChanged != 1 {
		t.Errorf("Expected 1 resource changed, got %d", diff.Summary.TotalResourcesChanged)
	}

	if diff.Summary.TotalOutputsChanged != 2 {
		t.Errorf("Expected 2 output changes, got %d", diff.Summary.TotalOutputsChanged)
	}

	// Verify resource changes
	foundAdded := false
	foundModified := false

	for _, rc := range diff.ResourceChanges {
		if rc.Address == "aws_s3_bucket.data" && rc.ChangeType == ChangeTypeAdded {
			foundAdded = true
		}
		if rc.Address == "aws_instance.web" && rc.ChangeType == ChangeTypeModified {
			foundModified = true
			if len(rc.AttributeChanges) == 0 {
				t.Error("Expected attribute changes for modified resource")
			}
		}
	}

	if !foundAdded {
		t.Error("Expected to find added resource aws_s3_bucket.data")
	}
	if !foundModified {
		t.Error("Expected to find modified resource aws_instance.web")
	}
}

func TestDiffAnalyzer_GetTimeline(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	sm := NewSnapshotManager(storage)
	ct := NewChangeTracker(storage)
	da := NewDiffAnalyzer(storage, sm, ct)

	// Create tfstate file
	tfstateContent := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "test",
  "outputs": {},
  "resources": [
    {
      "mode": "managed",
      "type": "null_resource",
      "name": "test",
      "provider": "provider[\"registry.terraform.io/hashicorp/null\"]",
      "instances": [{"attributes": {"id": "123"}}]
    }
  ]
}`

	tfstatePath := filepath.Join(tmpDir, "terraform.tfstate")

	// Capture 3 versions
	for i := 1; i <= 3; i++ {
		if err := os.WriteFile(tfstatePath, []byte(tfstateContent), 0644); err != nil {
			t.Fatalf("Failed to write tfstate: %v", err)
		}
		_, err := sm.Capture("test-workspace", tfstatePath)
		if err != nil {
			t.Fatalf("Failed to capture snapshot %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Small delay for distinct timestamps
	}

	// Get timeline
	timeline, err := da.GetTimeline("test-workspace")
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	if len(timeline.Entries) != 3 {
		t.Errorf("Expected 3 timeline entries, got %d", len(timeline.Entries))
	}

	if timeline.FromVersion != 1 {
		t.Errorf("Expected from version 1, got %d", timeline.FromVersion)
	}

	if timeline.ToVersion != 3 {
		t.Errorf("Expected to version 3, got %d", timeline.ToVersion)
	}

	if timeline.Summary.TotalVersions != 3 {
		t.Errorf("Expected 3 total versions, got %d", timeline.Summary.TotalVersions)
	}
}

func TestDiffAnalyzer_GetResourceHistory(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	sm := NewSnapshotManager(storage)
	ct := NewChangeTracker(storage)
	da := NewDiffAnalyzer(storage, sm, ct)

	// Create snapshots with evolving resource
	baseTime := time.Now()

	snapshots := []*StateSnapshot{
		{
			ID:          "snap-1",
			WorkspaceID: "test-workspace",
			Version:     1,
			Timestamp:   baseTime,
			Resources: []ResourceState{
				{
					Address: "aws_instance.web",
					Type:    "aws_instance",
					Name:    "web",
					Instances: []ResourceInstance{
						{Attributes: map[string]interface{}{"id": "i-123", "instance_type": "t2.micro"}},
					},
				},
			},
		},
		{
			ID:          "snap-2",
			WorkspaceID: "test-workspace",
			Version:     2,
			Timestamp:   baseTime.Add(1 * time.Minute),
			Resources: []ResourceState{
				{
					Address: "aws_instance.web",
					Type:    "aws_instance",
					Name:    "web",
					Instances: []ResourceInstance{
						{Attributes: map[string]interface{}{"id": "i-123", "instance_type": "t2.small"}}, // Changed
					},
				},
			},
		},
		{
			ID:          "snap-3",
			WorkspaceID: "test-workspace",
			Version:     3,
			Timestamp:   baseTime.Add(2 * time.Minute),
			Resources: []ResourceState{
				{
					Address: "aws_instance.web",
					Type:    "aws_instance",
					Name:    "web",
					Instances: []ResourceInstance{
						{Attributes: map[string]interface{}{"id": "i-123", "instance_type": "t2.medium"}}, // Changed again
					},
				},
			},
		},
	}

	for _, snap := range snapshots {
		if err := storage.SaveSnapshot(snap); err != nil {
			t.Fatalf("Failed to save snapshot: %v", err)
		}
	}

	// Get resource history
	history, err := da.GetResourceHistory("test-workspace", "aws_instance.web")
	if err != nil {
		t.Fatalf("Failed to get resource history: %v", err)
	}

	if history.ResourceAddress != "aws_instance.web" {
		t.Errorf("Expected address aws_instance.web, got %s", history.ResourceAddress)
	}

	if history.ResourceType != "aws_instance" {
		t.Errorf("Expected type aws_instance, got %s", history.ResourceType)
	}

	if len(history.Versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(history.Versions))
	}

	// Should have 2 changes (v1→v2 and v2→v3)
	if history.TotalChanges != 2 {
		t.Errorf("Expected 2 total changes, got %d", history.TotalChanges)
	}

	// Verify versions have changes
	if len(history.Versions[1].ChangesFromPrev) == 0 {
		t.Error("Expected changes in version 2")
	}
	if len(history.Versions[2].ChangesFromPrev) == 0 {
		t.Error("Expected changes in version 3")
	}
}

func TestFormatVersionDiff(t *testing.T) {
	diff := &VersionDiff{
		WorkspaceID: "test-workspace",
		FromVersion: 1,
		ToVersion:   2,
		FromTimestamp: time.Now(),
		ToTimestamp:   time.Now().Add(5 * time.Minute),
		Duration:      5 * time.Minute,
		Summary: DiffSummary{
			TotalResourcesAdded:   1,
			TotalResourcesRemoved: 1,
			TotalResourcesChanged: 1,
			TotalResourcesUnchanged: 5,
			TotalAttributesChanged: 3,
			SignificantChange:     false,
		},
		ResourceChanges: []ResourceDiff{
			{
				Address:    "aws_instance.new",
				Type:       "aws_instance",
				ChangeType: ChangeTypeAdded,
				Impact:     ImpactMedium,
			},
			{
				Address:    "aws_instance.old",
				Type:       "aws_instance",
				ChangeType: ChangeTypeRemoved,
				Impact:     ImpactHigh,
			},
			{
				Address:    "aws_instance.web",
				Type:       "aws_instance",
				ChangeType: ChangeTypeModified,
				Impact:     ImpactLow,
				AttributeChanges: []AttributeChange{
					{Path: "tags.Name", OldValue: "Old", NewValue: "New"},
				},
			},
		},
	}

	output := FormatVersionDiff(diff)

	// Check that output contains key elements
	if len(output) == 0 {
		t.Error("Expected non-empty formatted output")
	}

	// Should contain version info
	if !contains(output, "v1") || !contains(output, "v2") {
		t.Error("Output should contain version numbers")
	}

	// Should contain summary
	if !contains(output, "Summary") {
		t.Error("Output should contain summary section")
	}

	// Should contain resource types
	if !contains(output, "Added") || !contains(output, "Removed") || !contains(output, "Modified") {
		t.Error("Output should contain change type sections")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
