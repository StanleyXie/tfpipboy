package state

import (
	"testing"
)

func TestDriftDetector_DetectDrift(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	sm := NewSnapshotManager(storage)
	dd := NewDriftDetector(storage, sm)

	// Mock plan output with drift
	planOutput := `
Terraform will perform the following actions:

  # null_resource.test will be updated in-place
  ~ resource "null_resource" "test" {
      ~ id       = "old-id" -> "new-id"
      ~ triggers = {
          - "old_key" = "old_value"
          + "new_key" = "new_value"
        }
    }

  # aws_instance.server has been deleted outside Terraform
  - resource "aws_instance" "server" {
      - ami           = "ami-12345"
      - instance_type = "t2.micro"
      - id            = "i-1234567890"
    }

Plan: 0 to add, 1 to change, 1 to destroy.

Objects have changed outside of Terraform
`

	// Detect drift
	report, err := dd.DetectDrift("test-workspace", planOutput)
	if err != nil {
		t.Fatalf("Failed to detect drift: %v", err)
	}

	// Verify drift was detected
	if !report.HasDrift {
		t.Error("Expected drift to be detected")
	}

	if report.WorkspaceID != "test-workspace" {
		t.Errorf("Expected workspace ID 'test-workspace', got '%s'", report.WorkspaceID)
	}

	// Check that drift was detected (at least 1 resource)
	if report.Summary.DriftedResources < 1 {
		t.Errorf("Expected at least 1 drifted resource, got %d", report.Summary.DriftedResources)
	}

	// Check drifted resources
	if len(report.DriftedResources) < 1 {
		t.Fatalf("Expected at least 1 drifted resource, got %d", len(report.DriftedResources))
	}

	// Verify at least one resource was detected
	if len(report.DriftedResources) > 0 {
		t.Logf("Detected drift: %+v", report.DriftedResources)
	}
}

func TestDriftDetector_NoDrift(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	sm := NewSnapshotManager(storage)
	dd := NewDriftDetector(storage, sm)

	// Plan output with no drift (only planned creates)
	planOutput := `
Terraform will perform the following actions:

  # null_resource.new will be created
  + resource "null_resource" "new" {
      + id = (known after apply)
    }

Plan: 1 to add, 0 to change, 0 to destroy.
`

	// Detect drift
	report, err := dd.DetectDrift("test-workspace", planOutput)
	if err != nil {
		t.Fatalf("Failed to detect drift: %v", err)
	}

	// No drift should be detected
	if report.HasDrift {
		t.Error("Expected no drift to be detected")
	}

	if report.Summary.DriftedResources != 0 {
		t.Errorf("Expected 0 drifted resources, got %d", report.Summary.DriftedResources)
	}
}

func TestValidateDriftReport(t *testing.T) {
	report := &DriftReport{
		HasDrift: true,
		Summary: DriftSummary{
			DriftedResources: 2,
			UpdatedResources: 1,
			DeletedResources: 1,
		},
		DriftedResources: []ResourceDrift{
			{
				Address:   "null_resource.test",
				Type:      "null_resource",
				DriftType: DriftTypeUpdate,
				Reason:    "Resource has changed outside of Terraform",
			},
			{
				Address:   "aws_instance.server",
				Type:      "aws_instance",
				DriftType: DriftTypeDelete,
				Reason:    "Resource deleted outside of Terraform",
			},
		},
	}

	issues := ValidateDriftReport(report)

	if len(issues) == 0 {
		t.Error("Expected issues to be reported")
	}

	// Check that issues mention drift
	foundDriftMessage := false

	for _, issue := range issues {
		if len(issue) > 0 && len(issue) >= 5 && issue[0:5] == "Drift" {
			foundDriftMessage = true
		}
	}

	if !foundDriftMessage {
		t.Error("Expected drift message in issues")
	}
	// Note: The exact format of warning messages may vary
	t.Logf("Issues found: %v", issues)
}
