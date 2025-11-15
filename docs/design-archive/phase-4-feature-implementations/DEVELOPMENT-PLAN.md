# Development Plan & User Stories
**Date**: 2025-10-18  
**Based on**: DD-Arch001-Orchestrator-Runner-Architecture.md  
**Version**: 1.0

## Executive Summary

This document provides a comprehensive development plan with user stories for implementing the **tfpipboy Orchestrator-Runner architecture**. The plan is organized into 7 sprints over 14 weeks, focusing on delivering an MVP that enables declarative multi-module Terraform orchestration.

---

## Architecture Changes Summary

### Key Paradigm Shift

| Aspect | Previous (CLI Wrapper) | New (Orchestrator-Runner) |
|--------|----------------------|---------------------------|
| **Scope** | Single module execution | Multi-module orchestration |
| **Interface** | CLI wrapper | TUI Orchestrator + CLI Runners |
| **Configuration** | User preferences | Declarative pipeline (YAML) |
| **Execution** | Pass-through | Pipeline stages with DAG |
| **Process Model** | Single process | Multi-process (1 Orchestrator + N Runners) |

### Impact on CLAUDE.md

**Updates Required:**
1. ✅ Project overview: CLI wrapper → CI/CD Orchestrator
2. ✅ Core features: Context display → Multi-module pipelines
3. ✅ Architecture: Single component → Orchestrator-Runner dual system
4. ✅ Configuration: User config → Project pipeline config (tfpipboy.yaml)
5. ✅ Technology stack: Undecided → Go with Bubble Tea
6. ✅ Implementation priority: CLI features → Pipeline orchestration sprints

**Note**: CLAUDE.md should be updated to reflect the new architecture. The original CLI wrapper design is now superseded.

---

## Development Roadmap

### Timeline Overview

```
Sprint 1-2: Foundation (Weeks 1-4)
  └─> Configuration parsing, JSON-RPC protocol
  
Sprint 3-4: Core Engine (Weeks 5-8)
  └─> DAG execution, TUI dashboard, lifecycle management
  
Sprint 5-6: Smart Features (Weeks 9-12)
  └─> Environment sensing, auth tracking
  
Sprint 7: Polish (Weeks 13-14)
  └─> Testing, documentation, optimization
```

---

## Sprint 1: Foundation (Week 1-2)

### Goals
- Define and validate configuration schema
- Implement basic Runner CLI
- Establish communication protocol

### User Stories

#### US-101: Configuration Schema Definition
**As a** DevOps engineer  
**I want** to define my infrastructure pipeline in a declarative YAML file  
**So that** I can version-control my orchestration configuration

**Acceptance Criteria:**
- [ ] `tfpipboy.yaml` schema is documented with examples
- [ ] Schema includes: project metadata, modules, pipelines, environments
- [ ] Schema supports dependency declaration (`depends_on`)
- [ ] Schema supports parallel execution configuration
- [ ] Schema validation rules are defined

**Technical Tasks:**
- Define YAML schema structure
- Create example configurations for common patterns
- Document all configuration options
- Create JSON schema for validation

**Files to Create:**
- `design/tfpipboy-schema.yaml` (schema definition)
- `examples/tfpipboy.yaml` (sample configuration)

---

#### US-102: YAML Configuration Parser
**As a** developer  
**I want** to parse and validate tfpipboy.yaml configuration files  
**So that** I can load pipeline definitions programmatically

**Acceptance Criteria:**
- [ ] Parser reads tfpipboy.yaml from project root
- [ ] Parser validates schema against defined rules
- [ ] Parser returns structured Go data structures
- [ ] Parser provides helpful error messages for invalid config
- [ ] Parser supports environment variable substitution

**Technical Tasks:**
- Implement YAML parser using `gopkg.in/yaml.v3`
- Create configuration structs in Go
- Implement validation logic
- Add error handling with context
- Write unit tests for parser

