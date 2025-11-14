# Changelog

All notable changes to tf-pipboy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Enhanced Error Handling & Categorization
- **Intelligent error categorization**: Automatically detects and categorizes 15+ error types
  - Orchestrator errors (config file missing, path not found)
  - Terraform command errors (variable errors, function errors)
  - Backend/state errors (backend config, state access)
  - Authentication errors (expired credentials, access denied)
  - Provider errors (installation, configuration)
  - Resource errors (conflicts, not found)
  - Syntax errors (invalid HCL)
  - Module errors (not found, version incompatible)
  - Version errors (Terraform version mismatch)
- **User-friendly error messages**: Clear category labels and extracted error details
- **File location extraction**: Shows file path and line number for errors
- **Suggested resolutions**: Specific steps to fix each error type
- **Contextual troubleshooting**: Error-type-specific troubleshooting guides
- **Enhanced error logs**: Structured error logs with category, resolution, and full output

#### Backend Authentication Validation
- **Proactive auth checking**: Validates backend authentication before running terraform init
- **Azure (azurerm) validation**: Checks `az account show` for valid, non-expired sessions
- **AWS (s3) validation**: Checks `aws sts get-caller-identity` for valid credentials
- **GCP (gcs) validation**: Checks `gcloud auth application-default print-access-token`
- **Token expiration detection**: Detects expired Azure tokens (AADSTS errors) and AWS tokens
- **Timeout protection**: 5-second timeout for auth checks to prevent hanging
- **Clear auth guidance**: Shows exact commands to authenticate for each provider
- **Session validation**: Catches Azure cached sessions that have timed out
- **Account details display**: Shows authenticated account/subscription information

#### Plan Results Display
- **Plan results in execution summary**: Shows plan outcomes for each instance
- **Terraform-style color coding**: 
  - Green for additions (+N) and "No changes"
  - Yellow for modifications (~N)
  - Red for deletions (-N)
- **Real-time plan results in live board**: Updates as plans complete
- **Compact plan format**: e.g., "+3 ~2 -1" or "No changes"
- **"No changes" detection**: Properly detects and displays when no changes are needed

#### Parallel Execution Improvements
- **--targets-all flag**: Automatically targets all instances without specifying names
- **--concurrent flag**: Renamed from --parallel for clarity (sets concurrent execution limit)
- **Auto-discovery**: Automatically finds all instances in configuration
- **Improved column alignment**: Fixed execution summary table formatting

### Changed

#### Error Display Format
- **Categorized error headers**: Shows error type (Orchestrator/Terraform/Backend/Auth)
- **Color-coded error output**: Red for errors, yellow for warnings, cyan for info
- **Structured error sections**: Separate sections for details, resolution, and troubleshooting
- **Enhanced error logs**: Includes category, file location, and resolution in log files

#### Backend Initialization
- **Pre-init auth validation**: Authentication checked before attempting terraform init
- **Early failure detection**: Fails fast with clear message if auth is missing/expired
- **Detailed auth logging**: Logs authentication status for debugging

### Fixed
- **Azure session timeout detection**: Now catches expired Azure CLI sessions
- **Plan result parsing**: Fixed race condition in output parsing
- **Column alignment**: Fixed execution summary table alignment with color codes
- **Variable error clarity**: Better error messages for missing/invalid variables

### Technical Details

#### New Files
- `pkg/orchestrator/errors.go`: Complete error categorization system
- `pkg/orchestrator/backend_auth.go`: Backend authentication validation

#### Modified Files
- `pkg/orchestrator/executor.go`: Integrated error categorization and auth validation
- `pkg/orchestrator/display.go`: Added plan result colors and alignment
- `pkg/orchestrator/liveboard.go`: Added plan result display to live board
- `pkg/orchestrator/types.go`: Added PlanResult field to ExecutionJob
- `cmd/tfpipboy/main.go`: Added --targets-all and renamed --parallel to --concurrent

#### Error Detection Patterns
- 15+ regex patterns for common Terraform errors
- File location extraction (path/to/file.tf:123)
- Multi-line error block extraction
- ANSI code stripping for clean logs

#### Authentication Validation
- Parallel checks with context timeout
- JSON parsing for account details
- Specific error message detection (token expired, not logged in, etc.)
- CLI availability checking (az, aws, gcloud)

