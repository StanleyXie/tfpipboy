# Version-to-Version Change Tracking

## Overview

tfpipboy tracks changes between every state version, making it easy to understand what changed from version to version. Using full snapshots, the system can compute detailed diffs at any time without requiring incremental deltas.

## Key Capabilities

### 1. **Version Diff** - Compare Any Two Versions
Compare any two state versions to see exactly what changed:

```go
manager, _ := state.NewManager(".tfpipboy")

// Compare version 5 with version 10
diff, _ := manager.DiffVersions("workspace-id", 5, 10)

// Print human-readable diff
fmt.Println(state.FormatVersionDiff(diff))
```

**Output Example:**
```
=== Version Diff: v5 → v10 ===
Workspace: prod-infrastructure
Duration: 2h 30m
Timestamp: 2025-11-17 10:00:00 → 2025-11-17 12:30:00

Summary:
  Resources: 45 unchanged, 3 changed, 2 added, 1 removed
  Attributes: 7 changed
  Outputs: 2 changed
  ⚠️  Significant infrastructure change detected

Added Resources (2):
  + aws_s3_bucket.new_data (aws_s3_bucket)
  + aws_lambda_function.processor (aws_lambda_function)

Removed Resources (1):
  - aws_instance.old_server (aws_instance) [Impact: high]

Modified Resources (3):
  ⚠ aws_instance.web (aws_instance) [Impact: high, 2 attributes]
      instance_type: t2.micro → t2.large
      ami: ami-12345 → ami-67890
  ~ aws_vpc.main (aws_vpc) [Impact: low, 1 attributes]
      tags.Name: Old → New
```

### 2. **Timeline View** - Chronological Change History
See all versions and changes over time:

```go
timeline, _ := manager.GetTimeline("workspace-id")

fmt.Printf("=== Infrastructure Timeline ===\n")
fmt.Printf("Versions: v%d → v%d\n", timeline.FromVersion, timeline.ToVersion)
fmt.Printf("Total Changes: %d\n", timeline.Summary.TotalChanges)
fmt.Printf("Most Active Version: v%d\n", timeline.Summary.MostActiveVersion)

for _, entry := range timeline.Entries {
    fmt.Printf("v%-3d %s | %3d resources | %s\n",
        entry.Version,
        entry.Timestamp.Format("2006-01-02 15:04:05"),
        entry.ResourceCount,
        entry.ChangesSummary)
}
```

**Output Example:**
```
=== Infrastructure Timeline ===
Versions: v1 → v15
Total Changes: 47
Most Active Version: v8

v1   2025-11-17 09:00:00 |  20 resources | no changes
v2   2025-11-17 09:30:00 |  22 resources | +2 added
v3   2025-11-17 10:00:00 |  22 resources | ~1 changed
v4   2025-11-17 10:30:00 |  25 resources | +3 added
v5   2025-11-17 11:00:00 |  24 resources | -1 removed, ~2 changed
...
```

### 3. **Resource History** - Track Specific Resources
See how a specific resource evolved over time:

```go
history, _ := manager.GetResourceHistory("workspace-id", "aws_instance.web_server")

fmt.Printf("=== Resource History: %s ===\n", history.ResourceAddress)
fmt.Printf("Type: %s\n", history.ResourceType)
fmt.Printf("Total Changes: %d\n", history.TotalChanges)

for _, ver := range history.Versions {
    fmt.Printf("v%-3d %s", ver.Version, ver.Timestamp.Format("2006-01-02 15:04:05"))

    if len(ver.ChangesFromPrev) > 0 {
        fmt.Printf(" | %d attributes changed:\n", len(ver.ChangesFromPrev))
        for _, change := range ver.ChangesFromPrev {
            fmt.Printf("      - %s: %v → %v\n", change.Path, change.OldValue, change.NewValue)
        }
    }
}
```

**Output Example:**
```
=== Resource History: aws_instance.web_server ===
Type: aws_instance
Total Changes: 5

v1   2025-11-17 09:00:00
v2   2025-11-17 09:30:00 | 1 attributes changed:
      - instance_type: t2.micro → t2.small
v3   2025-11-17 10:00:00
v4   2025-11-17 10:30:00 | 2 attributes changed:
      - instance_type: t2.small → t2.medium
      - tags.Environment: dev → staging
v5   2025-11-17 11:00:00 | 1 attributes changed:
      - ami: ami-12345 → ami-67890
```

### 4. **Version Changes** - What Changed in Latest Version
Quickly see what changed in the most recent apply:

```go
// Get latest snapshot
snapshot, _ := manager.GetSnapshot("workspace-id")

// Compare with previous version
diff, _ := manager.GetVersionChanges("workspace-id", snapshot.Version)

fmt.Printf("Changes in v%d:\n", snapshot.Version)
fmt.Println(state.FormatVersionDiff(diff))
```