**Files to Create:**
- `source/orchestrator/config/parser.go`
- `source/orchestrator/config/types.go`
- `source/orchestrator/config/validator.go`
- `tests/config/parser_test.go`

---

#### US-103: Runner CLI Structure
**As a** Runner process  
**I want** a basic CLI structure for executing terraform commands  
**So that** I can be invoked by the Orchestrator

**Acceptance Criteria:**
- [ ] Runner CLI accepts module path as argument
- [ ] Runner CLI accepts action (init/plan/apply/destroy)
- [ ] Runner CLI accepts configuration via flags or stdin
- [ ] Runner CLI executes terraform commands
- [ ] Runner CLI returns exit codes correctly

**Technical Tasks:**
- Create Runner CLI using Cobra
- Implement command structure: `runner exec --module <path> --action <action>`
- Add configuration parsing from flags/stdin
- Implement basic terraform execution
- Add logging infrastructure

**Files to Create:**
- `source/runner/main.go`
- `source/runner/cmd/root.go`
- `source/runner/cmd/exec.go`
- `source/runner/terraform/executor.go`

---

#### US-104: JSON-RPC Communication Protocol
**As an** Orchestrator  
**I want** to communicate with Runner processes via JSON-RPC  
**So that** I can send commands and receive status updates

**Acceptance Criteria:**
- [ ] JSON-RPC 2.0 protocol implemented over stdio
- [ ] Orchestrator can send `execute` command to Runner
- [ ] Runner sends `status_update` notifications to Orchestrator
- [ ] Runner sends `complete` notification on finish
- [ ] Protocol supports error reporting

**Technical Tasks:**
- Implement JSON-RPC 2.0 protocol handler
- Define message types: execute, status_update, complete, error
- Create serialization/deserialization logic
- Implement stdio-based transport
- Write protocol tests

**Files to Create:**
- `source/common/jsonrpc/protocol.go`
- `source/common/jsonrpc/types.go`
- `source/common/jsonrpc/transport.go`
- `tests/jsonrpc/protocol_test.go`

---

## Sprint 2: Core Orchestration (Week 3-4)

### Goals
- Build dependency graph engine
- Implement pipeline execution logic
- Create Orchestrator-Runner lifecycle management

### User Stories

#### US-201: Dependency Graph (DAG) Engine
**As an** Orchestrator  
**I want** to build a dependency graph from module definitions  
**So that** I can determine execution order and identify parallel opportunities

**Acceptance Criteria:**
- [ ] DAG builder parses module dependencies from config
- [ ] DAG detects circular dependencies and reports errors
- [ ] DAG calculates topological sort for execution order
- [ ] DAG identifies modules that can run in parallel
- [ ] DAG supports multi-stage pipelines

**Technical Tasks:**
- Implement DAG data structure
- Create topological sort algorithm
- Add circular dependency detection
- Implement parallel execution planning
- Write comprehensive graph tests

**Files to Create:**
- `source/orchestrator/graph/dag.go`
- `source/orchestrator/graph/builder.go`
- `source/orchestrator/graph/sort.go`
- `tests/graph/dag_test.go`

---

#### US-202: Pipeline Execution Engine
**As an** Orchestrator  
**I want** to execute pipeline stages in correct order  
**So that** dependent modules run after their dependencies complete

**Acceptance Criteria:**
- [ ] Executor respects dependency order from DAG
- [ ] Executor runs independent modules in parallel
- [ ] Executor respects `parallel_limit` configuration
- [ ] Executor stops on failure (fail-fast mode)
- [ ] Executor tracks per-module execution state

**Technical Tasks:**
- Implement pipeline executor with goroutine pool
- Create execution scheduler based on DAG
- Add parallel execution with concurrency limits
- Implement state tracking for all modules
- Add failure handling logic