## [0.2.0] - 2025-11-02

### Added

#### Instance-Based Architecture
- **Instance-based orchestration model**: Modules are now templates, instances are deployments
- **Flexible instance naming**: No enforced patterns, user-defined instance names (e.g., `seed-main-gwc`, `app-prod-eastus`)
- **First-class environment and region fields**: Optional metadata fields for instances
- **Instance-level dependencies**: Dependencies between instances instead of modules
- **Instance metadata tracking**: Each execution creates `instance-metadata.json` with full context

#### Execution Plan Preview & Confirmation
- **Interactive execution plan preview**: Shows detailed plan before execution
- **Authentication status detection**: Checks Azure/AWS/GCP CLI authentication before execution
- **Execution sequence table**: Displays stages, instances, backends, and dependencies
- **Confirmation prompt**: User must confirm before execution (type yes/no)
- **`--auto-confirm` flag**: Skip confirmation for CI/CD and automated workflows
- **Parallel execution indicators**: Shows which instances run in parallel (∥ symbol)

#### Enhanced Output & Visualization
- **Operation-specific progress indicators**: Color-coded spinners for init/plan/apply/destroy
  - Init: Cyan
  - Plan: Blue  
  - Apply: Yellow
  - Destroy: Red
- **Animated spinners**: Real-time progress with elapsed time (⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)
- **Color-coded results**: Green ✓ for success, Red ✗ for failure, Yellow ○ for skipped
- **Multi-line command formatting**: Terraform commands split across lines for readability
- **Execution summary table**: Shows instance status, duration, backend, and metadata

#### Plan Artifacts & Management
- **Plan file persistence**: Saves terraform.tfplan (binary format)
- **JSON plan export**: Saves terraform.tfplan.json for programmatic access
- **Human-readable plan**: Saves terraform.tfplan.txt with ANSI codes stripped
- **Plan reuse**: Apply operations use saved plan if available
- **Plan summary display**: Shows resources to add/change/destroy after successful plan

#### Backend Configuration
- **Auto-detection of backend types**: Automatically detects azurerm, s3, gcs, or local
- **External backend file support**: Reads and detects type from backend.hcl files
- **Inline backend configuration**: Supports inline backend config in YAML
- **Dual field support**: Handles both `backend:` and `backend-config:` in YAML
- **Backend type in metadata**: Tracks backend type in instance metadata

#### Logging & Artifacts
- **ANSI-clean log files**: Strips color codes from log files for clean output
- **Per-instance workspaces**: Each instance gets isolated workspace directory
- **Per-instance logs**: Separate log directory for each instance execution
- **Persistent artifacts**: GitHub Actions-style artifact preservation
- **Artifact cleanup commands**: Manual cleanup with retention policies

### Changed

#### Architecture Changes
- **Module references**: Now resolves instance names across all modules
- **Dependency validation**: Validates standalone instance names
- **Job identification**: Uses instance names as job IDs (e.g., `seed-main-gwc` instead of `job-1`)
- **Workspace naming**: Workspaces named by instance (`.tfpipboy/workspaces/seed-main-gwc/`)
- **Log naming**: Logs organized by instance (`.tfpipboy/logs/seed-main-gwc/`)
- **Metadata files**: Renamed from `job-metadata.json` to `instance-metadata.json`

#### Configuration Format
- **Config structure**: Migrated to instance-based format with `modules.{module}.instances.{instance-id}`
- **Flexible metadata**: Environment and region are optional first-class fields
- **Instance dependencies**: `depends_on` references instance names directly

### Fixed
- **Backend type detection**: Now correctly detects backend type from external files
- **BackendCfg field handling**: Properly checks both `backend:` and `backend-config:` fields
- **Detection order**: Checks external files before inline config for correct backend type
- **Type propagation**: Backend type correctly propagated to execution summary

### Documentation
- **INSTANCE-FLEXIBILITY-EXAMPLES.md**: 6 real-world examples for different use cases:
  - Multi-environment, multi-region deployments
  - Multi-cloud infrastructure
  - Multi-tenant SaaS
  - Feature branch deployments
  - Traditional workspace-style
  - Custom metadata (versions, tenants, etc.)
- **CONFIG-FORMAT-SPEC.md**: Complete configuration format specification
- **ORCHESTRATOR-README.md**: Orchestrator implementation guide

