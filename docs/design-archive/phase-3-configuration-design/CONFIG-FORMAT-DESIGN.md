# Declarative Configuration Format Design
**Document No.**: CFG-001  
**Date**: 2025-10-19  
**Version**: 1.0

## Executive Summary

This document defines the declarative configuration format for tfpipboy's pipeline orchestration system. The design follows the **DRY principle** (Don't Repeat Yourself) with a **three-layer hierarchy**:

1. **Variables Layer** (`variables.yaml`) - Shared variables and defaults
2. **Modules Layer** (`modules.yaml`) - Module definitions and configurations
3. **Pipelines Layer** (`pipelines.yaml`) - Pipeline workflows and operations

This separation allows maximum flexibility while maintaining simplicity and avoiding duplication.

---

## Design Principles

### 1. Separation of Concerns
- **Variables**: Reusable values (DRY)
- **Modules**: Infrastructure component definitions (WHAT)
- **Pipelines**: Execution workflows (HOW + WHEN)

### 2. Flexibility First
- Per-pipeline operation override (plan/apply/destroy)
- Per-module configuration override
- Environment-specific overrides
- Conditional execution support

### 3. Progressive Complexity
- Simple cases require minimal configuration
- Complex cases are possible without breaking simplicity
- Sensible defaults reduce boilerplate

### 4. Composability
- Modules can be composed into pipelines
- Pipelines can reference other pipelines
- Variables can be composed and overridden

---

## Configuration Architecture

### File Structure

```
project-root/
├── .tfpipboy/
│   ├── config.yaml              # Main orchestrator config (optional)
│   ├── variables.yaml           # Shared variables
│   ├── modules.yaml             # Module catalog
│   └── pipelines.yaml           # Pipeline definitions
│
├── modules/                     # Terraform modules
│   ├── vpc/
│   ├── eks/
│   └── rds/
│
└── environments/                # Environment-specific configs
    ├── dev/
    │   ├── variables.yaml       # Dev-specific variables
    │   └── backend.tfvars       # Dev backend config
    └── prod/
        ├── variables.yaml       # Prod-specific variables
        └── backend.tfvars       # Prod backend config
```

### Configuration Loading Priority (Highest to Lowest)

```
1. CLI flags                     (--var key=value)
2. Environment variables         (TFPIPBOY_VAR_key)
3. Pipeline-level overrides      (in pipelines.yaml)
4. Environment-specific configs  (environments/{env}/variables.yaml)
5. Module defaults               (in modules.yaml)
6. Global variables              (.tfpipboy/variables.yaml)
7. Built-in defaults
```

---

## Layer 1: Variables Configuration

**Purpose**: Define reusable variables to avoid duplication

**File**: `.tfpipboy/variables.yaml`

### Basic Structure

```yaml
# Global variables available to all modules and pipelines
variables:
  # Project-level variables
  project_name: "my-infrastructure"
  organization: "acme-corp"
  
  # Cloud provider settings
  azure:
    subscription_id: "12345678-1234-1234-1234-123456789012"
    resource_group_prefix: "rg-${project_name}"
    location: "eastus"
  
  aws:
    region: "us-east-1"
    account_id: "123456789012"
  
  # Naming conventions
  naming:
    prefix: "${organization}-${environment}"
    separator: "-"
  
  # Common tags
  common_tags:
    project: "${project_name}"
    managed_by: "tfpipboy"
    terraform: "true"

# Environment-specific variable overrides
environments:
  dev:
    environment: "dev"
    auto_approve: true
    azure:
      location: "eastus"
    common_tags:
      environment: "dev"
      cost_center: "engineering"
  
  staging:
    environment: "staging"
    auto_approve: true
    azure:
      location: "eastus2"
    common_tags:
      environment: "staging"
      cost_center: "engineering"
  
  prod:
    environment: "prod"
    auto_approve: false
    azure:
      location: "eastus"
      redundancy: true
    common_tags:
      environment: "prod"
      cost_center: "production"
      sla: "critical"

# Variable templates (for complex interpolation)
templates:
  resource_name: "${naming.prefix}${naming.separator}${resource_type}"
  backend_key: "${module_name}/${environment}.tfstate"
```

