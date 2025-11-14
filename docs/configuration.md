# Configuration Reference

Complete reference for tf-pipboy configuration files.

## Overview

tf-pipboy uses YAML-based configuration stored in the `.tfpipboy/` directory. The main configuration file is `tfproject.yaml`.

## Configuration File Location

```
.tfpipboy/
└── tfproject.yaml    # Main configuration file
```

## Configuration Schema

### Root Level

```yaml
version: "1.0"                # Configuration format version (required)

project:                      # Project metadata (optional)
  name: "string"
  description: "string"

variables:                    # Global variables (optional)
  key: value

modules:                      # Module definitions (required)
  - name: "string"
    path: "string"
    # ... module configuration

instances:                    # Instance definitions (required)
  - module: "string"
    workspace: "string"
    # ... instance configuration

pipelines:                    # Pipeline definitions (optional)
  pipeline-name:
    # ... pipeline configuration

backends:                     # Backend templates (optional)
  backend-name:
    # ... backend configuration
```

---

## Project Configuration

Project metadata for documentation and display purposes.

```yaml
project:
  name: "my-infrastructure"
  description: "Production infrastructure deployment"
```

### Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | No | Project name |
| `description` | string | No | Project description |

---

## Variables

Global variables available to all modules and instances.

```yaml
variables:
  environment: production
  region: us-east-1
  project_name: my-app
  
  # Nested objects
  azure:
    subscription_id: "xxx"
    tenant_id: "yyy"
    
  # Arrays
  allowed_regions:
    - us-east-1
    - eu-west-1
```

### Variable Substitution

Variables can be referenced using `${variable_name}` syntax:

```yaml
instances:
  - module: networking
    workspace: "${environment}-${region}-network"
    variables:
      env: "${environment}"
      location: "${azure.location}"
```

### Supported Types

- **String**: `"value"` or `value`
- **Number**: `123` or `45.67`
- **Boolean**: `true` or `false`
- **Object**: Nested key-value pairs
- **Array**: List of values

---

## Module Definitions

Modules define Terraform configurations that can be deployed.

```yaml
modules:
  - name: networking
    path: terraform/modules/networking
    description: "Network infrastructure"
    depends_on: []
    backend:
      type: azurerm
      # ... backend configuration
```

### Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Unique module identifier |
| `path` | string | Yes | Path to Terraform module directory |
| `description` | string | No | Human-readable description |
| `depends_on` | array | No | List of module dependencies |
| `backend` | object | No | Backend configuration for this module |

### Example

```yaml
modules:
  - name: networking
    path: terraform/modules/networking
    description: "Network infrastructure with VPC and subnets"
    depends_on: []
    
  - name: database
    path: terraform/modules/database
    description: "PostgreSQL database"
    depends_on:
      - networking
    backend:
      type: azurerm
      resource_group_name: "rg-terraform-state"
      storage_account_name: "tfstate"
      container_name: "tfstate"
      key: "database.tfstate"
```

---

## Instance Definitions

Instances are specific executions of modules with their own workspaces and configurations.

```yaml
instances:
  - module: networking
    workspace: "prod-us-east-1-network"
    description: "US East 1 production network"
    variables:
      region: us-east-1
      cidr_block: "10.0.0.0/16"
    backend:
      key: "prod-us-east-1-network.tfstate"
```

### Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `module` | string | Yes | Reference to module name |
| `workspace` | string | Yes | Unique workspace identifier |
| `description` | string | No | Instance description |
| `variables` | object | No | Terraform variables for this instance |
| `backend` | object | No | Backend configuration override |
| `depends_on` | array | No | Additional instance dependencies |

### Instance-Specific Dependencies

Instances can have additional dependencies beyond module dependencies:

```yaml
instances:
  - module: database
    workspace: "prod-primary-db"
    
  - module: database
    workspace: "prod-replica-db"
    depends_on:
      - "prod-primary-db"  # Replica depends on primary
```

### Multi-Region Example

```yaml
modules:
  - name: regional-infrastructure
    path: terraform/modules/regional

instances:
  # US East 1
  - module: regional-infrastructure
    workspace: "prod-us-east-1"
    variables:
      region: us-east-1
      cidr_block: "10.0.0.0/16"
      
  # EU West 1  
  - module: regional-infrastructure
    workspace: "prod-eu-west-1"
    variables:
      region: eu-west-1
      cidr_block: "10.1.0.0/16"
```

