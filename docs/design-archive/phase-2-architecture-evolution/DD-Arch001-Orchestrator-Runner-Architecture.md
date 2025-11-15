# Architecture Design Document
**Document No.**: DD-Arch001  
**Topic**: Orchestrator-Runner Architecture  
**Based on**: UR-Arch001-Architecture-ReDesign.md  
**Date**: 2024-01-18

## Executive Summary

tfpipboy is a **CI/CD Workflow Pipeline Orchestrator** for Terraform modules, designed to run in terminal environments. It uses a two-component architecture:
- **Orchestrator**: TUI application for pipeline orchestration and monitoring
- **Runner**: Dedicated terminal process for executing individual Terraform modules

## Core Architecture

### Naming Convention
- **Orchestrator** (formerly "Commander"): The control plane
- **Runner** (formerly "Soldier"): The execution plane

This naming better reflects the CI/CD pipeline paradigm and lifecycle management nature of the system.

## Product Objectives Analysis

### 1. CI/CD Workflow Pipeline Orchestrator
**What this means:**
- Treat infrastructure deployment like CI/CD pipelines
- Stages/steps execute in defined order
- Dependencies between modules like pipeline stages
- Parallel execution where possible
- Full lifecycle management (init → plan → apply → destroy)

**Architecture implications:**
```
Pipeline Stages (Sequential):
  Stage 1: [vpc] 
    └─> VPC must complete before stage 2
  
  Stage 2: [eks, rds]  (Parallel)
    ├─> eks depends on vpc
    └─> rds depends on vpc
  
  Stage 3: [app]
    └─> app depends on eks AND rds
```

### 2. Declarative Configuration
**What this means:**
- Configuration-as-code for the orchestration itself
- Define WHAT to deploy, not HOW
- Version-controlled pipeline definitions
- Reusable configurations across environments

**Design considerations:**
- YAML-based configuration (industry standard for CI/CD)
- Template support for multi-environment
- Module registry/catalog concept
- Pipeline inheritance and composition

### 3. Automatic Environment Sensing
**What this means:**
- Auto-detect current environment context
- Auto-apply correct configurations
- Seamless switching between dev/staging/prod
- Smart defaults based on context

**Key features:**
- Git branch detection → environment mapping
- Workspace detection
- Backend configuration auto-selection
- Variable file auto-loading

### 4. Multi-Cloud Auth Status Tracking
**What this means:**
- Real-time auth status for AWS/Azure/GCP
- GitHub authentication tracking
- Environment variable tracking
- Credential expiration warnings

**Display in Orchestrator:**
```
╭─ Authentication Status ─────────────────╮
│ ✓ Azure   (Stanley.Xie@...) Exp: 2h    │
│ ✗ AWS     Not logged in                │
│ ✓ GitHub  (StanleyXie)   Exp: 30d     │
╰─────────────────────────────────────────╯
```

## MVP Feature Breakdown

### Feature 1: Multiple Terraform Module Management

**Requirement:**
> Multiple terraform module management in a single repo

**Design:**
```
project-root/
├── tfpipboy.yaml           # Orchestrator configuration
├── modules/
│   ├── vpc/
│   │   ├── main.tf
│   │   └── variables.tf
│   ├── eks/
│   │   ├── main.tf
│   │   └── variables.tf
│   └── rds/
│       ├── main.tf
│       └── variables.tf
└── environments/
    ├── dev/
    │   ├── vpc.tfvars
    │   ├── eks.tfvars
    │   └── rds.tfvars
    └── prod/
        ├── vpc.tfvars
        ├── eks.tfvars
        └── rds.tfvars
```

**Key capabilities:**
- Single repository contains all infrastructure modules
- Centralized orchestration configuration
- Environment-specific variable files
- Module dependencies declared explicitly

### Feature 2: Declarative Configuration Format

**Requirement:**
> Declarative configuration format design for terraform module configuration

