# Live Board - Real-time Parallel Execution Monitoring

## Overview

The Live Board is a real-time dashboard that provides visual feedback during parallel Terraform module executions. It displays the status, progress, and timing of each instance being executed simultaneously, making it easy to monitor complex orchestrations at a glance.

## Features

### Real-time Status Display
- **Color-coded Status Indicators**: Visual feedback with colors and icons
  - `✓` Green: Completed successfully
  - `✗` Red: Failed with error
  - `▶` Blue: Currently running
  - `○` Gray: Pending/waiting to start
  - `⊘` Yellow: Skipped

### Live Progress Tracking
- **Spinner Animation**: Rotating spinner for running jobs
- **Elapsed Time**: Real-time duration tracking for each instance
- **Progress Messages**: Current operation phase (e.g., "Initializing...", "Planning...")
- **Error Display**: First line of error message for failed jobs

### Summary Statistics
- Total jobs count
- Completed count
- Running count
- Failed count
- Pending count

### Smart Activation
- **Automatic**: Only activates when executing 2+ jobs in parallel
- **Non-intrusive**: Single job executions use traditional output
- **Graceful**: Falls back if terminal doesn't support required features

## Visual Layout

```
╭─────────────────────────────────────────────────────────────────────────────╮
│ PARALLEL EXECUTION IN PROGRESS                               Elapsed: 00:15s │
╰─────────────────────────────────────────────────────────────────────────────╯
  ▶ ⠋ seed                 Initializing...                       [  15s]
  ✓ core                   Completed successfully                [  12s]
  ▶ ⠹ baseline.connectivity Planning...                          [  10s]
  ○ vending.connectivity   Waiting...                            [   -  ]
  ✗ dns.connectivity       Failed                                [   8s] - Error: module not found
─────────────────────────────────────────────────────────────────────────────
Total: 5  |  Completed: 1  |  Running: 2  |  Failed: 1  |  Pending: 1
```

### Layout Components

1. **Header Box**
   - Title: "PARALLEL EXECUTION IN PROGRESS"
   - Overall elapsed time
   - Blue border with Unicode box drawing characters

2. **Job Status Rows**
   - Status icon + spinner (for running jobs)
   - Instance name (truncated to 20 chars if needed)
   - Progress message (truncated to 30 chars if needed)
   - Duration in brackets (or "-" if not started)
   - Error message (if failed, truncated to 30 chars)

3. **Footer**
   - Separator line
   - Summary statistics

## Usage

### Automatic Activation

The Live Board automatically activates when:
- Using `--parallel-all` flag
- Executing multiple jobs with `parallel: true` in config
- Multiple instances in the same stage with dependencies allowing parallelism

### Example Commands

```bash
# Parallel execution with live board
tfpipboy --config . --targets seed,core,baseline.connectivity --operation plan --parallel-all

# Multiple stages with parallel instances
tfpipboy --config . --pipeline full-deployment --operation apply
```

### Configuration

No configuration needed - Live Board activates automatically based on execution plan.

Future configuration options may include:
```yaml
# .tfpipboy/config.yaml (future)
display:
  live_board:
    enabled: true
    refresh_rate: 200ms  # Update frequency
    max_name_length: 20  # Instance name truncation
    max_progress_length: 30  # Progress message truncation
    show_errors: true  # Display error messages inline
```

## Technical Details

### Architecture

**Component**: `LiveBoard` struct in `pkg/orchestrator/liveboard.go`

**Key Methods**:
- `Start(jobs []*ExecutionJob)` - Initialize and begin display
- `Stop()` - Finalize and restore terminal
- `UpdateJobStatus(jobID, status, progress)` - Update job state
- `UpdateJobProgress(jobID, progress)` - Update progress message only
- `UpdateJobError(jobID, error)` - Mark job as failed with error

**Refresh Loop**:
- Runs at 200ms intervals (5 FPS)
- Updates spinner frames
- Redraws entire board to terminal
- Uses ANSI escape codes for cursor positioning

### Terminal Control

**ANSI Escape Sequences Used**:
- `\033[?25l` - Hide cursor (on start)
- `\033[?25h` - Show cursor (on stop)
- `\033[NA` - Move cursor up N lines
- `\033[0J` - Clear from cursor to end of screen
- `\033[1A` - Move cursor up one line (for spinner refresh)

**Color Codes**:
- `\033[0m` - Reset
- `\033[31m` - Red (failed)
- `\033[32m` - Green (completed)
- `\033[33m` - Yellow (skipped)
- `\033[34m` - Blue (running/headers)
- `\033[36m` - Cyan (info)
- `\033[90m` - Gray (pending)
- `\033[1m` - Bold

### Integration with Parallel Executor

The Live Board is integrated into `ParallelExecutor.ExecuteJobs()`:

1. **Initialize**: Create LiveBoard before starting jobs
2. **Start**: Call `Start()` and print initial empty board space
3. **Update**: Jobs update status via LiveBoard methods during execution
4. **Stop**: Call `Stop()` after all jobs complete

**Thread Safety**: All LiveBoard methods use mutex locks for safe concurrent access from multiple goroutines.

## Performance Considerations

### Minimal Overhead
- Refresh rate: 200ms (5 updates/second)
- Only active during parallel execution
- Lightweight terminal operations

### Resource Usage
- Single goroutine for refresh loop
- Minimal memory (tracks status for each job)
- CPU usage negligible (simple string formatting)

