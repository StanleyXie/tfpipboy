# Variable Configuration Examples

This document demonstrates all the ways to configure variables in tfpipboy's `var-config` section.

## Overview

The `var-config` section supports flexible variable configuration with both backward compatibility and new features for multiple files and variables.

### Supported Fields

- `file`: Single variable file path (backward compatible)
- `files`: Array of variable file paths for multiple `-var-file` arguments
- `json`: Inline variables as key-value pairs (backward compatible)
- `vars`: Inline variables as key-value pairs (alias for json)

## Examples

### Example 1: Single Variable File (Backward Compatible)

```yaml
modules:
  seed:
    path: "./modules/seed"
    var-config:
      file: "variables/seed.tfvars"
```

**Terraform command generated:**
```bash
terraform plan -var-file=/path/to/workspace/terraform.tfvars
```

### Example 2: Multiple Variable Files

```yaml
modules:
  seed:
    path: "./modules/seed"
    var-config:
      files:
        - "variables/common.tfvars"
        - "variables/seed-specific.tfvars"
        - "variables/environment-prod.tfvars"
```

**Terraform command generated:**
```bash
terraform plan \
  -var-file=/path/to/workspace/terraform.tfvars \
  -var-file=/path/to/workspace/vars-1-seed-specific.tfvars \
  -var-file=/path/to/workspace/vars-2-environment-prod.tfvars
```

### Example 3: Single File + Inline Variables

```yaml
modules:
  seed:
    path: "./modules/seed"
    var-config:
      file: "variables/seed.tfvars"
      json:
        environment: "production"
        location: "germanywestcentral"
        subscription_id: "xxxxxxxx-xxxx-4xxx-xxxx-000000000005"
```

**Terraform command generated:**
```bash
terraform plan \
  -var-file=/path/to/workspace/terraform.tfvars \
  -var=environment=production \
  -var=location=germanywestcentral \
  -var=subscription_id=xxxxxxxx-xxxx-4xxx-xxxx-000000000005
```

### Example 4: Multiple Files + Inline Variables

```yaml
modules:
  seed:
    path: "./modules/seed"
    var-config:
      files:
        - "variables/common.tfvars"
        - "variables/seed.tfvars"
      vars:
        environment: "production"
        location: "germanywestcentral"
        override_setting: "true"
```

**Terraform command generated:**
```bash
terraform plan \
  -var-file=/path/to/workspace/terraform.tfvars \
  -var-file=/path/to/workspace/vars-1-seed.tfvars \
  -var=environment=production \
  -var=location=germanywestcentral \
  -var=override_setting=true
```

### Example 5: Only Inline Variables (No Files)

```yaml
modules:
  seed:
    path: "./modules/seed"
    var-config:
      vars:
        environment: "production"
        location: "germanywestcentral"
        subscription_id: "xxxxxxxx-xxxx-4xxx-xxxx-000000000005"
        tags:
          project: "landing-zone"
          managed_by: "tfpipboy"
```

**Terraform command generated:**
```bash
terraform plan \
  -var=environment=production \
  -var=location=germanywestcentral \
  -var=subscription_id=xxxxxxxx-xxxx-4xxx-xxxx-000000000005 \
  -var=tags=map[managed_by:tfpipboy project:landing-zone]
```

### Example 6: Mixed - Both `file` and `files`

```yaml
modules:
  seed:
    path: "./modules/seed"
    var-config:
      file: "variables/base.tfvars"      # Will be added first
      files:
        - "variables/environment.tfvars"
        - "variables/overrides.tfvars"
      json:
        debug_mode: "true"
```

**Terraform command generated:**
```bash
terraform plan \
  -var-file=/path/to/workspace/terraform.tfvars \
  -var-file=/path/to/workspace/vars-1-environment.tfvars \
  -var-file=/path/to/workspace/vars-2-overrides.tfvars \
  -var=debug_mode=true
```

### Example 7: Instance-Level Variable Override

```yaml
modules:
  seed:
    path: "./modules/seed"
    var-config:
      file: "variables/seed-common.tfvars"
      json:
        location: "germanywestcentral"
    
    instances:
      seed-dev-gwc:
        environment: "dev"
        region: "gwc"
        var-config:
          # Override with instance-specific files
          files:
            - "variables/seed-common.tfvars"
            - "variables/seed-dev.tfvars"
          vars:
            location: "germanywestcentral"
            environment: "dev"
      
      seed-prod-sdc:
        environment: "prod"
        region: "sdc"
        var-config:
          files:
            - "variables/seed-common.tfvars"
            - "variables/seed-prod.tfvars"
          vars:
            location: "swedencentral"
            environment: "prod"
```

