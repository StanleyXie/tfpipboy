# tfpipboy Orchestrator

A powerful Terraform module orchestrator that manages complex infrastructure deployments with dependency-aware sequencing, parallel execution, and isolated environments.

## Overview

The tfpipboy orchestrator solves the challenge of managing multiple Terraform modules with complex dependencies. It provides:

- **Dependency-aware execution**: Automatically determines the correct order to execute modules based on their dependencies
- **Parallel execution**: Runs independent modules in parallel to reduce deployment time
- **Isolated environments**: Each module runs in its own isolated workspace to prevent conflicts
- **Pipeline support**: Define reusable deployment workflows with stages and hooks
- **Flexible configuration**: YAML-based configuration for modules, dependencies, and pipelines

## Key Features

### 🔄 Dependency Management
- Automatic dependency graph construction
- Topological sorting for correct execution order
- Circular dependency detection
- Support for module instances (one module, multiple configurations)

### ⚡ Parallel Execution
- Configurable concurrency limits
- Automatic parallelization of independent modules
- Stage-based execution with parallel capabilities within stages

### 🏗️ Isolated Workspaces
- Each module execution gets its own temporary workspace
- Isolated backend configurations
- Environment variable isolation
- No cross-module state conflicts

### 📋 Pipeline Orchestration
- Define multi-stage deployment workflows
- Before/after hooks for custom commands
- Environment-specific configurations
- Built-in error handling and rollback

### 🎯 Flexible Targeting
- Execute specific modules or groups
- Pipeline-based execution
- Support for module instances
- Group expansion

## Installation

Build from source:

```bash
git clone https://github.com/StanleyXie/tfpipboy.git
cd tfpipboy
go build -o tfpipboy cmd/tfpipboy/main.go
```

## Quick Start

1. **Create configuration directory**:
```bash
mkdir .tfpipboy
```

2. **Define your modules** in `.tfpipboy/modules.yaml`:
```yaml
version: "1.0"

modules:
  foundation:
    description: "Basic infrastructure foundation"
    path: "./modules/foundation"
    depends_on: []
    
  network:
    description: "Network infrastructure" 
    path: "./modules/network"
    depends_on: ["foundation"]
    
  application:
    description: "Application infrastructure"
    path: "./modules/application"
    depends_on: ["network"]

groups:
  all:
    - foundation
    - network
    - application
```

3. **Execute modules**:
```bash
# Plan all modules
./tfpipboy --targets all --operation plan

# Apply specific modules
./tfpipboy --targets foundation,network --operation apply

# Execute a pipeline
./tfpipboy --pipeline deploy-all
```

## Configuration

### Module Configuration

Modules are defined in `.tfpipboy/modules.yaml`:

```yaml
version: "1.0"

modules:
  # Simple module
  foundation:
    description: "Infrastructure foundation"
    path: "./modules/foundation"
    depends_on: []
    backend:
      type: "azurerm"
      subscription_id: "xxx"
      resource_group_name: "rg-terraform"
      storage_account_name: "terraformstate"
      container_name: "tfstate" 
      key: "foundation.tfstate"
    var-config:
      file: "./vars/foundation.tfvars"
      json:
        environment: "production"
        region: "eastus"
    outputs:
      - vpc_id
      - resource_group_name

  # Module with instances
  database:
    description: "Database infrastructure"
    path: "./modules/database"
    instances:
      primary:
        description: "Primary database"
        depends_on: ["network"]
        var-config:
          json:
            db_name: "primary"
            instance_type: "Standard_D2s_v3"
      replica:
        description: "Read replica"
        depends_on: ["database.primary"]
        var-config:
          json:
            db_name: "replica" 
            instance_type: "Standard_D1s_v3"

groups:
  infrastructure:
    - foundation
    - network
  database:
    - database.primary
    - database.replica
  all:
    - foundation
    - network
    - database.primary
    - database.replica
```

### Pipeline Configuration

Pipelines are defined in `.tfpipboy/pipelines.yaml`:

```yaml
version: "1.0"

pipelines:
  deploy-all:
    description: "Deploy complete infrastructure"
    default_operation: apply
    
    settings:
      parallel_limit: 3
      timeout: "60m"
      auto_approve: false
      
    before:
      - name: "Pre-deployment validation"
        command: "terraform --version"
        
    stages:
      - name: "Foundation"
        modules:
          - foundation
        parallel: false
        
      - name: "Core Infrastructure"
        modules:
          - network
          - database.primary
        parallel: true
        
      - name: "Application Layer"
        modules:
          - database.replica
          - application
        parallel: true
        
    after:
      - name: "Deployment complete"
        command: "echo 'Infrastructure deployed successfully'"
        
    on_failure:
      - name: "Cleanup on failure"
        command: "echo 'Deployment failed, check logs'"
```

