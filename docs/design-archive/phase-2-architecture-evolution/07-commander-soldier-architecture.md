# Commander-Soldier Architecture

## Overview

Split tf-pipboy into two separate applications:
1. **pipboy-commander**: TUI orchestration dashboard (control plane)
2. **pipboy-soldier**: CLI wrapper for individual module execution (data plane)

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    pipboy-commander (TUI)                        │
│  - Reads tfpipboy.yaml                                          │
│  - Builds dependency graph                                      │
│  - Orchestrates execution                                       │
│  - Shows real-time dashboard                                    │
│  - Manages soldier processes                                    │
└──────────────────┬──────────────────────────────────────────────┘
                   │
    ┌──────────────┼──────────────┐
    │              │              │
    ▼              ▼              ▼
┌─────────┐  ┌─────────┐  ┌─────────┐
│ soldier │  │ soldier │  │ soldier │
│  (vpc)  │  │  (eks)  │  │  (rds)  │
└─────────┘  └─────────┘  └─────────┘
```

## Component Design

### pipboy-commander

**Purpose:** Orchestration, coordination, and monitoring

**Responsibilities:**
- Parse `tfpipboy.yaml` configuration
- Build dependency graph
- Spawn soldier processes for each module
- Monitor soldier status and output
- Display TUI dashboard
- Handle user interactions (pause, resume, cancel)
- Aggregate results and report summary

**Technology:**
- Bubble Tea for TUI
- Go channels for inter-process communication
- JSON-RPC or stdio for soldier communication

**Key Features:**
```go
type Commander struct {
    config      *Config
    depGraph    *DependencyGraph
    soldiers    map[string]*Soldier
    dashboard   *DashboardModel
}

type Soldier struct {
    ModuleName  string
    Cmd         *exec.Cmd
    Status      Status
    Output      chan string
    Result      chan Result
    StartTime   time.Time
}
```

### pipboy-soldier

**Purpose:** Execute Terraform commands for a single module

**Responsibilities:**
- Receive instructions from commander (which command to run)
- Execute terraform in specific module directory
- Stream output back to commander
- Report status updates (starting, running, completed, failed)
- Handle terraform-specific logic (init, plan, apply, etc.)
- Manage authentication and context for the module

**Technology:**
- Simple CLI application
- Communicates via JSON on stdin/stdout
- Wraps terraform commands

**Key Features:**
```go
type Soldier struct {
    modulePath  string
    tfvarsPath  string
    workspace   string
    backend     map[string]string
}

// Commands soldier can execute
type Command struct {
    Action  string   // "init", "plan", "apply", "destroy"
    Args    []string // Additional args
}

// Messages soldier sends to commander
type StatusUpdate struct {
    Status  string   // "started", "running", "success", "failed"
    Output  string   // Latest output line
    Error   string   // Error message if failed
    Metadata map[string]string // Resources created, etc.
}
```

## Communication Protocol

### Option 1: JSON-RPC over stdio (Recommended)

**Commander → Soldier:**
```json
{
  "jsonrpc": "2.0",
  "method": "execute",
  "params": {
    "action": "apply",
    "module_path": "./modules/vpc",
    "tfvars": ".env/dev/vpc.tfvars",
    "workspace": "dev",
    "backend_config": {
      "storage_account": "mystorageaccount",
      "container_name": "tfstate",
      "key": "vpc.tfstate"
    },
    "args": ["--auto-approve"]
  },
  "id": 1
}
```

**Soldier → Commander (Status Updates):**
```json
{
  "jsonrpc": "2.0",
  "method": "status_update",
  "params": {
    "status": "running",
    "output": "Creating VPC...",
    "progress": 45
  }
}
```

**Soldier → Commander (Result):**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "status": "success",
    "duration": "2m34s",
    "resources_created": 5,
    "summary": "VPC and subnets created successfully"
  },
  "id": 1
}
```

### Option 2: Simple Line-Based Protocol

**Commander → Soldier:**
```
CMD:apply:./modules/vpc:.env/dev/vpc.tfvars:dev
```

**Soldier → Commander:**
```
STATUS:started
OUTPUT:Initializing Terraform...
OUTPUT:Creating VPC...
PROGRESS:45
STATUS:success:2m34s
```

## File Structure

```
tf-pipboy/
├── cmd/
│   ├── commander/
│   │   └── main.go          # TUI orchestration app
│   └── soldier/
│       └── main.go          # CLI wrapper for single module
├── pkg/
│   ├── commander/
│   │   ├── dashboard.go     # TUI dashboard
│   │   ├── orchestrator.go  # Execution coordination
│   │   ├── soldier_manager.go # Spawn and manage soldiers
│   │   └── config.go        # Config parsing
│   ├── soldier/
│   │   ├── executor.go      # Terraform command execution
│   │   ├── protocol.go      # Communication protocol
│   │   └── context.go       # Module context management
│   ├── protocol/
│   │   ├── types.go         # Shared message types
│   │   └── jsonrpc.go       # JSON-RPC implementation
│   └── shared/
│       ├── dependency.go    # Dependency graph
│       └── types.go         # Common types
└── design/
    └── 07-commander-soldier-architecture.md
```