## Variable Precedence

Variables are applied in the following order (later values override earlier ones):

1. **Variable files** (in order specified):
   - `file` (if specified) - copied as `terraform.tfvars`
   - `files` (in order) - copied as `vars-1-*.tfvars`, `vars-2-*.tfvars`, etc.

2. **Inline variables** (applied as `-var` flags):
   - `json` values
   - `vars` values
   - Module-level `variables` field

This matches Terraform's variable precedence where:
- `-var` flags override `-var-file` values
- Later `-var-file` arguments override earlier ones

## Path Resolution

All file paths in `var-config` are resolved relative to the `.tfpipboy` directory (where `tfproject.yaml` is located):

```yaml
# Assuming .tfpipboy is at: /project/.tfpipboy/

var-config:
  file: "variables/seed.tfvars"
  # Resolves to: /project/.tfpipboy/variables/seed.tfvars
  
  files:
    - "../shared-vars/common.tfvars"
    # Resolves to: /project/shared-vars/common.tfvars
    
    - "/absolute/path/to/overrides.tfvars"
    # Used as-is: /absolute/path/to/overrides.tfvars
```

## Use Cases

### Use Case 1: Shared Variables Across Modules

```yaml
modules:
  seed:
    path: "./modules/seed"
    var-config:
      files:
        - "../shared/common.tfvars"      # Shared across all modules
        - "../shared/environment.tfvars"  # Environment-specific
        - "variables/seed.tfvars"         # Module-specific
  
  core:
    path: "./modules/core"
    var-config:
      files:
        - "../shared/common.tfvars"      # Same shared file
        - "../shared/environment.tfvars"
        - "variables/core.tfvars"         # Different module-specific
```

### Use Case 2: Environment-Specific Overrides

```yaml
modules:
  networking:
    path: "./modules/networking"
    instances:
      net-dev:
        environment: "dev"
        var-config:
          files:
            - "variables/networking-base.tfvars"
          vars:
            cidr_block: "10.0.0.0/16"
            enable_nat: "false"  # Dev doesn't need NAT
      
      net-prod:
        environment: "prod"
        var-config:
          files:
            - "variables/networking-base.tfvars"
          vars:
            cidr_block: "172.16.0.0/16"
            enable_nat: "true"   # Prod requires NAT
            high_availability: "true"
```

### Use Case 3: Multi-Region Deployments

```yaml
modules:
  regional-infra:
    path: "./modules/regional"
    instances:
      infra-gwc:
        region: "germanywestcentral"
        var-config:
          files:
            - "variables/infra-common.tfvars"
            - "variables/regions/gwc.tfvars"
      
      infra-sdc:
        region: "swedencentral"
        var-config:
          files:
            - "variables/infra-common.tfvars"
            - "variables/regions/sdc.tfvars"
```

## Migration from v0.1.0

### Old Configuration (v0.1.0)

```yaml
var-config:
  file: "variables/seed.tfvars"
  json:
    environment: "production"
```

### New Configuration (v0.2.0) - Backward Compatible

The old configuration still works! No changes needed.

### Enhanced Configuration (v0.2.0)

```yaml
var-config:
  # Now you can specify multiple files
  files:
    - "variables/base.tfvars"
    - "variables/seed.tfvars"
    - "variables/overrides.tfvars"
  
  # Both 'json' and 'vars' work the same way
  vars:
    environment: "production"
    location: "germanywestcentral"
```

## Best Practices

1. **Use multiple files for better organization:**
   - `common.tfvars` - Variables shared across all modules
   - `environment.tfvars` - Environment-specific variables
   - `module-name.tfvars` - Module-specific variables

2. **Use `vars` for runtime overrides:**
   - Dynamic values that change per instance
   - Values that override file-based defaults
   - Secrets that should be passed via environment variables (not committed)

3. **Keep files DRY:**
   - Place common variables in shared files
   - Use instance-level overrides for differences
   - Leverage variable precedence for flexibility

4. **Path conventions:**
   - Relative paths from `.tfpipboy/` directory
   - Use `../shared/` for cross-module variables
   - Use `variables/` for module-specific variables