### Variable Interpolation

Variables support string interpolation using `${variable.path}` syntax:

```yaml
variables:
  project: "myapp"
  environment: "dev"
  
  # Simple interpolation
  cluster_name: "${project}-${environment}-eks"
  
  # Nested interpolation
  azure:
    resource_group: "rg-${project}-${environment}"
    storage_account: "st${project}${environment}"  # No hyphens for Azure storage
  
  # Computed values
  workspace_name: "${environment}"
  backend_key: "terraform/${environment}/${module_name}.tfstate"
```

---

## Layer 2: Modules Configuration

**Purpose**: Define module catalog with metadata, paths, dependencies, and defaults

**File**: `.tfpipboy/modules.yaml`

### Basic Structure

```yaml
# Module catalog version
version: "1.0"

# Module definitions
modules:
  # Module name (unique identifier)
  vpc:
    # Metadata
    description: "Virtual Private Cloud infrastructure"
    category: "networking"
    owner: "platform-team"
    
    # Module location
    path: "./modules/vpc"
    
    # Dependencies (execution order)
    depends_on: []
    
    # Terraform configuration
    terraform:
      version: "~> 1.9.0"
      required_providers:
        azurerm: "~> 3.0"
    
    # Default variables for this module
    variables:
      cidr_block: "10.0.0.0/16"
      enable_nat_gateway: true
      enable_vpn_gateway: false
    
    # Backend configuration (can reference variables)
    backend:
      type: "azurerm"
      resource_group_name: "${azure.resource_group_prefix}-tfstate"
      storage_account_name: "${organization}tfstate"
      container_name: "tfstate"
      key: "${templates.backend_key}"  # References template from variables.yaml
    
    # Outputs that other modules might need
    outputs:
      - vpc_id
      - subnet_ids
      - route_table_ids
  
  eks:
    description: "Kubernetes cluster (EKS)"
    category: "compute"
    owner: "platform-team"
    path: "./modules/eks"
    
    # This module depends on VPC
    depends_on:
      - vpc
    
    # Can reference outputs from dependencies
    variables:
      vpc_id: "${modules.vpc.outputs.vpc_id}"
      subnet_ids: "${modules.vpc.outputs.subnet_ids}"
      cluster_version: "1.28"
      node_instance_type: "t3.medium"
      desired_capacity: 3
    
    backend:
      type: "azurerm"
      resource_group_name: "${azure.resource_group_prefix}-tfstate"
      storage_account_name: "${organization}tfstate"
      container_name: "tfstate"
      key: "eks/${environment}.tfstate"
    
    outputs:
      - cluster_endpoint
      - cluster_ca_certificate
      - cluster_name
  
  rds:
    description: "PostgreSQL database (RDS)"
    category: "database"
    owner: "data-team"
    path: "./modules/rds"
    
    depends_on:
      - vpc
    
    variables:
      vpc_id: "${modules.vpc.outputs.vpc_id}"
      subnet_ids: "${modules.vpc.outputs.subnet_ids}"
      engine: "postgres"
      engine_version: "15.4"
      instance_class: "db.t3.medium"
      allocated_storage: 100
    
    backend:
      type: "azurerm"
      resource_group_name: "${azure.resource_group_prefix}-tfstate"
      storage_account_name: "${organization}tfstate"
      container_name: "tfstate"
      key: "rds/${environment}.tfstate"
    
    # Sensitive outputs (won't be displayed in TUI)
    sensitive_outputs:
      - db_password
    
    outputs:
      - db_endpoint
      - db_port
  
  app:
    description: "Application deployment"
    category: "application"
    owner: "app-team"
    path: "./modules/app"
    
    # Multiple dependencies
    depends_on:
      - eks
      - rds
    
    variables:
      cluster_endpoint: "${modules.eks.outputs.cluster_endpoint}"
      db_endpoint: "${modules.rds.outputs.db_endpoint}"
      app_version: "1.0.0"
      replicas: 3
    
    backend:
      type: "azurerm"
      resource_group_name: "${azure.resource_group_prefix}-tfstate"
      storage_account_name: "${organization}tfstate"
      container_name: "tfstate"
      key: "app/${environment}.tfstate"

# Module groups (for convenience)
groups:
  networking:
    - vpc
  
  data_layer:
    - eks
    - rds
  
  application:
    - app
  
  infrastructure:
    - vpc
    - eks
    - rds
  
  all:
    - vpc
    - eks
    - rds
    - app
```

