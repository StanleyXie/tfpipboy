# Parallel Execution Control

## Overview

tf-pipboy provides flexible control over parallel execution of Terraform modules at both global and instance levels. You can execute multiple instances in parallel for faster deployments while maintaining fine-grained control over which instances can run concurrently.

## Features

### 1. Global Parallel Execution (`--parallel-all`)

Force all targeted instances to run in parallel, ignoring dependency order.

**Usage:**
```bash
tfpipboy --targets instance1,instance2,instance3 --operation apply --parallel-all
```

**Behavior:**
- All instances are placed in stage 0
- All instances are marked as parallelizable
- Dependencies are ignored (instances run simultaneously)
- Respects `--parallel N` limit (default: 3 concurrent executions)

**Use Cases:**
- Independent infrastructure components
- Testing/development environments where order doesn't matter
- Fast teardown with `--operation destroy --parallel-all`
- Batch operations on isolated resources

### 2. Instance-Level Parallel Control

Configure individual instances to run in parallel or sequentially via YAML configuration.

**Configuration:**
```yaml
modules:
  connectivity:
    path: "./modules/connectivity"
    
    instances:
      conn-prod-gwc:
        environment: "prod"
        region: "germanywestcentral"
        parallel: true  # Force this instance to run in parallel
        
      conn-prod-sdc:
        environment: "prod"
        region: "swedencentral"
        parallel: false  # Force this instance to run sequentially
        
      conn-dev-gwc:
        environment: "dev"
        region: "germanywestcentral"
        # parallel: not set - inherits from dependency graph
```

**Settings:**
- `parallel: true` - Forces instance to run in parallel with others in the same stage
- `parallel: false` - Forces instance to run sequentially (blocks parallel execution)
- `parallel: <not set>` - Default behavior based on dependency graph

### 3. Parallel Limit (`--parallel N`)

Control the maximum number of concurrent executions.

**Usage:**
```bash
tfpipboy --targets all --parallel 5 --operation apply
```

**Default:** 3 concurrent executions  
**Range:** 1 to unlimited (practically limited by system resources)

## Configuration Examples

### Example 1: Multi-Region Deployment (Parallel)

Deploy the same module to multiple regions in parallel:

```yaml
modules:
  regional-infra:
    path: "./modules/regional"
    
    instances:
      infra-gwc:
        region: "germanywestcentral"
        parallel: true
        
      infra-sdc:
        region: "swedencentral"
        parallel: true
        
      infra-neu:
        region: "northeurope"
        parallel: true
```

**Execution:**
```bash
tfpipboy --targets infra-gwc,infra-sdc,infra-neu --operation apply
# All three regions deploy simultaneously
```

### Example 2: Sequential Critical Infrastructure

Force critical infrastructure to deploy sequentially:

```yaml
modules:
  bootstrap:
    path: "./modules/bootstrap"
    
    instances:
      bootstrap-networking:
        parallel: false  # Must complete first
        depends_on: []
        
      bootstrap-security:
        parallel: false  # Must complete second
        depends_on: [bootstrap-networking]
        
      bootstrap-monitoring:
        parallel: false  # Must complete last
        depends_on: [bootstrap-security]
```

### Example 3: Mixed Parallel and Sequential

Combine parallel and sequential execution:

```yaml
modules:
  platform:
    path: "./modules/platform"
    
    instances:
      # Foundation (sequential - must be in order)
      platform-network:
        parallel: false
        depends_on: []
        
      platform-identity:
        parallel: false
        depends_on: [platform-network]
      
      # Applications (parallel - can run together)
      platform-app1:
        parallel: true
        depends_on: [platform-identity]
        
      platform-app2:
        parallel: true
        depends_on: [platform-identity]
        
      platform-app3:
        parallel: true
        depends_on: [platform-identity]
```

**Execution Flow:**
1. **Stage 1:** `platform-network` (sequential)
2. **Stage 2:** `platform-identity` (sequential)
3. **Stage 3:** `platform-app1`, `platform-app2`, `platform-app3` (all parallel)

