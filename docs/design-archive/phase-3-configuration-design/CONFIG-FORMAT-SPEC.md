# tf-pipboy Configuration Format Specification

**Version**: 1.0  
**Status**: Draft  
**Last Updated**: 2025-10-19

---

## Overview

This document defines the complete configuration format for tf-pipboy's declarative orchestration system. The configuration is split across 5 YAML files in the `.tfpipboy/` directory.

## Directory Structure

```
.tfpipboy/
├── variables.yaml        # Global variables and environment settings
├── backends.yaml         # Backend templates and configurations
├── landing-zones.yaml    # Landing zone definitions (optional)
├── modules.yaml          # Module catalog with dependencies
└── pipelines.yaml        # Deployment workflows
```

---

## 1. variables.yaml

Defines global variables, environment-specific overrides, and shared configuration.

### Schema

```yaml
version: "1.0"  # Configuration format version

# Global variables available to all modules
variables:
  # Simple key-value pairs
  project: "string"
  tenant_id: "string"
  location: "string"
  
  # Nested objects
  bootstrap:
    subscription_id: "string"
    resource_group: "string"
    storage_account: "string"
  
  # Arrays
  allowed_locations: ["location1", "location2"]

# Environment-specific overrides
environments:
  dev:
    # Override any global variable
    auto_approve: true
    
  staging:
    # Can add new variables
    additional_tags:
      environment: "staging"
  
  prod:
    # Production-specific settings
    auto_approve: false
    require_approval: true
```

### Key Features

- **Variable Types**: Strings, numbers, booleans, objects, arrays
- **Nested Objects**: Support dot notation access (e.g., `${bootstrap.subscription_id}`)
- **Environment Overrides**: Per-environment variable values
- **Interpolation**: Variables can reference other variables

### Example

```yaml
version: "1.0"

variables:
  # Project metadata
  project: "atlz-platform"
  root_id: "exp"
  root_name: "ExperimentDev"
  
  # Azure configuration
  tenant_id: "c3310862-ab75-4bc6-b8b6-26a14539537c"
  location: "germanywestcentral"
  location_slug: "gc"
  org_name: "explz"
  
  # Bootstrap backend
  bootstrap:
    subscription_id: "4b05673b-62ce-4723-99fa-c1030624561e"
    resource_group: "rg-exp-bootstrap-tfbackend"
    storage_account: "stexptfbackendbootstrap"
  
  # Billing
  billing_account_id: "bca92218-bdfe-407e-b2c0-02ef02b3b6f8:333013a0-..."
  billing_profile_id: "XU5N-S6UC-BG7-PGB"
  
  # Naming conventions
  naming:
    resource_group: "rg-${org_name}-${landing_zone}-${environment}-${location_slug}-${suffix}"

environments:
  dev:
    environment: "dev"
    auto_approve: true
  
  prod:
    environment: "prod"
    auto_approve: false
    org_name: "prodlz"  # Override for prod
```

---

## 2. backends.yaml

Defines backend templates for Terraform state storage and module-specific backend configurations.

### Schema

```yaml
version: "1.0"

# Reusable backend templates
backend_templates:
  template_name:
    type: "azurerm" | "s3" | "gcs" | "local"
    
    # Azure backend fields
    tenant_id: "string"
    subscription_id: "string"
    resource_group_name: "string"
    storage_account_name: "string"
    container_name: "string"
    use_azuread_auth: boolean
    
    # AWS S3 backend fields
    bucket: "string"
    region: "string"
    
    # Variables can be interpolated
    # Use ${variable_name} syntax

# Per-module backend configurations
modules:
  module_name:
    description: "string"
    
    # Simple module
    environments:
      env_name:
        backend:
          template: "template_name"  # Reference a template
          key: "path/to/state.tfstate"
          # Can override template values
          subscription_id: "override-value"
    
    # Module with instances
    instances:
      instance_name:
        description: "string"
        environments:
          env_name:
            backend:
              template: "template_name"
              key: "path/to/state.tfstate"
```

