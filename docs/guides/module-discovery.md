# Module Discovery Guide

This guide explains how to use tfpipboy's module discovery feature to automatically scan and generate configuration for Terraform modules.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Module Type Classification](#module-type-classification)
- [Generated Configuration](#generated-configuration)
- [Next Steps](#next-steps)
- [Advanced Usage](#advanced-usage)

---

## Overview

The module discovery feature automatically scans a directory tree for Terraform modules and generates a tfpipboy configuration template. This is particularly useful when:

- Initializing tfpipboy for an existing Terraform project
- Auditing your infrastructure codebase
- Understanding module dependencies and structure
- Creating a baseline configuration for orchestration

### What Gets Discovered

The discovery process identifies:

- **Terraform Modules**: All directories containing `.tf` files
- **Module Types**: Classification as root modules or source modules
- **Module Metadata**: Variables, outputs, resources, and dependencies
- **Tfvars Files**: All `.tfvars` and `.tfvars.json` files
- **Module Dependencies**: References to other modules

---

## Quick Start

### Basic Discovery

Discover all modules in the current directory:

```bash
tfpipboy --discover .
```

### Save Configuration to File

Generate configuration and save it to a file:

```bash
tfpipboy --discover ./terraform --discover-output .tfpipboy/discovered-modules.yaml
```

### Scan a Specific Directory

Discover modules in a specific path:

```bash
tfpipboy --discover /path/to/terraform/modules
```

---

## How It Works

### Discovery Process

1. **Recursive Scanning**: The tool recursively scans the specified path for directories containing Terraform files (`.tf`)

2. **Module Detection**: Each directory with `.tf` files is identified as a Terraform module

3. **Metadata Extraction**: For each module, the tool parses `.tf` files to extract:
   - Variable declarations (name, type, description, default, required)
   - Output declarations (name, description, sensitive)
   - Resource declarations (type, name)
   - Module dependencies (module block sources)
   - Provider configurations

4. **Type Classification**: Each module is classified as either a root module or source module

5. **Tfvars Discovery**: All `.tfvars` and `.tfvars.json` files are located and cataloged

6. **Configuration Generation**: A YAML configuration template is generated with all discovered modules

### Directories Skipped

The discovery process automatically skips:

- Hidden directories (starting with `.`)
- `.terraform` directories
- `node_modules` directories
- `vendor` directories
- `.git` directories

---

## Module Type Classification

Modules are classified into two types:

### Root Modules

**Definition**: Modules that can be deployed directly and independently.

**Characteristics**:
- Located at the project root or environment directories
- Have a `.git` directory (repository root)
- Standalone deployable units

**Example**:
```
my-infrastructure/
├── .git/
├── main.tf           # Root module
├── variables.tf
└── terraform.tfvars
```

### Source Modules

**Definition**: Reusable modules referenced by other modules.

**Characteristics**:
- Located in `modules/`, `terraform-modules/`, or similar directories
- Nested within other Terraform modules
- Designed to be referenced with `module` blocks

**Example**:
```
my-infrastructure/
├── main.tf
└── modules/
    ├── networking/   # Source module
    │   └── main.tf
    └── compute/      # Source module
        └── main.tf
```

---

## Generated Configuration

### Configuration Structure

The discovery process generates a YAML configuration with the following structure:

```yaml
version: "1.0"

modules:
  my-module:
    path: terraform/my-module
    description: "Auto-discovered root module"
    instances: {}
    _comment: "Root module - can be deployed directly. Add instances below to enable execution."

  shared-network:
    path: terraform/modules/shared-network
    description: "Auto-discovered source module"
    instances: {}
    _comment: "Source module - referenced by other modules. Add instances only if deploying independently."

_tfvars_reference:
  - terraform/dev.tfvars
  - terraform/prod.tfvars

_tfvars_note: "Tfvars files found. These can be used to define instance variables in the var-config section."
```

### Understanding the Output

**Module Entries**: Each discovered module becomes an entry in the `modules` section with:
- `path`: Relative path to the module
- `description`: Auto-generated description based on module type
- `instances`: Empty object for you to populate
- `_comment`: Guidance on how to use the module

**Tfvars Reference**: Lists all discovered `.tfvars` files that can be used for instance configuration

**Instances Required**: The configuration is a template - you must add instance definitions before execution

---

## Next Steps

### 1. Review the Generated Configuration

Examine the discovered modules and verify they match your expectations:

```bash
tfpipboy --discover ./terraform --discover-output discovered.yaml
cat discovered.yaml
```

### 2. Define Instances

For each module you want to deploy, add instance definitions:

```yaml
modules:
  networking:
    path: terraform/modules/networking
    instances:
      prod-network:
        description: "Production network infrastructure"
        environment: production
        region: us-east-1
        variables:
          vpc_cidr: "10.0.0.0/16"

      dev-network:
        description: "Development network infrastructure"
        environment: development
        region: us-west-2
        variables:
          vpc_cidr: "10.1.0.0/16"
```

### 3. Configure Backend

Add backend configuration for state management:

```yaml
modules:
  networking:
    path: terraform/modules/networking
    backend:
      type: azurerm
      resource_group_name: tfstate-rg
      storage_account_name: tfstatestorage
      container_name: tfstate
      key: "networking-${instance}.tfstate"  # Uses instance name
```

### 4. Define Dependencies

Add dependencies between module instances:

```yaml
modules:
  networking:
    path: terraform/modules/networking
    instances:
      prod-network:
        # ...

  compute:
    path: terraform/modules/compute
    instances:
      prod-app-servers:
        depends_on:
          - prod-network  # Wait for network to be ready
        # ...
```

### 5. Test the Configuration

Validate your configuration:

```bash
tfpipboy --config .tfpipboy --list-modules
```

Run a plan to verify:

```bash
tfpipboy --config .tfpipboy --targets prod-network --operation plan
```

---

## Advanced Usage

### Filtering Discovery Results

You can manually edit the generated configuration to:

- Remove modules you don't want to orchestrate
- Reorganize module groupings
- Add additional metadata

### Combining with Existing Configuration

If you already have a tfpipboy configuration:

1. Discover modules to a separate file:
   ```bash
   tfpipboy --discover ./new-modules --discover-output new-modules.yaml
   ```

2. Manually merge relevant sections into your existing configuration

### Automated Configuration Management

For large projects, you can:

1. Run discovery periodically to detect new modules
2. Use the output as a baseline for configuration updates
3. Diff against existing configuration to identify changes

### Example: Complete Workflow

```bash
# 1. Discover modules
tfpipboy --discover ./terraform --discover-output .tfpipboy/base-config.yaml

# 2. Edit the configuration to add instances
nano .tfpipboy/base-config.yaml

# 3. Rename to modules.yaml (or your preferred name)
mv .tfpipboy/base-config.yaml .tfpipboy/modules.yaml

# 4. Validate
tfpipboy --config .tfpipboy --list-modules

# 5. Run a dry-run
tfpipboy --config .tfpipboy --targets-all --operation plan --dry-run

# 6. Execute
tfpipboy --config .tfpipboy --targets-all --operation plan
```

---

## Discovery Output Example

Here's an example of what the discovery output looks like:

```
================================================================================
  TERRAFORM MODULE DISCOVERY
================================================================================
Scanning path: ./terraform
================================================================================

Discovery Summary:
  Total modules found:     8
  Root modules:            3
  Source modules:          5
  Tfvars files found:      4

Discovered Modules:
  [root] infrastructure
      Variables: 12 (5 required)
      Outputs: 6
  [source] modules/networking
      Variables: 8 (3 required)
      Outputs: 4
      Module dependencies: 0
  [source] modules/compute
      Variables: 15 (7 required)
      Outputs: 3
      Module dependencies: 1
  [root] environments/production
      Variables: 5 (2 required)
      Outputs: 2
      Module dependencies: 2

Discovered Tfvars Files:
  infrastructure/production.tfvars
  infrastructure/staging.tfvars
  environments/production/terraform.tfvars
  environments/staging/terraform.tfvars

================================================================================
  GENERATED CONFIGURATION
================================================================================

version: "1.0"
modules:
  infrastructure:
    path: infrastructure
    description: Auto-discovered root module
    instances: {}
...

================================================================================
NOTE: The generated configuration template requires you to define instances
for each module before execution. Edit the configuration file and add
instance definitions under the 'instances' section of each module.
================================================================================
```

---

## Best Practices

### 1. Use Discovery as a Starting Point

The discovery feature is designed to bootstrap your configuration, not replace manual configuration management. Always review and customize the generated configuration.

### 2. Version Control Generated Configurations

Commit the generated configuration to version control as a baseline:

```bash
tfpipboy --discover . --discover-output .tfpipboy/discovered-$(date +%Y%m%d).yaml
git add .tfpipboy/discovered-*.yaml
git commit -m "Add module discovery baseline"
```

### 3. Regular Discovery Audits

Run discovery periodically to identify:
- New modules added to the codebase
- Modules that were removed
- Changes in module dependencies

### 4. Combine with Manual Configuration

Use discovery for initial setup, then maintain configuration manually:

```yaml
# Base configuration from discovery
modules:
  networking:
    path: terraform/modules/networking

# Add your custom configuration
modules:
  networking:
    category: infrastructure
    owner: platform-team
    instances:
      # Your instances here
```

### 5. Document Custom Changes

Add comments to document why you deviated from the discovered configuration:

```yaml
modules:
  legacy-module:
    # Note: Excluded from orchestration - managed manually
    # path: terraform/legacy
    enabled: false
```

---

## Troubleshooting

### No Modules Found

**Problem**: Discovery reports 0 modules found

**Solutions**:
- Verify the path contains `.tf` files
- Check that you're not running discovery on a `.terraform` directory
- Ensure you have read permissions on the directory

### Incorrect Module Type

**Problem**: Module classified as wrong type (root vs source)

**Understanding**:
- Modules in `modules/` directories are classified as source modules
- Modules with `.git` directory are classified as root modules
- Standalone modules (no parent with `.tf` files) are root modules

**Solution**: Module type is informational only and doesn't affect functionality

### Missing Metadata

**Problem**: Variables or outputs not extracted

**Cause**: The parser uses simple regex-based extraction, not a full HCL parser

**Workaround**: Manually add missing metadata to the configuration

### Performance on Large Codebases

**Problem**: Discovery is slow on very large repositories

**Solutions**:
- Run discovery on specific subdirectories
- Exclude unnecessary paths by running on targeted directories
- Use the output caching strategy (discover once, reuse)

---

## See Also

- [User Guide](../user-guide.md) - Complete tfpipboy usage guide
- [Configuration Reference](../configuration.md) - Detailed configuration options
- [Getting Started](../getting-started.md) - Quick start guide