### Example 4: Environment-Based Parallelization

Different parallel settings for different environments:

```yaml
modules:
  application:
    path: "./modules/application"
    
    instances:
      # Production - sequential for safety
      app-prod:
        environment: "prod"
        parallel: false
        
      # Staging - parallel for speed
      app-staging:
        environment: "staging"
        parallel: true
        
      # Development - parallel for speed
      app-dev:
        environment: "dev"
        parallel: true
```

## Command-Line Usage

### Basic Parallel Execution

```bash
# Execute with default parallelism (max 3 concurrent)
tfpipboy --targets module1,module2,module3 --operation plan

# Execute with custom parallelism (max 5 concurrent)
tfpipboy --targets module1,module2,module3 --operation plan --parallel 5

# Force all to run in parallel (ignoring dependencies)
tfpipboy --targets module1,module2,module3 --operation apply --parallel-all
```

### Combined with Other Flags

```bash
# Parallel execution with auto-confirm (for CI/CD)
tfpipboy --targets all --operation apply --parallel-all --auto-confirm

# Parallel dry-run
tfpipboy --targets all --operation plan --parallel-all --dry-run

# Parallel destroy with high concurrency
tfpipboy --targets all --operation destroy --parallel-all --parallel 10
```

## Execution Plan Preview

The execution plan shows parallelization clearly:

**Without `--parallel-all`:**
```
Execution Sequence:
--------------------------------------------------------------------------------
Stage  Instance             Backend      Details      Dependencies
--------------------------------------------------------------------------------
1      seed-main            azurerm      main         -
2      core-main-gwc        azurerm      main         seed-main
3∥     conn-gwc-prod        azurerm      prod         core-main-gwc
3∥     conn-sdc-prod        azurerm      prod         core-main-gwc
4      mgmt-sdc-prod        azurerm      prod         conn-sdc-prod
--------------------------------------------------------------------------------
```

**With `--parallel-all`:**
```
Execution Sequence:
--------------------------------------------------------------------------------
Stage  Instance             Backend      Details      Dependencies
--------------------------------------------------------------------------------
0∥     seed-main            azurerm      main         -
0∥     core-main-gwc        azurerm      main         -
0∥     conn-gwc-prod        azurerm      prod         -
0∥     conn-sdc-prod        azurerm      prod         -
0∥     mgmt-sdc-prod        azurerm      prod         -
--------------------------------------------------------------------------------
Note: ∥ indicates parallel execution within the same stage
```

## Performance Considerations

### Benefits of Parallel Execution

1. **Faster Deployments:** Multiple instances execute simultaneously
2. **Better Resource Utilization:** Uses available CPU/network capacity
3. **Reduced Total Time:** Especially beneficial for large infrastructures

### Considerations

1. **API Rate Limits:** Cloud providers may throttle parallel requests
2. **Resource Contention:** Too many parallel executions can cause timeouts
3. **Dependency Violations:** `--parallel-all` ignores dependencies - use carefully
4. **Log Readability:** Parallel execution interleaves output

### Recommended Settings

**Small Infrastructure (< 10 instances):**
```bash
--parallel 3  # Default, safe for most scenarios
```

**Medium Infrastructure (10-50 instances):**
```bash
--parallel 5  # Good balance of speed and safety
```

**Large Infrastructure (> 50 instances):**
```bash
--parallel 10  # Maximum recommended for cloud APIs
```

**Development/Testing:**
```bash
--parallel-all --parallel 20  # Maximum speed, ignore dependencies
```

## Safety Guidelines

### When to Use `--parallel-all`

✅ **Safe:**
- Independent infrastructure components (different projects/regions)
- Development/test environments
- Destroying resources (`--operation destroy`)
- Read-only operations (`--operation plan`)

❌ **Risky:**
- Production environments with dependencies
- Infrastructure with strict ordering requirements
- Initial deployments (bootstrap scenarios)
- Stateful resources with dependencies