**Configuration Schema: `tfpipboy.yaml`**
```yaml
version: "1.0"

# Project metadata
project:
  name: "my-infrastructure"
  description: "Complete infrastructure stack"
  
# Environment detection and mapping
environment:
  auto_detect: true              # Auto-detect from git branch
  branch_mapping:
    main: prod
    develop: dev
    "feature/*": dev
  default: dev

# Module catalog
modules:
  vpc:
    description: "Virtual Private Cloud"
    path: "./modules/vpc"
    category: "networking"
    
  eks:
    description: "Kubernetes Cluster"
    path: "./modules/eks"
    category: "compute"
    depends_on: [vpc]
    
  rds:
    description: "PostgreSQL Database"
    path: "./modules/rds"
    category: "database"
    depends_on: [vpc]
    
  app:
    description: "Application Deployment"
    path: "./modules/app"
    category: "application"
    depends_on: [eks, rds]

# Pipeline definitions
pipelines:
  deploy-all:
    description: "Deploy complete infrastructure"
    stages:
      - name: "Networking"
        modules: [vpc]
        
      - name: "Data Layer"
        modules: [eks, rds]
        parallel: true
        
      - name: "Application"
        modules: [app]
        
  destroy-all:
    description: "Destroy all infrastructure"
    stages:
      - name: "Application"
        modules: [app]
        action: destroy
        
      - name: "Data Layer"
        modules: [eks, rds]
        action: destroy
        parallel: true
        
      - name: "Networking"
        modules: [vpc]
        action: destroy

# Environment-specific configurations
environments:
  dev:
    workspace: "dev"
    backend:
      type: "azurerm"
      storage_account: "devtfstate"
      container: "tfstate"
    variables_path: "./environments/dev"
    auto_approve: true
    
  prod:
    workspace: "production"
    backend:
      type: "azurerm"
      storage_account: "prodtfstate"
      container: "tfstate"
    variables_path: "./environments/prod"
    auto_approve: false
    require_approval: true

# Orchestrator settings
orchestrator:
  parallel_limit: 3           # Max runners in parallel
  runner_timeout: 30m         # Per-module timeout
  retry_failed: 2             # Retry count for failures
  log_level: "info"
  
# Runner settings
runner:
  terraform_version: "1.9.0"  # Required TF version
  init_args: ["-upgrade"]
  plan_args: ["-out=tfplan"]
  apply_args: ["-auto-approve"]
```

**Key design decisions:**

1. **Auto-detection**: Reduce manual configuration
2. **Pipeline abstraction**: Define workflows, not individual commands
3. **Environment contexts**: Complete isolation per environment
4. **Module catalog**: Central registry of all modules
5. **Dependency graph**: Explicit, verifiable dependencies

### Feature 3: Orchestrator TUI Application

**Requirement:**
> TUI application for managing and orchestrating multiple terraform modules

**Orchestrator Responsibilities:**
- Parse `tfpipboy.yaml` configuration
- Build dependency graph
- Manage Runner lifecycle
- Display real-time status
- Capture and display logs
- Handle errors and retries
- Provide interactive controls

**TUI Design:**

#### Main View: Pipeline Execution
```
╭──────────────────────────────────────────────────────────────────╮
│ 🎮 Orchestrator | Pipeline: deploy-all | Environment: dev       │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│ Stage 1: Networking                                      [DONE]  │
│   ✓ vpc          SUCCESS   2m 34s   5 resources                 │
│                                                                   │
│ Stage 2: Data Layer                                  [RUNNING]   │
│   ↻ eks          RUNNING   1m 12s   ████████░░ 80%              │
│      └─ Creating EKS cluster...                                  │
│   ↻ rds          RUNNING   0m 45s   █████░░░░░ 50%              │
│      └─ Provisioning RDS instance...                            │
│                                                                   │
│ Stage 3: Application                                 [PENDING]   │
│   ⏸ app          PENDING   -        Waiting for eks, rds        │
│                                                                   │
├──────────────────────────────────────────────────────────────────┤
│ Progress: Stage 2/3 │ 2 runners active │ ETA: 3m 20s            │
│                                                                   │
│ [Enter] View Logs  [P] Pause  [R] Resume  [Q] Quit              │
╰──────────────────────────────────────────────────────────────────╯
```

#### Log Viewer: Module Details
```
╭──────────────────────────────────────────────────────────────────╮
│ 📋 Runner Logs: eks                               [Esc] Close   │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│ [14:23:45] Runner started for module: eks                        │
│ [14:23:45] Environment: dev                                      │
│ [14:23:45] Workspace: dev                                        │
│ [14:23:46] Initializing Terraform...                            │
│ [14:23:47] Terraform v1.9.0                                     │
│ [14:23:48] Backend: azurerm                                     │
│ [14:23:50] Running: terraform plan                              │
│ [14:24:01] Plan: 12 to add, 0 to change, 0 to destroy          │
│ [14:24:02] Running: terraform apply                             │
│ [14:24:15] Creating aws_eks_cluster.main...                     │
│ [14:25:30] aws_eks_cluster.main: Creating... [1m15s elapsed]    │
│ [14:26:45] aws_eks_cluster.main: Still creating... [2m30s]      │
│                                                                   │
├──────────────────────────────────────────────────────────────────┤
│ [↑↓] Scroll  [/] Search  [Home/End] Jump  [S] Save              │
╰──────────────────────────────────────────────────────────────────╯
```