### Technical Details

#### New Files
- `pkg/orchestrator/auth.go`: Cloud provider authentication checking
- `pkg/orchestrator/display.go`: Enhanced progress display and formatting
- `pkg/orchestrator/logwriter.go`: ANSI-clean log writer
- `design/INSTANCE-FLEXIBILITY-EXAMPLES.md`: Usage examples

#### Modified Files
- `pkg/orchestrator/types.go`: Added instance-based types and metadata
- `pkg/orchestrator/orchestrator.go`: Instance-based job creation and backend detection
- `pkg/orchestrator/executor.go`: Instance naming, metadata, and plan artifacts
- `pkg/orchestrator/workspace.go`: Backend detection from files and inline config
- `pkg/orchestrator/config.go`: Instance-level dependency validation
- `pkg/orchestrator/graph.go`: Instance name resolution
- `cmd/tfpipboy/main.go`: Execution plan preview and confirmation
- `examples/demo-project/tfproject.yaml`: Migrated to instance-based format

#### Dependency Detection
- Parallel execution with 3-second timeout per auth check
- Cached results to avoid repeated checks
- Only checks authentication for backends actually used

## [0.1.0] - 2025-10-18

### Added
- Initial CLI wrapper implementation
- Basic Terraform command passthrough
- Module dependency resolution
- Sequential execution support
- Basic workspace management
- Configuration file support (tfproject.yaml)

---

## Migration Guide: 0.1.0 → 0.2.0

### Configuration File Changes

**Old Format (0.1.0):**
```yaml
modules:
  seed:
    path: "terraform/seed"
    depends_on: []
    backend:
      type: "azurerm"
      key: "seed.tfstate"
```

**New Format (0.2.0):**
```yaml
modules:
  seed:
    path: "terraform/seed"
    instances:
      seed-prod-eastus:
        environment: "production"
        region: "eastus"
        depends_on: []
        backend:
          type: "azurerm"
          key: "seed-prod-eastus.tfstate"
```

### Command Changes

**Targeting:**
- Old: `--targets seed,core`
- New: `--targets seed-prod-eastus,core-prod-eastus` (use instance names)

**New Flags:**
- `--auto-confirm`: Skip confirmation prompt (useful for CI/CD)

### Workspace Structure Changes

**Old:**
```
.tfpipboy/
├── workspaces/
│   ├── job-1/
│   └── job-2/
└── logs/
    ├── job-1.log
    └── job-2.log
```

**New:**
```
.tfpipboy/
├── workspaces/
│   ├── seed-prod-eastus/
│   │   ├── instance-metadata.json
│   │   ├── terraform.tfplan
│   │   ├── terraform.tfplan.json
│   │   └── terraform.tfplan.txt
│   └── core-prod-eastus/
└── logs/
    ├── seed-prod-eastus/
    │   └── seed-prod-eastus.log
    └── core-prod-eastus/
```

### Breaking Changes
- Configuration file format changed to instance-based model
- Job IDs changed from `job-N` to instance names
- Metadata file renamed from `job-metadata.json` to `instance-metadata.json`
- Workspace and log directories now use instance names
- `--targets` flag now expects instance names, not module names

### Deprecations
- Module-level targeting (use instance names instead)
- Sequential job IDs (use meaningful instance names)

---

## Upgrade Instructions

1. **Update Configuration File**:
   ```bash
   # Backup existing config
   cp .tfpipboy/tfproject.yaml .tfpipboy/tfproject.yaml.backup
   
   # Migrate to instance-based format (see Migration Guide above)
   ```

2. **Clean Old Artifacts**:
   ```bash
   tfpipboy --cleanup --all
   ```

3. **Update Scripts/CI**:
   - Update target references to use instance names
   - Add `--auto-confirm` for automated workflows
   - Update artifact paths in CI/CD pipelines

4. **Test New Configuration**:
   ```bash
   # Preview without execution
   tfpipboy --config . --targets your-instance-name --operation plan
   
   # Answer 'yes' to confirm, or use --auto-confirm
   ```

---

## Notes

- All instance names are flexible and user-defined
- Environment and region fields are optional metadata
- Backend type auto-detection works with both files and inline config
- Authentication checks are smart and only run for required providers
- Plan artifacts are preserved for review and reuse