---

## Backend Configuration

Backend configuration for Terraform state storage.

### Module-Level Backend

Applied to all instances of a module:

```yaml
modules:
  - name: networking
    backend:
      type: azurerm
      resource_group_name: "rg-terraform-state"
      storage_account_name: "tfstate"
      container_name: "tfstate"
      key: "networking.tfstate"
```

### Instance-Level Backend

Overrides module backend for specific instance:

```yaml
instances:
  - module: networking
    workspace: "prod-network"
    backend:
      key: "prod-network.tfstate"
      # Inherits other properties from module backend
```

### Supported Backend Types

#### Azure (azurerm)

```yaml
backend:
  type: azurerm
  resource_group_name: "rg-terraform-state"
  storage_account_name: "terraformstate"
  container_name: "tfstate"
  key: "terraform.tfstate"
  subscription_id: "xxx"  # Optional
  tenant_id: "yyy"        # Optional
```

#### AWS (s3)

```yaml
backend:
  type: s3
  bucket: "terraform-state"
  key: "terraform.tfstate"
  region: "us-east-1"
  dynamodb_table: "terraform-locks"  # Optional
  encrypt: true                       # Optional
```

#### GCP (gcs)

```yaml
backend:
  type: gcs
  bucket: "terraform-state"
  prefix: "terraform/state"
```

#### Local

```yaml
backend:
  type: local
  path: "terraform.tfstate"
```

---

## Pipeline Configuration

Pipelines define reusable deployment workflows with multiple stages.

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
        description: "Base infrastructure"
        instances:
          - networking
        parallel: false
        
      - name: "Services"
        description: "Database and services"
        instances:
          - database-primary
          - database-replica
        parallel: true
        
      - name: "Applications"
        description: "Application layer"
        instances:
          - application
        parallel: false
```

### Pipeline Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `description` | string | No | Pipeline description |
| `default_operation` | string | No | Default operation (plan, apply, destroy) |
| `settings` | object | No | Pipeline-level settings |
| `stages` | array | Yes | List of execution stages |

### Settings Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `parallel_limit` | number | 3 | Maximum concurrent jobs |
| `timeout` | string | "30m" | Maximum execution time |
| `auto_approve` | boolean | false | Skip confirmation prompts |

### Stage Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Stage name |
| `description` | string | No | Stage description |
| `instances` | array | Yes | List of instance workspaces |
| `parallel` | boolean | No | Allow parallel execution within stage |

### Pipeline Example

```yaml
pipelines:
  full-deployment:
    description: "Complete infrastructure deployment"
    default_operation: apply
    
    settings:
      parallel_limit: 5
      timeout: "90m"
      auto_approve: false
    
    stages:
      - name: "Network Foundation"
        instances:
          - prod-us-east-1-network
          - prod-eu-west-1-network
        parallel: true
        
      - name: "Database Layer"
        instances:
          - prod-us-east-1-db-primary
          - prod-us-east-1-db-replica
          - prod-eu-west-1-db-primary
        parallel: true
        
      - name: "Application Layer"
        instances:
          - prod-us-east-1-app
          - prod-eu-west-1-app
        parallel: true
```

---

## Complete Example

Comprehensive example showing all configuration options:

```yaml
version: "1.0"

project:
  name: "multi-region-infrastructure"
  description: "Multi-region production infrastructure"

variables:
  project: "myapp"
  environment: "production"
  
  aws:
    account_id: "123456789012"
    
  azure:
    subscription_id: "xxx-xxx-xxx"
    tenant_id: "yyy-yyy-yyy"

modules:
  - name: networking
    path: terraform/modules/networking
    description: "VPC and networking"
    depends_on: []
    backend:
      type: s3
      bucket: "terraform-state-${variables.project}"
      region: us-east-1
      dynamodb_table: "terraform-locks"
      
  - name: database
    path: terraform/modules/database
    description: "RDS PostgreSQL"
    depends_on:
      - networking
    backend:
      type: s3
      bucket: "terraform-state-${variables.project}"
      region: us-east-1
      
  - name: application
    path: terraform/modules/application
    description: "Application servers"
    depends_on:
      - networking
      - database