**Files to Create:**
- `source/orchestrator/executor/pipeline.go`
- `source/orchestrator/executor/scheduler.go`
- `source/orchestrator/executor/state.go`
- `tests/executor/pipeline_test.go`

---

#### US-203: Runner Process Lifecycle Management
**As an** Orchestrator  
**I want** to spawn, monitor, and cleanup Runner processes  
**So that** I can manage the full lifecycle of module execution

**Acceptance Criteria:**
- [ ] Orchestrator spawns Runner process per module
- [ ] Orchestrator passes configuration to Runner via JSON-RPC
- [ ] Orchestrator receives status updates from Runner
- [ ] Orchestrator handles Runner process termination (graceful + forced)
- [ ] Orchestrator cleans up resources on completion

**Technical Tasks:**
- Implement process spawning using `os/exec`
- Create Runner lifecycle manager
- Add stdio-based JSON-RPC communication
- Implement graceful shutdown with timeout
- Add resource cleanup logic

**Files to Create:**
- `source/orchestrator/runner/manager.go`
- `source/orchestrator/runner/lifecycle.go`
- `source/orchestrator/runner/process.go`
- `tests/runner/manager_test.go`

---

#### US-204: Basic TUI Shell with Bubble Tea
**As a** user  
**I want** a basic TUI interface for the Orchestrator  
**So that** I can see that the foundation is working

**Acceptance Criteria:**
- [ ] TUI initializes with Bubble Tea framework
- [ ] TUI displays basic pipeline information
- [ ] TUI shows placeholder for module list
- [ ] TUI supports keyboard quit (q/Ctrl+C)
- [ ] TUI renders without errors

**Technical Tasks:**
- Set up Bubble Tea application structure
- Create main TUI model and update loop
- Implement basic view rendering
- Add keyboard input handling
- Create TUI initialization

**Files to Create:**
- `source/orchestrator/tui/app.go`
- `source/orchestrator/tui/model.go`
- `source/orchestrator/tui/view.go`
- `source/orchestrator/tui/update.go`

---

## Sprint 3: TUI Dashboard (Week 5-6)

### Goals
- Build professional TUI interface
- Implement real-time status display
- Create log viewer component

### User Stories

#### US-301: Pipeline Execution View
**As a** user  
**I want** to see the current pipeline execution status  
**So that** I know what stages are running and what's completed

**Acceptance Criteria:**
- [ ] TUI displays pipeline name and environment
- [ ] TUI shows all pipeline stages with status (PENDING/RUNNING/DONE/FAILED)
- [ ] TUI displays per-module status with icons (✓ ↻ ✗ ⏸)
- [ ] TUI shows execution duration for each module
- [ ] TUI updates in real-time as modules progress

**Technical Tasks:**
- Design pipeline view layout
- Implement stage rendering with status
- Add module status indicators
- Create real-time update mechanism
- Add color coding for different states

**Files to Create:**
- `source/orchestrator/tui/views/pipeline.go`
- `source/orchestrator/tui/components/stage.go`
- `source/orchestrator/tui/components/module.go`
- `source/orchestrator/tui/styles.go`

---

#### US-302: Module Status Display
**As a** user  
**I want** to see detailed status for each module  
**So that** I understand what each Runner is doing

**Acceptance Criteria:**
- [ ] Display shows module name and current action
- [ ] Display shows execution time
- [ ] Display shows progress percentage (if available)
- [ ] Display shows resource count (e.g., "5 resources")
- [ ] Display shows last log line for running modules

**Technical Tasks:**
- Create module detail component
- Implement progress bar rendering
- Add resource count display
- Implement log line preview
- Add status color coding

**Files to Create:**
- `source/orchestrator/tui/components/module_detail.go`
- `source/orchestrator/tui/components/progress.go`

---

#### US-303: Log Viewer Component
**As a** user  
**I want** to view full logs for a selected module  
**So that** I can debug issues and see detailed terraform output

