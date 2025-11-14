# Message Pipeline System Design

## Overview

Redesign the output message pipeline to have a structured, event-driven architecture with dedicated pipelines for each instance and the orchestrator, along with flexible console output filtering.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                       Message Pipeline System                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Instance 1  │  │  Instance 2  │  │  Instance N  │              │
│  │   Pipeline   │  │   Pipeline   │  │   Pipeline   │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                  │                  │                       │
│         │ Messages         │ Messages         │ Messages              │
│         ├─────────┬────────┴────────┬────────┴────────┐             │
│         │         │                 │                  │             │
│         v         v                 v                  v             │
│    ┌────────┐  ┌──────────────────────────────────────┐             │
│    │Instance│  │    Message Router & Processors       │             │
│    │  Log   │  │                                       │             │
│    │ Writer │  │  • LiveBoard Processor               │             │
│    └────────┘  │  • Console Processor (filtered)      │             │
│                 │  • File Log Processor                │             │
│  ┌──────────┐  └──────────────────────────────────────┘             │
│  │Orchestr. │            │         │           │                     │
│  │ Pipeline │            v         v           v                     │
│  └────┬─────┘     ┌──────────┐ ┌────────┐ ┌──────────┐             │
│       │           │LiveBoard │ │Console │ │ Instance │             │
│       │           │  Update  │ │ Output │ │   Logs   │             │
│       └───────────►          │ │        │ │          │             │
│                   └──────────┘ └────────┘ └──────────┘             │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Message Structure

```go
type Message struct {
    Timestamp  time.Time
    Level      MessageLevel  // TRACE, DEBUG, INFO, WARN, ERROR
    Source     MessageSource // orchestrator, instance, terraform, system
    InstanceID string        // Empty for orchestrator messages
    Component  string        // e.g., "init", "plan", "apply"
    Content    string        // Original message (may contain ANSI codes)
    Metadata   map[string]interface{}
}
```

### 2. Message Pipeline

Each instance and the orchestrator have their own dedicated pipeline:

- **Buffered channel** for async message processing
- **Filters** to control which messages get processed
- **Processors** to handle messages (log files, console, liveboard)
- **Independent goroutines** for non-blocking operation

### 3. Message Router

Central coordinator that:
- Manages all pipelines (orchestrator + instances)
- Creates and initializes pipelines
- Routes messages to appropriate destinations
- Configures output modes
- Buffers messages during liveboard execution

### 4. Message Processors

#### FileLogProcessor
- Writes messages to log files
- Separate log file per instance: `.tfpipboy/logs/instances/{instance-id}.log`
- Orchestrator log: `.tfpipboy/logs/orchestrator.log`
- **Preserves ANSI color codes** in instance logs
- Optional ANSI stripping for orchestrator log

#### ConsoleProcessor
- Writes messages to stdout
- Respects output mode configuration
- Optional ANSI stripping

#### LiveBoardProcessor
- Updates liveboard display in real-time
- Only processes instance messages
- Updates job status, steps, errors, plan results

#### BufferProcessor
- Buffers messages during liveboard execution
- Used in "liveboard_details" mode
- Messages displayed after liveboard completion

### 5. Message Filters

- **MinLevelFilter**: Only pass messages at or above level (INFO, DEBUG, TRACE)
- **SourceFilter**: Filter by message source
- **ComponentFilter**: Filter by component
- **ExcludeComponentFilter**: Exclude specific components

## Output Modes

### 1. LiveBoardOnly (Default)
- Only liveboard shown during execution
- No instance details to console
- All messages logged to files
- **Console**: LiveBoard only
- **Instance Logs**: Full terraform output with ANSI colors

### 2. LiveBoardDetails
- Liveboard during execution
- After completion, show grouped instance details
- **Console**: LiveBoard + buffered instance details after
- **Instance Logs**: Full terraform output with ANSI colors

### 3. AllMessages
- Real-time output from all pipelines
- No liveboard (too much output would conflict)
- **Console**: All messages in real-time
- **Instance Logs**: Full terraform output with ANSI colors

### 4. Quiet
- No console output except errors
- All messages logged to files
- **Console**: Errors only
- **Instance Logs**: Full terraform output with ANSI colors

## Message Flow

### Instance Execution Flow

