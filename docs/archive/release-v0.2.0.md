# Release Notes: v0.2.0

**Release Date**: November 2, 2025  
**Tag**: v0.2.0  
**Branch**: feature/cli-wrapper

## 🎉 Major Release: Instance-Based Orchestration

This is a major feature release introducing instance-based orchestration, execution plan preview, and enhanced visual output.

---

## 📦 Installation

### From Source
```bash
git clone https://github.com/yourusername/tfpipboy.git
cd tfpipboy
git checkout v0.2.0
go build -o tfpipboy cmd/tfpipboy/main.go
```

### Binary
```bash
# Download binary (when available)
./tfpipboy --version
# tfpipboy version 0.2.0
```

---

## 🚀 What's New

### 1. Instance-Based Architecture

**Modules are now templates, instances are deployments.**

```yaml
modules:
  seed:
    path: "terraform/seed"
    instances:
      seed-prod-eastus:      # User-defined instance name
        environment: "production"
        region: "eastus"
        backend:
          type: "azurerm"
          key: "seed-prod-eastus.tfstate"
```

**Benefits:**
- ✅ Flexible instance naming (no patterns enforced)
- ✅ Deploy same module multiple times with different configs
- ✅ Environment and region as optional metadata
- ✅ Instance-level dependencies
- ✅ Per-instance workspaces and logs

---

### 2. Interactive Execution Plan Preview

**See what will execute before running:**

```
================================================================================
   EXECUTION PLAN PREVIEW
================================================================================
Operation:   plan
Environment: default
Total Jobs:  2

Authentication Status:
  ✓ Azure:   atlz-bootstrap-dev

Execution Sequence:
--------------------------------------------------------------------------------
Stage  Instance             Backend      Details      Dependencies
--------------------------------------------------------------------------------
1      seed-main-gwc        azurerm      main/gwc     -
2      core-main-gwc        azurerm      main/gwc     seed-main-gwc
--------------------------------------------------------------------------------

Do you want to proceed with this execution plan? (yes/no):
```

**Features:**
- ✅ Authentication status for Azure/AWS/GCP
- ✅ Execution order with stage numbers
- ✅ Backend types and metadata
- ✅ Dependencies visualization
- ✅ User confirmation required
- ✅ `--auto-confirm` flag for automation

---

### 3. Enhanced Visual Output

**Animated progress with color-coded indicators:**

```bash
[20:04:23] terraform init seed-main-gwc... ⠋ (4s)
[20:04:27] ✓ terraform init seed-main-gwc (6.974s)
[20:04:27] terraform plan seed-main-gwc... ⠙ (2s)
[20:04:47] ✓ terraform plan seed-main-gwc (18.346s)
```

**Features:**
- ✅ Operation-specific colors (init=cyan, plan=blue, apply=yellow, destroy=red)
- ✅ Animated spinners with elapsed time
- ✅ Color-coded results (✓ green, ✗ red, ○ yellow)
- ✅ Multi-line command formatting
- ✅ Real-time progress updates

---

### 4. Plan Artifact Persistence

**Terraform plans are saved in multiple formats:**

```
.tfpipboy/workspaces/seed-main-gwc/
├── terraform.tfplan          # Binary format
├── terraform.tfplan.json     # Machine-readable
└── terraform.tfplan.txt      # Human-readable (ANSI-clean)
```

**Benefits:**
- ✅ Review plans before applying
- ✅ Apply uses saved plan automatically
- ✅ JSON format for programmatic analysis
- ✅ Clean text logs without ANSI codes

---

### 5. Backend Type Auto-Detection

**Automatically detects backend type:**

- ✅ **Azure (azurerm)**: Detects from `storage_account_name`, `resource_group_name`
- ✅ **AWS (s3)**: Detects from `bucket` + `region`
- ✅ **GCP (gcs)**: Detects from `bucket` + `prefix`
- ✅ **External files**: Reads and parses backend.hcl files
- ✅ **Inline config**: Detects from YAML configuration fields

---

## 🎯 Use Cases

### Multi-Environment Deployments
```yaml
instances:
  app-prod-eastus:
    environment: "production"
    region: "eastus"
  
  app-staging-westus:
    environment: "staging"
    region: "westus"
```