**Acceptance Criteria:**
- [ ] User can press Enter to view logs for selected module
- [ ] Log viewer displays full terraform output
- [ ] Log viewer supports scrolling (↑↓, PgUp/PgDn)
- [ ] Log viewer supports search (/)
- [ ] Log viewer can be closed (Esc)

**Technical Tasks:**
- Create log viewer model
- Implement scrolling buffer
- Add search functionality
- Create log viewer UI layout
- Add keyboard navigation

**Files to Create:**
- `source/orchestrator/tui/views/logs.go`
- `source/orchestrator/tui/components/scrollable.go`
- `source/orchestrator/tui/components/search.go`

---

#### US-304: Keyboard Navigation
**As a** user  
**I want** to navigate the TUI with keyboard shortcuts  
**So that** I can efficiently interact with the Orchestrator

**Acceptance Criteria:**
- [ ] ↑↓ keys navigate between modules
- [ ] Enter opens log viewer for selected module
- [ ] P pauses execution (if supported)
- [ ] R resumes execution
- [ ] Q quits the application
- [ ] Help screen (?) shows all shortcuts

**Technical Tasks:**
- Implement keyboard event handling
- Create navigation state management
- Add help screen view
- Implement pause/resume logic
- Add keyboard shortcut hints in UI

**Files to Create:**
- `source/orchestrator/tui/keyboard.go`
- `source/orchestrator/tui/views/help.go`

---

## Sprint 4: Lifecycle Management (Week 7-8)

### Goals
- Implement Runner state machine
- Add error handling and retry logic
- Implement cancellation and cleanup

### User Stories

#### US-401: Runner State Machine
**As a** Runner  
**I want** a well-defined state machine for my lifecycle  
**So that** the Orchestrator can track my progress accurately

**Acceptance Criteria:**
- [ ] Runner transitions through states: INIT → WAITING → STARTING → RUNNING → SUCCESS/FAILED
- [ ] Runner reports state changes to Orchestrator
- [ ] Runner validates state transitions (no invalid jumps)
- [ ] Runner handles errors at each state
- [ ] Runner supports CANCELLED state

**Technical Tasks:**
- Define state enum and transitions
- Implement state machine logic
- Add state validation
- Create state change notifications
- Add state-specific behavior

**Files to Create:**
- `source/runner/state/machine.go`
- `source/runner/state/types.go`
- `source/runner/state/transitions.go`
- `tests/state/machine_test.go`

---

#### US-402: Error Handling and Retry Logic
**As an** Orchestrator  
**I want** to retry failed modules automatically  
**So that** transient errors don't fail the entire pipeline

**Acceptance Criteria:**
- [ ] Orchestrator retries failed modules based on `retry_failed` config
- [ ] Orchestrator waits between retries (exponential backoff)
- [ ] Orchestrator shows retry count in TUI
- [ ] Orchestrator marks module as permanently failed after max retries
- [ ] Orchestrator logs retry attempts

**Technical Tasks:**
- Implement retry logic with exponential backoff
- Add retry counter to module state
- Create retry configuration handling
- Add retry status to TUI
- Implement logging for retries

**Files to Create:**
- `source/orchestrator/executor/retry.go`
- `source/orchestrator/executor/backoff.go`
- `tests/executor/retry_test.go`

---

#### US-403: Cancellation and Graceful Shutdown
**As a** user  
**I want** to cancel pipeline execution gracefully  
**So that** I can stop long-running operations cleanly

**Acceptance Criteria:**
- [ ] User can press Ctrl+C to cancel execution
- [ ] Orchestrator sends cancellation signal to all running Runners
- [ ] Runners complete current terraform command then exit
- [ ] Orchestrator waits for graceful shutdown (with timeout)
- [ ] Orchestrator force-kills Runners after timeout
- [ ] TUI shows cancellation progress

