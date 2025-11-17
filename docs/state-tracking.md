# Terraform State Tracking & Drift Detection

## Overview

tfpipboy now includes comprehensive state tracking and drift detection capabilities inspired by Change Data Capture (CDC) patterns used in database systems. This feature automatically tracks all Terraform state changes and detects drift between desired and actual infrastructure state.

## Features

### 1. State Snapshots
- **Automatic Capture**: After each successful `terraform apply`, a snapshot of the complete state is captured
- **Versioning**: Each snapshot is versioned incrementally (v1, v2, v3, etc.)
- **Historical Tracking**: All snapshots are preserved, allowing you to view state at any point in time
- **Resource Details**: Full resource attributes, outputs, and dependencies are tracked

### 2. Change Data Capture (CDC)
- **Change Events**: Every modification to infrastructure is logged as a discrete change event
- **Operation Types**:
  - `create`: New resources added
  - `update`: Existing resources modified
  - `delete`: Resources removed
  - `recreate`: Resources replaced (delete + create)
- **Attribute-Level Tracking**: Detailed tracking of which specific attributes changed
- **Change History**: Complete audit trail of all infrastructure modifications

### 3. Drift Detection
- **Automatic Detection**: Analyzes `terraform plan` output to identify drift
- **Drift Types**:
  - `update`: Resources modified outside Terraform
  - `delete`: Resources deleted outside Terraform
  - `out-of-sync`: State inconsistencies
- **Detailed Reporting**: Identifies specific resources and attributes that have drifted
- **Severity Classification**: Critical (deletions) vs Warning (updates)

## Architecture

### Components

```
pkg/state/
├── types.go          # Core data structures
├── snapshot.go       # State snapshot management
├── tracker.go        # Change tracking (CDC)
├── drift.go          # Drift detection
├── storage.go        # Persistent storage
└── manager.go        # Unified facade
```

### Data Storage

State tracking data is stored in `.tfpipboy/state-tracking/`:

```
.tfpipboy/
└── state-tracking/
    ├── snapshots/
    │   └── <workspace-id>/
    │       ├── v1_<timestamp>.json
    │       ├── v2_<timestamp>.json
    │       └── latest.json -> v2_<timestamp>.json
    ├── changes/
    │   └── <workspace-id>/
    │       └── changes.jsonl       # JSON Lines format (CDC log)
    └── drift/
        └── <workspace-id>/
            ├── <timestamp>.json
            └── latest.json
```

### State Snapshot Structure

```json
{
  "id": "uuid",
  "workspace_id": "job-id",
  "module_path": "/path/to/module",
  "timestamp": "2025-11-17T12:00:00Z",
  "version": 1,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "abc-123",
  "resources": [
    {
      "address": "aws_instance.server",
      "type": "aws_instance",
      "name": "server",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "mode": "managed",
      "instances": [
        {
          "attributes": {
            "id": "i-1234567890",
            "instance_type": "t2.micro",
            "ami": "ami-12345"
          }
        }
      ],
      "dependencies": ["aws_vpc.main"]
    }
  ],
  "outputs": {
    "server_ip": "1.2.3.4"
  }
}
```

### Change Event Structure (CDC)

```json
{
  "id": "uuid",
  "workspace_id": "job-id",
  "snapshot_before": "snapshot-v1-id",
  "snapshot_after": "snapshot-v2-id",
  "timestamp": "2025-11-17T12:05:00Z",
  "operation": "update",
  "resource_address": "aws_instance.server",
  "resource_type": "aws_instance",
  "attribute_changes": [
    {
      "path": "instances[0].instance_type",
      "old_value": "t2.micro",
      "new_value": "t2.small"
    }
  ]
}
```

### Drift Report Structure

```json
{
  "id": "uuid",
  "workspace_id": "job-id",
  "timestamp": "2025-11-17T12:10:00Z",
  "has_drift": true,
  "drifted_resources": [
    {
      "address": "aws_instance.server",
      "type": "aws_instance",
      "drift_type": "update",
      "changes": [
        {
          "path": "tags",
          "old_value": "{\"Name\": \"Server\"}",
          "new_value": "{\"Name\": \"Server\", \"Env\": \"prod\"}"
        }
      ],
      "reason": "Resource has changed outside of Terraform"
    }
  ],
  "summary": {
    "total_resources": 10,
    "drifted_resources": 1,
    "updated_resources": 1,
    "deleted_resources": 0,
    "out_of_sync_count": 0
  }
}
```

## Integration

### Automatic Integration