### When to Use `parallel: false`

Use `parallel: false` for:
- **Networking:** VNets, subnets must exist before VMs
- **Identity:** AAD groups must exist before role assignments
- **Secrets:** Key Vaults must exist before storing secrets
- **State:** Backend storage must exist before Terraform state
- **Critical Path:** Any resource that others depend on

### When to Use `parallel: true`

Use `parallel: true` for:
- **Independent Regions:** Same module deployed to different regions
- **Isolated Applications:** App services without cross-dependencies
- **Batch Resources:** Multiple VMs, storage accounts, etc.
- **Testing:** Test instances that don't interfere with each other

## Troubleshooting

### Issue: Parallel Execution Fails with Timeouts

**Cause:** Too many concurrent executions overwhelming cloud APIs

**Solution:**
```bash
# Reduce parallelism
tfpipboy --targets all --parallel 2 --operation apply
```

### Issue: Dependency Errors with `--parallel-all`

**Cause:** Resources deployed out of order

**Solution:**
```bash
# Remove --parallel-all and let dependency graph control order
tfpipboy --targets all --operation apply
```

### Issue: Some Instances Not Running in Parallel

**Cause:** Instance has `parallel: false` in configuration

**Solution:**
```yaml
# Change instance configuration
instances:
  my-instance:
    parallel: true  # or remove this line to use default
```

## Implementation Details

### How `--parallel-all` Works

1. **Modifies Execution Plan:** All jobs moved to stage 0
2. **Clears Dependencies:** `DependsOn` set to empty array
3. **Marks Parallelizable:** `CanParallel` set to `true`
4. **Respects Limit:** Still honors `--parallel N` limit

### How Instance `parallel` Setting Works

1. **Plan Creation:** During `PlanExecution()`, instance settings are checked
2. **Override CanParallel:** If `parallel: true/false`, overrides default logic
3. **Respects Dependencies:** Does not break dependency order
4. **Stage Assignment:** Parallel instances can be in same stage

### Execution Flow

```
┌─────────────────────────────────────────────┐
│ 1. Parse targets                            │
└────────────────┬────────────────────────────┘
                 ▼
┌─────────────────────────────────────────────┐
│ 2. Build dependency graph                   │
└────────────────┬────────────────────────────┘
                 ▼
┌─────────────────────────────────────────────┐
│ 3. Create execution plan (stages)           │
│    - Check instance.Parallel setting        │
│    - Apply to job.CanParallel               │
└────────────────┬────────────────────────────┘
                 ▼
┌─────────────────────────────────────────────┐
│ 4. Apply --parallel-all (if specified)      │
│    - Move all to stage 0                    │
│    - Clear dependencies                     │
│    - Force CanParallel = true               │
└────────────────┬────────────────────────────┘
                 ▼
┌─────────────────────────────────────────────┐
│ 5. Execute stages                           │
│    - Respect --parallel N limit             │
│    - Run parallel jobs concurrently         │
└─────────────────────────────────────────────┘
```

## Future Enhancements

### Planned Features

1. **Module-Level Parallel Setting:**
   ```yaml
   modules:
     my-module:
       parallel: true  # All instances inherit this
   ```

2. **Dynamic Parallel Limit:**
   ```yaml
   instances:
     my-instance:
       parallel: true
       parallel_limit: 5  # Custom limit for this instance
   ```

3. **Conditional Parallelization:**
   ```yaml
   instances:
     my-instance:
       parallel: "${env == 'dev' ? true : false}"
   ```

4. **Parallel Execution Groups:**
   ```yaml
   groups:
     fast-deploy:
       modules: [app1, app2, app3]
       parallel: true
   ```

## Related Documentation

- [Dependency Management](./DEPENDENCY-MANAGEMENT.md)
- [Execution Plan Preview](./EXECUTION-PLAN-PREVIEW.md)
- [Configuration Format](./CONFIG-FORMAT-SPEC.md)
