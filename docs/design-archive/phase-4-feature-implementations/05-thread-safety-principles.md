# Thread-Safety Principles for Parallel Execution

**Status**: CRITICAL DESIGN PRINCIPLE  
**Created**: 2025-11-11  
**Context**: Emerged from debugging race condition in output suppression during parallel job execution

## Overview

tfpipboy executes multiple Terraform jobs concurrently for performance. All code handling parallel execution **MUST be thread-safe by design**.

## The Thread-Safety Rule

**NEVER use shared mutable state in concurrent execution paths.**

Instead, use one of these thread-safe patterns:

1. **Immutable Parameters (PREFERRED)**: Pass values as function parameters through the call chain
2. **Synchronization**: Use mutexes to protect shared state (only when absolutely necessary)
3. **Message Passing**: Use channels for communication between goroutines
4. **Thread-Local Storage**: Each goroutine has its own copy of the data

## Real-World Case Study: Output Suppression Race Condition

### The Problem

Error boxes appeared randomly during LiveBoard execution despite having suppression checks in place. The behavior was non-deterministic - sometimes errors were suppressed correctly, sometimes they appeared on the console.

### Root Cause

Multiple goroutines executing jobs in parallel modified a shared field:

```go
// ❌ WRONG - Race condition!
type ParallelExecutor struct {
    executor *TerraformExecutor
}

type TerraformExecutor struct {
    suppressProgressBar bool  // SHARED MUTABLE STATE!
}

func (pe *ParallelExecutor) executeJobWithLiveUpdates(..., useLiveBoard bool) error {
    // Multiple goroutines modify the same field
    pe.executor.suppressProgressBar = useLiveBoard  // RACE CONDITION!
    defer func() {
        pe.executor.suppressProgressBar = originalValue
    }()
    
    // Execute job (reads suppressProgressBar)
    err := pe.executor.ExecuteJob(ctx, job)
    return err
}
```

### Why This Fails

When Job A and Job B run in parallel:

1. **T=0ms**: Job A goroutine sets `suppressProgressBar = true`
2. **T=5ms**: Job B goroutine sets `suppressProgressBar = false`
3. **T=10ms**: Job A encounters an error and reads `suppressProgressBar`
4. **Result**: Job A sees `false` instead of `true` → error box appears!

The defer also creates race conditions when goroutines finish at different times.

### The Solution: Immutable Parameters

Pass the value as a parameter through the entire call chain:

```go
// ✅ CORRECT - Thread-safe!
type TerraformExecutor struct {
    // NO shared mutable state for suppressOutput
}

func (pe *ParallelExecutor) executeJobWithLiveUpdates(..., useLiveBoard bool) error {
    // Pass useLiveBoard as immutable parameter
    err := pe.executor.ExecuteJobWithWorkspace(ctx, job, module, instance, useLiveBoard)
    return err
}

func (e *TerraformExecutor) ExecuteJobWithWorkspace(ctx context.Context, job *ExecutionJob, 
    module *Module, instance *Instance, suppressOutput bool) error {
    // Continue passing down the call chain
    return e.ExecuteJob(ctx, job, suppressOutput)
}

func (e *TerraformExecutor) ExecuteJob(ctx context.Context, job *ExecutionJob, 
    suppressOutput bool) error {
    // Pass to individual execute methods
    switch job.Operation {
    case OpInit:
        err = e.executeInit(ctx, job, workspace, suppressOutput)
    case OpPlan:
        err = e.executePlan(ctx, job, workspace, suppressOutput)
    // ... other operations
    }
    return err
}

func (e *TerraformExecutor) executeInit(ctx context.Context, job *ExecutionJob, 
    workspace *Workspace, suppressOutput bool) error {
    // Pass to command execution
    err := e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)
    return err
}

func (e *TerraformExecutor) runTerraformCommand(ctx context.Context, job *ExecutionJob, 
    workspace *Workspace, args []string, suppressOutput bool) error {
    // Use the parameter safely
    if !suppressOutput {
        e.printCategorizedErrorToConsole(job, tfError, args)
    }
    return err
}
```

### Why This Works

Each goroutine executing a job has its own immutable `suppressOutput` value:
- Job A (useLiveBoard=true) → suppressOutput=true throughout entire call chain
- Job B (useLiveBoard=false) → suppressOutput=false throughout entire call chain
- No shared state → no race condition → reliable behavior

## Thread-Safety Checklist

Before implementing parallel execution features, verify:

- [ ] **No shared mutable fields** modified by concurrent goroutines
- [ ] **Pass values as parameters** instead of storing in shared fields
- [ ] **Use mutexes** only when shared state is absolutely necessary
- [ ] **Document synchronization** strategy in comments if using mutexes
- [ ] **Test with race detector**: `go test -race ./...`
- [ ] **Test with high concurrency**: `--concurrent 16` or higher

## When Shared State Is Required

Sometimes shared state is unavoidable (e.g., LiveBoard updating job statuses from multiple goroutines). In these cases:

### Rules for Shared State

1. **Protect with mutex**: Always use `sync.Mutex` or `sync.RWMutex`
2. **Lock before read/write**: Acquire lock, do operation, release lock
3. **Keep critical sections small**: Minimize time holding the lock
4. **Document locking order**: Prevent deadlocks
5. **Never call external code while holding lock**: Avoid callbacks, defer functions

### Example: Properly Synchronized Shared State

