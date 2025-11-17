package state

import (
	"time"
)

// StateSnapshot represents a point-in-time snapshot of Terraform state
type StateSnapshot struct {
	ID               string                 `json:"id"`
	WorkspaceID      string                 `json:"workspace_id"`
	ModulePath       string                 `json:"module_path"`
	Timestamp        time.Time              `json:"timestamp"`
	Version          int                    `json:"version"`
	Resources        []ResourceState        `json:"resources"`
	Outputs          map[string]interface{} `json:"outputs"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           int                    `json:"serial"`
	Lineage          string                 `json:"lineage"`
}

// ResourceState represents a single resource in Terraform state
type ResourceState struct {
	Address      string                 `json:"address"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Provider     string                 `json:"provider"`
	Mode         string                 `json:"mode"` // managed, data
	Instances    []ResourceInstance     `json:"instances"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// ResourceInstance represents a specific instance of a resource (for count/for_each)
type ResourceInstance struct {
	IndexKey   interface{}            `json:"index_key,omitempty"` // for count/for_each
	Attributes map[string]interface{} `json:"attributes"`
	Status     string                 `json:"status,omitempty"` // tainted, etc.
}

// ChangeEvent represents a tracked change between state snapshots (CDC pattern)
type ChangeEvent struct {
	ID               string            `json:"id"`
	WorkspaceID      string            `json:"workspace_id"`
	SnapshotBefore   string            `json:"snapshot_before,omitempty"`
	SnapshotAfter    string            `json:"snapshot_after"`
	Timestamp        time.Time         `json:"timestamp"`
	Operation        ChangeOperation   `json:"operation"`
	ResourceAddress  string            `json:"resource_address"`
	ResourceType     string            `json:"resource_type"`
	Before           *ResourceState    `json:"before,omitempty"`
	After            *ResourceState    `json:"after,omitempty"`
	AttributeChanges []AttributeChange `json:"attribute_changes,omitempty"`
}

// ChangeOperation represents the type of change
type ChangeOperation string

const (
	// OperationCreate indicates a resource was created
	OperationCreate ChangeOperation = "create"
	// OperationUpdate indicates a resource was updated
	OperationUpdate ChangeOperation = "update"
	// OperationDelete indicates a resource was deleted
	OperationDelete ChangeOperation = "delete"
	// OperationNoOp indicates no change
	OperationNoOp ChangeOperation = "no-op"
	// OperationRecreate indicates a resource was replaced (delete + create)
	OperationRecreate ChangeOperation = "recreate"
)

// AttributeChange represents a change to a specific attribute
type AttributeChange struct {
	Path     string      `json:"path"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}

// DriftReport represents the result of drift detection
type DriftReport struct {
	ID               string          `json:"id"`
	WorkspaceID      string          `json:"workspace_id"`
	ModulePath       string          `json:"module_path"`
	Timestamp        time.Time       `json:"timestamp"`
	HasDrift         bool            `json:"has_drift"`
	DriftedResources []ResourceDrift `json:"drifted_resources,omitempty"`
	Summary          DriftSummary    `json:"summary"`
}

// ResourceDrift represents drift detected for a specific resource
type ResourceDrift struct {
	Address   string            `json:"address"`
	Type      string            `json:"type"`
	DriftType DriftType         `json:"drift_type"`
	Changes   []AttributeChange `json:"changes,omitempty"`
	Reason    string            `json:"reason,omitempty"`
}

// DriftType represents the type of drift
type DriftType string

const (
	// DriftTypeUpdate indicates resource attributes have changed
	DriftTypeUpdate DriftType = "update"
	// DriftTypeDelete indicates resource was deleted outside Terraform
	DriftTypeDelete DriftType = "delete"
	// DriftTypeOutOfSync indicates resource state is out of sync
	DriftTypeOutOfSync DriftType = "out-of-sync"
)

// DriftSummary provides aggregate drift statistics
type DriftSummary struct {
	TotalResources   int `json:"total_resources"`
	DriftedResources int `json:"drifted_resources"`
	UpdatedResources int `json:"updated_resources"`
	DeletedResources int `json:"deleted_resources"`
	OutOfSyncCount   int `json:"out_of_sync_count"`
}

// ChangeHistory represents a collection of change events for a workspace
type ChangeHistory struct {
	WorkspaceID string        `json:"workspace_id"`
	Events      []ChangeEvent `json:"events"`
	From        time.Time     `json:"from,omitempty"`
	To          time.Time     `json:"to,omitempty"`
}