### Module Configuration Features

#### 1. Conditional Dependencies

```yaml
modules:
  monitoring:
    path: "./modules/monitoring"
    depends_on:
      - vpc
      # Conditional dependency based on variable
      - eks:
          when: "${monitoring.enable_kubernetes}"
      - rds:
          when: "${monitoring.enable_database}"
```

#### 2. Environment-Specific Overrides

```yaml
modules:
  rds:
    path: "./modules/rds"
    variables:
      instance_class: "db.t3.medium"
    
    # Override for specific environments
    environments:
      prod:
        variables:
          instance_class: "db.r6g.xlarge"
          multi_az: true
          backup_retention_period: 30
      
      dev:
        variables:
          instance_class: "db.t3.micro"
          multi_az: false
          backup_retention_period: 1
```

#### 3. Module Variants

```yaml
modules:
  # Base module definition
  eks:
    path: "./modules/eks"
    variables:
      cluster_version: "1.28"
  
  # Variant for small clusters
  eks-small:
    extends: eks
    variables:
      node_instance_type: "t3.small"
      desired_capacity: 2
  
  # Variant for large clusters
  eks-large:
    extends: eks
    variables:
      node_instance_type: "m5.2xlarge"
      desired_capacity: 10
      max_capacity: 50
```

---

## Layer 3: Pipelines Configuration

**Purpose**: Define execution workflows with operation types and module selection

**File**: `.tfpipboy/pipelines.yaml`

### Basic Structure