**Technical Tasks:**
- Implement signal handling (SIGINT, SIGTERM)
- Create cancellation context propagation
- Add graceful shutdown logic with timeout
- Implement force-kill fallback
- Add cancellation status to TUI

**Files to Create:**
- `source/orchestrator/executor/cancel.go`
- `source/orchestrator/signals/handler.go`
- `source/runner/cancel.go`

---

#### US-404: Resource Cleanup
**As an** Orchestrator  
**I want** to clean up all resources on completion  
**So that** no processes or files are left behind

**Acceptance Criteria:**
- [ ] Orchestrator closes all Runner stdio pipes
- [ ] Orchestrator archives logs to `~/.tfpipboy/logs/`
- [ ] Orchestrator removes temporary files
- [ ] Orchestrator releases all file handles
- [ ] Cleanup happens even on error/cancellation

**Technical Tasks:**
- Implement cleanup manager
- Add deferred cleanup handlers
- Create log archival logic
- Implement temp file management
- Add cleanup to all exit paths

**Files to Create:**
- `source/orchestrator/cleanup/manager.go`
- `source/orchestrator/cleanup/logs.go`
- `source/orchestrator/cleanup/temp.go`

---

## Sprint 5: Environment Sensing (Week 9-10)

### Goals
- Implement Git branch detection
- Create environment auto-mapping
- Build configuration auto-loading

### User Stories

#### US-501: Git Branch Detection
**As an** Orchestrator  
**I want** to detect the current Git branch  
**So that** I can automatically determine the target environment

**Acceptance Criteria:**
- [ ] Orchestrator detects current Git branch using git commands
- [ ] Orchestrator handles non-git directories gracefully
- [ ] Orchestrator caches branch detection result
- [ ] Orchestrator re-detects on branch switch (if possible)
- [ ] Orchestrator shows detected branch in TUI

**Technical Tasks:**
- Implement Git branch detection using `git rev-parse --abbrev-ref HEAD`
- Add git command execution wrapper
- Create branch detection cache
- Add error handling for non-git directories
- Display branch in TUI header

**Files to Create:**
- `source/orchestrator/git/detector.go`
- `source/orchestrator/git/branch.go`
- `tests/git/detector_test.go`

---

#### US-502: Environment Auto-Mapping
**As a** user  
**I want** environments to be selected based on my Git branch  
**So that** I don't have to manually specify dev/staging/prod

**Acceptance Criteria:**
- [ ] Orchestrator maps branch to environment using `branch_mapping` config
- [ ] Orchestrator supports exact match (e.g., `main: prod`)
- [ ] Orchestrator supports pattern match (e.g., `feature/*: dev`)
- [ ] Orchestrator falls back to `default` environment if no match
- [ ] Orchestrator allows manual override via flag

**Technical Tasks:**
- Implement branch pattern matching
- Create environment resolver
- Add mapping logic from config
- Implement CLI flag override (`--env prod`)
- Add environment display to TUI

**Files to Create:**
- `source/orchestrator/environment/resolver.go`
- `source/orchestrator/environment/matcher.go`
- `tests/environment/resolver_test.go`

---

#### US-503: Configuration Auto-Loading
**As an** Orchestrator  
**I want** to automatically load environment-specific configuration  
**So that** modules get the correct variables and backend settings

**Acceptance Criteria:**
- [ ] Orchestrator loads backend config for detected environment
- [ ] Orchestrator loads workspace for detected environment
- [ ] Orchestrator loads variable files from `variables_path`
- [ ] Orchestrator applies auto-approve setting
- [ ] Orchestrator passes all config to Runners

**Technical Tasks:**
- Implement environment config loader
- Create variable file discovery
- Add backend config resolution
- Implement config merging logic
- Pass config to Runners via JSON-RPC

**Files to Create:**
- `source/orchestrator/environment/loader.go`
- `source/orchestrator/environment/variables.go`
- `source/orchestrator/environment/backend.go`

---