### Key Features

- **Templates**: DRY principle - define once, use many times
- **Variable Interpolation**: Reference variables from `variables.yaml`
- **Module Instances**: Same module deployed multiple times with different configs
- **Template Overrides**: Override any template field per module
- **Multi-Environment**: Different backends per environment

### Example

```yaml
version: "1.0"

backend_templates:
  # Bootstrap backend for platform modules
  bootstrap:
    type: "azurerm"
    tenant_id: "${tenant_id}"
    subscription_id: "${bootstrap.subscription_id}"
    resource_group_name: "${bootstrap.resource_group}"
    storage_account_name: "${bootstrap.storage_account}"
    container_name: "platform"
    use_azuread_auth: true
  
  # Landing zone backend for deployed resources
  landing_zone:
    type: "azurerm"
    tenant_id: "${tenant_id}"
    subscription_id: "${landing_zones.${landing_zone}.subscriptions.${environment}}"
    resource_group_name: "${landing_zones.${landing_zone}.backend.resource_group}"
    storage_account_name: "${landing_zones.${landing_zone}.backend.storage_account}"
    container_name: "tfstate"
    use_azuread_auth: true

modules:
  # Simple module
  core:
    description: "Core platform module"
    environments:
      exp:
        backend:
          template: "bootstrap"
          key: "exp-core.tfstate"
  
  # Module with instances
  vending:
    description: "Subscription vending"
    instances:
      connectivity:
        description: "Vend connectivity subscription"
        environments:
          exp:
            backend:
              template: "bootstrap"
              key: "exp-vending-conn.tfstate"
      
      management:
        description: "Vend management subscription"
        environments:
          exp:
            backend:
              template: "bootstrap"
              key: "exp-vending-mgmt.tfstate"
```

---

## 3. landing-zones.yaml (Optional)

Defines landing zone metadata and configuration. This is optional and specific to Azure Landing Zone pattern.

### Schema

```yaml
version: "1.0"

landing_zones:
  landing_zone_name:
    description: "string"
    org_name: "string"
    abbreviation: "string"
    archetype: "string"
    workload_type: "string"
    
    # Network configuration
    network:
      enable_resources: boolean
      enable_peering: boolean
      address_spaces:
        env_name: ["cidr1", "cidr2"]
      hub_vnet_id:
        env_name: "vnet_resource_id"
    
    # Subscription mappings
    subscriptions:
      env_name: "subscription_id"
    
    # Backend configuration
    backend:
      resource_group: "string (can use ${environment})"
      storage_account: "string (can use ${environment})"
    
    # Tags
    tags:
      key: "value"
    
    # Environments
    environments:
      env_name:
        # Environment-specific config
```

### Key Features

- **Landing Zone Abstraction**: First-class representation of landing zones
- **Network Configuration**: Per-environment address spaces
- **Subscription Mapping**: Map landing zones to Azure subscriptions
- **Backend Abstraction**: Landing zone backend configuration
- **Variable Support**: Use variables in any field

### Example

```yaml
version: "1.0"

landing_zones:
  connectivity:
    description: "Hub networking infrastructure"
    org_name: "explz"
    abbreviation: "conn"
    archetype: "connectivity"
    workload_type: "Production"
    
    network:
      enable_resources: true
      enable_peering: false
      address_spaces:
        dev: ["10.200.208.0/23"]
        exp: ["10.200.208.0/23"]
        prod: ["10.100.0.0/16"]
      hub_vnet_id:
        dev: "/subscriptions/.../vnet-explz-conn-dev-gc-hub"
    
    subscriptions:
      dev: "e786a45d-030c-4dfb-a827-fd18444ed496"
      exp: "e786a45d-030c-4dfb-a827-fd18444ed496"
      prod: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    
    backend:
      resource_group: "rg-explz-conn-${environment}-gc-base"
      storage_account: "stexplzconn${environment}gcbase"
    
    tags:
      BEE360_ID: "SE1070"
      archetype: "connectivity"
```

