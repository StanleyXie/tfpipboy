# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**tf-pipboy** is a CLI tool that wraps Terraform to provide an enhanced command-line environment with real-time context awareness. It maintains full Terraform compatibility while adding workspace context and authentication status visibility.

REPO Local path: `/Users/stanleyxie/Workspace/Projects/tf-pipboy`

### Project Structure
- `source/`: Source code for the project
- `design/`: Design documents and diagrams (see below)
- `docs/`: Documentation for the project
- `tests/`: Test files for the project
- `.CLAUDE/`: Context for Claude Code
  - `CLAUDE.md`: Guidance for Claude Code
  - `Conversation.md`: Conversation logs with Claude Code

## Design Documents

Comprehensive design documentation is available in `design/`:

1. **01-technology-stack-decision.md**: Language and framework comparison (Python vs Go vs Rust)
2. **02-open-source-components.md**: Analysis of available open-source libraries and tools
3. **03-architecture-design.md**: Detailed system architecture, components, and data flow
4. **04-ghostty-integration.md**: Strategy for integrating with Ghostty terminal emulator

**Read these documents first** before making architectural decisions or starting implementation.

## Technology Stack Recommendations

### Primary Recommendation: Go
- **CLI Framework**: Cobra (industry standard, used by Terraform/kubectl/docker)
- **Display Library**: pterm (simple, beautiful) or Bubble Tea (full TUI)
- **Benefits**: Single binary distribution, fast startup, strong Terraform ecosystem fit

### Alternative: Python
- **CLI Framework**: Typer (modern, type-safe) or Click (battle-tested)
- **Display Library**: Rich (beautiful terminal output, excellent status bars)
- **Benefits**: Fastest development, excellent for rapid prototyping

**Decision Criteria**: See `design/01-technology-stack-decision.md` for detailed scoring matrix.

## Core Features

### 1. Terraform Command Passthrough
- Execute all Terraform commands as if running in native terminal
- Maintain 100% compatibility with Terraform CLI
- No breaking changes to existing Terraform workflows
- Preserve exit codes and interactive input

### 2. Runtime Environment Context Display
Real-time status display showing:
- **Backend Configuration**: Current backend type, location, and settings
- **Target State File**: Active tfstate file being used
- **Working Module**: Current Terraform module directory
- **Workspace**: Active Terraform workspace name
- **Terraform Version**: Installed Terraform version and initialization status

### 3. Authentication Status Monitoring
Track and display login status for:
- **AWS**: `aws sts get-caller-identity` (exit code 0 = authenticated)
- **Azure**: `az account show` (exit code 0 = authenticated)
- **GCP**: `gcloud auth list --filter=status:ACTIVE` (returns active accounts)
- **GitHub**: `gh auth status` (shows authentication state)

**Implementation**: Run checks in parallel, cache results (60s TTL), 2s timeout per check.

### 4. Workspace Context Visualization

**Display Modes:**
- **Header** (default): Full context box before terraform command
- **Minimal**: Single line with key info
- **Quiet**: No context display
- **Status**: Detailed status command output

**Example Header Display:**
```
╭─────────────────────── tf-pipboy context ───────────────────────╮
│ Module:    aws/vpc                    Workspace: production     │
│ Backend:   s3://my-terraform-state/vpc                          │
│ Auth:      ✓ AWS (123456789012)  ✓ Azure  ✗ GCP  ✓ GitHub     │
│ Terraform: v1.9.0 (initialized)                                 │
╰─────────────────────────────────────────────────────────────────╯
```

## Architecture

See `design/03-architecture-design.md` for complete architecture documentation.

### Key Components

1. **CLI Interface Layer**: Command parser and Terraform passthrough (Cobra/Typer)
2. **Context Aggregator**: Coordinate parallel context collection
   - **Terraform Context Manager**: Detect backend, workspace, module, version
   - **Auth Status Manager**: Check cloud provider authentication (parallel)
   - **System Info Manager**: Git branch, environment variables, user info
3. **Display Engine**: Render context with Rich/pterm/Bubble Tea
4. **Terraform Executor**: Execute terraform preserving all arguments and exit codes

### Data Flow

```
User Input → CLI Parser → Context Aggregator (parallel) → Display Engine → Terraform Executor
```

**Performance Target**: < 200ms startup overhead (mostly auth checks, cached after first run)

### Design Principles

- **Non-invasive**: Don't modify Terraform behavior or state
- **Read-only monitoring**: Query status without changing system state
- **Performance**: Context checks should be fast and non-blocking (parallel + caching)
- **Extensibility**: Easy to add new cloud provider auth checkers
- **Fail Gracefully**: If context check fails, show "?" and continue
- **Thread-Safe Execution**: All parallel execution must be thread-safe by design (see below)