## Execution Flow

### 1. Commander Startup
```
1. Commander reads tfpipboy.yaml
2. Parses module definitions and dependencies
3. Builds dependency graph
4. Initializes TUI dashboard
5. Waits for user to trigger execution
```

### 2. Execution Triggered
```
User presses "Start" in TUI:

1. Commander determines execution order (topological sort)
2. For each module (respecting dependencies):
   a. Spawn soldier process: `pipboy-soldier`
   b. Send execute command via stdin (JSON-RPC)
   c. Listen for status updates on stdout
   d. Update TUI dashboard in real-time
3. Wait for all soldiers to complete
4. Show summary in TUI
```

### 3. Soldier Execution
```
1. Soldier starts, reads JSON command from stdin
2. Parse command and extract parameters
3. Navigate to module directory
4. Execute terraform commands:
   - Run: terraform init
   - Send status: "initialized"
   - Run: terraform apply --auto-approve
   - Stream output lines to commander
   - Send progress updates
5. On completion:
   - Send result (success/failure)
   - Exit with appropriate code
```

## Benefits of This Architecture

### 1. Separation of Concerns ✅
- **Commander**: Focus on orchestration and UX
- **Soldier**: Focus on terraform execution
- Clean interfaces between components

### 2. Independent Development ✅
- Can develop/test soldier independently
- Can improve commander UI without touching execution
- Can replace soldier with different implementation

### 3. Scalability ✅
- Easy to parallelize (spawn multiple soldiers)
- Soldiers are isolated processes
- No shared state or locking issues

### 4. Debugging ✅
- Can run soldier standalone for testing
- Commander doesn't need to know terraform details
- Clear communication protocol

### 5. Future Enhancements ✅
- Could make soldier a network service (remote execution)
- Could implement different soldier types (AWS vs Azure vs GCP)
- Could add soldier pooling/reuse

### 6. Testability ✅
- Mock soldiers for commander testing
- Test soldiers independently
- Integration tests via protocol

## Commander TUI Design

### Main Dashboard View
```
╭─────────────────────────────────────────────────────────────────╮
│ 🎮 pipboy-commander                      Environment: dev      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Module          Status      Duration    Progress               │
│  ────────────────────────────────────────────────────────────  │
│                                                                  │
│  ✓ vpc           SUCCESS     2m 34s      [████████████] 100%   │
│    └─ 5 resources created                                       │
│                                                                  │
│  ↻ eks           RUNNING     1m 12s      [████████░░░░]  67%   │
│    └─ Creating EKS cluster...                                   │
│                                                                  │
│  ↻ rds           RUNNING     0m 45s      [█████░░░░░░░]  42%   │
│    └─ Initializing database...                                 │
│                                                                  │
│  ⏸ app           PENDING     -           [░░░░░░░░░░░░]   0%   │
│    └─ Waiting for: eks, rds                                    │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│ Progress: 1/4 complete │ 2 running │ 1 pending                 │
│                                                                  │
│ Keys: [↑↓] Navigate [l] Logs [p] Pause [r] Resume [q] Quit    │
╰─────────────────────────────────────────────────────────────────╯
```

### Log Viewer (When 'l' pressed on selected module)
```
╭─────────────────────────────────────────────────────────────────╮
│ 📋 Module Logs: eks                                    [x] Close│
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  [14:23:45] Initializing Terraform...                          │
│  [14:23:46] Terraform v1.9.0                                   │
│  [14:23:47] Initializing provider plugins...                   │
│  [14:23:50] Provider aws v5.0.0                                │
│  [14:24:01] Creating EKS cluster...                            │
│  [14:24:15] Creating IAM roles...                              │
│  [14:24:32] Configuring VPC settings...                        │
│  [14:24:45] EKS cluster provisioning (this may take 10-15min)  │
│  [14:25:12] Cluster endpoint: https://xxxxx.eks.amazonaws.com  │
│  [14:25:13] Configuring security groups...                     │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│ [↑↓] Scroll [Home/End] Jump [/] Search [Esc] Back             │
╰─────────────────────────────────────────────────────────────────╯
```

## Soldier CLI Interface

### Standalone Usage (for testing/debugging)
```bash
# Run soldier directly for a single module
pipboy-soldier execute \
  --action apply \
  --module ./modules/vpc \
  --tfvars .env/dev/vpc.tfvars \
  --workspace dev \
  --auto-approve

# Or use JSON input for commander compatibility
echo '{
  "action": "apply",
  "module_path": "./modules/vpc",
  "tfvars": ".env/dev/vpc.tfvars",
  "workspace": "dev"
}' | pipboy-soldier
```

### Soldier Output
```
{"status":"started","timestamp":"2024-01-15T14:23:45Z"}
{"status":"running","output":"Initializing Terraform..."}
{"status":"running","output":"Creating VPC...","progress":45}
{"status":"running","output":"VPC created successfully"}
{"status":"success","duration":"2m34s","resources":5}
```