```
1. Orchestrator creates execution plan
2. For each instance:
   a. MessageRouter.InitializeInstance(instanceID)
      - Creates instance pipeline
      - Adds FileLogProcessor → .tfpipboy/logs/instances/{id}.log
      - Adds appropriate processors based on output mode
   b. Instance starts execution
   c. Terraform output captured line-by-line
   d. Each line sent to instance pipeline:
      - MessageRouter.SendInstanceTerraformOutput(instanceID, component, line)
   e. Pipeline processors handle message:
      - FileLogProcessor → writes to instance log
      - LiveBoardProcessor → updates liveboard display
      - ConsoleProcessor/BufferProcessor → based on mode
3. All pipelines shut down gracefully after execution
```

### Orchestrator Message Flow

```
1. Orchestrator initialization
   a. MessageRouter.InitializeOrchestrator()
      - Creates orchestrator pipeline
      - Adds FileLogProcessor → .tfpipboy/logs/orchestrator.log
      - Adds ConsoleProcessor (filtered by verbose/trace flags)
2. Orchestrator sends messages:
   - MessageRouter.SendOrchestratorInfo(component, message)
   - MessageRouter.SendOrchestratorDebug(component, message)
   - MessageRouter.SendOrchestratorError(component, message)
3. Messages filtered by level (INFO, DEBUG, TRACE)
4. Written to orchestrator log and optionally console
```

## Implementation Steps

### Phase 1: Core Infrastructure
- [x] Create message types and structures
- [x] Implement MessagePipeline
- [x] Implement MessageRouter
- [x] Implement message processors (File, Console, LiveBoard, Buffer)
- [x] Implement message filters

### Phase 2: Integration
- [ ] Add MessageRouter to Orchestrator
- [ ] Update ParallelExecutor to initialize instance pipelines
- [ ] Modify terraform execution to send output to pipelines
- [ ] Update liveboard to work with message pipeline
- [ ] Remove old direct stdout writes

### Phase 3: Configuration
- [ ] Add CLI flags for output modes
- [ ] Add verbose/trace flags
- [ ] Update configuration options
- [ ] Add output mode to help text

### Phase 4: Testing
- [ ] Test each output mode
- [ ] Verify log files contain correct content
- [ ] Verify ANSI codes preserved in instance logs
- [ ] Test buffering in LiveBoardDetails mode
- [ ] Test concurrent message handling

## Benefits

1. **Separation of Concerns**: Each instance has isolated logging
2. **Original Output Preserved**: Full terraform output with colors in instance logs
3. **Flexible Display**: Easy to switch between output modes
4. **Non-blocking**: Async message processing doesn't slow execution
5. **Debugging**: Full trace available in log files
6. **Clean Console**: LiveBoard-only mode keeps console clean
7. **Post-Execution Review**: LiveBoardDetails mode allows review after

## Log File Structure

```
.tfpipboy/
└── logs/
    ├── orchestrator.log          # Orchestrator messages
    └── instances/
        ├── seed-main.log         # Instance 1 full output
        ├── core-main-gwc.log     # Instance 2 full output
        ├── vending-conn-main.log # Instance 3 full output
        └── ...
```

## Example Usage

```bash
# Default: LiveBoard only
tfpipboy plan --targets-all

# With instance details after execution
tfpipboy plan --targets-all --output-mode liveboard_details

# Real-time all messages
tfpipboy plan --targets-all --output-mode all_messages

# Quiet mode (only errors)
tfpipboy plan --targets-all --output-mode quiet

# Verbose mode (DEBUG level in orchestrator.log)
tfpipboy plan --targets-all --verbose

# Trace mode (TRACE level in all logs)
tfpipboy plan --targets-all --trace
```

## Migration Path

1. **Phase 1**: Add message pipeline alongside existing output
2. **Phase 2**: Gradually migrate to pipeline (dual write)
3. **Phase 3**: Remove old output mechanisms
4. **Phase 4**: Clean up and optimize

## Configuration Options

```yaml
# Future: .tfpipboy/config.yaml
output:
  mode: liveboard_only  # liveboard_only, liveboard_details, all_messages, quiet
  verbose: false        # Enable DEBUG level
  trace: false          # Enable TRACE level
  strip_ansi_logs: false      # Keep ANSI in logs
  strip_ansi_console: false   # Keep ANSI in console
```

## Current Status

- ✅ Core infrastructure implemented
- ✅ Message types and pipeline structure
- ✅ Message router with processors and filters
- ⏳ Integration with executor (in progress)
- ⏳ CLI flags and configuration
- ⏳ Testing and verification
