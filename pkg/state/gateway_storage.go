package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// gatewayStorage implements Storage by calling the HTTP/HTTPS gateway,
// which in turn stores all data in HashiCorp Vault (KV v2).
//
// Vault path layout (managed by the gateway, not by this client):
//
//	secret/tfpipboy/workspaces/{workspaceID}/
//	  snapshots/meta          → SnapshotMeta {latest_version, count}
//	  snapshots/v{N}          → StateSnapshot
//	  changes/log             → []ChangeEvent  (append-only list)
//	  drift/{unix-timestamp}  → DriftReport
//	  drift/latest            → DriftReport    (copy of most recent)
type gatewayStorage struct {
	cfg    *GatewayConfig
	client *http.Client
}

// NewGatewayStorage creates a Storage implementation backed by the HTTP/HTTPS gateway.
func NewGatewayStorage(cfg *GatewayConfig) (Storage, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid gateway config: %w", err)
	}

	httpClient, err := cfg.BuildHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("building HTTP client: %w", err)
	}

	return &gatewayStorage{cfg: cfg, client: httpClient}, nil
}

// ── Storage interface implementation ──────────────────────────────────────────

func (g *gatewayStorage) SaveSnapshot(snapshot *StateSnapshot) error {
	body := CreateSnapshotRequest{Snapshot: snapshot}
	url := g.url("workspaces/%s/snapshots", snapshot.WorkspaceID)
	return g.doWithRetry(http.MethodPost, url, body, nil)
}

func (g *gatewayStorage) LoadSnapshot(id string) (*StateSnapshot, error) {
	// id here is the UUID; the gateway can resolve it via its own index
	url := g.url("workspaces/_/snapshots/%s", id)
	var resp SnapshotResponse
	if err := g.doWithRetry(http.MethodGet, url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Snapshot, nil
}

func (g *gatewayStorage) ListSnapshots(workspaceID string) ([]StateSnapshot, error) {
	url := g.url("workspaces/%s/snapshots", workspaceID)
	var resp SnapshotListResponse
	if err := g.doWithRetry(http.MethodGet, url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Snapshots, nil
}

func (g *gatewayStorage) SaveChangeEvent(event *ChangeEvent) error {
	body := AppendChangeEventRequest{Event: event}
	url := g.url("workspaces/%s/changes", event.WorkspaceID)
	return g.doWithRetry(http.MethodPost, url, body, nil)
}

func (g *gatewayStorage) LoadChangeEvents(workspaceID string) ([]ChangeEvent, error) {
	url := g.url("workspaces/%s/changes", workspaceID)
	var resp ChangeEventListResponse
	if err := g.doWithRetry(http.MethodGet, url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Events, nil
}

func (g *gatewayStorage) SaveDriftReport(report *DriftReport) error {
	body := CreateDriftReportRequest{Report: report}
	url := g.url("workspaces/%s/drift", report.WorkspaceID)
	return g.doWithRetry(http.MethodPost, url, body, nil)
}

func (g *gatewayStorage) LoadDriftReport(id string) (*DriftReport, error) {
	url := g.url("workspaces/_/drift/%s", id)
	var resp DriftReportResponse
	if err := g.doWithRetry(http.MethodGet, url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Report, nil
}

func (g *gatewayStorage) ListDriftReports(workspaceID string) ([]DriftReport, error) {
	url := g.url("workspaces/%s/drift", workspaceID)
	var resp DriftReportListResponse
	if err := g.doWithRetry(http.MethodGet, url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Reports, nil
}

// ── HTTP helpers ───────────────────────────────────────────────────────────────

// url builds a full API endpoint URL
func (g *gatewayStorage) url(pathFmt string, args ...interface{}) string {
	path := fmt.Sprintf(pathFmt, args...)
	return fmt.Sprintf("%s/api/%s/%s", g.cfg.BaseURL, g.cfg.APIVersion, path)
}

// doWithRetry executes an HTTP request with exponential-backoff retry.
// reqBody is JSON-marshalled into the request body (may be nil for GET).
// respBody is JSON-unmarshalled from the response body (may be nil to discard).
func (g *gatewayStorage) doWithRetry(method, url string, reqBody, respBody interface{}) error {
	var lastErr error
	delay := g.cfg.RetryDelay

	for attempt := 0; attempt <= g.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay *= 2
		}

		err := g.do(method, url, reqBody, respBody)
		if err == nil {
			return nil
		}
		lastErr = err

		// Only retry on transient errors (5xx, network)
		if !isRetryable(err) {
			return err
		}
	}
	return fmt.Errorf("after %d attempts: %w", g.cfg.MaxRetries+1, lastErr)
}

// do performs a single HTTP request
func (g *gatewayStorage) do(method, url string, reqBody, respBody interface{}) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshalling request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader) //nolint:noctx
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Attach authentication header
	switch g.cfg.AuthType {
	case GatewayAuthToken:
		req.Header.Set("Authorization", "Bearer "+g.cfg.Token)
	case GatewayAuthVaultToken:
		req.Header.Set("X-Vault-Token", g.cfg.Token)
	// GatewayAuthTLS: no header needed – the mTLS client cert is the credential
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return &retryableError{err: err}
	}
	defer resp.Body.Close() //nolint:errcheck

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	// Handle non-2xx as structured gateway errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var gErr GatewayError
		if jsonErr := json.Unmarshal(rawBody, &gErr); jsonErr != nil {
			gErr = GatewayError{Code: resp.StatusCode, Message: string(rawBody)}
		} else {
			gErr.Code = resp.StatusCode
		}
		if resp.StatusCode >= 500 {
			return &retryableError{err: &gErr}
		}
		return &gErr
	}

	// Unmarshal response body if requested
	if respBody != nil && len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, respBody); err != nil {
			return fmt.Errorf("unmarshalling response: %w", err)
		}
	}

	return nil
}

// ── Retry sentinel ─────────────────────────────────────────────────────────────

type retryableError struct{ err error }

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	_, ok := err.(*retryableError)
	return ok
}
