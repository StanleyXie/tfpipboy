# Configuration Design Summary
**Date**: 2025-10-19  
**Status**: Ready for Review

## Quick Reference

This document provides a quick overview of the declarative configuration format design for tf-pipboy.

## The Three-Layer Model

```
┌──────────────────────────────────────────────┐
│ Layer 1: VARIABLES (DRY Principle)          │
│ File: .tfpipboy/variables.yaml              │
│ Purpose: Shared values, environment configs  │
└──────────────────────────────────────────────┘
                    ▼
┌──────────────────────────────────────────────┐
│ Layer 2: MODULES (WHAT to deploy)           │
│ File: .tfpipboy/modules.yaml                │
│ Purpose: Infrastructure catalog, deps        │
└──────────────────────────────────────────────┘
                    ▼
┌──────────────────────────────────────────────┐
│ Layer 3: PIPELINES (HOW to deploy)          │
│ File: .tfpipboy/pipelines.yaml              │
│ Purpose: Execution workflows, operations     │
└──────────────────────────────────────────────┘
```

## Design Decisions

### ✅ Accepted Decisions

1. **Three-layer separation**: Variables, Modules, Pipelines
2. **YAML format**: Industry standard, human-readable
3. **Variable interpolation**: `${variable.path}` syntax
4. **Module groups**: Logical grouping for convenience
5. **Per-pipeline operations**: Flexible operation override (plan/apply/destroy)
6. **Environment overrides**: Override at any layer
7. **Progressive complexity**: Start simple, grow as needed

### Key Features

| Feature | Description | Example |
|---------|-------------|---------|
| **Variables** | Shared values, DRY | `project: "myapp"` |
| **Interpolation** | Reference variables | `name: "${org}-${project}"` |
| **Dependencies** | Explicit module deps | `depends_on: [vpc]` |
| **Groups** | Logical module sets | `groups: {infra: [vpc, eks]}` |
| **Operations** | Per-pipeline ops | `default_operation: apply` |
| **Hooks** | Pre/post commands | `before: [...]` |
| **Conditionals** | When to deploy | `when: "${features.enable_x}"` |
| **Environments** | Dev/staging/prod | `environments: {prod: {...}}` |

## File Structure Options

### Option A: Single File (Simple Projects)

```yaml
# tfpipboy.yaml
version: "1.0"
variables: {...}
modules: {...}
pipelines: {...}
```

**Best for**: Small projects (< 5 modules), prototypes, learning

### Option B: Multi-File (Production Projects)

```
.tfpipboy/
├── variables.yaml    # Shared variables
├── modules.yaml      # Module catalog
└── pipelines.yaml    # Workflows
```

**Best for**: Production, multiple environments, > 5 modules

### Option C: Full Hierarchy (Enterprise)

```
.tfpipboy/
├── variables.yaml
├── modules.yaml
├── pipelines.yaml
└── environments/
    ├── dev/
    │   └── variables.yaml
    ├── staging/
    │   └── variables.yaml
    └── prod/
        └── variables.yaml
```

**Best for**: Enterprise, complex overrides, multiple teams

## Configuration Examples

### Minimal Configuration

```yaml
# tfpipboy.yaml
version: "1.0"

modules:
  app:
    path: "./modules/app"

pipelines:
  deploy:
    default_operation: apply
    stages:
      - name: "Deploy"
        modules: [app]
```

### Production Configuration

```yaml
# .tfpipboy/variables.yaml
variables:
  project: "ecommerce"
  organization: "acme"

environments:
  prod:
    auto_approve: false

# .tfpipboy/modules.yaml
modules:
  vpc:
    path: "./modules/vpc"
    depends_on: []
  
  app:
    path: "./modules/app"
    depends_on: [vpc]

# .tfpipboy/pipelines.yaml
pipelines:
  deploy-all:
    default_operation: apply
    stages:
      - name: "Network"
        modules: [vpc]
      - name: "Application"
        modules: [app]
```

## Key Capabilities

### 1. Flexible Operations

```yaml
pipelines:
  mixed-operations:
    stages:
      - name: "Different Ops"
        modules:
          - name: vpc
            operation: plan      # Just plan
          - name: app
            operation: apply     # Actually apply
          - name: old-db
            operation: destroy   # Destroy old
```

### 2. Conditional Deployment

```yaml
modules:
  monitoring:
    path: "./modules/monitoring"
    when: "${features.enable_monitoring}"  # Only if true
```

### 3. Environment Overrides

```yaml
modules:
  database:
    variables:
      instance_type: "small"
    
    environments:
      prod:
        variables:
          instance_type: "large"  # Override for prod
```

### 4. Module Dependencies

```yaml
modules:
  app:
    depends_on: [vpc, database, cache]  # Explicit ordering
```

### 5. Parallel Execution

```yaml
pipelines:
  deploy:
    stages:
      - name: "Parallel Stage"
        modules: [eks, rds, redis]
        parallel: true  # Run concurrently
```

### 6. Hooks

```yaml
pipelines:
  deploy:
    before:
      - name: "Run tests"
        command: "npm test"
    after:
      - name: "Notify team"
        command: "slack-notify.sh"
```