## Usage Examples

### Basic Operations

```bash
# List available modules
./tfpipboy --list-modules

# List available pipelines  
./tfpipboy --list-pipelines

# List available groups
./tfpipboy --list-groups

# Plan specific modules
./tfpipboy --targets foundation,network --operation plan

# Apply with dry run
./tfpipboy --targets all --operation apply --dry-run

# Execute pipeline
./tfpipboy --pipeline deploy-all --env production

# Validate configurations
./tfpipboy --targets all --operation validate
```

### Advanced Usage

```bash
# Custom configuration path
./tfpipboy --config ./custom-config --targets all --operation plan

# Parallel execution control
./tfpipboy --targets all --operation apply --parallel 5

# Custom timeout
./tfpipboy --pipeline deploy-all --timeout 90m

# Verbose logging
./tfpipboy --targets all --operation plan --verbose
```

## Supported Operations

- `plan` - Create execution plan
- `apply` - Apply changes
- `destroy` - Destroy resources  
- `validate` - Validate configuration
- `refresh` - Refresh state
- `output` - Show outputs

## Backend Support

The orchestrator supports multiple Terraform backends:

- **Azure (`azurerm`)**: Azure Storage Account backend
- **AWS (`s3`)**: S3 bucket backend  
- **GCP (`gcs`)**: Google Cloud Storage backend
- **Local (`local`)**: Local state files

Each module can have its own backend configuration with proper isolation.

## Environment Isolation

Each module execution runs in an isolated environment:

- **Workspace isolation**: Temporary workspace directory per job
- **Backend isolation**: Module-specific backend configuration
- **Variable isolation**: Module-specific variables and tfvars files
- **Environment variables**: Controlled environment variable inheritance
- **State isolation**: No shared state between modules

## Examples

See the `examples/` directory for complete working examples:

- `examples/azure-landing-zone/`: Complex Azure Landing Zone with multiple modules and instances
- `examples/simple-demo/`: Simple demonstration with foundation, network, database, and application modules

## Architecture

### Core Components

1. **ConfigParser**: Loads and validates YAML configuration
2. **DependencyGraphBuilder**: Builds dependency graphs and execution order
3. **WorkspaceManager**: Manages isolated execution workspaces
4. **TerraformExecutor**: Executes Terraform commands with proper isolation
5. **ParallelExecutor**: Handles parallel execution with concurrency control
6. **DefaultOrchestrator**: Main orchestrator coordinating all components

### Execution Flow

1. Load and validate configuration
2. Expand target groups to individual modules
3. Build dependency graph
4. Calculate execution stages (topological sort)
5. Create isolated workspaces for each module
6. Execute stages sequentially, modules within stages in parallel
7. Clean up workspaces

## Best Practices

### Module Design
- Keep modules focused and single-purpose
- Use clear, descriptive module names
- Define explicit dependencies
- Include comprehensive outputs

### Configuration Management
- Use groups to organize related modules
- Leverage module instances for similar modules with different configurations
- Keep sensitive values in external tfvars files
- Use consistent naming conventions

### Pipeline Design
- Start with validation stages
- Group independent modules for parallel execution
- Include meaningful hooks for monitoring and validation
- Plan for failure scenarios with appropriate cleanup

### Dependency Management
- Minimize dependencies where possible
- Use data sources instead of direct dependencies when appropriate
- Avoid circular dependencies
- Document dependency rationale

## Troubleshooting

### Common Issues

1. **Circular Dependencies**
   ```
   Error: circular dependency detected involving module1
   ```
   Solution: Review module dependencies and break circular references

2. **Module Not Found**
   ```
   Error: module 'xyz' not found
   ```
   Solution: Check module name spelling and ensure module exists in configuration

3. **Backend Configuration Issues**
   ```
   Error: failed to setup backend
   ```
   Solution: Verify backend configuration and credentials

4. **Timeout Issues**
   ```
   Error: terraform command timed out
   ```
   Solution: Increase timeout or check for hanging Terraform processes

### Debug Mode

Enable verbose logging for detailed execution information:

```bash
./tfpipboy --targets all --operation plan --verbose
```

### Log Files

Execution logs include:
- Module execution start/completion
- Terraform command output
- Error details and stack traces
- Timing information

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Submit a pull request

## License

MIT License - see LICENSE file for details