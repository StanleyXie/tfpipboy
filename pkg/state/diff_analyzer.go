package state

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// DiffAnalyzer provides version-to-version comparison capabilities
type DiffAnalyzer interface {
	// Compare two specific versions
	DiffVersions(workspaceID string, fromVersion, toVersion int) (*VersionDiff, error)

	// Get timeline of all changes
	GetTimeline(workspaceID string) (*ChangeTimeline, error)

	// Get history of a specific resource
	GetResourceHistory(workspaceID string, resourceAddress string) (*ResourceHistory, error)

	// Get changes for a specific version (compared to previous)
	GetVersionChanges(workspaceID string, version int) (*VersionDiff, error)
}

type diffAnalyzer struct {
	storage         Storage
	snapshotManager SnapshotManager
	changeTracker   ChangeTracker
}

// NewDiffAnalyzer creates a new diff analyzer
func NewDiffAnalyzer(storage Storage, snapshotManager SnapshotManager, changeTracker ChangeTracker) DiffAnalyzer {
	return &diffAnalyzer{
		storage:         storage,
		snapshotManager: snapshotManager,
		changeTracker:   changeTracker,
	}
}

// DiffVersions compares two specific versions and returns detailed differences
func (da *diffAnalyzer) DiffVersions(workspaceID string, fromVersion, toVersion int) (*VersionDiff, error) {
	// Get snapshots
	fromSnapshot, err := da.snapshotManager.GetByVersion(workspaceID, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot v%d: %w", fromVersion, err)
	}

	toSnapshot, err := da.snapshotManager.GetByVersion(workspaceID, toVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot v%d: %w", toVersion, err)
	}

	// Create diff
	diff := &VersionDiff{
		WorkspaceID:    workspaceID,
		FromVersion:    fromVersion,
		ToVersion:      toVersion,
		FromSnapshotID: fromSnapshot.ID,
		ToSnapshotID:   toSnapshot.ID,
		FromTimestamp:  fromSnapshot.Timestamp,
		ToTimestamp:    toSnapshot.Timestamp,
		Duration:       toSnapshot.Timestamp.Sub(fromSnapshot.Timestamp),
	}

	// Build resource maps
	fromResources := make(map[string]*ResourceState)
	for i := range fromSnapshot.Resources {
		res := &fromSnapshot.Resources[i]
		fromResources[res.Address] = res
	}

	toResources := make(map[string]*ResourceState)
	for i := range toSnapshot.Resources {
		res := &toSnapshot.Resources[i]
		toResources[res.Address] = res
	}

	// Compare resources
	resourceChanges := []ResourceDiff{}
	added := 0
	removed := 0
	changed := 0
	unchanged := 0
	totalAttrChanges := 0

	// Check all resources in "to" snapshot
	for address, toRes := range toResources {
		fromRes, existed := fromResources[address]

		if !existed {
			// Resource added
			resourceChanges = append(resourceChanges, ResourceDiff{
				Address:    address,
				Type:       toRes.Type,
				ChangeType: ChangeTypeAdded,
				After:      toRes,
				Impact:     ImpactMedium,
			})
			added++
		} else {
			// Check for changes
			attrChanges := compareResources(fromRes, toRes)
			if len(attrChanges) > 0 {
				impact := assessChangeImpact(attrChanges)
				resourceChanges = append(resourceChanges, ResourceDiff{
					Address:          address,
					Type:             toRes.Type,
					ChangeType:       ChangeTypeModified,
					Before:           fromRes,
					After:            toRes,
					AttributeChanges: attrChanges,
					Impact:           impact,
				})
				changed++
				totalAttrChanges += len(attrChanges)
			} else {
				unchanged++
			}
		}
	}

	// Check for removed resources
	for address, fromRes := range fromResources {
		if _, exists := toResources[address]; !exists {
			resourceChanges = append(resourceChanges, ResourceDiff{
				Address:    address,
				Type:       fromRes.Type,
				ChangeType: ChangeTypeRemoved,
				Before:     fromRes,
				Impact:     ImpactHigh,
			})
			removed++
		}
	}

	// Compare outputs
	outputChanges := compareOutputs(fromSnapshot.Outputs, toSnapshot.Outputs)

	// Build summary
	diff.Summary = DiffSummary{
		TotalResourcesAdded:     added,
		TotalResourcesRemoved:   removed,
		TotalResourcesChanged:   changed,
		TotalResourcesUnchanged: unchanged,
		TotalAttributesChanged:  totalAttrChanges,
		TotalOutputsChanged:     len(outputChanges),
		SignificantChange:       removed > 0 || added > 5 || changed > 10,
	}

	diff.ResourceChanges = resourceChanges
	diff.OutputChanges = outputChanges

	return diff, nil
}