---

## 4. modules.yaml

Defines the complete module catalog with dependencies, variables, and outputs.

### Schema

```yaml
version: "1.0"

modules:
  module_name:
    description: "string"
    category: "string"
    owner: "string"
    path: "string"  # Relative path to module
    depends_on: ["module1", "module2.instance"]  # Dependencies
    
    # Backend reference (details in backends.yaml)
    backend:
      template: "template_name"
      key: "state-file.tfstate"
      # Optional overrides
    
    # Variables passed to Terraform
    variables:
      var_name: "value"  # Can use ${variable} interpolation
      nested:
        key: "value"
    
    # Expected outputs from this module
    outputs:
      - output_name1
      - output_name2
    
    # Module instances (optional)
    instances:
      instance_name:
        description: "string"
        depends_on: ["other_module"]
        backend:
          template: "template_name"
          key: "state-file.tfstate"
        variables:
          var_name: "value"
        outputs:
          - output_name

# Module groups for batch operations
groups:
  group_name:
    - module1
    - module2.instance
    - module3
```

### Key Features

- **Dependency Management**: Explicit `depends_on` relationships
- **Module Instances**: Deploy same module multiple times
- **Variable Interpolation**: Reference variables and outputs
- **Outputs**: Track expected outputs for validation
- **Groups**: Logical grouping for batch operations
- **Path Mapping**: Filesystem path to Terraform code

### Example

```yaml
version: "1.0"

modules:
  core:
    description: "Core management groups and policies"
    category: "bootstrap"
    owner: "platform-team"
    path: "./plz/core"
    depends_on: []
    
    backend:
      template: "bootstrap"
      key: "${root_id}-core.tfstate"
    
    variables:
      root_id: "${root_id}"
      location: "${location}"
      billing_account_id: "${billing_account_id}"
    
    outputs:
      - root_management_group_id
      - management_groups
  
  vending:
    description: "Subscription vending"
    category: "bootstrap"
    path: "./plz/bootstrap/vending"
    
    instances:
      connectivity:
        description: "Vend connectivity subscription"
        depends_on: [core]
        
        backend:
          template: "bootstrap"
          key: "${root_id}-vending-conn.tfstate"
        
        variables:
          landing_zone_name: "connectivity"
          environment: "${environment}"
        
        outputs:
          - subscription_id
          - subscription_name

groups:
  bootstrap:
    - core
    - vending.connectivity
    - vending.management
```

---

## 5. pipelines.yaml

Defines orchestration workflows with stages, hooks, and execution settings.

### Schema

```yaml
version: "1.0"

pipelines:
  pipeline_name:
    description: "string"
    default_operation: "plan" | "apply" | "destroy" | "validate"
    
    # Pipeline-level settings
    settings:
      parallel_limit: number  # Max parallel modules
      timeout: "duration"  # e.g., "30m", "2h"
      retry_failed: number  # Retry count
      auto_approve: boolean
    
    # Before pipeline starts
    before:
      - name: "Hook name"
        command: "shell command"
    
    # Execution stages
    stages:
      - name: "Stage name"
        
        # Option 1: List specific modules
        modules:
          - module_name
          - module_name.instance
          - module_name:  # With overrides
              variables:
                var: "value"
              operation: "apply"
        
        # Option 2: Use module groups
        groups: [group_name]
        
        # Parallel execution
        parallel: boolean  # true = parallel, false = sequential
        
        # Stage settings
        settings:
          timeout: "duration"
          retry_failed: number
        
        # After stage completes
        after:
          - name: "Hook name"
            command: "shell command"
    
    # After pipeline completes
    after:
      - name: "Hook name"
        command: "shell command"
    
    # On pipeline failure
    on_failure:
      - name: "Hook name"
        command: "shell command"
    
    # Confirmation requirement
    require_confirmation: boolean
    confirmation_message: "string"
    
    # Environment-specific overrides
    environments:
      env_name:
        settings:
          auto_approve: boolean
        require_confirmation: boolean
        before: [...]
        after: [...]
```