#### Environment Selector
```
╭──────────────────────────────────────────────────────────────────╮
│ 🌍 Select Environment                                            │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│   ○ dev          (auto-detected from branch: develop)           │
│   ○ staging                                                      │
│   ○ prod         ⚠️  Requires approval                           │
│                                                                   │
│ Current branch: develop                                          │
│ Auto-detect: enabled                                             │
│                                                                   │
│ [↑↓] Select  [Enter] Confirm  [Esc] Cancel                      │
╰──────────────────────────────────────────────────────────────────╯
```

### Feature 4: Runner Process Management

**Requirement:**
> Each module will be orchestrated to run in a dedicated terminal process "Runner"

**Runner Characteristics:**
- Isolated process per module
- Full terraform output capture
- Lifecycle managed by Orchestrator
- Communicates via JSON-RPC or stdio

**Runner Lifecycle:**
```
1. INIT
   ↓
2. WAITING (for dependencies)
   ↓
3. STARTING
   ↓
4. RUNNING
   ↓
5. SUCCESS / FAILED / CANCELLED
   ↓
6. CLEANUP
```

**Runner State Machine:**
```go
type RunnerState int

const (
    StateInit RunnerState = iota
    StateWaiting
    StateStarting
    StateRunning
    StateSuccess
    StateFailed
    StateCancelled
    StateCleanup
)

type Runner struct {
    ID          string
    ModuleName  string
    State       RunnerState
    Process     *exec.Cmd
    Output      *OutputBuffer
    StartTime   time.Time
    EndTime     time.Time
    Error       error
    Metadata    map[string]interface{}
}
```

**Full Lifecycle Management:**

1. **Initialization**
   - Orchestrator creates Runner instance
   - Assigns module configuration
   - Sets up output capture
   - Registers with lifecycle manager

2. **Waiting**
   - Check dependencies
   - Wait for dependent modules to complete
   - Monitor dependency failures (cancel if dependency fails)

3. **Execution**
   ```
   Runner executes:
   1. terraform init
   2. terraform workspace select/create
   3. terraform plan
   4. terraform apply (if approved)
   ```

4. **Monitoring**
   - Stream output to Orchestrator
   - Report progress percentage
   - Send status updates
   - Handle errors

5. **Completion**
   - Report final status
   - Save output logs
   - Update state
   - Notify Orchestrator

6. **Cleanup**
   - Close file handles
   - Release resources
   - Archive logs
   - Remove temporary files

## Automatic Environment Sensing

### Git Branch Detection
```go
type EnvironmentDetector struct {
    branchMapping map[string]string
}

func (d *EnvironmentDetector) DetectEnvironment() string {
    branch := getCurrentGitBranch()
    
    // Check exact match
    if env, ok := d.branchMapping[branch]; ok {
        return env
    }
    
    // Check pattern match
    for pattern, env := range d.branchMapping {
        if matchPattern(pattern, branch) {
            return env
        }
    }
    
    return "dev" // default
}
```

### Configuration Auto-Alignment
```yaml
# Based on detected environment, automatically:
1. Select workspace
2. Load backend configuration
3. Load variable files
4. Set approval requirements
5. Configure logging level
```

**Example Flow:**
```
1. User runs: orchestrator run deploy-all
2. Detect git branch: "develop"
3. Map to environment: "dev"
4. Auto-load:
   - Workspace: dev
   - Backend: devtfstate
   - Variables: ./environments/dev/*.tfvars
   - Auto-approve: true
5. Execute pipeline with detected config
```

## Authentication Status Tracking

### Multi-Provider Auth Manager
```go
type AuthManager struct {
    providers map[string]AuthProvider
    cache     map[string]*AuthStatus
}

type AuthProvider interface {
    Name() string
    Check() (*AuthStatus, error)
    Login() error
    Logout() error
}

type AuthStatus struct {
    Provider      string
    Authenticated bool
    User          string
    ExpiresAt     time.Time
    Details       map[string]string
}

// Providers to implement:
- AzureAuthProvider
- AWSAuthProvider
- GCPAuthProvider
- GitHubAuthProvider
```

### Real-time Status Display
```
Authentication Status (refreshed every 30s):
┌──────────────────────────────────────────┐
│ ✓ Azure    Stanley.Xie@...  Exp: 2h 15m │
│ ✗ AWS      Not logged in               │
│ ✓ GCP      user@gmail.com  Exp: 1h 45m │
│ ✓ GitHub   StanleyXie      Exp: 29d    │
└──────────────────────────────────────────┘
```

## Communication Protocol

### Orchestrator ←→ Runner

