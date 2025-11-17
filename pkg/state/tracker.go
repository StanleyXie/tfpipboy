package state

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// ChangeTracker tracks changes between state snapshots using CDC approach
type ChangeTracker interface {
	TrackChanges(previous, current *StateSnapshot) ([]ChangeEvent, error)
	GetChangeHistory(workspaceID string) (*ChangeHistory, error)
	GetChangesSince(workspaceID string, since time.Time) ([]ChangeEvent, error)
}

type changeTracker struct {
	storage Storage
}

// NewChangeTracker creates a new change tracker
func NewChangeTracker(storage Storage) ChangeTracker {
	return &changeTracker{
		storage: storage,
	}
}

// TrackChanges compares two snapshots and generates change events (CDC pattern)
func (ct *changeTracker) TrackChanges(previous, current *StateSnapshot) ([]ChangeEvent, error) {
	if current == nil {
		return nil, fmt.Errorf("current snapshot cannot be nil")
	}

	events := []ChangeEvent{}

	// Build resource maps for easier comparison
	previousResources := make(map[string]*ResourceState)
	if previous != nil {
		for i := range previous.Resources {
			resource := &previous.Resources[i]
			previousResources[resource.Address] = resource
		}
	}

	currentResources := make(map[string]*ResourceState)
	for i := range current.Resources {
		resource := &current.Resources[i]
		currentResources[resource.Address] = resource
	}

	// Detect creates and updates
	for address, currentResource := range currentResources {
		previousResource, existed := previousResources[address]

		if !existed {
			// Resource created
			event := ChangeEvent{
				ID:              uuid.New().String(),
				WorkspaceID:     current.WorkspaceID,
				SnapshotAfter:   current.ID,
				Timestamp:       current.Timestamp,
				Operation:       OperationCreate,
				ResourceAddress: address,
				ResourceType:    currentResource.Type,
				After:           currentResource,
			}
			if previous != nil {
				event.SnapshotBefore = previous.ID
			}
			events = append(events, event)
		} else {
			// Check for updates
			changes := compareResources(previousResource, currentResource)
			if len(changes) > 0 {
				event := ChangeEvent{
					ID:               uuid.New().String(),
					WorkspaceID:      current.WorkspaceID,
					SnapshotBefore:   previous.ID,
					SnapshotAfter:    current.ID,
					Timestamp:        current.Timestamp,
					Operation:        OperationUpdate,
					ResourceAddress:  address,
					ResourceType:     currentResource.Type,
					Before:           previousResource,
					After:            currentResource,
					AttributeChanges: changes,
				}
				events = append(events, event)
			}
		}
	}

	// Detect deletions
	if previous != nil {
		for address, previousResource := range previousResources {
			if _, exists := currentResources[address]; !exists {
				// Resource deleted
				event := ChangeEvent{
					ID:              uuid.New().String(),
					WorkspaceID:     current.WorkspaceID,
					SnapshotBefore:  previous.ID,
					SnapshotAfter:   current.ID,
					Timestamp:       current.Timestamp,
					Operation:       OperationDelete,
					ResourceAddress: address,
					ResourceType:    previousResource.Type,
					Before:          previousResource,
				}
				events = append(events, event)
			}
		}
	}

	// Save all change events
	for _, event := range events {
		if err := ct.storage.SaveChangeEvent(&event); err != nil {
			return nil, fmt.Errorf("failed to save change event: %w", err)
		}
	}

	return events, nil
}

// GetChangeHistory retrieves all change events for a workspace
func (ct *changeTracker) GetChangeHistory(workspaceID string) (*ChangeHistory, error) {
	events, err := ct.storage.LoadChangeEvents(workspaceID)
	if err != nil {
		return nil, err
	}

	history := &ChangeHistory{
		WorkspaceID: workspaceID,
		Events:      events,
	}

	// Set time range if events exist
	if len(events) > 0 {
		history.From = events[0].Timestamp
		history.To = events[len(events)-1].Timestamp
	}

	return history, nil
}

// GetChangesSince retrieves change events since a specific time
func (ct *changeTracker) GetChangesSince(workspaceID string, since time.Time) ([]ChangeEvent, error) {
	allEvents, err := ct.storage.LoadChangeEvents(workspaceID)
	if err != nil {
		return nil, err
	}

	filtered := []ChangeEvent{}
	for _, event := range allEvents {
		if event.Timestamp.After(since) || event.Timestamp.Equal(since) {
			filtered = append(filtered, event)
		}
	}

	return filtered, nil
}

// compareResources compares two resource states and returns attribute changes
func compareResources(before, after *ResourceState) []AttributeChange {
	changes := []AttributeChange{}

	// Compare instances
	if len(before.Instances) != len(after.Instances) {
		changes = append(changes, AttributeChange{
			Path:     "instances.count",
			OldValue: len(before.Instances),
			NewValue: len(after.Instances),
		})
	}

	// Compare each instance's attributes
	maxInstances := len(before.Instances)
	if len(after.Instances) > maxInstances {
		maxInstances = len(after.Instances)
	}

	for i := 0; i < maxInstances; i++ {
		var beforeInst, afterInst *ResourceInstance

		if i < len(before.Instances) {
			beforeInst = &before.Instances[i]
		}
		if i < len(after.Instances) {
			afterInst = &after.Instances[i]
		}

		// Instance was added
		if beforeInst == nil && afterInst != nil {
			changes = append(changes, AttributeChange{
				Path:     fmt.Sprintf("instances[%d]", i),
				OldValue: nil,
				NewValue: afterInst.Attributes,
			})
			continue
		}

		// Instance was removed
		if beforeInst != nil && afterInst == nil {
			changes = append(changes, AttributeChange{
				Path:     fmt.Sprintf("instances[%d]", i),
				OldValue: beforeInst.Attributes,
				NewValue: nil,
			})
			continue
		}

		// Compare attributes
		if beforeInst != nil && afterInst != nil {
			attrChanges := compareAttributes(beforeInst.Attributes, afterInst.Attributes, fmt.Sprintf("instances[%d]", i))
			changes = append(changes, attrChanges...)
		}
	}

	return changes
}

// compareAttributes recursively compares attribute maps
func compareAttributes(before, after map[string]interface{}, basePath string) []AttributeChange {
	changes := []AttributeChange{}

	// Check all keys in both maps
	allKeys := make(map[string]bool)
	for key := range before {
		allKeys[key] = true
	}
	for key := range after {
		allKeys[key] = true
	}

	for key := range allKeys {
		path := basePath + "." + key
		beforeVal, beforeExists := before[key]
		afterVal, afterExists := after[key]

		// Key added
		if !beforeExists && afterExists {
			changes = append(changes, AttributeChange{
				Path:     path,
				OldValue: nil,
				NewValue: afterVal,
			})
			continue
		}

		// Key removed
		if beforeExists && !afterExists {
			changes = append(changes, AttributeChange{
				Path:     path,
				OldValue: beforeVal,
				NewValue: nil,
			})
			continue
		}

		// Key exists in both - check if values differ
		if !reflect.DeepEqual(beforeVal, afterVal) {
			// For nested maps, recurse
			beforeMap, beforeIsMap := beforeVal.(map[string]interface{})
			afterMap, afterIsMap := afterVal.(map[string]interface{})

			if beforeIsMap && afterIsMap {
				nestedChanges := compareAttributes(beforeMap, afterMap, path)
				changes = append(changes, nestedChanges...)
			} else {
				// Simple value change
				changes = append(changes, AttributeChange{
					Path:     path,
					OldValue: beforeVal,
					NewValue: afterVal,
				})
			}
		}
	}

	return changes
}