## API Reference

### DiffVersions
```go
func (m *Manager) DiffVersions(workspaceID string, fromVersion, toVersion int) (*VersionDiff, error)
```

Compares two specific versions and returns detailed differences.

**Parameters:**
- `workspaceID`: Workspace identifier
- `fromVersion`: Starting version number
- `toVersion`: Ending version number

**Returns:**
- `*VersionDiff`: Detailed diff with summary, resource changes, and output changes

### GetVersionChanges
```go
func (m *Manager) GetVersionChanges(workspaceID string, version int) (*VersionDiff, error)
```

Gets changes for a specific version compared to the previous version.

**Parameters:**
- `workspaceID`: Workspace identifier
- `version`: Version number to analyze (must be > 1)

**Returns:**
- `*VersionDiff`: Diff between `version-1` and `version`

### GetTimeline
```go
func (m *Manager) GetTimeline(workspaceID string) (*ChangeTimeline, error)
```

Creates a chronological view of all changes across all versions.

**Returns:**
- `*ChangeTimeline`: Complete timeline with all version entries and summary statistics

### GetResourceHistory
```go
func (m *Manager) GetResourceHistory(workspaceID string, resourceAddress string) (*ResourceHistory, error)
```

Gets the complete history of a specific resource across all versions.

**Parameters:**
- `workspaceID`: Workspace identifier
- `resourceAddress`: Full resource address (e.g., "aws_instance.web_server")

**Returns:**
- `*ResourceHistory`: Complete version history for the resource

## Data Structures

### VersionDiff
```go
type VersionDiff struct {
    WorkspaceID      string
    FromVersion      int
    ToVersion        int
    Duration         time.Duration
    Summary          DiffSummary
    ResourceChanges  []ResourceDiff
    OutputChanges    []OutputChange
}
```

### DiffSummary
```go
type DiffSummary struct {
    TotalResourcesAdded     int
    TotalResourcesRemoved   int
    TotalResourcesChanged   int
    TotalResourcesUnchanged int
    TotalAttributesChanged  int
    TotalOutputsChanged     int
    SignificantChange       bool  // Major infrastructure change
}
```

### ResourceDiff
```go
type ResourceDiff struct {
    Address          string
    Type             string
    ChangeType       ResourceChangeType  // added, removed, modified, recreated
    Before           *ResourceState      // State before change
    After            *ResourceState      // State after change
    AttributeChanges []AttributeChange   // Detailed attribute changes
    Impact           ChangeImpact        // low, medium, high
}
```

### ChangeTimeline
```go
type ChangeTimeline struct {
    WorkspaceID string
    FromVersion int
    ToVersion   int
    Entries     []TimelineEntry
    Summary     TimelineSummary
}
```

### ResourceHistory
```go
type ResourceHistory struct {
    ResourceAddress string
    ResourceType    string
    FirstSeen       time.Time
    LastSeen        time.Time
    TotalChanges    int
    Versions        []ResourceVersionEntry
}
```

## Use Cases

### 1. Post-Apply Review
After running `terraform apply`, review what changed:

```go
snapshot, _ := manager.GetSnapshot("workspace-id")
diff, _ := manager.GetVersionChanges("workspace-id", snapshot.Version)

if diff.Summary.SignificantChange {
    fmt.Println("⚠️  Significant change detected!")
}

// Check for high-impact changes
for _, rc := range diff.ResourceChanges {
    if rc.Impact == state.ImpactHigh {
        fmt.Printf("High-Impact: %s (%s)\n", rc.Address, rc.ChangeType)
    }
}
```

### 2. Audit & Compliance
Track all changes over a time period:

```go
timeline, _ := manager.GetTimeline("workspace-id")

fmt.Printf("Audit Report:\n")
fmt.Printf("Time Period: %v\n", timeline.Summary.TimeSpan)
fmt.Printf("Total Changes: %d\n", timeline.Summary.TotalChanges)

for _, entry := range timeline.Entries {
    if entry.ChangesFromPrev > 0 {
        fmt.Printf("v%d: %d changes at %s\n",
            entry.Version,
            entry.ChangesFromPrev,
            entry.Timestamp)
    }
}
```

### 3. Rollback Planning
Understand what will change if you rollback to a previous version:

```go
// Current version
current, _ := manager.GetSnapshot("workspace-id")

// Compare with version 2 days ago
diff, _ := manager.DiffVersions("workspace-id", 5, current.Version)

fmt.Printf("Rollback Impact (v%d → v5):\n", current.Version)
fmt.Printf("Will add: %d resources\n", diff.Summary.TotalResourcesRemoved)  // Reversed
fmt.Printf("Will remove: %d resources\n", diff.Summary.TotalResourcesAdded) // Reversed
fmt.Printf("Will revert: %d resources\n", diff.Summary.TotalResourcesChanged)
```