```yaml
# Pipelines version
version: "1.0"

# Pipeline definitions
pipelines:
  # Pipeline name
  plan-all:
    description: "Plan all infrastructure changes"
    
    # Default operation for all modules in this pipeline
    default_operation: plan
    
    # Stages define execution order
    stages:
      - name: "Network Planning"
        modules:
          - vpc
      
      - name: "Compute & Data Planning"
        modules:
          - eks
          - rds
        # Modules in same stage run in parallel (respecting dependencies)
        parallel: true
      
      - name: "Application Planning"
        modules:
          - app
  
  deploy-networking:
    description: "Deploy only networking infrastructure"
    default_operation: apply
    
    stages:
      - name: "Deploy VPC"
        modules:
          - vpc
  
  deploy-all:
    description: "Deploy complete infrastructure stack"
    default_operation: apply
    
    # Pre-pipeline hooks
    before:
      - name: "Validate Configuration"
        command: "terraform fmt -check -recursive ."
      - name: "Security Scan"
        command: "tfsec ./modules"
    
    stages:
      - name: "Networking"
        modules:
          - vpc
      
      - name: "Data Layer"
        modules:
          - eks
          - rds
        parallel: true
      
      - name: "Application"
        modules:
          - app
        
        # Stage-level hooks
        after:
          - name: "Health Check"
            command: "kubectl get pods -A"
    
    # Post-pipeline hooks
    after:
      - name: "Notify Success"
        command: "echo 'Deployment completed successfully'"
  
  destroy-all:
    description: "Destroy all infrastructure (reverse order)"
    default_operation: destroy
    
    # Destroy in reverse dependency order
    stages:
      - name: "Remove Application"
        modules:
          - app
      
      - name: "Remove Data Layer"
        modules:
          - rds
          - eks
        parallel: true
      
      - name: "Remove Networking"
        modules:
          - vpc
    
    # Confirmation required for destroy
    require_confirmation: true
    confirmation_message: "⚠️  This will DESTROY all infrastructure. Type 'yes' to continue."

# Advanced Pipeline: Per-Module Operations
pipelines:
  deploy-with-mixed-ops:
    description: "Deploy with different operations per module"
    
    stages:
      - name: "Mixed Operations"
        modules:
          # Just plan VPC (don't apply)
          - name: vpc
            operation: plan
          
          # Apply EKS
          - name: eks
            operation: apply
          
          # Destroy old RDS
          - name: rds-old
            operation: destroy
          
          # Apply new RDS
          - name: rds
            operation: apply
            # Module-specific variable override
            variables:
              instance_class: "db.r6g.large"

# Pipeline with Conditional Execution
pipelines:
  deploy-conditional:
    description: "Deploy with conditional modules"
    default_operation: apply
    
    stages:
      - name: "Always Deploy"
        modules:
          - vpc
      
      - name: "Conditional Deploy"
        modules:
          # Only deploy if variable is true
          - name: eks
            when: "${features.enable_kubernetes}"
          
          - name: rds
            when: "${features.enable_database}"
          
          - name: serverless
            when: "${features.enable_serverless}"

# Pipeline using Module Groups
pipelines:
  deploy-by-groups:
    description: "Deploy using module groups from modules.yaml"
    default_operation: apply
    
    stages:
      - name: "Infrastructure"
        groups:
          - infrastructure  # Expands to: vpc, eks, rds
      
      - name: "Application"
        groups:
          - application     # Expands to: app

# Pipeline with Retries and Timeouts
pipelines:
  deploy-resilient:
    description: "Deploy with retry logic"
    default_operation: apply
    
    # Pipeline-level settings
    settings:
      parallel_limit: 3        # Max 3 modules in parallel
      timeout: 30m             # Pipeline timeout
      retry_failed: 2          # Retry failed modules twice
      retry_delay: 30s         # Wait between retries
      continue_on_error: false # Stop on first error
    
    stages:
      - name: "Critical Infrastructure"
        modules:
          - vpc
          - eks
        
        # Stage-level overrides
        settings:
          timeout: 45m           # Longer timeout for this stage
          retry_failed: 3        # More retries for critical modules

# Pipeline Composition (referencing other pipelines)
pipelines:
  full-deployment:
    description: "Complete deployment workflow"
    
    # Run other pipelines in sequence
    includes:
      - pipeline: plan-all
        when: "${auto_plan}"
      
      - pipeline: deploy-all
        require_approval: true
      
      - pipeline: run-tests
        on_failure: rollback

# Environment-Specific Pipeline Behavior
pipelines:
  deploy-all:
    description: "Deploy infrastructure"
    default_operation: apply
    
    stages:
      - name: "Deploy"
        modules:
          - vpc
          - eks
          - rds
    
    # Different behavior per environment
    environments:
      dev:
        settings:
          auto_approve: true
          parallel_limit: 5
      
      staging:
        settings:
          auto_approve: true
          parallel_limit: 3
        before:
          - name: "Backup State"
            command: "scripts/backup-state.sh"
      
      prod:
        settings:
          auto_approve: false
          require_approval: true
          parallel_limit: 2
        before:
          - name: "Create Backup"
            command: "scripts/backup-prod.sh"
          - name: "Notify Team"
            command: "scripts/notify-deployment.sh"
        after:
          - name: "Smoke Tests"
            command: "scripts/smoke-tests.sh"
          - name: "Update Documentation"
            command: "scripts/update-docs.sh"
```

---

## Complete Configuration Examples

### Example 1: Simple Multi-Module Project

**`.tfpipboy/variables.yaml`**
```yaml
variables:
  project: "webapp"
  aws:
    region: "us-west-2"

environments:
  dev:
    environment: "dev"
  prod:
    environment: "prod"
```

**`.tfpipboy/modules.yaml`**
```yaml
version: "1.0"

modules:
  network:
    path: "./modules/network"
    depends_on: []
  
  compute:
    path: "./modules/compute"
    depends_on: [network]

groups:
  all: [network, compute]
```