## Variable Interpolation

### Syntax

```yaml
# Simple reference
"${variable_name}"

# Nested path
"${azure.location}"

# Module output
"${modules.vpc.outputs.vpc_id}"

# Template
"${templates.resource_name}"

# With default
"${variable_name | default_value}"
```

### Examples

```yaml
variables:
  org: "acme"
  project: "shop"
  env: "prod"
  
  # Interpolated
  prefix: "${org}-${project}-${env}"
  vpc_name: "${prefix}-vpc"
  
  # Module output reference
  app_config:
    db_host: "${modules.rds.outputs.endpoint}"
    cache_host: "${modules.redis.outputs.endpoint}"
```

## Validation Rules

### Required Fields
- ✅ `version` in all config files
- ✅ `modules[].path` must exist
- ✅ `pipelines[].stages[]` must have modules or groups

### Constraints
- ✅ Module names must be unique
- ✅ No circular dependencies
- ✅ `depends_on` must reference existing modules
- ✅ Valid operations: plan, apply, destroy, refresh, validate, init

### Warnings
- ⚠️ Unused modules
- ⚠️ Undefined variable references
- ⚠️ Missing outputs referenced by other modules

## Migration Strategy

### Phase 1: Start Simple
```bash
# Create single file
cat > tfpipboy.yaml <<EOF
version: "1.0"
modules:
  app:
    path: "./modules/app"
pipelines:
  deploy:
    default_operation: apply
    stages:
      - name: "Deploy"
        modules: [app]
EOF

# Test
tfpipboy run deploy --env dev
```

### Phase 2: Split Configuration
```bash
# As project grows, split files
mkdir .tfpipboy
mv tfpipboy.yaml .tfpipboy/

# Split into layers
tfpipboy config split  # (future command)
```

### Phase 3: Add Environments
```bash
# Add environment-specific configs
mkdir -p .tfpipboy/environments/{dev,prod}

# Create overrides
cat > .tfpipboy/environments/prod/variables.yaml <<EOF
variables:
  instance_size: "large"
  auto_approve: false
EOF
```

## CLI Usage

```bash
# Run pipeline
tfpipboy run deploy-all

# Specific environment
tfpipboy run deploy-all --env prod

# Override operation
tfpipboy run deploy-all --operation plan

# Override variables
tfpipboy run deploy-all --var instance_size=large

# Specific modules only
tfpipboy run deploy-all --modules vpc,app

# Dry-run
tfpipboy run deploy-all --dry-run

# Validate config
tfpipboy config validate

# Show resolved config
tfpipboy config show --env prod
```

## Trade-offs Analysis

### Complexity vs Flexibility

| Approach | Complexity | Flexibility | Best For |
|----------|-----------|-------------|----------|
| Single file | Low | Low | Prototypes |
| Multi-file | Medium | High | Production |
| Full hierarchy | High | Very High | Enterprise |

### DRY vs Explicit

**DRY (Variables Layer)**
- ✅ No duplication
- ✅ Single source of truth
- ❌ Indirection

**Explicit (Inline)**
- ✅ Clear and direct
- ❌ Duplication
- ❌ Hard to change

**Decision**: DRY by default, inline for edge cases

## Implementation Priority

### Sprint 1 (Weeks 1-2)
- [ ] Define YAML schemas
- [ ] Implement parser for all three layers
- [ ] Variable interpolation engine
- [ ] Basic validation

### Sprint 2 (Weeks 3-4)
- [ ] Dependency graph builder
- [ ] Module group resolution
- [ ] Environment override logic
- [ ] Configuration loader

### Sprint 3 (Weeks 5-6)
- [ ] Pipeline executor
- [ ] Operation override handling
- [ ] Conditional deployment
- [ ] Hook execution

## Open Questions

### 1. Single vs Multi-File Default?
**Options**:
- A) Start with single file, migrate to multi-file
- B) Always use multi-file

**Recommendation**: Option A (progressive complexity)

### 2. HCL Support?
**Options**:
- A) YAML only
- B) Support both YAML and HCL

**Recommendation**: YAML for MVP, HCL in v2.0

### 3. Remote Configuration?
**Options**:
- A) Local files only
- B) Support remote (Git, S3, HTTP)

**Recommendation**: Local for MVP, remote in v2.0

### 4. Configuration Inheritance?
**Options**:
- A) Simple overrides only
- B) Full inheritance with extends

**Recommendation**: Start with overrides, add extends if needed

## Next Steps

1. **Review and approve** this configuration design
2. **Create JSON schemas** for IDE autocomplete
3. **Implement YAML parser** (US-102)
4. **Build validator** with helpful error messages
5. **Create example configurations** for testing
6. **Write user documentation**

## Related Documents

- [CONFIG-FORMAT-DESIGN.md](./CONFIG-FORMAT-DESIGN.md) - Complete specification
- [DEVELOPMENT-PLAN.md](./DEVELOPMENT-PLAN.md) - Implementation roadmap
- [examples/](../examples/) - Example configurations

---

**Status**: ✅ Ready for review  
**Blockers**: None  
**Decision needed**: Approve configuration format before implementation
