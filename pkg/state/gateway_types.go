package state

import "time"

// ── Request types ──────────────────────────────────────────────────────────────

// CreateSnapshotRequest is the body sent to POST /workspaces/{id}/snapshots
type CreateSnapshotRequest struct {
	Snapshot *StateSnapshot `json:"snapshot"`
}

// AppendChangeEventRequest is the body sent to POST /workspaces/{id}/changes
type AppendChangeEventRequest struct {
	Event *ChangeEvent `json:"event"`
}

// CreateDriftReportRequest is the body sent to POST /workspaces/{id}/drift
type CreateDriftReportRequest struct {
	Report *DriftReport `json:"report"`
}

// ── Response types ─────────────────────────────────────────────────────────────

// GatewayError is the standard error response from the gateway
type GatewayError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *GatewayError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// SnapshotResponse wraps a single snapshot returned by the gateway
type SnapshotResponse struct {
	Snapshot *StateSnapshot `json:"snapshot"`
}

// SnapshotListResponse wraps a list of snapshots
type SnapshotListResponse struct {
	Snapshots []StateSnapshot `json:"snapshots"`
	Total     int             `json:"total"`
}

// ChangeEventListResponse wraps a list of change events
type ChangeEventListResponse struct {
	Events []ChangeEvent `json:"events"`
	Total  int           `json:"total"`
}

// DriftReportResponse wraps a single drift report
type DriftReportResponse struct {
	Report *DriftReport `json:"report"`
}

// DriftReportListResponse wraps a list of drift reports
type DriftReportListResponse struct {
	Reports []DriftReport `json:"reports"`
	Total   int           `json:"total"`
}

// HealthResponse is returned by GET /health
type HealthResponse struct {
	Status    string    `json:"status"`    // "ok" | "degraded" | "unavailable"
	VaultOK   bool      `json:"vault_ok"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}
