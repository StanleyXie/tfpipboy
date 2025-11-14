# tf-pipboy User Guide

Complete guide to using tf-pipboy for Terraform orchestration.

## Table of Contents

- [Overview](#overview)
- [Core Concepts](#core-concepts)
- [Command Line Interface](#command-line-interface)
- [Configuration](#configuration)
- [Operations](#operations)
- [Parallel Execution](#parallel-execution)
- [Authentication Monitoring](#authentication-monitoring)
- [Live Board Interface](#live-board-interface)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Overview

tf-pipboy is a Terraform orchestration tool that manages complex, multi-module deployments with dependency-aware execution, parallel processing, and real-time monitoring.

### Key Features

- **Dependency Management**: Automatic execution ordering based on module dependencies
- **Parallel Execution**: Run independent modules concurrently
- **Authentication Monitoring**: Real-time status for AWS, Azure, GCP, and GitHub
- **Live Board**: Beautiful TUI showing job status and progress
- **Isolated Workspaces**: Each module runs in its own environment
- **Pipeline Support**: Define reusable deployment workflows
- **Configuration Validation**: Comprehensive validation before execution

---

## Core Concepts

### Projects

A project is the top-level container for your infrastructure configuration.

```yaml
project:
  name: "my-infrastructure"
  description: "Production infrastructure"
```

### Modules

A module is a Terraform configuration directory that manages related resources.

```yaml
modules:
  - name: networking
    path: terraform/modules/networking
    description: "Network infrastructure"
    depends_on: []
```

**Key Properties:**
- `name`: Unique identifier for the module
- `path`: Path to Terraform module directory
- `description`: Human-readable description
- `depends_on`: List of module dependencies

### Instances

An instance is a specific execution of a module with its own workspace and configuration.

```yaml
instances:
  - module: networking
    workspace: "prod-us-east-1-network"
    variables:
      region: us-east-1
      environment: production
```

**Why Instances?**
- Deploy the same module to multiple regions
- Separate environments (dev, staging, prod)
- Different configurations for the same infrastructure pattern

### Dependencies

Dependencies define the execution order of modules.

```yaml
modules:
  - name: database
    depends_on:
      - networking
      
  - name: application
    depends_on:
      - networking
      - database
```

**Dependency Resolution:**
- tf-pipboy automatically determines execution order
- Modules with no dependencies run first
- Dependent modules wait for their dependencies to complete
- Independent modules can run in parallel

### Pipelines

Pipelines are reusable deployment workflows with multiple stages.

```yaml
pipelines:
  deploy-all:
    description: "Deploy complete infrastructure"
    stages:
      - name: Foundation
        instances: [networking]
      - name: Services
        instances: [database, application]
```

---

## Command Line Interface

### Basic Syntax

```bash
tfpipboy [OPTIONS]
```

### Common Options

#### Operation Selection
```bash
--operation OP           # Terraform operation to perform
                        # Values: plan, apply, destroy, validate
```

#### Target Selection
```bash
--targets MODULES       # Comma-separated list of modules or instances
--targets-all          # Target all instances in the configuration
--pipeline NAME        # Execute a specific pipeline
```

#### Execution Control
```bash
--concurrent N         # Maximum concurrent executions (default: 3)
--parallel-all        # Ignore dependencies, run everything in parallel
--auto-confirm        # Skip confirmation prompts
--dry-run            # Show what would be executed without running
```

#### Configuration
```bash
--config PATH         # Path to configuration directory (default: .tfpipboy)
--env NAME           # Environment name (default: default)
```

#### Output Control
```bash
--verbose            # Enable verbose logging (DEBUG level)
--trace             # Enable trace logging (TRACE level, most verbose)
--output-mode MODE   # Console output mode (default: liveboard_only)
```

### Examples

```bash
# Plan all modules
tfpipboy --operation plan --targets-all

# Apply specific modules
tfpipboy --operation apply --targets networking,database

# Execute pipeline
tfpipboy --pipeline deploy-all --env production

# Destroy with high concurrency
tfpipboy --operation destroy --targets-all --concurrent 10

# Dry run to see execution plan
tfpipboy --operation apply --targets-all --dry-run

# Verbose logging for debugging
tfpipboy --operation plan --targets-all --verbose
```

---

## Configuration

tf-pipboy uses a YAML-based configuration in the `.tfpipboy/` directory.

### Basic Configuration

Minimal configuration in `.tfpipboy/tfproject.yaml`:

```yaml
version: "1.0"

project:
  name: "my-infrastructure"
  description: "My infrastructure project"

modules:
  - name: networking
    path: terraform/modules/networking
    
  - name: database
    path: terraform/modules/database
    depends_on: [networking]

instances:
  - module: networking
    workspace: "prod-network"
    
  - module: database
    workspace: "prod-db"
```

### Module Configuration

```yaml
modules:
  - name: networking
    path: terraform/modules/networking
    description: "Network infrastructure"
    depends_on: []
    backend:
      type: azurerm
      resource_group_name: "rg-terraform-state"
      storage_account_name: "terraformstate"
      container_name: "tfstate"
      key: "networking.tfstate"
```

### Instance Configuration

```yaml
instances:
  - module: networking
    workspace: "prod-us-east-1-network"
    description: "US East 1 network"
    variables:
      region: us-east-1
      cidr_block: "10.0.0.0/16"
    backend:
      key: "prod-us-east-1-network.tfstate"
```

### Pipeline Configuration

```yaml
pipelines:
  deploy-all:
    description: "Deploy complete infrastructure"
    default_operation: apply
    
    settings:
      parallel_limit: 3
      timeout: "60m"
      auto_approve: false
    
    stages:
      - name: "Foundation"
        instances:
          - networking
        
      - name: "Services"
        instances:
          - database
          - application
```

See [Configuration Reference](configuration.md) for complete details.

---

## Operations

### Plan

Preview infrastructure changes without applying them.

```bash
tfpipboy --operation plan --targets-all
```

**Use Cases:**
- Review changes before applying
- Validate configuration
- Check for errors
- Generate execution plan

### Apply

Create or update infrastructure.

```bash
tfpipboy --operation apply --targets-all
```

**Options:**
- `--auto-confirm`: Skip confirmation prompt
- `--timeout DURATION`: Custom timeout (e.g., `60m`)

**Safety:**
- Prompts for confirmation by default
- Shows plan summary before applying
- Validates authentication before execution

### Destroy

Remove infrastructure.

```bash
tfpipboy --operation destroy --targets-all
```

**Warning:** This is destructive! Always review targets carefully.

### Validate

Validate Terraform configuration without executing.

```bash
tfpipboy --operation validate --targets-all
```

**Checks:**
- Terraform syntax
- Provider configuration
- Resource references
- Module structure

---

## Parallel Execution

tf-pipboy automatically parallelizes independent modules.

### How It Works

1. **Dependency Graph**: Build graph of module dependencies
2. **Topological Sort**: Determine safe execution order
3. **Stage Grouping**: Group independent modules into stages
4. **Concurrent Execution**: Run modules within each stage in parallel

### Example

Given this configuration:

```yaml
modules:
  - name: networking
    depends_on: []
    
  - name: database-primary
    depends_on: [networking]
    
  - name: database-replica
    depends_on: [networking]
    
  - name: application
    depends_on: [database-primary]
```

**Execution Stages:**
1. `networking` (runs alone)
2. `database-primary`, `database-replica` (run in parallel)
3. `application` (runs after database-primary)

### Concurrency Control

```bash
# Default: 3 concurrent jobs
tfpipboy --operation apply --targets-all

# Custom: 10 concurrent jobs
tfpipboy --operation apply --targets-all --concurrent 10

# Maximum: run everything in parallel (ignores dependencies!)
tfpipboy --operation apply --targets-all --parallel-all
```

**Recommendation:** Start with default (3) and increase based on:
- Available system resources
- Cloud provider API limits
- Network bandwidth
- Number of independent modules

---

## Authentication Monitoring

tf-pipboy monitors authentication status for cloud providers.

### Supported Providers

#### AWS
Checks via: `aws sts get-caller-identity`

```
✓ AWS (account: 123456789012, role: OrganizationAccountAccessRole)
```

#### Azure
Checks via: `az account show`

```
✓ Azure (subscription: prod-subscription, tenant: contoso.onmicrosoft.com)
```

#### GCP
Checks via: `gcloud auth list`

```
✓ GCP (account: user@example.com, project: my-project)
```

#### GitHub
Checks via: `gh auth status`

```
✓ GitHub (user: myusername)
```

### Authentication Errors

tf-pipboy blocks execution if required authentication is missing:

```
ERROR: Required authentication missing or expired

Missing authentication for:
  - Azure: Please run 'az login'
  - AWS: Please configure AWS credentials

Cannot proceed with terraform operations.
```

### Handling Expired Credentials

```bash
# AWS
aws configure
aws sts get-caller-identity

# Azure
az login
az account set --subscription "your-subscription"

# GCP
gcloud auth login
gcloud config set project your-project

# GitHub
gh auth login
```

---

## Live Board Interface

tf-pipboy includes a beautiful TUI (Text User Interface) for real-time status.

### Layout

```
╭─────────────────────── tf-pipboy Status ───────────────────────╮
│ Project: my-infrastructure        Operation: apply            │
│ Environment: production           Concurrent: 3                │
╰────────────────────────────────────────────────────────────────╯

╭─────────────────────── Authentication ────────────────────────╮
│ ✓ AWS     account: 123456789012                              │
│ ✓ Azure   subscription: prod-subscription                     │
│ ✗ GCP     not authenticated                                  │
│ ✓ GitHub  user: yourname                                     │
╰────────────────────────────────────────────────────────────────╯

╭─────────────────────── Execution Progress ────────────────────╮
│ Stage 1/3: Foundation                                         │
│   [✓] networking            Completed    45s                  │
│                                                                │
│ Stage 2/3: Services (2 running)                              │
│   [→] database-primary      Running      23s                  │
│   [→] database-replica      Running      23s                  │
│                                                                │
│ Stage 3/3: Application                                        │
│   [⧖] application           Waiting                           │
╰────────────────────────────────────────────────────────────────╯

╭─────────────────────── Recent Logs ───────────────────────────╮
│ [networking] Terraform apply complete! Resources: 15 added    │
│ [database-primary] Creating database instance...              │
│ [database-replica] Creating read replica...                   │
╰────────────────────────────────────────────────────────────────╯
```

### Status Indicators

- `✓` - Completed successfully
- `→` - Currently running
- `⧖` - Waiting for dependencies
- `✗` - Failed
- `○` - Pending (not started)

### Keyboard Controls

- `q` or `Ctrl+C` - Graceful shutdown
- Arrow keys - Scroll through logs
- `Enter` - View detailed logs for selected job

---

## Advanced Features

### Module Instances

Deploy the same module multiple times with different configurations:

```yaml
modules:
  - name: regional-app
    path: terraform/modules/application

instances:
  - module: regional-app
    workspace: "prod-us-east-1-app"
    variables:
      region: us-east-1
      
  - module: regional-app
    workspace: "prod-eu-west-1-app"
    variables:
      region: eu-west-1
```

### Variable Substitution

Use variables in configuration:

```yaml
variables:
  environment: production
  region: us-east-1

instances:
  - module: application
    workspace: "${environment}-${region}-app"
    variables:
      env: "${environment}"
```

### Backend Configuration

Per-module or per-instance backend configuration:

```yaml
modules:
  - name: networking
    backend:
      type: azurerm
      resource_group_name: "rg-tfstate"
      storage_account_name: "tfstate"
      container_name: "tfstate"
      key: "networking.tfstate"
```

### Timeout Configuration

Custom timeouts for long-running operations:

```bash
# Global timeout
tfpipboy --operation apply --targets-all --timeout 90m

# Pipeline timeout
pipelines:
  deploy-all:
    settings:
      timeout: "120m"
```

---

## Best Practices

### 1. Start Small

Begin with a few modules and gradually increase complexity.

### 2. Use Dependencies Wisely

- Define explicit dependencies only when necessary
- Use Terraform data sources for loose coupling
- Avoid circular dependencies

### 3. Leverage Parallelism

- Group independent modules for parallel execution
- Increase concurrency based on resources
- Monitor system performance

### 4. Environment Separation

- Use separate workspaces for environments
- Use pipelines for environment-specific deployments
- Keep environment variables in separate files

### 5. Version Control

- Commit all configuration files
- Use `.gitignore` for sensitive values
- Tag releases for important milestones

### 6. Testing

- Always run `plan` before `apply`
- Use `--dry-run` to preview execution
- Test in non-production environments first

### 7. Security

- Never commit credentials to version control
- Use environment variables for sensitive values
- Enable MFA on cloud accounts
- Rotate credentials regularly

---

## Troubleshooting

### Common Issues

#### Module Not Found

**Error**: `module 'xyz' not found`

**Solution**: Check module name in configuration matches exactly

#### Circular Dependency

**Error**: `circular dependency detected`

**Solution**: Review `depends_on` and break the cycle

#### Authentication Failed

**Error**: `authentication missing or expired`

**Solution**: Re-authenticate with cloud provider CLI

#### Timeout

**Error**: `operation timed out`

**Solution**: Increase timeout or check for hanging processes

#### Backend Error

**Error**: `failed to configure backend`

**Solution**: Verify backend configuration and credentials

### Debug Mode

Enable verbose logging:

```bash
tfpipboy --operation plan --targets-all --verbose
```

Enable trace logging (very detailed):

```bash
tfpipboy --operation plan --targets-all --trace
```

### Log Files

Check logs in `.tfpipboy/logs/`:
- `orchestrator.log` - Main orchestration logs
- Individual job logs in workspace directories

---

## Next Steps

- [Configuration Reference](configuration.md) - Detailed configuration options
- [Examples](../examples/) - Real-world examples
- [Architecture](architecture.md) - System architecture *(to be created)*
- [Development Guide](development.md) - Contributing *(to be created)*

---

**Need Help?** See [GitHub Issues](https://github.com/StanleyXie/tf-pipboy/issues) or [Discussions](https://github.com/StanleyXie/tf-pipboy/discussions)
