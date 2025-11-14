# Getting Started with tf-pipboy

Welcome to tf-pipboy! This guide will help you get up and running quickly.

## What is tf-pipboy?

tf-pipboy is a Terraform orchestration tool that helps you manage complex, multi-module infrastructure deployments with:

- **Parallel Execution**: Run independent modules concurrently
- **Dependency Management**: Automatic execution ordering based on dependencies
- **Authentication Monitoring**: Real-time status for AWS, Azure, GCP, and GitHub
- **Live Progress Display**: Beautiful TUI showing real-time job status
- **Isolated Workspaces**: Each module runs in its own environment

## Prerequisites

Before you begin, ensure you have:

- **Terraform** installed (v1.0.0 or later)
- **Go** 1.21+ (if building from source)
- **Cloud CLI tools** (optional, for authentication monitoring):
  - AWS CLI (`aws`)
  - Azure CLI (`az`)
  - GCP CLI (`gcloud`)
  - GitHub CLI (`gh`)

## Installation

### Option 1: Homebrew (Recommended for macOS/Linux)

```bash
# Add the tap
brew tap stanleyxie/tap

# Install tfpipboy
brew install tfpipboy

# Verify installation
tfpipboy --version
```

### Option 2: Go Install

```bash
go install github.com/StanleyXie/tf-pipboy/cmd/tfpipboy@latest
```

### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/StanleyXie/tf-pipboy.git
cd tf-pipboy

# Build
make build

# Install to $GOPATH/bin
make install
```

## Quick Start

### 1. Create Your First Configuration

Navigate to your Terraform project directory and create the configuration directory:

```bash
cd your-terraform-project
mkdir -p .tfpipboy
```

### 2. Define Your Project

Create `.tfpipboy/tfproject.yaml`:

```yaml
version: "1.0"

project:
  name: "my-infrastructure"
  description: "My infrastructure project"

modules:
  - name: networking
    path: terraform/modules/networking
    description: "Network infrastructure"
    
  - name: database
    path: terraform/modules/database
    description: "Database infrastructure"
    depends_on:
      - networking
      
  - name: application
    path: terraform/modules/application
    description: "Application infrastructure"
    depends_on:
      - networking
      - database

instances:
  - module: networking
    workspace: "prod-network"
    
  - module: database
    workspace: "prod-db"
    
  - module: application
    workspace: "prod-app"
```

### 3. Run Your First Orchestration

```bash
# Plan all modules (dry-run)
tfpipboy --operation plan --targets-all

# Apply specific modules
tfpipboy --operation apply --targets networking,database

# Destroy everything
tfpipboy --operation destroy --targets-all
```

## Basic Operations

### Planning

Preview what changes will be made:

```bash
# Plan all modules
tfpipboy --operation plan --targets-all

# Plan specific modules
tfpipboy --operation plan --targets networking,database

# Dry run (show what would be executed)
tfpipboy --operation plan --targets-all --dry-run
```

### Applying

Apply infrastructure changes:

```bash
# Apply all modules (will prompt for confirmation)
tfpipboy --operation apply --targets-all

# Apply with auto-confirm (use with caution!)
tfpipboy --operation apply --targets-all --auto-confirm

# Apply with custom concurrency
tfpipboy --operation apply --targets-all --concurrent 5
```

### Destroying

Remove infrastructure:

```bash
# Destroy all modules
tfpipboy --operation destroy --targets-all

# Destroy specific modules
tfpipboy --operation destroy --targets application,database
```

## Understanding the Live Board

tf-pipboy includes a beautiful TUI (Text User Interface) that shows real-time status:

```
╭─────────────────────── tf-pipboy Status ───────────────────────╮
│ Project: my-infrastructure                                     │
│ Operation: apply          Concurrent: 3                        │
╰────────────────────────────────────────────────────────────────╯