## Fundamental Design Principle: Thread-Safe Parallel Execution

**CRITICAL**: tf-pipboy executes multiple Terraform jobs concurrently. All code handling parallel execution MUST be thread-safe by design.

### The Thread-Safety Rule

**NEVER use shared mutable state in concurrent execution paths.**

Instead, use one of these thread-safe patterns:

1. **Immutable Parameters (PREFERRED)**: Pass values as function parameters through the call chain
2. **Synchronization**: Use mutexes to protect shared state (only when absolutely necessary)
3. **Message Passing**: Use channels for communication between goroutines
4. **Thread-Local Storage**: Each goroutine has its own copy of the data

### Real-World Example: Output Suppression Race Condition

**Problem**: Error boxes appeared randomly during LiveBoard execution despite suppression checks.

**Root Cause**: Multiple goroutines executing jobs in parallel modified a shared field:
```go
// ❌ WRONG - Race condition!
func executeJobWithLiveUpdates(..., useLiveBoard bool) {
    pe.executor.suppressProgressBar = useLiveBoard  // Shared mutable state!
    defer func() {
        pe.executor.suppressProgressBar = originalValue
    }()
    // ... execute job ...
}
```

When Job A and Job B run in parallel:
- Job A sets `suppressProgressBar = true`
- Job B sets `suppressProgressBar = false`
- Job A reads it and gets the wrong value → error box appears!

**Solution**: Pass as immutable parameter through entire call chain:
```go
// ✅ CORRECT - Thread-safe!
func executeJobWithLiveUpdates(..., useLiveBoard bool) {
    // Pass useLiveBoard as parameter (immutable)
    err := pe.executor.ExecuteJobWithWorkspace(ctx, job, module, instance, useLiveBoard)
}

func ExecuteJobWithWorkspace(..., suppressOutput bool) {
    return e.ExecuteJob(ctx, job, suppressOutput)
}

func ExecuteJob(..., suppressOutput bool) {
    err = e.executeInit(ctx, job, workspace, suppressOutput)
    // ... other execute methods ...
}

func executeInit(..., suppressOutput bool) {
    err := e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)
}

func runTerraformCommand(..., suppressOutput bool) {
    if !suppressOutput {
        // Use the parameter safely
    }
}
```

Each goroutine has its own immutable `suppressOutput` value → no race condition.

### Thread-Safety Checklist

Before implementing parallel execution features, verify:

- [ ] **No shared mutable fields** modified by concurrent goroutines
- [ ] **Pass values as parameters** instead of storing in shared fields
- [ ] **Use mutexes** only when shared state is absolutely necessary
- [ ] **Document synchronization** strategy in comments if using mutexes
- [ ] **Test with race detector**: `go test -race ./...`
- [ ] **Test with high concurrency**: `--concurrent 16` or higher

### When Shared State Is Required

If you MUST use shared state (e.g., LiveBoard updating job statuses):

1. **Protect with mutex**: Always use `sync.Mutex` or `sync.RWMutex`
2. **Lock before read/write**: Acquire lock, do operation, release lock
3. **Keep critical sections small**: Minimize time holding the lock
4. **Document locking order**: Prevent deadlocks

Example:
```go
// ✅ CORRECT - Properly synchronized
type LiveBoard struct {
    mu    sync.RWMutex
    jobs  map[string]*LiveJobStatus
}

func (lb *LiveBoard) UpdateJobStatus(jobID string, status JobStatus) {
    lb.mu.Lock()           // Acquire lock
    defer lb.mu.Unlock()   // Always release lock
    
    job := lb.jobs[jobID]
    job.Status = status    // Safe - protected by mutex
}
```

### Additional Resources