// GetVersionChanges gets changes for a specific version (compared to previous)
func (da *diffAnalyzer) GetVersionChanges(workspaceID string, version int) (*VersionDiff, error) {
	if version <= 1 {
		return nil, fmt.Errorf("version must be > 1 to compare with previous")
	}
	return da.DiffVersions(workspaceID, version-1, version)
}

// GetTimeline creates a chronological view of all changes
func (da *diffAnalyzer) GetTimeline(workspaceID string) (*ChangeTimeline, error) {
	// Get all snapshots
	snapshots, err := da.snapshotManager.List(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		return &ChangeTimeline{
			WorkspaceID: workspaceID,
			Entries:     []TimelineEntry{},
		}, nil
	}

	timeline := &ChangeTimeline{
		WorkspaceID: workspaceID,
		FromVersion: snapshots[0].Version,
		ToVersion:   snapshots[len(snapshots)-1].Version,
		Entries:     []TimelineEntry{},
	}

	totalChanges := 0
	mostActiveVersion := 0
	maxChanges := 0

	// Create timeline entries
	for i, snapshot := range snapshots {
		entry := TimelineEntry{
			Version:       snapshot.Version,
			SnapshotID:    snapshot.ID,
			Timestamp:     snapshot.Timestamp,
			ResourceCount: len(snapshot.Resources),
		}

		// Compare with previous version if not first
		if i > 0 {
			prevSnapshot := &snapshots[i-1]
			diff, err := da.diffSnapshots(prevSnapshot, &snapshot)
			if err == nil {
				changeCount := diff.Summary.TotalResourcesAdded +
					diff.Summary.TotalResourcesRemoved +
					diff.Summary.TotalResourcesChanged
				entry.ChangesFromPrev = changeCount
				entry.ChangesSummary = formatDiffSummary(diff.Summary)
				entry.ChangedResources = extractChangedResources(diff.ResourceChanges)

				totalChanges += changeCount
				if changeCount > maxChanges {
					maxChanges = changeCount
					mostActiveVersion = snapshot.Version
				}
			}
		}

		timeline.Entries = append(timeline.Entries, entry)
	}

	// Build timeline summary
	timeline.Summary = TimelineSummary{
		TotalVersions:    len(snapshots),
		TotalChanges:     totalChanges,
		MostActiveVersion: mostActiveVersion,
		TimeSpan:         snapshots[len(snapshots)-1].Timestamp.Sub(snapshots[0].Timestamp),
	}

	if len(snapshots) > 1 {
		timeline.Summary.AverageChangesPerVersion = float64(totalChanges) / float64(len(snapshots)-1)
	}

	return timeline, nil
}

