# Terraform Orchestration Architecture

## New Vision

tf-pipboy evolves from a simple CLI wrapper to a **Terraform orchestration engine** that manages complex multi-module deployments across multiple environments with dependency management and parallel execution capabilities.

## Core Concepts

### 1. Multi-Module Orchestration
- **Modules**: Individual Terraform modules (e.g., `vpc`, `eks`, `rds`)
- **Environments**: Multiple deployment targets (e.g., `dev`, `staging`, `prod`)
- **Dependencies**: Explicit dependency graph between modules
- **Configuration**: Declarative config files defining orchestration

### 2. Multi-Session Management
- Each module runs in its own terminal session/process
- Similar to CI/CD pipelines but running locally
- Visual representation of running sessions (like tmux/screen)
- Real-time status updates across all sessions

### 3. Dependency Management
- DAG (Directed Acyclic Graph) of module dependencies
- Modules with no dependencies run in parallel
- Dependent modules wait for their dependencies to complete
- Smart execution order based on dependency graph

## Architecture Components

### Configuration Layer

#### Module Definition (`tfpipboy.yaml`)
```yaml
version: "1.0"

# Define modules
modules:
  vpc:
    path: "./modules/vpc"
    tfvars: ".env/{{env}}/vpc.tfvars"
    backend:
      type: azurerm
      config: ".env/{{env}}/backend.tfvars"
  
  eks:
    path: "./modules/eks"
    tfvars: ".env/{{env}}/eks.tfvars"
    depends_on: [vpc]  # Explicit dependency
  
  rds:
    path: "./modules/rds"
    tfvars: ".env/{{env}}/rds.tfvars"
    depends_on: [vpc]
  
  app:
    path: "./modules/app"
    tfvars: ".env/{{env}}/app.tfvars"
    depends_on: [eks, rds]  # Multiple dependencies

# Define environments
environments:
  dev:
    workspace: dev
    region: eastus
  
  staging:
    workspace: staging
    region: eastus
  
  prod:
    workspace: production
    region: westus

# Execution strategies
execution:
  parallel_limit: 3  # Max parallel sessions
  timeout: 30m       # Per-module timeout
  auto_approve: false
```

#### Pipeline Definition (Optional)
```yaml
pipelines:
  deploy-all:
    description: "Deploy all infrastructure"
    steps:
      - modules: [vpc]
        action: apply
      - modules: [eks, rds]  # Run in parallel after vpc
        action: apply
      - modules: [app]
        action: apply
  
  destroy-all:
    description: "Destroy all infrastructure"
    steps:
      - modules: [app]
        action: destroy
      - modules: [eks, rds]
        action: destroy
      - modules: [vpc]
        action: destroy
```

### Execution Engine

#### 1. Dependency Resolver
```
┌─────────────────────────────────────┐
│     Dependency Resolver             │
│  - Parse module dependencies        │
│  - Build DAG                        │
│  - Determine execution order        │
│  - Identify parallel opportunities  │
└─────────────────────────────────────┘
```

**Algorithm:**
- Topological sort of dependency graph
- Group modules by dependency level
- Level 0: No dependencies (run first, in parallel)
- Level N: Wait for Level N-1 to complete

**Example Execution Order:**
```
Level 0: [vpc] (runs first)
         ↓
Level 1: [eks, rds] (run in parallel after vpc)
         ↓
Level 2: [app] (runs after eks and rds)
```

#### 2. Session Manager
```
┌─────────────────────────────────────┐
│       Session Manager               │
│  - Create isolated sessions         │
│  - Track session state              │
│  - Capture output                   │
│  - Handle termination               │
└─────────────────────────────────────┘
```

**Session Lifecycle:**
1. **Pending**: Module queued, waiting for dependencies
2. **Running**: Terraform command executing
3. **Success**: Completed successfully
4. **Failed**: Error occurred
5. **Skipped**: Dependency failed

**Session Isolation:**
- Each module runs in separate goroutine
- Separate working directory context
- Independent authentication/state
- Captured stdout/stderr

#### 3. Display Engine (Ghostty Integration)

**Option A: Multiplexed Terminal View**
```
╭─────────────────────────────────────────────────────────────╮
│ tf-pipboy Orchestration: dev environment                    │
├─────────────────────────────────────────────────────────────┤
│ Module    Status      Duration    Output                    │
├─────────────────────────────────────────────────────────────┤
│ ✓ vpc     SUCCESS     2m 34s      → 3 resources created    │
│ ↻ eks     RUNNING     1m 12s      Creating EKS cluster...  │
│ ↻ rds     RUNNING     0m 45s      Initializing database... │
│ ⏸ app     PENDING     -           Waiting for eks, rds     │
├─────────────────────────────────────────────────────────────┤
│ Progress: 2/4 modules | 1 running | 1 pending              │
╰─────────────────────────────────────────────────────────────╯
```

**Option B: Ghostty Multi-Window Layout**
```
┌─────────────────┬─────────────────┬─────────────────┐
│  vpc (SUCCESS)  │  eks (RUNNING)  │  rds (RUNNING)  │
│                 │                 │                 │
│ Created:        │ Creating EKS... │ Initializing... │
│ - VPC           │                 │                 │
│ - Subnets       │ [Live output]   │ [Live output]   │
│ - NAT Gateway   │                 │                 │
└─────────────────┴─────────────────┴─────────────────┘
│           app (PENDING - waiting)                   │
└─────────────────────────────────────────────────────┘
```

**Ghostty Features to Use:**
- Multiple windows/tabs for parallel sessions
- OSC 8 hyperlinks to Terraform cloud console
- OSC 133 prompt marking for session boundaries
- Synchronized rendering for smooth updates
- Custom keybindings for session control