**Option 1: JSON-RPC over stdio**
```json
// Orchestrator → Runner
{
  "method": "execute",
  "params": {
    "module": "vpc",
    "action": "apply",
    "config": {
      "workspace": "dev",
      "tfvars": ["./environments/dev/vpc.tfvars"],
      "backend_config": {...}
    }
  }
}

// Runner → Orchestrator (streaming)
{
  "method": "status_update",
  "params": {
    "state": "running",
    "output": "Creating VPC...",
    "progress": 45
  }
}

// Runner → Orchestrator (completion)
{
  "method": "complete",
  "params": {
    "state": "success",
    "duration": "2m34s",
    "resources": {
      "added": 5,
      "changed": 0,
      "destroyed": 0
    }
  }
}
```

## Technology Stack

### Orchestrator (TUI)
- **Framework**: Bubble Tea
- **Language**: Go
- **Config Parser**: gopkg.in/yaml.v3
- **Graph**: Custom DAG implementation
- **Logging**: zerolog

### Runner (CLI)
- **Framework**: Cobra (simple CLI)
- **Language**: Go
- **Terraform Execution**: os/exec
- **Communication**: JSON-RPC or line protocol

### Shared
- **Protocol**: JSON-RPC 2.0
- **Logging Format**: JSON Lines
- **Config Format**: YAML

## Implementation Roadmap

### Sprint 1: Foundation (Week 1-2)
- [ ] Define configuration schema (tfpipboy.yaml)
- [ ] Implement configuration parser with validation
- [ ] Create basic Runner CLI structure
- [ ] Implement JSON-RPC protocol

### Sprint 2: Core Orchestration (Week 3-4)
- [ ] Build dependency graph engine
- [ ] Implement pipeline execution logic
- [ ] Create Orchestrator-Runner communication
- [ ] Basic TUI shell with Bubble Tea

### Sprint 3: TUI Dashboard (Week 5-6)
- [ ] Pipeline execution view
- [ ] Module status display
- [ ] Log viewer component
- [ ] Keyboard navigation

### Sprint 4: Lifecycle Management (Week 7-8)
- [ ] Runner state machine
- [ ] Error handling and retries
- [ ] Cancellation and cleanup
- [ ] Resource management

### Sprint 5: Environment Sensing (Week 9-10)
- [ ] Git branch detection
- [ ] Environment auto-mapping
- [ ] Configuration auto-loading
- [ ] Smart defaults

### Sprint 6: Auth Tracking (Week 11-12)
- [ ] Multi-provider auth checkers
- [ ] Real-time status updates
- [ ] Expiration warnings
- [ ] Auth status display in TUI

### Sprint 7: Polish & Testing (Week 13-14)
- [ ] Integration testing
- [ ] Error scenario handling
- [ ] Documentation
- [ ] Performance optimization

## Open Questions for Discussion

### 1. Configuration Format Details
**Question**: Should we support both YAML and HCL (Terraform native)?
**Options**:
- YAML only (simpler, standard for CI/CD)
- HCL support (consistent with Terraform)
- Both formats

**Recommendation**: Start with YAML, add HCL later if needed

### 2. Runner Isolation Level
**Question**: Should Runners run in separate processes or goroutines?
**Options**:
- Separate processes (true isolation, easier debugging)
- Goroutines with isolation (lighter weight, shared memory issues)

**Recommendation**: Separate processes for true isolation

### 3. Interactive Approvals
**Question**: How to handle terraform approval prompts?
**Options**:
- Always use --auto-approve (based on environment)
- TUI shows plan, user approves, then auto-approve
- Suspend TUI, show native terraform prompt

**Recommendation**: Option 2 - TUI approval with plan preview

### 4. Error Recovery Strategy
**Question**: What happens when a module fails?
**Options**:
- Stop all (fail-fast)
- Continue with independent modules
- User chooses at runtime

**Recommendation**: Default stop-all, optional continue flag

### 5. State Management
**Question**: How to handle Terraform state inspection?
**Options**:
- Read-only state viewer in TUI
- No state inspection (use terraform state commands)
- Full state management UI

**Recommendation**: Read-only viewer for outputs/resources

## Next Steps

1. Review and approve this design document
2. Refine configuration schema
3. Create POC for Orchestrator-Runner communication
4. Build dependency graph engine
5. Implement basic TUI framework
6. Develop first working pipeline

## Conclusion

The Orchestrator-Runner architecture provides:
- ✅ Clean separation of concerns
- ✅ Scalable parallel execution
- ✅ Professional CI/CD-like experience
- ✅ Full lifecycle management
- ✅ Rich monitoring and debugging
- ✅ Future-proof extensibility

This design aligns with the MVP goals and provides a solid foundation for building a production-ready Terraform orchestration tool.