### Key Features

- **Multi-Stage Execution**: Sequential stages with dependencies
- **Parallel Execution**: Run independent modules concurrently
- **Lifecycle Hooks**: Before/after pipeline and stages
- **Custom Operations**: Override operation per module
- **Confirmation Gates**: User confirmation for critical operations
- **Environment Overrides**: Different behavior per environment
- **Timeout Management**: Per-stage and per-pipeline timeouts
- **Retry Logic**: Automatic retry on transient failures

### Example

```yaml
version: "1.0"

pipelines:
  deploy-bootstrap:
    description: "Deploy bootstrap infrastructure"
    default_operation: apply
    
    settings:
      parallel_limit: 2
      timeout: "60m"
    
    before:
      - name: "Validate Azure auth"
        command: "az account show"
    
    stages:
      - name: "Core Platform"
        modules:
          - core
        
        settings:
          timeout: "30m"
        
        after:
          - name: "Verify management groups"
            command: "az account management-group list"
      
      - name: "Subscription Vending"
        modules:
          - vending.connectivity
          - vending.management
        parallel: true
        
        settings:
          timeout: "30m"
          retry_failed: 2
    
    after:
      - name: "Generate report"
        command: "scripts/report-bootstrap.sh"
    
    environments:
      prod:
        require_confirmation: true
        confirmation_message: "Deploy bootstrap to PROD? Type 'yes':"
        settings:
          auto_approve: false
```

---

## Variable Interpolation

### Syntax

```
${variable_name}
${nested.variable.path}
${landing_zones.connectivity.subscriptions.dev}
${environment}  # Special variable - current environment
${landing_zone}  # Special variable - current landing zone
```

### Resolution Order

1. **Special variables**: `${environment}`, `${landing_zone}`, `${root_id}`
2. **Global variables**: From `variables.yaml`
3. **Landing zones**: From `landing-zones.yaml`
4. **Environment overrides**: From `environments` section

### Examples

```yaml
# In variables.yaml
variables:
  org_name: "myorg"
  location: "westus"

# In modules.yaml
modules:
  app:
    variables:
      resource_group: "rg-${org_name}-${environment}-${location}"
      # Resolves to: rg-myorg-dev-westus (when environment=dev)
```

---

## Module Instance Naming

### Format

- **Simple module**: `module_name`
- **Module instance**: `module_name.instance_name`

### Dependencies

```yaml
modules:
  app:
    depends_on: [database.primary, cache.redis]
```

### Pipeline References

```yaml
pipelines:
  deploy:
    stages:
      - name: "Deploy"
        modules:
          - database.primary
          - database.replica
          - app
```

---

## Environment Handling

### Current Environment

The current environment is passed via CLI:

```bash
tfpipboy run deploy-all --env dev
tfpipboy run deploy-all --env prod
```

### Environment Variables

Special variable `${environment}` is automatically set:

```yaml
backend:
  key: "${environment}-app.tfstate"  # dev-app.tfstate
```

### Environment Overrides

Override any configuration per environment:

```yaml
# In variables.yaml
variables:
  instance_type: "small"

environments:
  prod:
    instance_type: "large"  # Override for prod

# In pipelines.yaml
pipelines:
  deploy:
    settings:
      auto_approve: true
    
    environments:
      prod:
        settings:
          auto_approve: false  # Override for prod
```

---

## Configuration Validation

### Required Fields

**variables.yaml**:
- `version` (required)
- `variables` (required, can be empty)

**backends.yaml**:
- `version` (required)
- `backend_templates` (required if modules reference templates)

**modules.yaml**:
- `version` (required)
- `modules` (required)
- Each module must have: `path`

**pipelines.yaml**:
- `version` (required)
- `pipelines` (required)
- Each pipeline must have: `stages`