#### US-504: Smart Defaults and Validation
**As a** user  
**I want** sensible defaults and validation for environment config  
**So that** I catch configuration errors early

**Acceptance Criteria:**
- [ ] Validator checks that all referenced modules exist
- [ ] Validator checks that environment configs are complete
- [ ] Validator warns about missing variable files
- [ ] Validator provides default values where appropriate
- [ ] Validator shows helpful error messages

**Technical Tasks:**
- Implement config validator
- Add module path validation
- Create variable file checker
- Implement default value logic
- Add validation error formatting

**Files to Create:**
- `source/orchestrator/config/validator.go`
- `source/orchestrator/config/defaults.go`
- `tests/config/validator_test.go`

---

## Sprint 6: Auth Tracking (Week 11-12)

### Goals
- Implement multi-cloud auth checkers
- Add real-time status updates
- Display auth status in TUI

### User Stories

#### US-601: Multi-Provider Auth Checkers
**As an** Orchestrator  
**I want** to check authentication status for multiple cloud providers  
**So that** users know if they're logged in before running pipelines

**Acceptance Criteria:**
- [ ] Checker for Azure (`az account show`)
- [ ] Checker for AWS (`aws sts get-caller-identity`)
- [ ] Checker for GCP (`gcloud auth list --filter=status:ACTIVE`)
- [ ] Checker for GitHub (`gh auth status`)
- [ ] All checkers run in parallel
- [ ] All checkers have 2s timeout

**Technical Tasks:**
- Create AuthProvider interface
- Implement AzureAuthProvider
- Implement AWSAuthProvider
- Implement GCPAuthProvider
- Implement GitHubAuthProvider
- Add parallel execution logic

**Files to Create:**
- `source/orchestrator/auth/provider.go`
- `source/orchestrator/auth/azure.go`
- `source/orchestrator/auth/aws.go`
- `source/orchestrator/auth/gcp.go`
- `source/orchestrator/auth/github.go`
- `tests/auth/providers_test.go`

---

#### US-602: Auth Status Caching
**As an** Orchestrator  
**I want** to cache auth status checks  
**So that** I don't slow down the application with repeated checks

**Acceptance Criteria:**
- [ ] Auth status cached with 60s TTL (configurable)
- [ ] Cache invalidated after TTL expires
- [ ] Cache stored in memory (not persisted)
- [ ] Cache provides immediate results on hit
- [ ] Cache logs cache hit/miss for debugging

**Technical Tasks:**
- Implement in-memory cache with TTL
- Add cache key generation
- Create cache invalidation logic
- Add cache statistics
- Implement cache configuration

**Files to Create:**
- `source/orchestrator/auth/cache.go`
- `source/common/cache/ttl.go`
- `tests/auth/cache_test.go`

---

#### US-603: Real-time Auth Status Updates
**As a** user  
**I want** auth status to refresh periodically  
**So that** I see up-to-date login status during long-running pipelines

**Acceptance Criteria:**
- [ ] Auth status refreshes every 30s in TUI
- [ ] Refresh happens in background (non-blocking)
- [ ] TUI shows refresh indicator during check
- [ ] Expired credentials show warning icon
- [ ] User can manually refresh (Shift+R)

**Technical Tasks:**
- Implement background refresh ticker
- Add refresh indicator to TUI
- Create expiration time tracking
- Add manual refresh handler
- Display expiration warnings

**Files to Create:**
- `source/orchestrator/auth/refresher.go`
- `source/orchestrator/tui/components/auth_status.go`

---

#### US-604: Auth Status Display in TUI
**As a** user  
**I want** to see authentication status in the TUI header  
**So that** I'm aware of my login status at all times

**Acceptance Criteria:**
- [ ] TUI header shows auth status for all configured providers
- [ ] Status shows: ✓ (authenticated), ✗ (not authenticated), ? (checking)
- [ ] Status shows account/user identifier
- [ ] Status shows expiration time if available
- [ ] Status uses color coding (green/red/yellow)