### Multi-Cloud Infrastructure
```yaml
instances:
  infra-aws:
    backend: {type: "s3"}
  
  infra-azure:
    backend: {type: "azurerm"}
  
  infra-gcp:
    backend: {type: "gcs"}
```

### Multi-Tenant SaaS
```yaml
instances:
  tenant-acme:
    environment: "tenant-acme"
  
  tenant-globex:
    environment: "tenant-globex"
```

### Feature Branch Deployments
```yaml
instances:
  app-main:
    # Deploy from main branch
  
  app-feature-auth:
    # Deploy feature branch
```

See [INSTANCE-FLEXIBILITY-EXAMPLES.md](design/INSTANCE-FLEXIBILITY-EXAMPLES.md) for 6 complete examples.

---

## 📊 Execution Summary

**After execution, see detailed results:**

```
================================================================================
   EXECUTION SUMMARY
================================================================================
Status: ✓ SUCCESS (1m14s)
Jobs:   2 total, 2 completed, 0 failed, 0 skipped

Instance Execution Details:
--------------------------------------------------------------------------------
Instance             Status     Duration     Backend      Details
--------------------------------------------------------------------------------
seed-main-gwc        ✓          52s          azurerm      main/gwc
core-main-gwc        ✓          56s          azurerm      main/gwc
--------------------------------------------------------------------------------

Artifacts preserved:
  • Workspaces: .tfpipboy/workspaces/
  • Logs:       .tfpipboy/logs/
================================================================================
```

---

## 🔧 Command Reference

### Basic Usage
```bash
# Preview execution plan (interactive)
tfpipboy --config . --targets seed-prod-eastus --operation plan

# Auto-confirm for CI/CD
tfpipboy --config . --targets seed-prod-eastus --operation plan --auto-confirm

# Apply with saved plan
tfpipboy --config . --targets seed-prod-eastus --operation apply

# Multiple instances
tfpipboy --config . --targets seed-prod-eastus,core-prod-eastus --operation plan
```

### Cleanup
```bash
# Interactive cleanup
tfpipboy --cleanup

# Remove all artifacts
tfpipboy --cleanup --all

# Retention-based cleanup
tfpipboy --cleanup --older-than 7d
```

### Information
```bash
# Show version
tfpipboy --version

# List modules
tfpipboy --list-modules

# Help
tfpipboy --help
```

---

## 📝 Migration from v0.1.0

### Step 1: Update Configuration

**Before (v0.1.0):**
```yaml
modules:
  seed:
    path: "terraform/seed"
    backend:
      key: "seed.tfstate"
```

**After (v0.2.0):**
```yaml
modules:
  seed:
    path: "terraform/seed"
    instances:
      seed-prod-eastus:
        environment: "production"
        region: "eastus"
        backend:
          key: "seed-prod-eastus.tfstate"
```

### Step 2: Update Commands

**Before:** `--targets seed`  
**After:** `--targets seed-prod-eastus`

### Step 3: Clean Old Artifacts

```bash
tfpipboy --cleanup --all
```

See [CHANGELOG.md](CHANGELOG.md) for complete migration guide.

---

## 🐛 Known Issues

None currently. Please report issues at:
https://github.com/yourusername/tfpipboy/issues

---

## 📚 Documentation

- **[CHANGELOG.md](CHANGELOG.md)**: Complete change history
- **[ORCHESTRATOR-README.md](ORCHESTRATOR-README.md)**: Implementation guide
- **[CONFIG-FORMAT-SPEC.md](design/CONFIG-FORMAT-SPEC.md)**: Configuration reference
- **[INSTANCE-FLEXIBILITY-EXAMPLES.md](design/INSTANCE-FLEXIBILITY-EXAMPLES.md)**: Usage examples

---

## 🤝 Contributing

Contributions welcome! Please see CONTRIBUTING.md (coming soon).

---

## 📄 License

[Your License Here]

---

## 🙏 Acknowledgments

Built with:
- Go 1.21+
- Terraform CLI
- Azure CLI / AWS CLI / gcloud (optional for auth checks)

---

## 📞 Support

- Issues: https://github.com/yourusername/tfpipboy/issues
- Discussions: https://github.com/yourusername/tfpipboy/discussions

---

**Happy Orchestrating! 🎉**
