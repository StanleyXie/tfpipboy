package state

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DriftDetector detects drift between desired and actual state
type DriftDetector interface {
	DetectDrift(workspaceID string, planOutput string) (*DriftReport, error)
	DetectDriftFromPlanJSON(workspaceID string, planJSON []byte) (*DriftReport, error)
}

type driftDetector struct {
	storage         Storage
	snapshotManager SnapshotManager
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(storage Storage, snapshotManager SnapshotManager) DriftDetector {
	return &driftDetector{
		storage:         storage,
		snapshotManager: snapshotManager,
	}
}

// DetectDrift analyzes terraform plan output to detect drift
func (dd *driftDetector) DetectDrift(workspaceID string, planOutput string) (*DriftReport, error) {
	// Get latest snapshot for context
	snapshot, err := dd.snapshotManager.GetLatest(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest snapshot: %w", err)
	}

	modulePath := ""
	totalResources := 0
	if snapshot != nil {
		modulePath = snapshot.ModulePath
		totalResources = len(snapshot.Resources)
	}

	report := &DriftReport{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		ModulePath:  modulePath,
		Timestamp:   time.Now(),
		Summary: DriftSummary{
			TotalResources: totalResources,
		},
	}

	// Parse plan output for drift indicators
	driftedResources := parsePlanOutputForDrift(planOutput)

	report.DriftedResources = driftedResources
	report.HasDrift = len(driftedResources) > 0

	// Update summary
	for _, drift := range driftedResources {
		report.Summary.DriftedResources++
		switch drift.DriftType {
		case DriftTypeUpdate:
			report.Summary.UpdatedResources++
		case DriftTypeDelete:
			report.Summary.DeletedResources++
		case DriftTypeOutOfSync:
			report.Summary.OutOfSyncCount++
		}
	}

	// Save drift report
	if err := dd.storage.SaveDriftReport(report); err != nil {
		return nil, fmt.Errorf("failed to save drift report: %w", err)
	}

	return report, nil
}

// DetectDriftFromPlanJSON analyzes terraform plan JSON to detect drift
func (dd *driftDetector) DetectDriftFromPlanJSON(_ string, _ []byte) (*DriftReport, error) {
	// TODO: Implement JSON plan parsing for more accurate drift detection
	// This would parse the structured JSON plan format for precise analysis
	return nil, fmt.Errorf("JSON plan parsing not yet implemented")
}

// parsePlanOutputForDrift parses terraform plan text output for drift indicators
func parsePlanOutputForDrift(planOutput string) []ResourceDrift {
	drifted := []ResourceDrift{}

	// Regular expressions for detecting drift patterns
	// Pattern: ~ resource "type" "name" {
	updatePattern := regexp.MustCompile(`^\s*~\s+resource\s+"([^"]+)"\s+"([^"]+)"`)
	// Pattern: - resource "type" "name" {
	deletePattern := regexp.MustCompile(`^\s*-\s+resource\s+"([^"]+)"\s+"([^"]+)"`)
	// Pattern: + resource "type" "name" {
	createPattern := regexp.MustCompile(`^\s*\+\s+resource\s+"([^"]+)"\s+"([^"]+)"`)
	// Pattern: attribute = "value" -> "new_value"
	attrChangePattern := regexp.MustCompile(`^\s*~\s+(\S+)\s+=\s+(.+?)\s+->\s+(.+)$`)
	// Pattern for drift detection message
	driftMessagePattern := regexp.MustCompile(`Objects have changed outside of Terraform`)

	scanner := bufio.NewScanner(strings.NewReader(planOutput))

	var currentResource *ResourceDrift
	var currentChanges []AttributeChange
	hasDriftMessage := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check for drift message
		if driftMessagePattern.MatchString(line) {
			hasDriftMessage = true
		}

		// Check for update
		if matches := updatePattern.FindStringSubmatch(line); len(matches) == 3 {
			// Save previous resource if exists
			if currentResource != nil {
				currentResource.Changes = currentChanges
				drifted = append(drifted, *currentResource)
			}

			resourceType := matches[1]
			resourceName := matches[2]
			currentResource = &ResourceDrift{
				Address:   fmt.Sprintf("%s.%s", resourceType, resourceName),
				Type:      resourceType,
				DriftType: DriftTypeUpdate,
			}
			if hasDriftMessage {
				currentResource.Reason = "Resource has changed outside of Terraform"
			}
			currentChanges = []AttributeChange{}
			continue
		}

		// Check for delete
		if matches := deletePattern.FindStringSubmatch(line); len(matches) == 3 {
			resourceType := matches[1]
			resourceName := matches[2]
			drift := ResourceDrift{
				Address:   fmt.Sprintf("%s.%s", resourceType, resourceName),
				Type:      resourceType,
				DriftType: DriftTypeDelete,
				Reason:    "Resource deleted outside of Terraform",
			}
			drifted = append(drifted, drift)
			currentResource = nil
			currentChanges = nil
			continue
		}

		// Skip create patterns (not drift, these are planned creates)
		if createPattern.MatchString(line) {
			// Save previous resource if exists
			if currentResource != nil {
				currentResource.Changes = currentChanges
				drifted = append(drifted, *currentResource)
				currentResource = nil
				currentChanges = nil
			}
			continue
		}

		// Capture attribute changes for current resource
		if currentResource != nil {
			if matches := attrChangePattern.FindStringSubmatch(line); len(matches) == 4 {
				attrName := strings.TrimSpace(matches[1])
				oldValue := strings.TrimSpace(matches[2])
				newValue := strings.TrimSpace(matches[3])

				currentChanges = append(currentChanges, AttributeChange{
					Path:     attrName,
					OldValue: oldValue,
					NewValue: newValue,
				})
			}
		}
	}

	// Don't forget the last resource
	if currentResource != nil {
		currentResource.Changes = currentChanges
		drifted = append(drifted, *currentResource)
	}

	return drifted
}

// ValidateDriftReport checks if a drift report indicates issues
func ValidateDriftReport(report *DriftReport) []string {
	issues := []string{}

	if report.HasDrift {
		issues = append(issues, fmt.Sprintf("Drift detected: %d resource(s) have changed", report.Summary.DriftedResources))

		if report.Summary.DeletedResources > 0 {
			issues = append(issues, fmt.Sprintf("⚠️  Critical: %d resource(s) deleted outside Terraform", report.Summary.DeletedResources))
		}

		if report.Summary.UpdatedResources > 0 {
			issues = append(issues, fmt.Sprintf("⚠️  Warning: %d resource(s) modified outside Terraform", report.Summary.UpdatedResources))
		}

		// List specific resources
		for _, drift := range report.DriftedResources {
			issues = append(issues, fmt.Sprintf("  - %s (%s): %s", drift.Address, drift.DriftType, drift.Reason))
		}
	}

	return issues
}