### 4. Change Tracking by Resource Type
Find all changes to a specific resource type:

```go
timeline, _ := manager.GetTimeline("workspace-id")

fmt.Println("Changes to AWS Instances:")
for _, entry := range timeline.Entries {
    for _, resource := range entry.ChangedResources {
        if strings.HasPrefix(resource, "aws_instance.") {
            fmt.Printf("v%d: %s\n", entry.Version, resource)
        }
    }
}
```

### 5. Significant Change Alerts
Monitor for major infrastructure changes:

```go
snapshots, _ := manager.ListSnapshots("workspace-id")

for i := 1; i < len(snapshots); i++ {
    diff, _ := manager.DiffVersions("workspace-id", i, i+1)

    if diff.Summary.SignificantChange {
        fmt.Printf("⚠️  Alert: Significant change in v%d\n", i+1)
        fmt.Printf("   Resources removed: %d\n", diff.Summary.TotalResourcesRemoved)
        fmt.Printf("   Resources added: %d\n", diff.Summary.TotalResourcesAdded)

        // Send alert notification
        sendAlert(diff)
    }
}
```

## Impact Assessment

The system automatically assesses the impact of changes:

### Impact Levels

- **High Impact**: Resource deletions, recreations, changes to critical fields (id, arn, ami, instance_type, availability_zone)
- **Medium Impact**: Changes to networking (security_groups, subnet_id, vpc_id)
- **Low Impact**: Tag changes, configuration updates

### Example
```go
for _, rc := range diff.ResourceChanges {
    switch rc.Impact {
    case state.ImpactHigh:
        fmt.Printf("🔴 HIGH: %s\n", rc.Address)
    case state.ImpactMedium:
        fmt.Printf("🟡 MEDIUM: %s\n", rc.Address)
    case state.ImpactLow:
        fmt.Printf("🟢 LOW: %s\n", rc.Address)
    }
}
```

## Best Practices

### 1. Review Changes After Each Apply
Always check what changed after running terraform apply:

```go
// After terraform apply succeeds
snapshot, _ := manager.GetSnapshot(workspaceID)
diff, _ := manager.GetVersionChanges(workspaceID, snapshot.Version)

if diff.Summary.SignificantChange {
    // Require manual review
    fmt.Println("⚠️  Manual review required")
    fmt.Println(state.FormatVersionDiff(diff))
}
```

### 2. Track Long-Running Resources
Monitor resources that change frequently:

```go
history, _ := manager.GetResourceHistory(workspaceID, "aws_instance.web")

if history.TotalChanges > 10 {
    fmt.Printf("⚠️  %s has changed %d times - investigate instability\n",
        history.ResourceAddress, history.TotalChanges)
}
```

### 3. Compare Before Major Changes
Before applying major changes, understand the current state:

```go
// Before running terraform apply
snapshots, _ := manager.ListSnapshots(workspaceID)
latestVersion := snapshots[len(snapshots)-1].Version

fmt.Printf("Current state: v%d with %d resources\n",
    latestVersion,
    len(snapshots[len(snapshots)-1].Resources))

// After apply, compare
// ...
```

## Storage

All data is stored using full snapshots (no deltas):

```
.tfpipboy/state-tracking/
└── workspace-123/
    ├── snapshots/
    │   ├── v1_<timestamp>.json     (complete state)
    │   ├── v2_<timestamp>.json     (complete state)
    │   └── v3_<timestamp>.json     (complete state)
    └── changes/
        └── changes.jsonl           (CDC event log)
```

**Benefits:**
- ✅ Fast queries - no reconstruction needed
- ✅ Can compute any diff on-demand
- ✅ No delta corruption risk
- ✅ Simple backup/restore
- ✅ Easy to inspect manually

**Trade-offs:**
- ⚠️ Higher disk usage (mitigated by JSON compression)
- ⚠️ Not as space-efficient as pure delta approach

## Performance

### Query Performance

| Operation | Time Complexity | Notes |
|-----------|----------------|-------|
| Get snapshot | O(1) | Direct file read |
| Diff versions | O(n) | n = number of resources |
| Timeline | O(m*n) | m = versions, n = resources |
| Resource history | O(m*n) | Optimized with indexing |

### Optimization Tips

1. **Limit timeline queries**: Use date ranges or version ranges
2. **Cache diffs**: Store frequently accessed diffs
3. **Index resources**: Create resource → version index for faster history queries
4. **Compress snapshots**: Use gzip for storage

## Examples

See `/examples/state_tracking.go` for complete working examples.

## See Also

- [State Tracking Overview](state-tracking.md) - Complete feature documentation
- [CDC Pattern](state-tracking.md#cdc-pattern-benefits) - Change Data Capture architecture
- [Drift Detection](state-tracking.md#drift-detection) - Detecting external changes
