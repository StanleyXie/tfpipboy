# tfpipboy Architecture Design

**Date**: 2025-10-10  
**Version**: 1.0 (Initial Design)

## System Overview

tfpipboy is a context-aware Terraform CLI wrapper that provides real-time visibility into:
- Terraform runtime environment (backend, state, workspace, module)
- Cloud provider authentication status (AWS, Azure, GCP, GitHub)
- Current working context

**Design Principles:**
1. **Transparency**: Pass through all Terraform commands unchanged
2. **Non-invasive**: Read-only monitoring, no state modification
3. **Performance**: Fast context checks, minimal overhead
4. **Extensibility**: Easy to add new context providers
5. **User-Friendly**: Clear, actionable status information

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         User Input                           │
│                    $ tfpipboy plan -out=plan  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   CLI Interface Layer                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Command Parser (Cobra/Typer/Click)                 │   │
│  │  - Parse tfpipboy options                           │   │
│  │  - Extract terraform command & args                 │   │
│  └─────────────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  Context Aggregator                          │
│  ┌──────────────────┬──────────────────┬─────────────────┐ │
│  │  Terraform Ctx   │   Auth Status    │   System Info   │ │
│  │    Manager       │     Manager      │     Manager     │ │
│  └──────────────────┴──────────────────┴─────────────────┘ │
│                    (Parallel Execution)                      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   Display Engine                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Status Renderer (Rich/pterm/Bubble Tea)            │   │
│  │  - Format context information                       │   │
│  │  - Apply styling and colors                         │   │
│  │  - Handle terminal capabilities                     │   │
│  └─────────────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                 Terraform Executor                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Command Runner                                      │   │
│  │  - Execute: terraform <original args>               │   │
│  │  - Stream stdout/stderr to user                     │   │
│  │  - Preserve exit code                               │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. CLI Interface Layer

**Responsibility**: Parse user commands and options

**Interface:**
```python
# Pseudo-code (Python)
class CLIInterface:
    def parse_command(args: List[str]) -> Command:
        """Parse tfpipboy options and terraform command"""
        
    def get_display_mode() -> DisplayMode:
        """Get user preference: full, minimal, quiet"""
```

**Key Features:**
- Parse tfpipboy-specific flags (e.g., `--no-context`, `--quiet`)
- Extract terraform command and arguments
- Handle help and version commands
- Support for all terraform subcommands

**Example Usage:**
```bash
# tfpipboy options before terraform command
tfpipboy --minimal plan -out=plan.tfplan

# Just pass through to terraform
tfpipboy init
```

---

### 2. Context Aggregator

**Responsibility**: Coordinate context collection from all sources

**Interface:**
```python
class ContextAggregator:
    def collect_context() -> Context:
        """Collect all context information in parallel"""
        
    async def get_full_context() -> FullContext:
        """Async collection of all context"""
```

**Sub-components:**

#### 2.1 Terraform Context Manager

**Detects:**
- Current working directory / module path
- Backend configuration (type, location)
- Active workspace
- Terraform version
- Initialization status

**Data Sources:**
- `.terraform/` directory
- `.terraform/environment` file (workspace)
- `terraform.tf` or `backend.tf` files
- `terraform version` command
- `terraform workspace show` command

**Example Context:**
```json
{
  "module_path": "/home/user/infra/aws/vpc",
  "backend": {
    "type": "s3",
    "bucket": "my-terraform-state",
    "key": "vpc/terraform.tfstate",
    "region": "us-east-1"
  },
  "workspace": "production",
  "terraform_version": "1.9.0",
  "initialized": true
}
```

#### 2.2 Auth Status Manager

**Detects:**
- AWS CLI authentication status
- Azure CLI authentication status
- GCP (gcloud) authentication status
- GitHub CLI authentication status

**Detection Methods:**

| Provider | Command | Success Check | Cache TTL |
|----------|---------|---------------|-----------|
| AWS | `aws sts get-caller-identity` | Exit code 0 | 60s |
| Azure | `az account show` | Exit code 0 | 60s |
| GCP | `gcloud auth list --filter=status:ACTIVE` | Has output | 60s |
| GitHub | `gh auth status` | Exit code 0 | 300s |

**Optimization:**
- Run all checks in parallel
- Cache results with TTL
- Skip checks if CLI tool not installed
- Timeout: 2 seconds per check

**Example Context:**
```json
{
  "aws": {
    "authenticated": true,
    "account": "123456789012",
    "user": "arn:aws:iam::123456789012:user/developer",
    "region": "us-east-1"
  },
  "azure": {
    "authenticated": true,
    "subscription": "My Subscription",
    "account": "user@example.com"
  },
  "gcp": {
    "authenticated": true,
    "account": "user@example.com",
    "project": "my-project-123"
  },
  "github": {
    "authenticated": true,
    "user": "octocat"
  }
}
```