**`.tfpipboy/pipelines.yaml`**
```yaml
version: "1.0"

pipelines:
  deploy:
    default_operation: apply
    stages:
      - name: "All"
        groups: [all]
```

**Usage:**
```bash
# Deploy everything
tfpipboy run deploy

# Deploy to specific environment
tfpipboy run deploy --env prod
```

---

### Example 2: Complex Multi-Environment Project

**`.tfpipboy/variables.yaml`**
```yaml
variables:
  organization: "acme"
  project: "ecommerce"
  
  azure:
    subscription_id: "xxx"
    location: "eastus"
  
  naming:
    prefix: "${organization}-${project}-${environment}"

environments:
  dev:
    environment: "dev"
    auto_approve: true
    azure:
      location: "eastus2"
  
  prod:
    environment: "prod"
    auto_approve: false
    azure:
      location: "eastus"
```

**`.tfpipboy/modules.yaml`**
```yaml
version: "1.0"

modules:
  vpc:
    path: "./modules/vpc"
    depends_on: []
    variables:
      cidr: "10.0.0.0/16"
    environments:
      prod:
        variables:
          cidr: "10.1.0.0/16"
  
  aks:
    path: "./modules/aks"
    depends_on: [vpc]
    variables:
      node_count: 3
    environments:
      dev:
        variables:
          node_count: 1
      prod:
        variables:
          node_count: 5
  
  postgresql:
    path: "./modules/postgresql"
    depends_on: [vpc]
    variables:
      sku: "B_Gen5_1"
    environments:
      prod:
        variables:
          sku: "GP_Gen5_4"
  
  app:
    path: "./modules/app"
    depends_on: [aks, postgresql]

groups:
  infra: [vpc, aks, postgresql]
  app: [app]
```

**`.tfpipboy/pipelines.yaml`**
```yaml
version: "1.0"

pipelines:
  plan:
    description: "Plan all changes"
    default_operation: plan
    stages:
      - name: "Infrastructure"
        groups: [infra]
        parallel: true
      - name: "Application"
        groups: [app]
  
  deploy-infra:
    description: "Deploy infrastructure only"
    default_operation: apply
    stages:
      - name: "Network"
        modules: [vpc]
      - name: "Services"
        modules: [aks, postgresql]
        parallel: true
    
    environments:
      prod:
        require_confirmation: true
        before:
          - name: "Backup"
            command: "scripts/backup.sh"
  
  deploy-app:
    description: "Deploy application only"
    default_operation: apply
    stages:
      - name: "Application"
        modules: [app]
  
  deploy-all:
    description: "Full deployment"
    includes:
      - pipeline: deploy-infra
      - pipeline: deploy-app
        require_approval: true
  
  destroy:
    description: "Destroy all infrastructure"
    default_operation: destroy
    stages:
      - name: "App"
        modules: [app]
      - name: "Services"
        modules: [postgresql, aks]
        parallel: true
      - name: "Network"
        modules: [vpc]
    
    require_confirmation: true
    confirmation_message: "⚠️  DESTROY ALL? Type 'destroy-all' to confirm."
```

---

## Configuration Schema Reference

### Supported Operations

| Operation | Description | Terraform Commands |
|-----------|-------------|-------------------|
| `plan` | Show planned changes | `terraform plan` |
| `apply` | Apply changes | `terraform plan && terraform apply` |
| `destroy` | Destroy resources | `terraform plan -destroy && terraform destroy` |
| `refresh` | Refresh state | `terraform refresh` |
| `validate` | Validate configuration | `terraform validate` |
| `init` | Initialize only | `terraform init` |

### Variable Interpolation Syntax

```yaml
# Simple variable reference
"${variable_name}"

# Nested path
"${azure.location}"

# Output reference
"${modules.vpc.outputs.vpc_id}"

# Template reference
"${templates.resource_name}"

# Environment variable
"${env.AWS_REGION}"

# Conditional (ternary)
"${condition ? true_value : false_value}"

# Default value
"${variable_name | default_value}"
```

### Hook Types