// GetResourceHistory gets the complete history of a specific resource
func (da *diffAnalyzer) GetResourceHistory(workspaceID string, resourceAddress string) (*ResourceHistory, error) {
	// Get all snapshots
	snapshots, err := da.snapshotManager.List(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	history := &ResourceHistory{
		ResourceAddress: resourceAddress,
		WorkspaceID:     workspaceID,
		Versions:        []ResourceVersionEntry{},
	}

	var prevResourceState *ResourceState

	// Find resource in each snapshot
	for _, snapshot := range snapshots {
		for i := range snapshot.Resources {
			res := &snapshot.Resources[i]
			if res.Address == resourceAddress {
				if history.ResourceType == "" {
					history.ResourceType = res.Type
					history.FirstSeen = snapshot.Timestamp
				}
				history.LastSeen = snapshot.Timestamp

				entry := ResourceVersionEntry{
					Version:    snapshot.Version,
					SnapshotID: snapshot.ID,
					Timestamp:  snapshot.Timestamp,
					State:      res,
				}

				// Compare with previous version
				if prevResourceState != nil {
					changes := compareResources(prevResourceState, res)
					entry.ChangesFromPrev = changes
					if len(changes) > 0 {
						history.TotalChanges++
					}
				}

				history.Versions = append(history.Versions, entry)
				prevResourceState = res
				break
			}
		}
	}

	if len(history.Versions) == 0 {
		return nil, fmt.Errorf("resource %s not found in any snapshot", resourceAddress)
	}

	return history, nil
}

// Helper functions

func (da *diffAnalyzer) diffSnapshots(from, to *StateSnapshot) (*VersionDiff, error) {
	return da.DiffVersions(from.WorkspaceID, from.Version, to.Version)
}

func assessChangeImpact(changes []AttributeChange) ChangeImpact {
	// Heuristics for impact assessment
	highImpactFields := []string{"id", "arn", "ami", "instance_type", "availability_zone"}
	mediumImpactFields := []string{"security_groups", "subnet_id", "vpc_id"}

	for _, change := range changes {
		path := strings.ToLower(change.Path)
		for _, field := range highImpactFields {
			if strings.Contains(path, field) {
				return ImpactHigh
			}
		}
		for _, field := range mediumImpactFields {
			if strings.Contains(path, field) {
				return ImpactMedium
			}
		}
	}

	return ImpactLow
}

func compareOutputs(from, to map[string]interface{}) []OutputChange {
	changes := []OutputChange{}

	// Check all outputs in "to"
	for name, toValue := range to {
		fromValue, existed := from[name]
		if !existed {
			changes = append(changes, OutputChange{
				Name:       name,
				NewValue:   toValue,
				ChangeType: "added",
			})
		} else if fmt.Sprintf("%v", fromValue) != fmt.Sprintf("%v", toValue) {
			changes = append(changes, OutputChange{
				Name:       name,
				OldValue:   fromValue,
				NewValue:   toValue,
				ChangeType: "modified",
			})
		}
	}

	// Check for removed outputs
	for name, fromValue := range from {
		if _, exists := to[name]; !exists {
			changes = append(changes, OutputChange{
				Name:       name,
				OldValue:   fromValue,
				ChangeType: "removed",
			})
		}
	}

	return changes
}

func formatDiffSummary(summary DiffSummary) string {
	parts := []string{}
	if summary.TotalResourcesAdded > 0 {
		parts = append(parts, fmt.Sprintf("+%d added", summary.TotalResourcesAdded))
	}
	if summary.TotalResourcesChanged > 0 {
		parts = append(parts, fmt.Sprintf("~%d changed", summary.TotalResourcesChanged))
	}
	if summary.TotalResourcesRemoved > 0 {
		parts = append(parts, fmt.Sprintf("-%d removed", summary.TotalResourcesRemoved))
	}
	if len(parts) == 0 {
		return "no changes"
	}
	return strings.Join(parts, ", ")
}

func extractChangedResources(changes []ResourceDiff) []string {
	resources := []string{}
	for _, change := range changes {
		if change.ChangeType != ChangeTypeUnchanged {
			resources = append(resources, change.Address)
		}
	}
	// Limit to top 10
	if len(resources) > 10 {
		resources = resources[:10]
	}
	return resources
}

// FormatVersionDiff creates a human-readable diff output
func FormatVersionDiff(diff *VersionDiff) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n=== Version Diff: v%d → v%d ===\n", diff.FromVersion, diff.ToVersion))
	sb.WriteString(fmt.Sprintf("Workspace: %s\n", diff.WorkspaceID))
	sb.WriteString(fmt.Sprintf("Duration: %v\n", diff.Duration.Round(time.Second)))
	sb.WriteString(fmt.Sprintf("Timestamp: %s → %s\n\n",
		diff.FromTimestamp.Format("2006-01-02 15:04:05"),
		diff.ToTimestamp.Format("2006-01-02 15:04:05")))

	// Summary
	s := diff.Summary
	sb.WriteString(fmt.Sprintf("Summary:\n"))
	sb.WriteString(fmt.Sprintf("  Resources: %d unchanged, %d changed, %d added, %d removed\n",
		s.TotalResourcesUnchanged, s.TotalResourcesChanged, s.TotalResourcesAdded, s.TotalResourcesRemoved))
	sb.WriteString(fmt.Sprintf("  Attributes: %d changed\n", s.TotalAttributesChanged))
	sb.WriteString(fmt.Sprintf("  Outputs: %d changed\n", s.TotalOutputsChanged))
	if s.SignificantChange {
		sb.WriteString("  ⚠️  Significant infrastructure change detected\n")
	}
	sb.WriteString("\n")

	// Group changes by type
	added := []ResourceDiff{}
	removed := []ResourceDiff{}
	modified := []ResourceDiff{}

	for _, change := range diff.ResourceChanges {
		switch change.ChangeType {
		case ChangeTypeAdded:
			added = append(added, change)
		case ChangeTypeRemoved:
			removed = append(removed, change)
		case ChangeTypeModified:
			modified = append(modified, change)
		}
	}

	// Added resources
	if len(added) > 0 {
		sb.WriteString(fmt.Sprintf("Added Resources (%d):\n", len(added)))
		for _, res := range added {
			sb.WriteString(fmt.Sprintf("  + %s (%s)\n", res.Address, res.Type))
		}
		sb.WriteString("\n")
	}

	// Removed resources
	if len(removed) > 0 {
		sb.WriteString(fmt.Sprintf("Removed Resources (%d):\n", len(removed)))
		for _, res := range removed {
			sb.WriteString(fmt.Sprintf("  - %s (%s) [Impact: %s]\n", res.Address, res.Type, res.Impact))
		}
		sb.WriteString("\n")
	}

	// Modified resources
	if len(modified) > 0 {
		sb.WriteString(fmt.Sprintf("Modified Resources (%d):\n", len(modified)))

		// Sort by impact (high first)
		sort.Slice(modified, func(i, j int) bool {
			impactOrder := map[ChangeImpact]int{ImpactHigh: 0, ImpactMedium: 1, ImpactLow: 2}
			return impactOrder[modified[i].Impact] < impactOrder[modified[j].Impact]
		})

		for _, res := range modified {
			impactSymbol := "~"
			if res.Impact == ImpactHigh {
				impactSymbol = "⚠"
			}
			sb.WriteString(fmt.Sprintf("  %s %s (%s) [Impact: %s, %d attributes]\n",
				impactSymbol, res.Address, res.Type, res.Impact, len(res.AttributeChanges)))

			// Show first few attribute changes
			maxShow := 3
			for i, attr := range res.AttributeChanges {
				if i >= maxShow {
					remaining := len(res.AttributeChanges) - maxShow
					sb.WriteString(fmt.Sprintf("      ... and %d more\n", remaining))
					break
				}
				sb.WriteString(fmt.Sprintf("      %s: %v → %v\n", attr.Path, attr.OldValue, attr.NewValue))
			}
		}
		sb.WriteString("\n")
	}

	// Output changes
	if len(diff.OutputChanges) > 0 {
		sb.WriteString(fmt.Sprintf("Output Changes (%d):\n", len(diff.OutputChanges)))
		for _, out := range diff.OutputChanges {
			switch out.ChangeType {
			case "added":
				sb.WriteString(fmt.Sprintf("  + %s = %v\n", out.Name, out.NewValue))
			case "removed":
				sb.WriteString(fmt.Sprintf("  - %s = %v\n", out.Name, out.OldValue))
			case "modified":
				sb.WriteString(fmt.Sprintf("  ~ %s: %v → %v\n", out.Name, out.OldValue, out.NewValue))
			}
		}
	}

	return sb.String()
}