#### 2.3 System Info Manager

**Detects:**
- Current Git branch (if in repo)
- OS environment variables (TF_VAR_*, AWS_*, AZURE_*, etc.)
- Current user
- Hostname

**Example Context:**
```json
{
  "git_branch": "feature/new-vpc",
  "git_repo": "infrastructure",
  "user": "developer",
  "hostname": "workstation",
  "env_vars": {
    "TF_VAR_environment": "staging",
    "AWS_PROFILE": "default"
  }
}
```

---

### 3. Display Engine

**Responsibility**: Render context information to terminal

**Display Modes:**

#### Mode 1: Header Display (Default)
```
╭─────────────────────── tfpipboy context ───────────────────────╮
│ Module:    aws/vpc                    Workspace: production     │
│ Backend:   s3://my-terraform-state/vpc                          │
│ Auth:      ✓ AWS (123456789012)  ✓ Azure  ✗ GCP  ✓ GitHub     │
│ Terraform: v1.9.0 (initialized)                                 │
╰─────────────────────────────────────────────────────────────────╯

Running: terraform plan -out=plan.tfplan
```

#### Mode 2: Minimal Display
```
[tfpipboy] production @ aws/vpc | ✓ AWS ✓ Azure
Running: terraform plan
```

#### Mode 3: Quiet (No Display)
```
(Just runs terraform directly)
```

#### Mode 4: Status Command
```bash
$ tfpipboy status

Terraform Context:
  Module:         /home/user/infra/aws/vpc
  Backend:        s3 (my-terraform-state/vpc)
  Workspace:      production
  Version:        1.9.0
  Initialized:    Yes

Authentication Status:
  ✓ AWS           Authenticated as arn:aws:iam::123456789012:user/developer
                  Region: us-east-1
  ✓ Azure         Authenticated as user@example.com
                  Subscription: My Subscription (12345678-1234-1234-1234-123456789012)
  ✗ GCP           Not authenticated
  ✓ GitHub        Authenticated as octocat

System Info:
  Git Branch:     feature/new-vpc
  User:           developer
  Hostname:       workstation
```

**Display Configuration:**
```yaml
# ~/.tfpipboy/config.yaml
display:
  mode: header  # header, minimal, quiet
  show_auth: true
  show_system: false
  auth_providers:
    - aws
    - azure
    - gcp
    - github
  cache_ttl: 60  # seconds
```

---

### 4. Terraform Executor

**Responsibility**: Execute terraform with original arguments

**Key Requirements:**
- Preserve all arguments exactly
- Stream stdout/stderr in real-time
- Preserve exit code
- Handle signals (Ctrl+C)
- Support interactive input (prompts)

**Implementation:**
```python
def execute_terraform(args: List[str]) -> int:
    """Execute terraform with given arguments"""
    import subprocess
    
    process = subprocess.Popen(
        ['terraform'] + args,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        stdin=subprocess.STDIN,  # Preserve interactivity
        text=True
    )
    
    # Stream output
    for line in process.stdout:
        print(line, end='')
    
    return process.wait()  # Return terraform's exit code
```

---

## Data Flow

### Typical Command Execution Flow

```
User runs: tfpipboy plan -out=plan.tfplan

1. CLI Interface Layer
   ├─ Parse command: terraform plan -out=plan.tfplan
   ├─ Check for tfpipboy flags: none
   └─ Display mode: header (default)

2. Context Aggregator (Parallel)
   ├─ Terraform Context Manager
   │  ├─ Read .terraform/environment → "production"
   │  ├─ Read backend config → s3 backend
   │  └─ Run terraform version → 1.9.0
   │
   ├─ Auth Status Manager (Parallel)
   │  ├─ aws sts get-caller-identity → ✓ (cached, 30s old)
   │  ├─ az account show → ✓ (2s)
   │  ├─ gcloud auth list → ✗ (not installed, skip)
   │  └─ gh auth status → ✓ (cached, 60s old)
   │
   └─ System Info Manager
      ├─ git branch → feature/new-vpc
      └─ hostname → workstation

3. Display Engine
   └─ Render header with collected context

4. Terraform Executor
   ├─ Execute: terraform plan -out=plan.tfplan
   ├─ Stream output to user
   └─ Exit with terraform's exit code
```

**Total Overhead:** ~50-200ms (mostly auth checks, cached after first run)

---

## Caching Strategy

### Context Cache

**Purpose**: Avoid repeated expensive operations

**Cache Entries:**
- Auth status checks (60s TTL)
- Terraform version (300s TTL)
- Backend configuration (until directory change)

**Cache Implementation:**
```python
class ContextCache:
    cache: Dict[str, CacheEntry] = {}
    
    def get(key: str, ttl: int) -> Optional[Any]:
        entry = cache.get(key)
        if entry and entry.age() < ttl:
            return entry.value
        return None
    
    def set(key: str, value: Any):
        cache[key] = CacheEntry(value, time.now())
```