### Validation Rules

1. **Circular Dependencies**: Modules cannot have circular `depends_on`
2. **Module References**: Pipeline modules must exist in `modules.yaml`
3. **Template References**: Backend templates must exist in `backend_templates`
4. **Variable Interpolation**: All `${variable}` must resolve
5. **File Paths**: Module `path` must exist on filesystem

---

## Best Practices

### 1. Use Backend Templates

❌ **Bad** - Duplicate configuration:
```yaml
modules:
  app1:
    backend:
      type: "azurerm"
      subscription_id: "xxx"
      resource_group_name: "rg-backend"
      storage_account_name: "stbackend"
      container_name: "tfstate"
      key: "app1.tfstate"
  app2:
    backend:
      type: "azurerm"
      subscription_id: "xxx"  # Duplicated
      resource_group_name: "rg-backend"  # Duplicated
      storage_account_name: "stbackend"  # Duplicated
      container_name: "tfstate"  # Duplicated
      key: "app2.tfstate"
```

✅ **Good** - Use templates:
```yaml
backend_templates:
  default:
    type: "azurerm"
    subscription_id: "${backend_subscription}"
    resource_group_name: "rg-backend"
    storage_account_name: "stbackend"
    container_name: "tfstate"

modules:
  app1:
    backend:
      template: "default"
      key: "app1.tfstate"
  app2:
    backend:
      template: "default"
      key: "app2.tfstate"
```

### 2. Use Variables

❌ **Bad** - Hardcode values:
```yaml
modules:
  app:
    variables:
      location: "westus"
      resource_group: "rg-myapp-dev-westus"
```

✅ **Good** - Use variables:
```yaml
# variables.yaml
variables:
  location: "westus"
  org: "myapp"

# modules.yaml
modules:
  app:
    variables:
      location: "${location}"
      resource_group: "rg-${org}-${environment}-${location}"
```

### 3. Explicit Dependencies

❌ **Bad** - Implicit dependencies:
```yaml
modules:
  app:
    depends_on: []  # Missing database dependency
```

✅ **Good** - Explicit dependencies:
```yaml
modules:
  app:
    depends_on: [database, cache]
```

### 4. Use Module Groups

❌ **Bad** - List all modules:
```yaml
pipelines:
  deploy:
    stages:
      - modules:
          - vpc
          - subnets
          - security-groups
          - nat-gateway
          # ... 20 more modules
```

✅ **Good** - Use groups:
```yaml
groups:
  networking:
    - vpc
    - subnets
    - security-groups
    - nat-gateway

pipelines:
  deploy:
    stages:
      - groups: [networking]
```

---

## Migration Path

### From Manual Terraform

1. Start with single environment
2. Create `modules.yaml` with existing modules
3. Add `variables.yaml` for common values
4. Create simple `pipelines.yaml`
5. Test thoroughly
6. Add more environments
7. Extract backend templates
8. Add lifecycle hooks

### From Other Tools

1. Map existing modules to `modules.yaml`
2. Convert variables to `variables.yaml`
3. Create equivalent pipelines
4. Test side-by-side
5. Migrate gradually

---

## Complete Example

See `examples/azure-landing-zone/template/` for a complete, production-ready example.

---

## Questions for Review

1. **Backend Templates**: Is the template syntax clear?
2. **Module Instances**: Is `module.instance` naming intuitive?
3. **Variable Interpolation**: Should we support more syntax (e.g., `${var.name}` vs `${name}`)?
4. **Pipeline Hooks**: Are before/after/on_failure sufficient?
5. **Environment Handling**: Is the override mechanism clear?
6. **Landing Zones**: Should this be optional or required?
7. **Validation**: What additional validation rules needed?
8. **File Split**: Is 5 files the right split, or combine some?

---

## Next Steps

1. Review and approve this specification
2. Create JSON Schema for validation
3. Implement configuration parser
4. Add validation logic
5. Create example configurations
6. Write tests