╭─────────────────────── Authentication ────────────────────────╮
│ ✓ AWS     (account: 123456789012)                            │
│ ✓ Azure   (subscription: prod-subscription)                   │
│ ✗ GCP     (not authenticated)                                │
│ ✓ GitHub  (user: yourname)                                   │
╰────────────────────────────────────────────────────────────────╯

╭─────────────────────── Job Status ────────────────────────────╮
│ [✓] networking         Completed  (45s)                       │
│ [→] database           Running    (12s)                       │
│ [⧖] application        Waiting                                │
╰────────────────────────────────────────────────────────────────╯
```

**Status Indicators:**
- `✓` - Completed successfully
- `→` - Currently running
- `⧖` - Waiting (dependency not met)
- `✗` - Failed
- `○` - Pending

## Key Concepts

### Modules

A module is a Terraform configuration that manages a set of related resources.

```yaml
modules:
  - name: networking
    path: terraform/modules/networking
    description: "Network infrastructure"
```

### Instances

An instance is a specific execution of a module with its own workspace and variables.

```yaml
instances:
  - module: networking
    workspace: "prod-us-east-1"
    variables:
      region: us-east-1
      
  - module: networking
    workspace: "prod-eu-west-1"
    variables:
      region: eu-west-1
```

### Dependencies

Dependencies ensure modules are executed in the correct order.

```yaml
modules:
  - name: database
    depends_on:
      - networking  # Database requires networking to be deployed first
```

### Pipelines

Pipelines define reusable deployment workflows (advanced feature).

```yaml
pipelines:
  deploy-all:
    description: "Deploy complete infrastructure"
    stages:
      - name: Foundation
        instances:
          - networking
      - name: Services
        instances:
          - database
          - application
```

## Common Patterns

### Pattern 1: Multi-Region Deployment

```yaml
modules:
  - name: regional-infrastructure
    path: terraform/modules/regional
    
instances:
  - module: regional-infrastructure
    workspace: "prod-us-east-1"
    variables:
      region: us-east-1
      
  - module: regional-infrastructure
    workspace: "prod-eu-west-1"
    variables:
      region: eu-west-1
```

### Pattern 2: Environment Separation

```yaml
instances:
  - module: application
    workspace: "dev-app"
    variables:
      environment: development
      
  - module: application
    workspace: "prod-app"
    variables:
      environment: production
```

### Pattern 3: Layered Infrastructure

```yaml
modules:
  - name: foundation
    description: "Core infrastructure"
    
  - name: platform
    description: "Platform services"
    depends_on: [foundation]
    
  - name: application
    description: "Applications"
    depends_on: [platform]
```

## Troubleshooting

### "Module not found"

**Problem**: tfpipboy can't find your Terraform module.

**Solution**: Check that the `path` in your configuration is correct relative to your project root.

### "Authentication failed"

**Problem**: Cloud provider authentication is missing or expired.

**Solution**: 
```bash
# AWS
aws configure
aws sts get-caller-identity

# Azure
az login
az account show

# GCP
gcloud auth login
gcloud auth list
```

### "Circular dependency detected"

**Problem**: Module dependencies form a loop.

**Solution**: Review your `depends_on` configuration and remove circular references.

### "Timeout exceeded"

**Problem**: Terraform operation took too long.

**Solution**: Increase timeout:
```bash
tfpipboy --operation apply --targets-all --timeout 60m
```

## Next Steps

Now that you have the basics:

1. **Read the [User Guide](user-guide.md)** - Complete documentation of all features
2. **Review [Configuration Reference](configuration.md)** - Detailed configuration options
3. **Check [Examples](../examples/)** - Real-world configuration examples
4. **Join the Community** - See [CONTRIBUTING.md](../CONTRIBUTING.md)

## Getting Help

- **Documentation**: [docs/](../docs/)
- **Issues**: [GitHub Issues](https://github.com/StanleyXie/tf-pipboy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/StanleyXie/tf-pipboy/discussions)

---

**Ready to orchestrate?** Continue to the [User Guide](user-guide.md) for advanced features!
