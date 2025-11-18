package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
)

// SnapshotManager handles capturing and managing state snapshots
type SnapshotManager interface {
	Capture(workspaceID string, tfstatePath string) (*StateSnapshot, error)
	GetLatest(workspaceID string) (*StateSnapshot, error)
	GetByVersion(workspaceID string, version int) (*StateSnapshot, error)
	List(workspaceID string) ([]StateSnapshot, error)
}

type snapshotManager struct {
	storage Storage
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(storage Storage) SnapshotManager {
	return &snapshotManager{
		storage: storage,
	}
}

// Capture creates a new snapshot from a tfstate file
func (sm *snapshotManager) Capture(workspaceID string, tfstatePath string) (*StateSnapshot, error) {
	// Read tfstate file
	data, err := os.ReadFile(tfstatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tfstate file: %w", err)
	}

	// Parse tfstate JSON
	var tfstate map[string]interface{}
	if err := json.Unmarshal(data, &tfstate); err != nil {
		return nil, fmt.Errorf("failed to parse tfstate JSON: %w", err)
	}

	// Get module path from tfstate or workspace metadata
	modulePath := ""
	if workspaceDir := filepath.Dir(tfstatePath); workspaceDir != "" {
		modulePath = workspaceDir
	}

	// Determine version number (increment from latest)
	version := 1
	if latest, err := sm.GetLatest(workspaceID); err == nil && latest != nil {
		version = latest.Version + 1
	}

	// Extract resources
	resources, err := extractResources(tfstate)
	if err != nil {
		return nil, fmt.Errorf("failed to extract resources: %w", err)
	}

	// Extract outputs
	outputs := extractOutputs(tfstate)

	// Create snapshot
	snapshot := &StateSnapshot{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		ModulePath:  modulePath,
		Timestamp:   time.Now(),
		Version:     version,
		Resources:   resources,
		Outputs:     outputs,
	}

	// Extract terraform version
	if tfVersion, ok := tfstate["terraform_version"].(string); ok {
		snapshot.TerraformVersion = tfVersion
	}

	// Extract serial and lineage
	if serial, ok := tfstate["serial"].(float64); ok {
		snapshot.Serial = int(serial)
	}
	if lineage, ok := tfstate["lineage"].(string); ok {
		snapshot.Lineage = lineage
	}

	// Save snapshot
	if err := sm.storage.SaveSnapshot(snapshot); err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	return snapshot, nil
}

// GetLatest retrieves the most recent snapshot for a workspace
func (sm *snapshotManager) GetLatest(workspaceID string) (*StateSnapshot, error) {
	snapshots, err := sm.List(workspaceID)
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return nil, nil
	}

	// Sort by version (descending)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Version > snapshots[j].Version
	})

	return &snapshots[0], nil
}

// GetByVersion retrieves a specific snapshot version
func (sm *snapshotManager) GetByVersion(workspaceID string, version int) (*StateSnapshot, error) {
	snapshots, err := sm.List(workspaceID)
	if err != nil {
		return nil, err
	}

	for _, snapshot := range snapshots {
		if snapshot.Version == version {
			return &snapshot, nil
		}
	}

	return nil, fmt.Errorf("snapshot version %d not found for workspace %s", version, workspaceID)
}

// List retrieves all snapshots for a workspace
func (sm *snapshotManager) List(workspaceID string) ([]StateSnapshot, error) {
	return sm.storage.ListSnapshots(workspaceID)
}

// extractResources extracts resource information from tfstate
func extractResources(tfstate map[string]interface{}) ([]ResourceState, error) {
	resources := []ResourceState{}

	// Get resources array
	resourcesRaw, ok := tfstate["resources"]
	if !ok {
		return resources, nil
	}

	resourcesList, ok := resourcesRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("resources is not an array")
	}

	for _, resourceRaw := range resourcesList {
		resource, ok := resourceRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract resource fields
		resourceState := ResourceState{
			Mode: getStringField(resource, "mode"),
			Type: getStringField(resource, "type"),
			Name: getStringField(resource, "name"),
		}

		// Build address
		switch resourceState.Mode {
		case "managed":
			resourceState.Address = fmt.Sprintf("%s.%s", resourceState.Type, resourceState.Name)
		case "data":
			resourceState.Address = fmt.Sprintf("data.%s.%s", resourceState.Type, resourceState.Name)
		}

		// Provider
		resourceState.Provider = getStringField(resource, "provider")

		// Dependencies
		if deps, ok := resource["dependencies"].([]interface{}); ok {
			for _, dep := range deps {
				if depStr, ok := dep.(string); ok {
					resourceState.Dependencies = append(resourceState.Dependencies, depStr)
				}
			}
		}

		// Instances
		if instances, ok := resource["instances"].([]interface{}); ok {
			for _, instRaw := range instances {
				if inst, ok := instRaw.(map[string]interface{}); ok {
					instance := ResourceInstance{
						Attributes: make(map[string]interface{}),
					}

					// Index key (for count/for_each)
					if indexKey, exists := inst["index_key"]; exists {
						instance.IndexKey = indexKey
					}

					// Attributes
					if attrs, ok := inst["attributes"].(map[string]interface{}); ok {
						instance.Attributes = attrs
					}

					// Status (tainted, etc.)
					if status, ok := inst["status"].(string); ok {
						instance.Status = status
					}

					resourceState.Instances = append(resourceState.Instances, instance)
				}
			}
		}

		resources = append(resources, resourceState)
	}

	return resources, nil
}

// extractOutputs extracts output values from tfstate
func extractOutputs(tfstate map[string]interface{}) map[string]interface{} {
	outputs := make(map[string]interface{})

	outputsRaw, ok := tfstate["outputs"]
	if !ok {
		return outputs
	}

	outputsMap, ok := outputsRaw.(map[string]interface{})
	if !ok {
		return outputs
	}

	for key, valueRaw := range outputsMap {
		if valueMap, ok := valueRaw.(map[string]interface{}); ok {
			if value, exists := valueMap["value"]; exists {
				outputs[key] = value
			}
		}
	}

	return outputs
}

// getStringField safely gets a string field from a map
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