- [Go Memory Model](https://go.dev/ref/mem)
- [Race Detector](https://go.dev/doc/articles/race_detector)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)

## Ghostty Terminal Integration

See `design/04-ghostty-integration.md` for complete strategy.

### Phase 1: Basic Terminal Features (MVP)
- **Terminal Title**: Update with workspace and module context
- **Hyperlinks (OSC 8)**: Clickable links to cloud consoles
- **Color Indicators**: Cursor color for auth status
- **Prompt Marking (OSC 133)**: Enable jump-to-prompt in Ghostty

### Future Phases
- Enhanced Ghostty-specific features (synchronized rendering)
- libghostty embedding (when stable)
- Custom Terraform output rendering

**Compatibility**: All features degrade gracefully in non-Ghostty terminals.

## Development Commands

**Note**: Project not yet initialized. Choose language first, then set up development environment.

### Future Commands (TBD based on language choice)

**If Python:**
```bash
# Setup
python -m venv venv
source venv/bin/activate
pip install -e ".[dev]"

# Development
pytest                    # Run tests
mypy source/             # Type checking
black source/            # Format code

# Installation
pipx install .           # Install globally
```

**If Go:**
```bash
# Setup
go mod init github.com/yourusername/tf-pipboy
go mod tidy

# Development
go test ./...            # Run tests
go build -o tfpipboy    # Build binary
go run main.go          # Run locally

# Installation
go install              # Install to $GOPATH/bin
```

## Configuration

### User Configuration File

**Location**: `~/.tfpipboy/config.yaml`

```yaml
display:
  mode: header              # header, minimal, quiet
  show_auth: true
  show_system: false
  color: auto

auth:
  providers: [aws, azure, gcp, github]
  cache_ttl: 60            # seconds
  timeout: 2               # seconds per check

terraform:
  executable: terraform
  pass_through_args: true
```

### Environment Variables

```bash
TFPIPBOY_MODE=minimal      # Override display mode
TFPIPBOY_NO_AUTH=1         # Disable auth checks
TFPIPBOY_SKIP_GCP=1        # Skip specific provider
TFPIPBOY_DEBUG=1           # Debug mode
```

## Implementation Priority

### MVP (Week 1-2)
1. ✅ Basic command passthrough to Terraform
2. ✅ Terraform context detection (workspace, backend)
3. ✅ Simple status display (one mode)
4. ✅ AWS authentication check

### v1.0 (Month 1)
1. ✅ All authentication providers (AWS, Azure, GCP, GitHub)
2. ✅ Multiple display modes (header, minimal, quiet)
3. ✅ Configuration file support
4. ✅ Caching for performance
5. ✅ Terminal title integration
6. ✅ Hyperlinks to cloud consoles

### v2.0 (Month 3-6)
1. 🔮 Ghostty-optimized features
2. 🔮 Additional context providers
3. 🔮 Plugin system for extensibility
4. 🔮 Team sharing / remote context

## Open Source Components

See `design/02-open-source-components.md` for detailed analysis.

### Recommended Libraries

**Python Stack:**
- Typer (CLI) + Rich (Display) + asyncio (Subprocess)

**Go Stack:**
- Cobra (CLI) + pterm (Display) + os/exec (Subprocess)

**All external dependencies are MIT or Apache 2.0 licensed.**

## Technical Considerations

### Authentication Status Detection
- Use existing CLI tools' status commands
- **Cache** auth status (60s TTL) to avoid performance impact
- **Parallel execution**: Run all checks concurrently
- **Graceful degradation**: Skip if CLI tool not installed

### Terraform Context Detection
- **Preferred**: Use `terraform show -json` for structured output
- **Alternative**: Parse `.terraform/` directory and config files
- Read `.terraform/environment` for active workspace
- Detect backend from `.terraform/terraform.tfstate` (local state pointer)

### Performance Optimization
1. **Parallel Context Collection**: Run all auth checks concurrently
2. **Caching**: Cache expensive operations (auth checks, terraform version)
3. **Timeouts**: 2-second timeout on slow checks
4. **Lazy Loading**: Only check configured providers

### Security Considerations
- **Read-only**: Never modify Terraform state or configuration
- **No credential storage**: Only check existing auth via official CLI tools
- **Respect environment**: Use existing authentication mechanisms
- **Cache permissions**: Store cache with restricted permissions (0700)

## Testing Strategy

### Unit Tests
- Context parsers (backend, workspace detection)
- Auth status parsers
- Display formatters
- Cache logic

### Integration Tests
- Command passthrough with mock terraform
- Auth check integration with mock CLI tools
- End-to-end workflow tests

### Manual Tests
- Test with each cloud provider
- Test in Ghostty and other terminals
- Test with different Terraform versions

## Getting Started (For New Contributors)

1. **Read Design Docs**: Start with `design/` folder
2. **Choose Language**: Review `design/01-technology-stack-decision.md`
3. **Setup Environment**: Install chosen language toolchain
4. **Create POC**: Build minimal command passthrough + context display
5. **Iterate**: Add features incrementally per architecture design

## Resources

- [Terraform CLI Documentation](https://developer.hashicorp.com/terraform/cli)
- [AWS CLI Authentication](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-authentication.html)
- [Azure CLI Reference](https://learn.microsoft.com/en-us/cli/azure/)
- [gcloud CLI](https://cloud.google.com/sdk/gcloud)
- [GitHub CLI Manual](https://cli.github.com/manual/)
- [Ghostty Documentation](https://ghostty.org/docs)
- [OSC 8 Hyperlinks Spec](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda)
