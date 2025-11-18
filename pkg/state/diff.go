package state

import (
	"time"
)

// VersionDiff represents a detailed diff between two state versions
type VersionDiff struct {
	WorkspaceID      string              `json:"workspace_id"`
	FromVersion      int                 `json:"from_version"`
	ToVersion        int                 `json:"to_version"`
	FromSnapshotID   string              `json:"from_snapshot_id"`
	ToSnapshotID     string              `json:"to_snapshot_id"`
	FromTimestamp    time.Time           `json:"from_timestamp"`
	ToTimestamp      time.Time           `json:"to_timestamp"`
	Duration         time.Duration       `json:"duration"`
	Summary          DiffSummary         `json:"summary"`
	ResourceChanges  []ResourceDiff      `json:"resource_changes"`
	OutputChanges    []OutputChange      `json:"output_changes,omitempty"`
	DependencyChanges []DependencyChange `json:"dependency_changes,omitempty"`
}

// DiffSummary provides aggregate statistics for the diff
type DiffSummary struct {
	TotalResourcesAdded   int `json:"total_resources_added"`
	TotalResourcesRemoved int `json:"total_resources_removed"`
	TotalResourcesChanged int `json:"total_resources_changed"`
	TotalResourcesUnchanged int `json:"total_resources_unchanged"`
	TotalAttributesChanged int `json:"total_attributes_changed"`
	TotalOutputsChanged   int `json:"total_outputs_changed"`
	SignificantChange     bool `json:"significant_change"` // Major infrastructure change
}

// ResourceDiff represents the change for a specific resource
type ResourceDiff struct {
	Address         string            `json:"address"`
	Type            string            `json:"type"`
	ChangeType      ResourceChangeType `json:"change_type"`
	Before          *ResourceState    `json:"before,omitempty"`
	After           *ResourceState    `json:"after,omitempty"`
	AttributeChanges []AttributeChange `json:"attribute_changes,omitempty"`
	Impact          ChangeImpact      `json:"impact"` // Low, Medium, High
}

// ResourceChangeType indicates the type of resource change
type ResourceChangeType string

const (
	ChangeTypeAdded     ResourceChangeType = "added"
	ChangeTypeRemoved   ResourceChangeType = "removed"
	ChangeTypeModified  ResourceChangeType = "modified"
	ChangeTypeRecreated ResourceChangeType = "recreated"
	ChangeTypeUnchanged ResourceChangeType = "unchanged"
)

// ChangeImpact indicates the severity of a change
type ChangeImpact string

const (
	ImpactLow    ChangeImpact = "low"    // Config changes, tags
	ImpactMedium ChangeImpact = "medium" // Non-destructive updates
	ImpactHigh   ChangeImpact = "high"   // Recreates, deletions
)

// OutputChange represents a change in Terraform outputs
type OutputChange struct {
	Name      string      `json:"name"`
	OldValue  interface{} `json:"old_value,omitempty"`
	NewValue  interface{} `json:"new_value,omitempty"`
	ChangeType string     `json:"change_type"` // added, removed, modified
}

// DependencyChange represents a change in resource dependencies
type DependencyChange struct {
	Resource        string   `json:"resource"`
	OldDependencies []string `json:"old_dependencies,omitempty"`
	NewDependencies []string `json:"new_dependencies,omitempty"`
}

// ChangeTimeline represents a chronological view of all changes
type ChangeTimeline struct {
	WorkspaceID string              `json:"workspace_id"`
	FromVersion int                 `json:"from_version"`
	ToVersion   int                 `json:"to_version"`
	Entries     []TimelineEntry     `json:"entries"`
	Summary     TimelineSummary     `json:"summary"`
}

// TimelineEntry represents a single point in the timeline
type TimelineEntry struct {
	Version        int                `json:"version"`
	SnapshotID     string             `json:"snapshot_id"`
	Timestamp      time.Time          `json:"timestamp"`
	ResourceCount  int                `json:"resource_count"`
	ChangesFromPrev int               `json:"changes_from_prev"`
	ChangesSummary string             `json:"changes_summary"` // Human-readable
	ChangedResources []string         `json:"changed_resources"`
}

// TimelineSummary provides aggregate statistics for the timeline
type TimelineSummary struct {
	TotalVersions      int           `json:"total_versions"`
	TotalChanges       int           `json:"total_changes"`
	AverageChangesPerVersion float64 `json:"average_changes_per_version"`
	MostActiveVersion  int           `json:"most_active_version"` // Version with most changes
	TimeSpan           time.Duration `json:"time_span"`
}

// ResourceHistory tracks all changes to a specific resource over time
type ResourceHistory struct {
	ResourceAddress string                  `json:"resource_address"`
	ResourceType    string                  `json:"resource_type"`
	WorkspaceID     string                  `json:"workspace_id"`
	FirstSeen       time.Time               `json:"first_seen"`
	LastSeen        time.Time               `json:"last_seen"`
	TotalChanges    int                     `json:"total_changes"`
	Versions        []ResourceVersionEntry  `json:"versions"`
}

// ResourceVersionEntry represents a resource at a specific version
type ResourceVersionEntry struct {
	Version       int              `json:"version"`
	SnapshotID    string           `json:"snapshot_id"`
	Timestamp     time.Time        `json:"timestamp"`
	State         *ResourceState   `json:"state"`
	ChangesFromPrev []AttributeChange `json:"changes_from_prev,omitempty"`
}
