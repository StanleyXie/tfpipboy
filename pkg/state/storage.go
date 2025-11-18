package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Storage handles persistent storage of state tracking data
type Storage interface {
	// Snapshot operations
	SaveSnapshot(snapshot *StateSnapshot) error
	LoadSnapshot(id string) (*StateSnapshot, error)
	ListSnapshots(workspaceID string) ([]StateSnapshot, error)

	// Change event operations
	SaveChangeEvent(event *ChangeEvent) error
	LoadChangeEvents(workspaceID string) ([]ChangeEvent, error)

	// Drift report operations
	SaveDriftReport(report *DriftReport) error
	LoadDriftReport(id string) (*DriftReport, error)
	ListDriftReports(workspaceID string) ([]DriftReport, error)
}

type fileStorage struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileStorage creates a new file-based storage implementation
func NewFileStorage(baseDir string) (Storage, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &fileStorage{
		baseDir: baseDir,
	}, nil
}

// SaveSnapshot saves a snapshot to disk
func (fs *fileStorage) SaveSnapshot(snapshot *StateSnapshot) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Create workspace snapshot directory
	snapshotDir := filepath.Join(fs.baseDir, "snapshots", snapshot.WorkspaceID)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Save snapshot file
	filename := fmt.Sprintf("v%d_%d.json", snapshot.Version, snapshot.Timestamp.Unix())
	filePath := filepath.Join(snapshotDir, filename)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	// Update latest symlink
	latestLink := filepath.Join(snapshotDir, "latest.json")
	_ = os.Remove(latestLink) // Remove old symlink if exists
	if err := os.Symlink(filename, latestLink); err != nil {
		// If symlink fails (e.g., Windows), copy the file instead
		if err := os.WriteFile(latestLink, data, 0644); err != nil {
			return fmt.Errorf("failed to create latest snapshot reference: %w", err)
		}
	}

	return nil
}

// LoadSnapshot loads a snapshot from disk
func (fs *fileStorage) LoadSnapshot(id string) (*StateSnapshot, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Search all workspace directories for the snapshot
	snapshotsDir := filepath.Join(fs.baseDir, "snapshots")
	workspaces, err := os.ReadDir(snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("snapshot not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read snapshots directory: %w", err)
	}

	for _, workspace := range workspaces {
		if !workspace.IsDir() {
			continue
		}

		workspaceDir := filepath.Join(snapshotsDir, workspace.Name())
		files, err := os.ReadDir(workspaceDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
				continue
			}

			filePath := filepath.Join(workspaceDir, file.Name())
			snapshot, err := fs.loadSnapshotFile(filePath)
			if err != nil {
				continue
			}

			if snapshot.ID == id {
				return snapshot, nil
			}
		}
	}

	return nil, fmt.Errorf("snapshot not found: %s", id)
}

// ListSnapshots lists all snapshots for a workspace
func (fs *fileStorage) ListSnapshots(workspaceID string) ([]StateSnapshot, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	snapshotDir := filepath.Join(fs.baseDir, "snapshots", workspaceID)
	files, err := os.ReadDir(snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []StateSnapshot{}, nil
		}
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	snapshots := []StateSnapshot{}
	for _, file := range files {
		if file.IsDir() || file.Name() == "latest.json" {
			continue
		}

		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(snapshotDir, file.Name())
		snapshot, err := fs.loadSnapshotFile(filePath)
		if err != nil {
			// Skip invalid files
			continue
		}

		snapshots = append(snapshots, *snapshot)
	}

	// Sort by version
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Version < snapshots[j].Version
	})

	return snapshots, nil
}