### Scalability
- Tested with up to 20 parallel jobs
- Display truncates long names/messages automatically
- Footer statistics update in O(n) time

## Comparison with Traditional Output

### Before (Traditional Output)
```
[2025-11-08 20:50:56] INFO: Starting job seed
[2025-11-08 20:50:58] INFO: Starting job core
[2025-11-08 20:51:00] INFO: Starting job baseline.connectivity
[2025-11-08 20:51:05] INFO: Job seed completed
[2025-11-08 20:51:08] ERROR: Job dns.connectivity failed: module not found
[2025-11-08 20:51:10] INFO: Job core completed
[2025-11-08 20:51:12] INFO: Job baseline.connectivity completed
```

**Issues**:
- Hard to see overall progress
- Logs interleaved from parallel jobs
- No real-time status of running jobs
- Difficult to identify which jobs are still running

### After (Live Board)
```
╭─────────────────────────────────────────────────────────────────────────────╮
│ PARALLEL EXECUTION IN PROGRESS                               Elapsed: 00:15s │
╰─────────────────────────────────────────────────────────────────────────────╯
  ✓ seed                   Completed successfully                [  12s]
  ✓ core                   Completed successfully                [  15s]
  ✓ baseline.connectivity  Completed successfully                [  13s]
  ✗ dns.connectivity       Failed                                [   8s] - Error: module not found
─────────────────────────────────────────────────────────────────────────────
Total: 4  |  Completed: 3  |  Running: 0  |  Failed: 1  |  Pending: 0
```

**Benefits**:
- ✅ Clear overview of all jobs
- ✅ Real-time status updates
- ✅ Easy to identify problems
- ✅ Progress visible at a glance
- ✅ Professional appearance

## Future Enhancements

### Phase 2 (Planned)
- [ ] Detailed progress percentage for Terraform operations
- [ ] Resource counts (adding/changing/destroying)
- [ ] Collapsible error details (press key to expand)
- [ ] Color themes and customization
- [ ] Export live board snapshots to logs

### Phase 3 (Considered)
- [ ] Interactive controls (pause/resume/cancel individual jobs)
- [ ] Terminal size adaptation (responsive layout)
- [ ] Multiple page support for 50+ parallel jobs
- [ ] Integration with CI/CD (non-interactive mode with JSON output)
- [ ] WebSocket streaming for remote monitoring

### Phase 4 (Future)
- [ ] TUI mode with keyboard navigation
- [ ] Historical replay of execution
- [ ] Performance graphs (CPU/memory per job)
- [ ] Dependency graph visualization

## Troubleshooting

### Live Board Not Appearing

**Problem**: Live board doesn't show during parallel execution

**Possible Causes**:
1. Only one job in execution (Live Board requires 2+)
2. Terminal doesn't support ANSI escape codes
3. Output redirected to file/pipe

**Solution**: 
- Ensure `--parallel-all` flag or multiple parallel jobs
- Use terminal that supports ANSI (most modern terminals)
- Avoid piping output: `tfpipboy ... | tee log.txt` won't show Live Board

### Broken Display

**Problem**: Characters garbled or display corrupted

**Possible Causes**:
1. Terminal window too small
2. Unicode box characters not supported
3. Terminal encoding issues

**Solution**:
- Resize terminal to at least 80 columns
- Use UTF-8 terminal encoding
- Set `LANG=en_US.UTF-8` environment variable

### Missing Status Updates

**Problem**: Jobs shown as "Waiting..." but are actually running

**Possible Causes**:
1. Job execution not calling update methods
2. Race condition in status updates

**Solution**:
- Check logs for actual job status
- Report issue with reproduction steps

## Examples

### Example 1: Simple Parallel Plan
```bash
tfpipboy --config . \
  --targets seed,core,baseline.connectivity \
  --operation plan \
  --parallel-all
```

**Expected Output**: Live Board with 3 instances running in parallel

### Example 2: Pipeline with Parallel Stages
```bash
tfpipboy --config . \
  --pipeline full-deployment \
  --operation apply
```

**Expected Output**: Live Board appears for each stage with parallel jobs

### Example 3: Mixed Sequential and Parallel
```yaml
# .tfpipboy/modules.yaml
modules:
  seed:
    instances:
      main:
        parallel: false  # Runs sequentially
        
  core:
    instances:
      main:
        parallel: true  # Can run in parallel
        depends_on: [seed]
        
  baseline:
    instances:
      connectivity:
        parallel: true  # Can run in parallel
        depends_on: [core]
      management:
        parallel: true  # Can run in parallel (with connectivity)
        depends_on: [core]
```

**Expected Output**:
- Stage 1: seed (no Live Board, single job)
- Stage 2: core (no Live Board, single job)
- Stage 3: Live Board with connectivity + management in parallel

## Related Documentation

- [PARALLEL-EXECUTION.md](./PARALLEL-EXECUTION.md) - Parallel execution configuration
- [CONFIG-FORMAT-SPEC.md](./CONFIG-FORMAT-SPEC.md) - Configuration format
- [GITHUB-AUTH-INTEGRATION.md](./GITHUB-AUTH-INTEGRATION.md) - Authentication integration

## Implementation Files

- `pkg/orchestrator/liveboard.go` - Live Board component
- `pkg/orchestrator/executor.go` - Integration with ParallelExecutor
- `pkg/orchestrator/display.go` - Display utilities and colors
- `pkg/orchestrator/types.go` - Job status types