## Configuration Example

### tfpipboy.yaml (Read by Commander)
```yaml
version: "1.0"

# Soldier configuration
soldier:
  binary: "./bin/pipboy-soldier"  # Path to soldier binary
  timeout: 30m                     # Per-module timeout
  retry: 2                         # Retry failed modules

# Module definitions
modules:
  vpc:
    path: "./modules/vpc"
    tfvars: ".env/{{env}}/vpc.tfvars"
    backend:
      type: azurerm
      config_file: ".env/{{env}}/backend.tfvars"
  
  eks:
    path: "./modules/eks"
    tfvars: ".env/{{env}}/eks.tfvars"
    depends_on: [vpc]
  
  rds:
    path: "./modules/rds"
    tfvars: ".env/{{env}}/rds.tfvars"
    depends_on: [vpc]
  
  app:
    path: "./modules/app"
    tfvars: ".env/{{env}}/app.tfvars"
    depends_on: [eks, rds]

# Environments
environments:
  dev:
    workspace: dev
  prod:
    workspace: production
```

## Implementation Phases

### Phase 1: Soldier Foundation (Week 1)
- [x] Create soldier CLI structure
- [ ] Implement JSON-RPC protocol
- [ ] Add terraform command execution
- [ ] Output streaming
- [ ] Status reporting
- [ ] Error handling

### Phase 2: Commander Foundation (Week 2)
- [ ] Create commander TUI structure
- [ ] Config parser for tfpipboy.yaml
- [ ] Soldier process spawning
- [ ] Communication with soldiers
- [ ] Basic dashboard view

### Phase 3: Orchestration (Week 3)
- [ ] Dependency graph implementation
- [ ] Execution coordination
- [ ] Parallel execution
- [ ] Progress tracking
- [ ] Error handling and retries

### Phase 4: Advanced UI (Week 4)
- [ ] Log viewer per module
- [ ] Keyboard navigation
- [ ] Module filtering
- [ ] Search functionality
- [ ] Summary reports

### Phase 5: Polish (Week 5)
- [ ] Testing and bug fixes
- [ ] Documentation
- [ ] Performance optimization
- [ ] Configuration validation

## Example Usage

### Basic Orchestration
```bash
# Commander orchestrates multiple soldiers
pipboy-commander apply --env dev

# Commander shows TUI dashboard
# Spawns soldiers for each module
# Coordinates execution based on dependencies
# Shows real-time progress
```

### Soldier Standalone (for debugging)
```bash
# Test a single module directly
pipboy-soldier execute \
  --action plan \
  --module ./modules/vpc \
  --tfvars .env/dev/vpc.tfvars
```

## Advantages Over Single Application

| Aspect | Single App | Commander-Soldier |
|--------|-----------|-------------------|
| Complexity | High (everything in one) | Low (separated concerns) |
| Testability | Hard to test pieces | Easy to test separately |
| Debugging | Complex | Simple (test soldier alone) |
| Scalability | Limited | Excellent (spawn more soldiers) |
| Code Organization | Tangled | Clean separation |
| Development | One team needs to know all | Can split development |
| Future Options | Limited | Could distribute soldiers |

## Communication Protocol Details

### Message Types

**Commander → Soldier:**
```go
type ExecuteRequest struct {
    Action       string            `json:"action"`        // init, plan, apply, destroy
    ModulePath   string            `json:"module_path"`
    TFVars       string            `json:"tfvars"`
    Workspace    string            `json:"workspace"`
    BackendConfig map[string]string `json:"backend_config"`
    Args         []string          `json:"args"`
}

type ControlMessage struct {
    Command string `json:"command"` // pause, resume, cancel
}
```

**Soldier → Commander:**
```go
type StatusUpdate struct {
    Status    string    `json:"status"`    // started, running, success, failed
    Output    string    `json:"output"`    // Latest output line
    Progress  int       `json:"progress"`  // 0-100
    Timestamp time.Time `json:"timestamp"`
}

type ExecuteResult struct {
    Status       string        `json:"status"`
    Duration     time.Duration `json:"duration"`
    Resources    int           `json:"resources"`
    Summary      string        `json:"summary"`
    Error        string        `json:"error,omitempty"`
}
```

## Next Steps

1. ✅ Document commander-soldier architecture (this file)
2. Implement soldier prototype
3. Implement commander prototype
4. Define and test communication protocol
5. Build dependency orchestration
6. Create TUI dashboard
7. Integration testing
8. Polish and release

## Conclusion

The **commander-soldier architecture** is the optimal design for tf-pipboy:

- **Commander**: Beautiful TUI for orchestration and monitoring
- **Soldier**: Simple, focused CLI wrapper for execution
- **Clean separation**: Each component does one thing well
- **Scalable**: Easy to parallelize and extend
- **Maintainable**: Clear interfaces and responsibilities

This architecture gives you the best of both worlds: professional orchestration UX with clean, testable components.