State tracking is automatically integrated into the Terraform execution workflow:

1. **After `terraform apply`**: State snapshot is captured and changes are tracked
2. **During `terraform plan`**: Drift detection analyzes plan output for external changes
3. **All operations**: Events are logged to the message router and live board

### Executor Integration

The `TerraformExecutor` includes state tracking:

```go
// After successful apply
if err == nil && e.stateManager != nil {
    e.captureStateAfterApply(job, workspace, suppressOutput)
}
```

### Event Types

New event types are published to the event bus:

- `state_captured`: State snapshot successfully captured
- `drift_detected`: Drift identified in infrastructure

## Usage Examples

### Querying State History

```go
manager, _ := state.NewManager(".tfpipboy")

// Get latest snapshot
snapshot, _ := manager.GetSnapshot("workspace-id")
fmt.Printf("Current state: v%d with %d resources\n",
    snapshot.Version, len(snapshot.Resources))

// Get specific version
oldSnapshot, _ := manager.GetSnapshotByVersion("workspace-id", 1)

// List all snapshots
snapshots, _ := manager.ListSnapshots("workspace-id")
```

### Viewing Change History

```go
// Get all changes
history, _ := manager.GetChangeHistory("workspace-id")

for _, event := range history.Events {
    fmt.Printf("%s: %s %s\n",
        event.Timestamp,
        event.Operation,
        event.ResourceAddress)

    for _, change := range event.AttributeChanges {
        fmt.Printf("  %s: %v -> %v\n",
            change.Path,
            change.OldValue,
            change.NewValue)
    }
}
```

### Checking for Drift

```go
// Get latest drift report
report, _ := manager.GetLatestDriftReport("workspace-id")

if report.HasDrift {
    issues := state.ValidateDriftReport(report)
    for _, issue := range issues {
        fmt.Println(issue)
    }
}
```

## CDC Pattern Benefits

The Change Data Capture approach provides:

1. **Complete Audit Trail**: Every change is logged with full before/after details
2. **Point-in-Time Recovery**: Can analyze state at any historical moment
3. **Compliance**: Detailed records for auditing and compliance requirements
4. **Debugging**: Understand exactly when and how infrastructure changed
5. **Drift Analysis**: Compare desired vs actual state over time

## Storage Format

### JSON Lines (JSONL)

Change events use JSON Lines format for append-only CDC logs:
- Each line is a complete JSON object
- Efficient for streaming and incremental processing
- Easy to parse and analyze with standard tools
- Append-only for write performance

### Snapshot Files

Snapshots are stored as pretty-printed JSON for human readability:
- Easy to inspect manually
- Can be diffed with standard tools
- Symlinks point to latest version

## Performance Considerations

1. **Local State Only**: Currently tracks local tfstate files only
2. **Remote Backends**: State tracking is skipped for remote backends (can be enhanced to use `terraform state pull`)
3. **Storage Growth**: Snapshots are retained indefinitely (implement rotation as needed)
4. **Concurrent Access**: Thread-safe with RWMutex locking

## Future Enhancements

Potential improvements:

1. **Remote Backend Support**: Pull state from remote backends for tracking
2. **State Comparison UI**: Visual diff tool for comparing snapshots
3. **Automated Alerts**: Notify on critical drift detection
4. **State Retention Policy**: Configurable cleanup of old snapshots
5. **Export Capabilities**: Export change history to various formats
6. **Integration with CI/CD**: Drift detection in automated pipelines

## Testing

Comprehensive tests cover:

- Snapshot capture from tfstate files
- Change detection between snapshots
- Drift parsing from plan output
- Storage persistence and retrieval
- Concurrent access patterns

Run tests:
```bash
go test -v ./pkg/state/...
```

## Troubleshooting

### State Tracking Disabled

If you see "State tracking skipped (remote backend)":
- State tracking currently supports local tfstate files only
- Enhancement needed to support `terraform state pull` for remote backends

### Storage Permission Errors

Ensure `.tfpipboy/state-tracking` directory is writable:
```bash
chmod -R 755 .tfpipboy/state-tracking
```

### Drift Not Detected

- Drift detection parses `terraform plan` text output
- Ensure plan output contains drift indicators
- Run with `-detailed-exitcode` flag for accurate detection

## References

- [Terraform State Documentation](https://www.terraform.io/docs/language/state/index.html)
- [Change Data Capture (CDC) Pattern](https://en.wikipedia.org/wiki/Change_data_capture)
- [JSON Lines Format](https://jsonlines.org/)
