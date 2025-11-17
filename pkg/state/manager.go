package state

import (
	"fmt"
	"path/filepath"
)

// Manager is a facade that coordinates state tracking, change detection, and drift analysis
type Manager struct {
	storage         Storage
	snapshotManager SnapshotManager
	changeTracker   ChangeTracker
	driftDetector   DriftDetector
}

// NewManager creates a new state manager with all components initialized
func NewManager(baseDir string) (*Manager, error) {
	// Initialize storage
	storage, err := NewFileStorage(filepath.Join(baseDir, "state-tracking"))
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Initialize components
	snapshotManager := NewSnapshotManager(storage)
	changeTracker := NewChangeTracker(storage)
	driftDetector := NewDriftDetector(storage, snapshotManager)

	return &Manager{
		storage:         storage,
		snapshotManager: snapshotManager,
		changeTracker:   changeTracker,
		driftDetector:   driftDetector,
	}, nil
}

// CaptureState captures the current terraform state and tracks changes
func (m *Manager) CaptureState(workspaceID string, tfstatePath string) (*StateSnapshot, []ChangeEvent, error) {
	// Get previous snapshot for comparison
	previous, err := m.snapshotManager.GetLatest(workspaceID)
	if err != nil {
		// First snapshot - no previous state to compare
		previous = nil
	}

	// Capture new snapshot
	current, err := m.snapshotManager.Capture(workspaceID, tfstatePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to capture snapshot: %w", err)
	}

	// Track changes if there was a previous snapshot
	var changes []ChangeEvent
	if previous != nil {
		changes, err = m.changeTracker.TrackChanges(previous, current)
		if err != nil {
			return current, nil, fmt.Errorf("failed to track changes: %w", err)
		}
	}

	return current, changes, nil
}

// DetectDrift runs drift detection using terraform plan output
func (m *Manager) DetectDrift(workspaceID string, planOutput string) (*DriftReport, error) {
	return m.driftDetector.DetectDrift(workspaceID, planOutput)
}

// GetSnapshot retrieves the latest snapshot for a workspace
func (m *Manager) GetSnapshot(workspaceID string) (*StateSnapshot, error) {
	return m.snapshotManager.GetLatest(workspaceID)
}

// GetSnapshotByVersion retrieves a specific snapshot version
func (m *Manager) GetSnapshotByVersion(workspaceID string, version int) (*StateSnapshot, error) {
	return m.snapshotManager.GetByVersion(workspaceID, version)
}

// ListSnapshots lists all snapshots for a workspace
func (m *Manager) ListSnapshots(workspaceID string) ([]StateSnapshot, error) {
	return m.snapshotManager.List(workspaceID)
}

// GetChangeHistory retrieves all change events for a workspace
func (m *Manager) GetChangeHistory(workspaceID string) (*ChangeHistory, error) {
	return m.changeTracker.GetChangeHistory(workspaceID)
}

// ListDriftReports lists all drift reports for a workspace
func (m *Manager) ListDriftReports(workspaceID string) ([]DriftReport, error) {
	return m.storage.ListDriftReports(workspaceID)
}

// GetLatestDriftReport gets the most recent drift report for a workspace
func (m *Manager) GetLatestDriftReport(workspaceID string) (*DriftReport, error) {
	reports, err := m.storage.ListDriftReports(workspaceID)
	if err != nil {
		return nil, err
	}

	if len(reports) == 0 {
		return nil, nil
	}

	// Reports are already sorted by timestamp (descending)
	return &reports[0], nil
}