## Implementation Phases

### Phase 1: Configuration Parser (Week 1)
- [ ] Define `tfpipboy.yaml` schema
- [ ] Implement YAML parser
- [ ] Validate module paths and dependencies
- [ ] Template variable substitution ({{env}})

### Phase 2: Dependency Engine (Week 2)
- [ ] Build dependency graph from config
- [ ] Implement topological sort
- [ ] Detect circular dependencies
- [ ] Generate execution plan

### Phase 3: Session Management (Week 3)
- [ ] Create session abstraction
- [ ] Implement parallel execution with goroutines
- [ ] Output capture and buffering
- [ ] Status tracking and updates

### Phase 4: CLI Interface (Week 4)
- [ ] Commands: `plan`, `apply`, `destroy`, `status`
- [ ] Environment selection: `--env dev`
- [ ] Module filtering: `--modules vpc,eks`
- [ ] Dry-run mode: `--dry-run`
- [ ] Interactive approval

### Phase 5: Display Engine (Week 5-6)
- [ ] Terminal UI with real-time updates
- [ ] Log viewer for each module
- [ ] Progress indicators
- [ ] Error highlighting

### Phase 6: Ghostty Integration (Week 7-8)
- [ ] Multi-window layout
- [ ] Session switching
- [ ] Synchronized updates
- [ ] Rich terminal features (hyperlinks, etc.)

## Command Examples

### Basic Orchestration
```bash
# Plan all modules in dev environment
tfpipboy plan --env dev

# Apply specific modules
tfpipboy apply --env prod --modules vpc,eks

# Destroy with auto-approve
tfpipboy destroy --env dev --auto-approve

# Show status of running orchestration
tfpipboy status
```

### Pipeline Execution
```bash
# Run predefined pipeline
tfpipboy run deploy-all --env staging

# Custom pipeline from command line
tfpipboy apply --env prod \
  --stage vpc \
  --stage eks,rds \
  --stage app
```

### Interactive Mode
```bash
# Interactive session selection
tfpipboy interactive --env dev

# Shows:
# 1. Select modules to deploy
# 2. Review execution plan
# 3. Approve/reject each stage
# 4. Watch live progress
```

## Technical Stack

### Core Libraries
- **Config Parsing**: `gopkg.in/yaml.v3`
- **Dependency Graph**: Custom implementation or `gonum.org/v1/gonum/graph`
- **Parallel Execution**: Go channels and goroutines
- **Terminal UI**: 
  - Option 1: `github.com/charmbracelet/bubbletea` (TUI)
  - Option 2: Direct Ghostty integration via escape codes
- **Process Management**: `os/exec` with context cancellation

### State Management
```go
type Orchestrator struct {
    config      *Config
    sessions    map[string]*Session
    depGraph    *DependencyGraph
    execPlan    *ExecutionPlan
    statusMutex sync.RWMutex
}

type Session struct {
    ModuleName  string
    Status      SessionStatus
    StartTime   time.Time
    Duration    time.Duration
    Output      *bytes.Buffer
    Error       error
    Context     context.Context
    Cancel      context.CancelFunc
}

type SessionStatus int
const (
    Pending SessionStatus = iota
    Running
    Success
    Failed
    Skipped
)
```

## Comparison with Existing Tools

### vs. Terragrunt
**Similarities:**
- Module orchestration
- Dependency management
- DRY configuration

**Differences:**
- tf-pipboy: Focus on terminal UX and parallel visibility
- tf-pipboy: Built-in multi-session display
- tf-pipboy: Ghostty-optimized features
- Terragrunt: More mature, production-ready
- Terragrunt: AWS-centric features (S3 backend, etc.)

### vs. Terraform Cloud
**Similarities:**
- Workspace management
- Execution planning
- Status tracking

**Differences:**
- tf-pipboy: Local execution, no SaaS
- tf-pipboy: Terminal-native interface
- tf-pipboy: Developer-focused
- TF Cloud: Team collaboration features
- TF Cloud: Remote execution and state management

## Success Metrics

1. **Execution Time**: 30-50% faster than sequential execution
2. **User Experience**: Real-time visibility into all modules
3. **Reliability**: Proper dependency resolution, no race conditions
4. **Usability**: Simple YAML config, intuitive commands

## Next Steps

1. ✅ Document architecture (this file)
2. Create `pkg/orchestrator` package structure
3. Implement config parser with validation
4. Build dependency graph engine
5. Create session manager POC
6. Design terminal UI mockups

## Open Questions

1. **Error Handling**: Continue or stop on first failure?
   - Default: Stop on failure
   - Option: `--continue-on-error` flag

2. **State Locking**: How to handle concurrent Terraform state access?
   - Rely on Terraform's native state locking
   - Add additional validation

3. **Output Verbosity**: How much output to show per session?
   - Summary view by default
   - Full logs on demand
   - Filterable by module

4. **Resource Sharing**: How to pass outputs between modules?
   - Option 1: Terraform remote state data sources
   - Option 2: Built-in output passing mechanism
   - Recommendation: Use Terraform's native approach

5. **Rollback Strategy**: How to handle partial failures?
   - Manual rollback with destroy commands
   - Automatic rollback (optional flag)
   - State snapshot/restore

## References

- [Terragrunt Documentation](https://terragrunt.gruntwork.io/)
- [Terraform Best Practices](https://www.terraform-best-practices.com/)
- [Ghostty Documentation](https://ghostty.org/docs)
- [DAG Implementation in Go](https://pkg.go.dev/gonum.org/v1/gonum/graph)
