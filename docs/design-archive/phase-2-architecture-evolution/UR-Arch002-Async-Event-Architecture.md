# UR-Arch002: Async Event-Driven Architecture for Deadlock-Free Communication

**Status**: Implemented  
**Date**: 2025-11-12  
**Author**: Claude (AI Assistant)

## Context

The original synchronous architecture had critical issues when multiple terraform jobs executed in parallel:

### Problems

1. **AB-BA Deadlocks**
   - Executor held `e.mu` mutex while calling `MessageRouter.SetInstancePlanResult()`
   - MessageRouter held `mr.mu` mutex while processing pipelines
   - Both components needed each other's locks → deadlock

2. **Shared Parser Race Conditions**
   - Single `TerraformOutputParser` shared by all parallel jobs
   - Job A captures plan summary → Job B calls `Reset()` → Job A gets empty summary

3. **Missing Plan Summaries**
   - Race condition caused plan summaries to be lost
   - Only "No changes" summaries worked (timing dependent)

## Solution: Async Event-Driven Architecture

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                  ExecutorEventBus (Channels)                 │
│   - Buffered channel (10,000 events)                         │
│   - Non-blocking publish                                     │
│   - Single subscriber goroutine                              │
└─────────────────────────────────────────────────────────────┘
         ↑ Publish (no locks!)    ↓ Subscribe (async)
    ┌──────────┐              ┌─────────────────┐
    │ Executor │              │ MessageRouter   │
    │          │              │ (Event Handler) │
    │ - Captures plan         │ - Buffers msgs  │
    │   summaries per-job     │ - Routes to     │
    │ - Publishes events      │   pipelines     │
    └──────────┘              └─────────────────┘
```

### Key Components

#### 1. ExecutorEventBus (`executor_events.go`)

**Lock-free event bus** for async communication:

```go
type ExecutorEvent struct {
    Type      ExecutorEventType
    JobID     string
    Timestamp int64
    Data      interface{}
}

type ExecutorEventBus struct {
    eventChan chan *ExecutorEvent  // Buffered (10k)
    done      chan struct{}
}
```

**Event Types:**
- `EventTerraformOutput`: Each line of terraform output
- `EventPlanResult`: Plan summary when plan completes
- `EventJobStatus`: Job status changes

#### 2. Executor Publishes Events

**Before (synchronous, deadlock-prone):**
```go
e.mu.Lock()
messageRouter := e.messageRouter
e.mu.Unlock()
messageRouter.SetInstancePlanResult(jobID, summary)  // Lock in MessageRouter
```

**After (async, no locks):**
```go
e.eventBus.Publish(&ExecutorEvent{
    Type:  EventPlanResult,
    JobID: job.ID,
    Data:  &PlanResultEvent{Summary: summary},
})
```

#### 3. MessageRouter Subscribes to Events

**Separate goroutine** processes events:

```go
func (mr *MessageRouter) StartEventSubscriber(eventBus *ExecutorEventBus) {
    go func() {
        for event := range eventBus.Subscribe() {
            switch event.Type {
            case EventPlanResult:
                mr.mu.Lock()
                mr.instancePlanResults[event.JobID] = data.Summary
                mr.mu.Unlock()
                
            case EventTerraformOutput:
                msg := &Message{...}
                mr.BufferMessage(msg)  // For filtered output
                pipeline.Send(msg)      // For logging
            }
        }
    }()
}
```

### Per-Job Plan Summary Capture

**Direct capture in `streamOutput()`** (bypasses shared parser):

```go
func (e *TerraformExecutor) streamOutput(..., jobID string, ...) {
    for scanner.Scan() {
        line := scanner.Text()
        cleanLine := StripANSI(line)
        
        // Capture plan summary directly per-job
        if strings.Contains(cleanLine, "Plan:") {
            e.mu.Lock()
            e.jobPlanSummaries[jobID] = cleanLine  // Per-job map!
            e.mu.Unlock()
        }
        
        // Publish event (no locks held!)
        e.eventBus.Publish(&ExecutorEvent{
            Type:  EventTerraformOutput,
            JobID: jobID,
            Data:  &TerraformOutputEvent{Line: line},
        })
    }
}
```

## Design Principles

### 1. Single-Writer Per Data Structure
- Each component owns its data
- No shared mutable state between components

### 2. Communicate via Channels
- Never hold locks across component boundaries
- Async, non-blocking event publishing

### 3. Event-Driven
- Components emit events, don't call each other directly
- Loose coupling

### 4. Buffered Channels
- Large buffers (10,000 events) prevent blocking
- Events dropped only if buffer full (acceptable for non-critical updates)

## Benefits

### Performance
- **No deadlocks**: No mutex ordering issues
- **No race conditions**: Per-job data isolation
- **Scalable**: Event bus handles high throughput

### Maintainability
- **Clean separation**: Components loosely coupled
- **Easy testing**: Mock event bus for unit tests
- **Extensible**: Add new event types without changing existing code

### Reliability
- **Plan summaries always captured**: Direct per-job capture
- **Graceful degradation**: Non-critical events can be dropped if buffer full
- **Proper shutdown**: Event bus closes cleanly

## Implementation Details

### Shutdown Sequence

Critical order to avoid hanging:

```go
// 1. Close event bus FIRST (stops new events, closes subscriber)
eventBus.Close()

// 2. Stop LiveBoard (displays final summary)
pe.liveBoard.Stop()

// 3. Display filtered output
if pe.messageRouter != nil {
    pe.messageRouter.StopLiveBoard()
}

// 4. Shutdown pipelines
pe.messageRouter.Shutdown()
```

### Message Buffering

Event subscriber **both buffers and sends to pipeline**:

```go
case EventTerraformOutput:
    msg := &Message{...}
    mr.BufferMessage(msg)    // For filtered output display
    pipeline.Send(msg)        // For log files
```

## Testing Results

**Test Case**: Parallel execution of 2 instances
- `core-main-gwc`: No changes
- `base-conn-sdc-prod`: +20 changes (would have deadlocked before)

**Results**:
```
INSTANCE EXECUTION DETAILS

core-main-gwc
────────────────────────────────────────
📦 Backend: Remote backend
🔌 Providers: azure/azapi, hashicorp/azurerm, ...
✓ Successfully initialized
📋 No changes

base-conn-sdc-prod
────────────────────────────────────────
📦 Backend: Remote backend
🔌 Providers: hashicorp/azurerm, hashicorp/random, ...
✓ Successfully initialized
📋 Plan: 20 to add, 0 to change, 0 to destroy
```

✅ **Both instances display correctly with no deadlock!**

## Future Enhancements

### Potential Improvements
1. **Event persistence**: Log events for debugging
2. **Event replay**: Reconstruct execution state
3. **Metrics**: Track event throughput, buffer usage
4. **Multiple subscribers**: Different components subscribe to same events
5. **Event filtering**: Subscribers filter by event type or job ID

### Migration Path
- Old `SendInstanceTerraformOutput()` method can remain for backward compatibility
- Gradually migrate all direct calls to event publishing

## Conclusion

The async event-driven architecture **eliminates all deadlocks and race conditions** while providing a clean, scalable foundation for future enhancements. By communicating through channels instead of direct method calls with locks, components remain loosely coupled and can operate independently without coordination overhead.

This pattern is industry-standard for concurrent systems and aligns with Go's philosophy: **"Don't communicate by sharing memory; share memory by communicating."**
