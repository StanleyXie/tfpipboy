# TUI vs CLI Wrapper: Architecture Analysis

## The Question

For multi-session Terraform orchestration, which approach is better:
1. **Pure CLI Wrapper**: Manage multiple shell sessions from wrapper
2. **Hybrid TUI + Wrapper**: Use TUI for orchestration, wrapper for interactive commands

## Use Case Requirements

### What We Need:
1. **Multi-session management**: Run multiple Terraform modules in parallel
2. **Real-time monitoring**: See status of all modules simultaneously
3. **Interactive commands**: Some terraform commands need user input (approve, etc.)
4. **Output visibility**: View live output from each module
5. **Dependency coordination**: Wait for dependencies before starting modules
6. **Error handling**: Handle failures gracefully, show errors clearly

## Option 1: Pure CLI Wrapper

### Architecture
```
tfpipboy wrapper
    ├─> spawn: terraform apply (module: vpc)
    ├─> spawn: terraform apply (module: eks)  [waits for vpc]
    └─> spawn: terraform apply (module: rds)  [waits for vpc]
```

### Implementation
- Use `os/exec` to spawn multiple terraform processes
- Capture stdout/stderr to buffers
- Print status updates to terminal inline
- Commands run with direct terminal access

### Pros ✅
- **Simple implementation**: Direct process spawning
- **Native terminal feel**: Real terraform output
- **Interactive support**: Can pass stdin directly to terraform
- **No TUI complexity**: No event loop, no rendering engine
- **Easy debugging**: Standard output, standard errors
- **Shell integration**: Works with pipes, redirects

### Cons ❌
- **Poor visibility**: Hard to see multiple modules at once
- **Output mixing**: Multiple modules print to same terminal, gets messy
- **No status overview**: Can't see "module A running, module B done, module C waiting"
- **Scrollback pollution**: Lots of output scrolls away
- **No structured display**: Just raw text output
- **Hard to track progress**: Which module is at which step?

### Example Output Problem
```bash
$ tfpipboy apply --env dev

[vpc] Initializing terraform...
[eks] Waiting for vpc...
[vpc] Creating VPC...
[rds] Waiting for vpc...
[vpc] VPC created successfully
[eks] Initializing terraform...
[eks] Creating EKS cluster...
[rds] Initializing terraform...
[rds] Creating RDS instance...
[eks] EKS cluster created
[rds] Error: Database subnet group not found
[eks] Complete!

# Hard to parse! What succeeded? What failed?
```

## Option 2: Hybrid TUI + CLI Wrapper

### Architecture
```
┌─────────────────────────────────────────┐
│         TUI (Orchestration View)        │
│  - Shows all modules status             │
│  - Manages execution coordination       │
│  - Displays progress                    │
└─────────────────────────────────────────┘
           │
           ├─> Session 1: terraform apply (vpc)
           │   [spawned in background, output captured]
           │
           ├─> Session 2: terraform apply (eks)
           │   [waits for vpc, runs when ready]
           │
           └─> Session 3: terraform apply (rds)
               [waits for vpc, runs when ready]

When interactive input needed:
  TUI → suspend → native terminal → resume TUI
```

### Implementation
- TUI shows orchestration dashboard
- Spawn terraform processes in background
- Capture output to buffers
- TUI displays status, progress, summaries
- For interactive commands: suspend TUI, run in native terminal, resume

### Pros ✅
- **Clear overview**: See all modules at once
- **Structured display**: Organized status per module
- **Real-time updates**: Live progress bars, status indicators
- **Better UX**: Professional dashboard interface
- **Output organization**: Each module's output in its own section
- **Easy debugging**: Click/select module to see detailed logs
- **Progress tracking**: Visual indicators for each stage
- **Error highlighting**: Failed modules clearly marked in red
- **History tracking**: See what ran when

### Cons ❌
- **Complex implementation**: TUI event loop, rendering, state management
- **Interactive handling**: Need to suspend/resume TUI for user input
- **More code**: TUI framework, rendering logic, event handling
- **Potential bugs**: TUI rendering issues, cursor positioning
- **Learning curve**: Bubble Tea framework complexity
- **Terminal compatibility**: May not work in all terminals

### Example TUI Display
```
╭─────────────────────────────────────────────────────────────╮
│ tf-pipboy Orchestration: dev environment                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ✓  vpc     SUCCESS     2m 34s    [View Logs]              │
│      └─ Created VPC, Subnets, NAT Gateway                   │
│                                                              │
│  ↻  eks     RUNNING     1m 12s    [View Logs]              │
│      └─ Creating EKS cluster... ████████░░ 80%              │
│                                                              │
│  ↻  rds     RUNNING     0m 45s    [View Logs]              │
│      └─ Initializing database... ████░░░░░░ 40%            │
│                                                              │
│  ⏸  app     PENDING     -         [Waiting for: eks, rds]  │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│ Progress: 1/4 complete │ 2 running │ 1 pending              │
│ Keys: [l]ogs [p]ause [q]uit                                │
╰─────────────────────────────────────────────────────────────╯
```

## Option 3: Hybrid with Ghostty Multi-Window