```go
// ✅ CORRECT - Properly synchronized
type LiveBoard struct {
    mu       sync.RWMutex  // Protects all fields below
    jobs     map[string]*LiveJobStatus
    jobOrder []string
    isActive bool
}

func (lb *LiveBoard) UpdateJobStatus(jobID string, status JobStatus, progress string) {
    lb.mu.Lock()           // Acquire write lock
    defer lb.mu.Unlock()   // Always release lock (even if panic)
    
    if job, exists := lb.jobs[jobID]; exists {
        job.Status = status
        job.Progress = progress
        // Safe - protected by mutex
    }
}

func (lb *LiveBoard) GetJobStatus(jobID string) (JobStatus, bool) {
    lb.mu.RLock()          // Acquire read lock (multiple readers OK)
    defer lb.mu.RUnlock()
    
    job, exists := lb.jobs[jobID]
    if !exists {
        return JobStatusPending, false
    }
    return job.Status, true
}
```

### Example: Message Passing (Channels)

For event-driven updates, use channels instead of shared state:

```go
// ✅ CORRECT - Channel-based communication
type StatusUpdateEvent struct {
    JobID     string
    EventType string
    Status    JobStatus
    Progress  string
}

type LiveBoard struct {
    eventChan chan StatusUpdateEvent
    mu        sync.RWMutex
    jobs      map[string]*LiveJobStatus
}

// Goroutines send events (no shared state modification)
func (pe *ParallelExecutor) executeJob(job *ExecutionJob) {
    // Send status update event
    pe.liveBoard.eventChan <- StatusUpdateEvent{
        JobID:    job.ID,
        Status:   JobStatusRunning,
        Progress: "Initializing...",
    }
}

// Single goroutine processes all events (serialized access)
func (lb *LiveBoard) eventProcessingLoop() {
    for event := range lb.eventChan {
        lb.mu.Lock()
        if job, exists := lb.jobs[event.JobID]; exists {
            job.Status = event.Status
            job.Progress = event.Progress
        }
        lb.mu.Unlock()
    }
}
```

## Common Pitfalls to Avoid

### ❌ Pitfall 1: Defer with Shared State

```go
// ❌ WRONG - Defer creates race condition
func executeJob(job *ExecutionJob) {
    sharedState.field = newValue
    defer func() {
        sharedState.field = oldValue  // RACE: Multiple goroutines modifying
    }()
}
```

### ❌ Pitfall 2: Reading Then Writing

```go
// ❌ WRONG - Race condition between read and write
if !sharedState.initialized {  // Goroutine A reads: false
    sharedState.initialized = true  // Goroutine A writes: true
    // Goroutine B might also read false and write true (duplicate init)
}
```

### ✅ Solution: Atomic Operations or Mutex

```go
// ✅ CORRECT - Mutex protects read-modify-write
mutex.Lock()
if !sharedState.initialized {
    sharedState.initialized = true
    doInitialization()
}
mutex.Unlock()

// ✅ CORRECT - sync.Once for one-time initialization
var once sync.Once
once.Do(func() {
    doInitialization()
})
```

### ❌ Pitfall 3: Map Access Without Protection

```go
// ❌ WRONG - Concurrent map writes panic!
func updateStatus(jobID string, status Status) {
    statusMap[jobID] = status  // PANIC if concurrent writes
}
```

### ✅ Solution: sync.Map or Mutex-Protected Map

```go
// ✅ CORRECT - Use sync.Map for concurrent access
var statusMap sync.Map

func updateStatus(jobID string, status Status) {
    statusMap.Store(jobID, status)
}

// ✅ CORRECT - Mutex-protected regular map
type SafeStatusMap struct {
    mu   sync.RWMutex
    data map[string]Status
}

func (m *SafeStatusMap) Update(jobID string, status Status) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.data[jobID] = status
}
```

## Testing for Race Conditions

### 1. Race Detector

Always run tests with the race detector:

```bash
# Run all tests with race detection
go test -race ./...

# Run specific package
go test -race ./pkg/orchestrator

# Run with higher concurrency to expose races
go test -race -parallel 16 ./...
```

### 2. Manual Stress Testing

Test with high concurrency in real scenarios:

```bash
# Execute many jobs in parallel
tfpipboy --targets-all --parallel-all --concurrent 16 --operation plan

# Run multiple times to catch non-deterministic issues
for i in {1..10}; do
    tfpipboy --targets-all --parallel-all --concurrent 8 --operation plan
done
```

### 3. Code Review Checklist

For any PR touching parallel execution:

- [ ] No new shared mutable state introduced
- [ ] All shared state properly synchronized
- [ ] Parameters passed instead of modifying fields
- [ ] Race detector tests pass
- [ ] Manual testing with `--concurrent 16` successful

## Additional Resources

- [Go Memory Model](https://go.dev/ref/mem) - Understanding Go's concurrency guarantees
- [Race Detector](https://go.dev/doc/articles/race_detector) - Finding race conditions
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency) - Best practices
- [Mutex vs Channels](https://github.com/golang/go/wiki/MutexOrChannel) - When to use which
- [sync.Map](https://pkg.go.dev/sync#Map) - Concurrent map implementation

## Conclusion

Thread-safety is **non-negotiable** for tfpipboy's parallel execution. By following the principle of avoiding shared mutable state and using immutable parameters, we ensure reliable and predictable concurrent behavior.

**When in doubt, pass it as a parameter.**