**Cache Location**: `~/.tfpipboy/cache/`

---

## Error Handling

### Principle: Fail Gracefully

**Context Collection Errors:**
- Auth check times out → Show "?" status, continue
- Terraform not found → Show warning, exit
- Backend config unreadable → Show "unknown", continue

**Display Errors:**
- Terminal too narrow → Switch to minimal mode
- No color support → Use plain text

**Terraform Execution Errors:**
- Pass through exactly as terraform would show them

---

## Configuration

### Configuration File

**Location**: `~/.tfpipboy/config.yaml`

```yaml
# Display settings
display:
  mode: header              # header, minimal, quiet
  show_auth: true           # Show auth status
  show_system: false        # Show system info
  color: auto               # auto, always, never

# Auth providers to check
auth:
  providers:
    - aws
    - azure
    - gcp
    - github
  cache_ttl: 60             # seconds
  timeout: 2                # seconds per check

# Terraform settings
terraform:
  executable: terraform     # Path to terraform binary
  pass_through_args: true   # Always pass args unchanged

# Advanced
cache:
  enabled: true
  directory: ~/.tfpipboy/cache
```

### Environment Variables

```bash
# Override display mode
export TFPIPBOY_MODE=minimal

# Disable auth checks
export TFPIPBOY_NO_AUTH=1

# Skip specific provider
export TFPIPBOY_SKIP_GCP=1

# Debug mode
export TFPIPBOY_DEBUG=1
```

---

## Extension Points

### Adding New Auth Providers

```python
class AuthProvider(ABC):
    @abstractmethod
    def name(self) -> str:
        """Provider name (e.g., 'aws')"""
    
    @abstractmethod
    def check_auth(self) -> AuthStatus:
        """Check if authenticated"""
    
    @abstractmethod
    def get_details(self) -> Dict[str, str]:
        """Get account details"""
```

**Example: Adding DigitalOcean**
```python
class DigitalOceanProvider(AuthProvider):
    def name(self) -> str:
        return "digitalocean"
    
    def check_auth(self) -> AuthStatus:
        result = run_command("doctl auth list")
        return AuthStatus(
            authenticated=result.exit_code == 0,
            details={"account": parse_account(result.stdout)}
        )
```

### Adding New Context Sources

```python
class ContextProvider(ABC):
    @abstractmethod
    def collect(self) -> Dict[str, Any]:
        """Collect context information"""
```

---

## Performance Considerations

### Startup Time Target: < 200ms

**Breakdown:**
- CLI parsing: ~5ms
- Context collection (parallel): ~100ms
  - Terraform context: ~20ms (file reads)
  - Auth checks (parallel): ~100ms (network calls)
  - System info: ~10ms
- Display rendering: ~10ms
- Terraform execution: (pass-through, no overhead)

**Optimizations:**
1. **Parallel execution**: Run all auth checks concurrently
2. **Caching**: Cache auth status for 60 seconds
3. **Lazy loading**: Only check providers configured in config
4. **Timeout**: 2-second timeout on slow checks
5. **Skip unavailable**: Don't check providers without CLI installed

---

## Security Considerations

1. **Read-only**: Never modify Terraform state or configuration
2. **No credential storage**: Only check existing auth, never store credentials
3. **Respect environment**: Use existing CLI tools' authentication
4. **No network calls**: Except through official CLI tools
5. **Cache security**: Store cache in `~/.tfpipboy/` with restricted permissions (0700)

---

## Testing Strategy

### Unit Tests
- Context parsers (backend config, workspace detection)
- Auth status parsers
- Display formatters
- Cache logic

### Integration Tests
- Command passthrough
- Auth check integration
- End-to-end with mock terraform

### Manual Tests
- Test with each cloud provider
- Test in various terminal emulators
- Test with different Terraform versions

---

## Future Enhancements

### Phase 2
- **Watch mode**: Continuous status updates while terraform runs
- **Resource preview**: Show resources that will be changed
- **Cost estimation**: Integrate with Infracost
- **Notifications**: Desktop notifications for long-running operations

### Phase 3
- **Ghostty integration**: Use libghostty when stable
- **Web dashboard**: Optional web UI for team visibility
- **Remote state browser**: Navigate state file interactively
- **AI assistance**: Suggest fixes for common errors

---

## References

- [Terraform CLI Documentation](https://developer.hashicorp.com/terraform/cli)
- [AWS CLI Authentication](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-authentication.html)
- [Azure CLI Reference](https://learn.microsoft.com/en-us/cli/azure/)
- [gcloud CLI](https://cloud.google.com/sdk/gcloud)
- [GitHub CLI](https://cli.github.com/)