### Architecture
```
Main Control Window (TUI):
  ┌─────────────────────────────┐
  │ Orchestration Dashboard     │
  │ Module Status Overview      │
  └─────────────────────────────┘

Ghostty automatically opens module windows:
  ┌───────────┐  ┌───────────┐  ┌───────────┐
  │ vpc       │  │ eks       │  │ rds       │
  │ [live]    │  │ [live]    │  │ [live]    │
  │ output    │  │ output    │  │ output    │
  └───────────┘  └───────────┘  └───────────┘
```

### Implementation
- Main TUI window for orchestration control
- Use Ghostty's API to open new windows/tabs for each module
- Each module runs in its own Ghostty window with native terminal
- Main TUI coordinates execution and shows overview

### Pros ✅
- **Best of both worlds**: TUI overview + native terminal experience
- **Native interaction**: Each module has real terminal for input
- **Parallel visibility**: See all modules simultaneously
- **Ghostty features**: Tabs, splits, hyperlinks, etc.
- **Clean separation**: Control in one window, execution in others
- **Professional**: Like modern IDE with multiple panes

### Cons ❌
- **Ghostty dependency**: Only works with Ghostty terminal
- **Complex coordination**: Managing multiple windows/processes
- **Implementation effort**: Ghostty API integration
- **Window management**: User needs to arrange windows
- **Not portable**: Won't work in other terminals

## Recommendation: **Hybrid TUI + Wrapper**

### Reasoning:

1. **Your Use Case is Perfect for TUI:**
   - Multi-module orchestration NEEDS overview dashboard
   - Can't effectively monitor 5+ modules with plain text output
   - Status tracking is essential (which modules done/running/pending)
   - Progress visibility is critical for long-running operations

2. **CLI Wrapper Alone Won't Scale:**
   ```bash
   # With 5 modules running in parallel:
   [vpc] line 1
   [eks] line 1
   [rds] line 1
   [vpc] line 2
   [app] waiting...
   [eks] line 2
   # Total chaos!
   ```

3. **Interactive Commands Can Be Handled:**
   - Most terraform commands are non-interactive
   - For `terraform apply` without `--auto-approve`:
     - Option A: Always use `--auto-approve` flag
     - Option B: Suspend TUI, run interactively, resume
     - Option C: TUI shows plan, user confirms in TUI, then auto-approve

4. **Implementation Path:**
   - Start with TUI for orchestration view
   - Use `os/exec` to spawn terraform in background
   - Capture output to buffers
   - Display in TUI with real-time updates
   - Add interactive mode if needed later

## Recommended Architecture

### Phase 1: TUI Orchestration Dashboard
```go
type OrchestrationModel struct {
    modules     map[string]*ModuleState
    sessions    map[string]*exec.Cmd
    outputs     map[string]*bytes.Buffer
    depGraph    *DependencyGraph
}

type ModuleState struct {
    Name       string
    Status     Status  // Pending, Running, Success, Failed
    StartTime  time.Time
    Duration   time.Duration
    Progress   int     // 0-100
    Output     []string
    Error      error
}
```

### TUI Components:
1. **Header**: Environment, timestamp, overall progress
2. **Module List**: Scrollable list of modules with status
3. **Detail Pane**: Selected module's detailed output
4. **Footer**: Keybindings, help

### Key Features:
- Real-time status updates (every 100ms)
- Color-coded status (green=success, yellow=running, red=failed)
- Progress bars for running modules
- Expandable log viewer per module
- Keyboard navigation (↑↓ to select, Enter to view logs, Q to quit)

## Implementation Timeline

### Week 1: Core TUI Framework
- [ ] Set up Bubble Tea application
- [ ] Design module list view
- [ ] Implement status display
- [ ] Add color coding

### Week 2: Session Management
- [ ] Spawn terraform processes
- [ ] Capture stdout/stderr
- [ ] Update TUI from background goroutines
- [ ] Handle process completion

### Week 3: Orchestration Logic
- [ ] Load config from tfpipboy.yaml
- [ ] Build dependency graph
- [ ] Implement execution coordination
- [ ] Add parallel execution

### Week 4: Polish & Features
- [ ] Log viewer per module
- [ ] Error handling and display
- [ ] Progress indicators
- [ ] Keyboard shortcuts

## Future Enhancements

### Phase 2: Ghostty Integration (Optional)
- If TUI feels limiting, add Ghostty multi-window
- Keep TUI as main orchestration view
- Ghostty windows for detailed module output
- Best of both worlds

### Phase 3: Advanced Features
- Pipeline definitions
- Rollback capabilities
- Resource dependency visualization
- State inspection

## Conclusion

**Answer: Yes, move to Hybrid TUI + Wrapper approach**

**Why:**
1. Multi-session orchestration requires dashboard view
2. TUI provides essential visibility and organization
3. CLI wrapper alone becomes chaos with parallel execution
4. Implementation is manageable with Bubble Tea
5. Can always add Ghostty integration later

**Start with:**
- TUI for orchestration dashboard
- Background process execution
- Captured output display
- Basic keyboard navigation

**This gives you:**
- Professional, usable interface
- Clear visibility of all modules
- Proper status tracking
- Foundation for future enhancements

The TUI is the **right tool for this job**. The previous CLI wrapper exploration was valuable for simple commands, but orchestration needs the structure and visibility that only a TUI can provide.