**Technical Tasks:**
- Design auth status header layout
- Implement status icon rendering
- Add account details display
- Create expiration time formatter
- Add color coding logic

**Files to Create:**
- `source/orchestrator/tui/components/auth_header.go`
- `source/orchestrator/tui/formatters/auth.go`

---

## Sprint 7: Polish & Testing (Week 13-14)

### Goals
- Comprehensive testing
- Documentation
- Performance optimization
- Bug fixes

### User Stories

#### US-701: Integration Test Suite
**As a** developer  
**I want** comprehensive integration tests  
**So that** I'm confident the system works end-to-end

**Acceptance Criteria:**
- [ ] Test: Full pipeline execution with mock terraform
- [ ] Test: Dependency resolution and parallel execution
- [ ] Test: Failure handling and retry logic
- [ ] Test: Cancellation and cleanup
- [ ] Test: Environment auto-detection
- [ ] All tests pass in CI/CD

**Technical Tasks:**
- Create integration test framework
- Mock terraform executable
- Create test fixtures and configs
- Write end-to-end test scenarios
- Add CI/CD pipeline configuration

**Files to Create:**
- `tests/integration/pipeline_test.go`
- `tests/integration/fixtures/`
- `tests/mocks/terraform.go`
- `.github/workflows/test.yml`

---

#### US-702: User Documentation
**As a** new user  
**I want** comprehensive documentation  
**So that** I can get started quickly

**Acceptance Criteria:**
- [ ] README with quick start guide
- [ ] Configuration reference documentation
- [ ] Example pipelines for common scenarios
- [ ] Troubleshooting guide
- [ ] Architecture overview

**Technical Tasks:**
- Write README.md with installation and quick start
- Document tfpipboy.yaml schema
- Create example configurations
- Write troubleshooting guide
- Add architecture diagrams

**Files to Create:**
- `README.md`
- `docs/configuration.md`
- `docs/examples/`
- `docs/troubleshooting.md`
- `docs/architecture.md`

---

#### US-703: Performance Optimization
**As a** developer  
**I want** the Orchestrator to start quickly and run efficiently  
**So that** users have a smooth experience

**Acceptance Criteria:**
- [ ] Orchestrator startup < 500ms
- [ ] Auth checks complete in < 2s (parallel)
- [ ] TUI renders at 60fps (no lag)
- [ ] Memory usage < 50MB for Orchestrator
- [ ] Runner overhead < 50ms per module

**Technical Tasks:**
- Profile startup time and identify bottlenecks
- Optimize config parsing
- Tune goroutine pool size
- Optimize TUI rendering
- Add performance benchmarks

**Files to Create:**
- `tests/benchmarks/startup_test.go`
- `tests/benchmarks/execution_test.go`

---

#### US-704: Error Scenario Handling
**As a** developer  
**I want** graceful error handling for all edge cases  
**So that** users get helpful messages instead of crashes

**Acceptance Criteria:**
- [ ] Terraform not installed → Clear error message
- [ ] Invalid config → Validation errors with line numbers
- [ ] Module not found → Helpful path suggestion
- [ ] Network timeout → Retry with exponential backoff
- [ ] No TTY → Fallback to line-based output

**Technical Tasks:**
- Audit all error paths
- Add context to all errors
- Create user-friendly error messages
- Add fallback for non-TTY environments
- Test all error scenarios

**Files to Create:**
- `source/common/errors/types.go`
- `source/common/errors/formatter.go`
- `tests/errors/scenarios_test.go`

---

## User Story Summary

### Sprint Breakdown