// SaveChangeEvent saves a change event to disk (append to JSONL file)
func (fs *fileStorage) SaveChangeEvent(event *ChangeEvent) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Get workspace ID from event
	workspaceID := event.WorkspaceID
	if workspaceID == "" {
		return fmt.Errorf("change event missing workspace ID")
	}

	// Create changes directory
	changesDir := filepath.Join(fs.baseDir, "changes", workspaceID)
	if err := os.MkdirAll(changesDir, 0755); err != nil {
		return fmt.Errorf("failed to create changes directory: %w", err)
	}

	// Append to changes.jsonl (JSON Lines format for CDC log)
	changesFile := filepath.Join(changesDir, "changes.jsonl")
	file, err := os.OpenFile(changesFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open changes file: %w", err)
	}
	defer file.Close()

	// Marshal and write event
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal change event: %w", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write change event: %w", err)
	}

	return nil
}

// LoadChangeEvents loads all change events for a workspace
func (fs *fileStorage) LoadChangeEvents(workspaceID string) ([]ChangeEvent, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	changesFile := filepath.Join(fs.baseDir, "changes", workspaceID, "changes.jsonl")
	data, err := os.ReadFile(changesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []ChangeEvent{}, nil
		}
		return nil, fmt.Errorf("failed to read changes file: %w", err)
	}

	// Parse JSON Lines format
	events := []ChangeEvent{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event ChangeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip invalid lines
			continue
		}

		events = append(events, event)
	}

	return events, nil
}

// SaveDriftReport saves a drift report to disk
func (fs *fileStorage) SaveDriftReport(report *DriftReport) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Create drift directory
	driftDir := filepath.Join(fs.baseDir, "drift", report.WorkspaceID)
	if err := os.MkdirAll(driftDir, 0755); err != nil {
		return fmt.Errorf("failed to create drift directory: %w", err)
	}

	// Save drift report
	filename := fmt.Sprintf("%d.json", report.Timestamp.Unix())
	filePath := filepath.Join(driftDir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal drift report: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write drift report: %w", err)
	}

	// Update latest symlink
	latestLink := filepath.Join(driftDir, "latest.json")
	_ = os.Remove(latestLink)
	if err := os.Symlink(filename, latestLink); err != nil {
		// Fallback to copy if symlink fails
		if err := os.WriteFile(latestLink, data, 0644); err != nil {
			return fmt.Errorf("failed to create latest drift report reference: %w", err)
		}
	}

	return nil
}

// LoadDriftReport loads a specific drift report
func (fs *fileStorage) LoadDriftReport(id string) (*DriftReport, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Search all workspace directories
	driftDir := filepath.Join(fs.baseDir, "drift")
	workspaces, err := os.ReadDir(driftDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("drift report not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read drift directory: %w", err)
	}

	for _, workspace := range workspaces {
		if !workspace.IsDir() {
			continue
		}

		workspaceDir := filepath.Join(driftDir, workspace.Name())
		files, err := os.ReadDir(workspaceDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
				continue
			}

			filePath := filepath.Join(workspaceDir, file.Name())
			report, err := fs.loadDriftReportFile(filePath)
			if err != nil {
				continue
			}

			if report.ID == id {
				return report, nil
			}
		}
	}

	return nil, fmt.Errorf("drift report not found: %s", id)
}

// ListDriftReports lists all drift reports for a workspace
func (fs *fileStorage) ListDriftReports(workspaceID string) ([]DriftReport, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	driftDir := filepath.Join(fs.baseDir, "drift", workspaceID)
	files, err := os.ReadDir(driftDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []DriftReport{}, nil
		}
		return nil, fmt.Errorf("failed to read drift directory: %w", err)
	}

	reports := []DriftReport{}
	for _, file := range files {
		if file.IsDir() || file.Name() == "latest.json" {
			continue
		}

		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(driftDir, file.Name())
		report, err := fs.loadDriftReportFile(filePath)
		if err != nil {
			continue
		}

		reports = append(reports, *report)
	}

	// Sort by timestamp (descending)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Timestamp.After(reports[j].Timestamp)
	})

	return reports, nil
}

// Helper functions

func (fs *fileStorage) loadSnapshotFile(filePath string) (*StateSnapshot, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var snapshot StateSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (fs *fileStorage) loadDriftReportFile(filePath string) (*DriftReport, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var report DriftReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	return &report, nil
}