instances:
  # US East 1
  - module: networking
    workspace: "prod-us-east-1-network"
    variables:
      region: us-east-1
      cidr_block: "10.0.0.0/16"
    backend:
      key: "prod/us-east-1/networking.tfstate"
      
  - module: database
    workspace: "prod-us-east-1-db"
    variables:
      region: us-east-1
      instance_class: "db.r5.large"
    backend:
      key: "prod/us-east-1/database.tfstate"
      
  - module: application
    workspace: "prod-us-east-1-app"
    variables:
      region: us-east-1
      instance_type: "t3.medium"
    backend:
      key: "prod/us-east-1/application.tfstate"
      
  # EU West 1
  - module: networking
    workspace: "prod-eu-west-1-network"
    variables:
      region: eu-west-1
      cidr_block: "10.1.0.0/16"
    backend:
      key: "prod/eu-west-1/networking.tfstate"
      
  - module: database
    workspace: "prod-eu-west-1-db"
    variables:
      region: eu-west-1
      instance_class: "db.r5.large"
    backend:
      key: "prod/eu-west-1/database.tfstate"
      
  - module: application
    workspace: "prod-eu-west-1-app"
    variables:
      region: eu-west-1
      instance_type: "t3.medium"
    backend:
      key: "prod/eu-west-1/application.tfstate"

pipelines:
  deploy-all:
    description: "Deploy all regions"
    default_operation: apply
    
    settings:
      parallel_limit: 4
      timeout: "120m"
      
    stages:
      - name: "Networks"
        instances:
          - prod-us-east-1-network
          - prod-eu-west-1-network
        parallel: true
        
      - name: "Databases"
        instances:
          - prod-us-east-1-db
          - prod-eu-west-1-db
        parallel: true
        
      - name: "Applications"
        instances:
          - prod-us-east-1-app
          - prod-eu-west-1-app
        parallel: true
```

---

## Validation

tf-pipboy validates configuration before execution:

- **Schema validation**: Ensures required fields are present
- **Dependency validation**: Detects circular dependencies
- **Reference validation**: Verifies module references exist
- **Workspace uniqueness**: Ensures workspace names are unique

### Common Validation Errors

**Missing required field**:
```
Error: missing required field 'name' in module definition
```

**Circular dependency**:
```
Error: circular dependency detected: networking -> database -> networking
```

**Invalid module reference**:
```
Error: instance references unknown module 'xyz'
```

**Duplicate workspace**:
```
Error: duplicate workspace name 'prod-network'
```

---

## Best Practices

### 1. Use Meaningful Names

```yaml
# Good
workspace: "prod-us-east-1-database-primary"

# Bad
workspace: "db1"
```

### 2. Organize by Environment and Region

```yaml
workspace: "${environment}-${region}-${service}"
# Example: prod-us-east-1-application
```

### 3. Use Variables for Common Values

```yaml
variables:
  environment: production
  
instances:
  - module: networking
    workspace: "${environment}-network"
```

### 4. Group Related Instances in Pipelines

```yaml
pipelines:
  deploy-region-us:
    stages:
      - name: "US Infrastructure"
        instances:
          - prod-us-east-1-network
          - prod-us-east-1-database
```

### 5. Document Dependencies

```yaml
modules:
  - name: application
    depends_on:
      - networking  # Application needs VPC
      - database    # Application needs database connection
```

---

## Migration Guide

### From Simple Modules to Instances

**Before**:
```yaml
modules:
  - name: networking
    path: terraform/modules/networking
```

**After**:
```yaml
modules:
  - name: networking
    path: terraform/modules/networking
    
instances:
  - module: networking
    workspace: "default-network"
```

### Adding Multi-Region Support

```yaml
# Original single region
instances:
  - module: networking
    workspace: "prod-network"

# Multi-region
instances:
  - module: networking
    workspace: "prod-us-network"
    variables:
      region: us-east-1
      
  - module: networking
    workspace: "prod-eu-network"
    variables:
      region: eu-west-1
```

---

## See Also

- [Getting Started](getting-started.md) - Quick start guide
- [User Guide](user-guide.md) - Complete usage documentation
- [Examples](../examples/) - Real-world configuration examples

---

**Questions?** See [GitHub Discussions](https://github.com/StanleyXie/tf-pipboy/discussions)