| Sprint | User Stories | Focus Area |
|--------|--------------|------------|
| Sprint 1 | US-101 to US-104 (4 stories) | Configuration & Protocol |
| Sprint 2 | US-201 to US-204 (4 stories) | Core Engine |
| Sprint 3 | US-301 to US-304 (4 stories) | TUI Dashboard |
| Sprint 4 | US-401 to US-404 (4 stories) | Lifecycle & Errors |
| Sprint 5 | US-501 to US-504 (4 stories) | Environment Sensing |
| Sprint 6 | US-601 to US-604 (4 stories) | Auth Tracking |
| Sprint 7 | US-701 to US-704 (4 stories) | Testing & Polish |

**Total**: 28 User Stories across 7 sprints (14 weeks)

---

## Technical Debt & Future Work

### Known Limitations (To Address Post-MVP)
- [ ] No HCL configuration format support (only YAML)
- [ ] No remote state inspection in TUI
- [ ] No cost estimation integration
- [ ] No web dashboard for team visibility
- [ ] No plugin system for custom providers

### Future Enhancements (Post-v1.0)
- [ ] Ghostty-specific optimizations
- [ ] Interactive plan approval in TUI
- [ ] Module output dependency resolution
- [ ] Remote execution support
- [ ] Team collaboration features

---

## Success Criteria for MVP

### Functional Requirements
- ✅ Parse and validate tfpipboy.yaml configuration
- ✅ Execute multi-module pipelines with dependency resolution
- ✅ Display real-time status in TUI
- ✅ Auto-detect environment from Git branch
- ✅ Track multi-cloud authentication status
- ✅ Handle errors gracefully with retry logic

### Non-Functional Requirements
- ✅ Startup time < 500ms
- ✅ Support 10+ modules in parallel
- ✅ Memory usage < 50MB (Orchestrator)
- ✅ Works on macOS, Linux, Windows
- ✅ Single binary distribution

### Quality Requirements
- ✅ 80%+ test coverage
- ✅ Zero critical bugs
- ✅ Comprehensive documentation
- ✅ Pass all integration tests

---

## Next Steps

1. **Review this development plan** with stakeholders
2. **Update CLAUDE.md** to reflect new architecture
3. **Create Sprint 1 tasks** in issue tracker
4. **Set up Go project structure** (`go mod init`)
5. **Begin US-101**: Configuration schema definition

---

## Appendix: File Structure

```
tfpipboy/
├── source/
│   ├── orchestrator/
│   │   ├── main.go
│   │   ├── config/
│   │   │   ├── parser.go
│   │   │   ├── types.go
│   │   │   └── validator.go
│   │   ├── graph/
│   │   │   ├── dag.go
│   │   │   └── builder.go
│   │   ├── executor/
│   │   │   ├── pipeline.go
│   │   │   ├── scheduler.go
│   │   │   └── retry.go
│   │   ├── runner/
│   │   │   ├── manager.go
│   │   │   └── lifecycle.go
│   │   ├── tui/
│   │   │   ├── app.go
│   │   │   ├── model.go
│   │   │   ├── views/
│   │   │   └── components/
│   │   ├── auth/
│   │   │   ├── provider.go
│   │   │   └── azure.go
│   │   ├── environment/
│   │   │   └── resolver.go
│   │   └── git/
│   │       └── detector.go
│   ├── runner/
│   │   ├── main.go
│   │   ├── cmd/
│   │   │   └── exec.go
│   │   ├── terraform/
│   │   │   └── executor.go
│   │   └── state/
│   │       └── machine.go
│   └── common/
│       ├── jsonrpc/
│       │   └── protocol.go
│       └── cache/
│           └── ttl.go
├── tests/
│   ├── integration/
│   ├── benchmarks/
│   └── mocks/
├── examples/
│   └── tfpipboy.yaml
├── docs/
│   ├── configuration.md
│   └── examples/
└── design/
    ├── tfpipboy-schema.yaml
    └── DEVELOPMENT-PLAN.md (this file)
```

---

**Document Status**: Draft v1.0  
**Last Updated**: 2025-10-18  
**Next Review**: Start of Sprint 1