```yaml
before:  # Execute before stage/pipeline
after:   # Execute after stage/pipeline
on_success:  # Execute only if successful
on_failure:  # Execute only if failed
```

---

## Migration Path

### Phase 1: Single File (MVP)
Start with everything in one file for simplicity:

```yaml
# tfpipboy.yaml (single file)
version: "1.0"

variables:
  project: "myapp"

modules:
  vpc:
    path: "./modules/vpc"

pipelines:
  deploy:
    default_operation: apply
    stages:
      - name: "Deploy"
        modules: [vpc]
```

### Phase 2: Split Configuration
As complexity grows, split into multiple files:

```
.tfpipboy/
├── variables.yaml
├── modules.yaml
└── pipelines.yaml
```

### Phase 3: Environment-Specific
Add environment-specific overrides:

```
.tfpipboy/
├── variables.yaml
├── modules.yaml
├── pipelines.yaml
└── environments/
    ├── dev/
    │   └── variables.yaml
    └── prod/
        └── variables.yaml
```

---

## Best Practices

### 1. Use Module Groups for Common Patterns
```yaml
groups:
  core: [vpc, security, logging]
  data: [rds, redis, s3]
  compute: [eks, lambda]
```

### 2. Define Reusable Pipelines
```yaml
pipelines:
  plan-all:
    default_operation: plan
    stages:
      - name: "Plan"
        groups: [all]
  
  deploy-all:
    includes:
      - pipeline: plan-all
      - pipeline: apply-all
        require_approval: true
```

### 3. Use Environment Overrides Sparingly
```yaml
# Good: Override only what's different
environments:
  prod:
    variables:
      instance_size: "large"  # Only this is different

# Bad: Duplicate everything
environments:
  prod:
    variables:
      instance_size: "large"
      region: "us-east-1"      # Already in global vars
      project: "myapp"         # Already in global vars
```

### 4. Leverage Variable Templates
```yaml
templates:
  resource_name: "${org}-${project}-${env}-${type}"
  
modules:
  vpc:
    variables:
      name: "${templates.resource_name}"  # Consistent naming
```

---

## Validation Rules

### Required Fields
- `version` in all configuration files
- `modules[].path` - module path must exist
- `pipelines[].stages[].modules` - at least one module or group

### Constraints
- Module names must be unique
- Pipeline names must be unique
- No circular dependencies in modules
- `depends_on` must reference existing modules
- Operations must be valid: plan, apply, destroy, refresh, validate, init

### Warnings
- Unused modules defined but not referenced in any pipeline
- Undefined variables referenced in interpolation
- Missing outputs referenced by other modules

---

## CLI Usage Examples

```bash
# Run specific pipeline
tfpipboy run deploy-all

# Run pipeline for specific environment
tfpipboy run deploy-all --env prod

# Override variables from CLI
tfpipboy run deploy-all --var instance_size=large

# Dry-run (show what would be executed)
tfpipboy run deploy-all --dry-run

# Run with specific modules only
tfpipboy run deploy-all --modules vpc,eks

# Skip specific modules
tfpipboy run deploy-all --skip-modules app

# List available pipelines
tfpipboy pipelines list

# Validate configuration
tfpipboy config validate

# Show resolved configuration
tfpipboy config show --env prod
```

---

## Next Steps

1. **Review and approve** this configuration format design
2. **Implement YAML parser** with validation (US-102)
3. **Create JSON schema** for IDE autocomplete
4. **Build example configurations** for testing
5. **Document migration guide** from single-file to multi-file

---

## Appendix: JSON Schema

To enable IDE autocomplete and validation, JSON schemas should be created:

- `.tfpipboy/schemas/variables.schema.json`
- `.tfpipboy/schemas/modules.schema.json`
- `.tfpipboy/schemas/pipelines.schema.json`

These can be referenced in YAML files:
```yaml
# yaml-language-server: $schema=.tfpipboy/schemas/pipelines.schema.json
version: "1.0"
```

---

**Document Status**: Draft v1.0  
**Needs Review**: Configuration hierarchy, variable interpolation syntax  
**Next**: Implement parser and validator
