# Configuration Format Design Summary

**Date:** 2025-10-24  
**Status:** Complete  
**Based On:** Real Azure Landing Zone deployment (`examples/azure-landing-zone/exp-alz`)

## Objective

Design a comprehensive, DRY (Don't Repeat Yourself) configuration format for tfpipboy that:
1. Eliminates configuration duplication
2. Provides clear separation of concerns
3. Supports module instances and dependencies
4. Enables multi-environment deployments
5. Allows orchestrated deployment workflows

## Solution: 5-File Configuration Format

### File Structure

```
.tfpipboy/
├── variables.yaml        # Global variables and environment settings
├── backends.yaml         # Backend templates (✅ Already implemented)
├── landing-zones.yaml    # Landing zone definitions (Azure-specific)
├── modules.yaml          # Module catalog with dependencies
└── pipelines.yaml        # Deployment workflows
```

## Key Achievements

### 1. Backend Configuration Reduction: 58%

**Before (HCL):**
```hcl
# 11 modules × 8 lines each = 88 lines
tenant_id            = "xxxxxxxx-xxx"
subscription_id      = "xxxxxxxx-001"
resource_group_name  = "rg-exp-bootstrap-tfbackend"
storage_account_name = "stexptfbackendbootstrap"
container_name       = "platform"
key                  = "exp-core.tfstate"
use_azuread_auth     = true
```

**After (YAML with templates):**
```yaml
# 4 templates + module references = 40 lines total
backend_templates:
  bootstrap:
    subscription_id: "${bootstrap.subscription_id}"
    # ... (defined once)

modules:
  core:
    backend:
      template: "bootstrap"
      key: "${root_id}-core.tfstate"
```

**Result:** 88 → 40 lines (58% reduction)

### 2. Module Instance Pattern

**Challenge:** Deploy same module multiple times with different configs

**Solution:**
```yaml
vending:
  description: "Subscription vending"
  instances:
    connectivity:
      backend:
        key: "exp-vending-conn.tfstate"
      variables:
        landing_zone_name: "connectivity"
    
    management:
      backend:
        key: "exp-vending-mgmt.tfstate"
      variables:
        landing_zone_name: "management"
```

**Modules with instances:**
- `vending` (2 instances: connectivity, management)
- `baseline` (2 instances: connectivity, management)
- `firewall_rules` (1 instance per landing zone)
- `dns` (1 instance per landing zone)

### 3. Landing Zone Abstraction

**First-class landing zone definitions:**
```yaml
landing_zones:
  connectivity:
    subscriptions:
      exp: "xxxxxxxx-xxxx-4xxx-xxxx-000000000004"
    backend:
      storage_account: "st${org_name}conn${environment}${location_slug}base"
    network:
      address_spaces:
        exp: ["10.200.208.0/23"]
    vpn_config:
      exp:
        azure_vgw_sku: "VpnGw1"
```

**Benefits:**
- Centralized landing zone configuration
- Reference via `${landing_zones.connectivity.subscriptions.exp}`
- Backend template uses landing zone definitions
- Clear separation of landing zone concerns

### 4. Explicit Dependency Management

**Dependency graph:**
```
seed
  ↓
core
  ↓
├─→ vending.connectivity ──→ baseline.connectivity ──→ connectivity
│                                                          ↓
│                                                    firewall_rules.connectivity
│                                                          ↓
│                                                    dns.connectivity
│
└─→ vending.management ──→ baseline.management ──→ management
                              (depends on baseline.connectivity)
```

**Expressed in config:**
```yaml
modules:
  baseline:
    instances:
      management:
        depends_on: 
          - vending.management
          - baseline.connectivity  # Peering dependency
```

### 5. Multi-Stage Pipeline Orchestration

**Example: Complete platform deployment**
```yaml
pipelines:
  deploy-platform-complete:
    stages:
      # Stage 1: Bootstrap (sequential)
      - name: "Bootstrap Infrastructure"
        modules: [seed, core]
        parallel: false
      
      # Stage 2: Vending (parallel)
      - name: "Subscription Provisioning"
        modules: [vending.connectivity, vending.management]
        parallel: true
      
      # Stage 3: Baseline (sequential)
      - name: "Baseline Infrastructure"
        modules: [baseline.connectivity, baseline.management]
        parallel: false
      
      # Stage 4-6: Landing zones...
```

**Features:**
- Multi-stage execution
- Parallel execution where possible
- Lifecycle hooks (before/after)
- Confirmation gates for production
- Environment-specific overrides

## Real-World Module Catalog

### Bootstrap Tier (Shared Backend)

| Module | Instances | Backend | Purpose |
|--------|-----------|---------|---------|
| seed | 1 | bootstrap | Create backend infrastructure |
| core | 1 | bootstrap | Management groups, policies |
| vending | 2 | bootstrap | Subscription provisioning |
| baseline | 2 | bootstrap | Resource groups, storage, network baseline |

**State storage:** Bootstrap subscription (`xxxxxxxx-001`) → `platform` container

### Landing Zone Tier (Per-LZ Backend)

| Module | Instances | Backend | Purpose |
|--------|-----------|---------|---------|
| connectivity | 1 | connectivity LZ | Hub VPN, Firewall, Bastion |
| firewall_rules | 1 | connectivity LZ | Firewall policies |
| dns | 1 | connectivity LZ | Private DNS zones |
| management | 1 | management LZ | Log Analytics, Automation |

**State storage:** Landing zone's own subscription → `tfstate` container

## Variable Interpolation System

### Variable Resolution Order

1. **Environment overrides** (`environments.exp.*`)
2. **Global variables** (`variables.*`)
3. **Landing zones** (`landing_zones.*`)
4. **Special variables** (`${environment}`, `${landing_zone}`)

### Examples

```yaml
# variables.yaml
variables:
  org_name: "explz"
  location_slug: "gc"

# Interpolation in backends.yaml
backend:
  storage_account: "st${org_name}conn${environment}${location_slug}base"
  # exp: stexplzconnexpgcbase
  # dev: stexplzconndevgcbase

# Interpolation in modules.yaml
modules:
  core:
    variables:
      root_id: "${root_id}"
      location: "${location}"

# Nested access
${landing_zones.connectivity.subscriptions.exp}
${landing_zones.connectivity.vpn_config.exp.azure_vgw_sku}
```

## Environment Strategy

### Environment-Specific Overrides

```yaml
environments:
  exp:
    bootstrap:
      subscription_id: "xxxxxxxx-001"  # Dev bootstrap
    firewall_config:
      sku_tier: "Standard"
    terraform:
      auto_approve: false
  
  prod:
    bootstrap:
      subscription_id: "xxxxxxxx-005"  # Prod bootstrap
    firewall_config:
      sku_tier: "Premium"
    terraform:
      auto_approve: false
    require_confirmation: true
```

### Usage

```bash
# Deploy to exp environment
tfpipboy pipeline deploy-platform-complete --env exp

# Deploy to prod environment (requires confirmation)
tfpipboy pipeline deploy-platform-complete --env prod
```

## Configuration Best Practices

### 1. Use Backend Templates

✅ **Good:**
```yaml
backend_templates:
  bootstrap:
    subscription_id: "${bootstrap.subscription_id}"

modules:
  core:
    backend:
      template: "bootstrap"
      key: "exp-core.tfstate"
```

❌ **Bad:**
```yaml
modules:
  core:
    backend:
      subscription_id: "xxxxxxxx-001"  # Duplicated
      resource_group: "rg-..."          # Duplicated
```

### 2. Use Variables for Shared Values

✅ **Good:**
```yaml
variables:
  location: "germanywestcentral"

modules:
  core:
    variables:
      location: "${location}"
```

❌ **Bad:**
```yaml
modules:
  core:
    variables:
      location: "germanywestcentral"  # Hardcoded
```

### 3. Explicit Dependencies

✅ **Good:**
```yaml
baseline:
  instances:
    management:
      depends_on: [vending.management, baseline.connectivity]
```

❌ **Bad:**
```yaml
baseline:
  instances:
    management:
      depends_on: []  # Missing dependencies
```

### 4. Use Module Groups

✅ **Good:**
```yaml
groups:
  bootstrap:
    - seed
    - core

pipelines:
  deploy:
    stages:
      - groups: [bootstrap]
```

❌ **Bad:**
```yaml
pipelines:
  deploy:
    stages:
      - modules: [seed, core, vending.conn, vending.mgmt, ...]  # Long list
```

## Migration Path

### Phase 1: Variables and Backends
1. ✅ Extract variables to `variables.yaml`
2. ✅ Create backend templates in `backends.yaml`
3. Test with existing modules

### Phase 2: Modules and Dependencies
1. Create `modules.yaml` with module catalog
2. Define dependencies explicitly
3. Add module instances

### Phase 3: Landing Zones (Azure-specific)
1. Create `landing-zones.yaml`
2. Migrate landing zone configs
3. Update backend templates to use landing zones

### Phase 4: Pipelines
1. Create `pipelines.yaml`
2. Define deployment workflows
3. Add lifecycle hooks

## Comparison with Alternatives

### vs. Terragrunt

| Feature | tfpipboy | Terragrunt |
|---------|-----------|------------|
| Configuration format | YAML (5 files) | HCL (directory hierarchy) |
| Backend config | Templates + interpolation | generate blocks |
| Dependencies | Explicit in modules.yaml | Implicit via dependency blocks |
| Orchestration | Pipeline stages | terraform-all |
| Module instances | Native support | Duplicate directories |

**Advantages:**
- More intuitive YAML format
- Centralized configuration
- Native module instance support
- More flexible pipeline orchestration

### vs. Terraform Cloud/Enterprise

| Feature | tfpipboy | TFC/TFE |
|---------|-----------|---------|
| Hosting | Self-hosted | SaaS/Enterprise |
| Configuration | Local YAML files | Web UI + VCS |
| State storage | Azure Storage, S3, etc. | TFC managed |
| Cost | Free | Paid |
| Customization | Full control | Limited |

**Advantages:**
- No vendor lock-in
- Full customization
- Works with existing backends
- Free and open-source

## Next Steps

### Implementation Priority

1. **Parser** (Week 1-2)
   - YAML parsing for all 5 files
   - Variable interpolation engine
   - Schema validation

2. **Backend Management** (Week 2-3)
   - Template expansion
   - Backend configuration generation
   - Module instance handling

3. **Dependency Resolution** (Week 3-4)
   - Dependency graph construction
   - Topological sorting
   - Cycle detection

4. **Pipeline Execution** (Week 4-6)
   - Stage execution
   - Parallel execution
   - Lifecycle hooks
   - Error handling

5. **CLI Interface** (Week 6-8)
   - Commands: module, group, pipeline
   - Environment selection
   - Interactive confirmation
   - Progress reporting

### Documentation

- [ ] JSON Schema for each YAML file
- [ ] Configuration reference guide
- [ ] Migration guide from raw Terraform
- [ ] Best practices guide
- [ ] Example configurations

## Conclusion

The 5-file configuration format provides:

1. **58% reduction** in backend configuration duplication
2. **Clear separation** of concerns across 5 files
3. **Module instance pattern** for repeated deployments
4. **Landing zone abstraction** for Azure Landing Zones
5. **Multi-stage pipelines** with dependencies and parallelization
6. **Environment management** with overrides
7. **Variable interpolation** for DRY configuration

**Status:** Design complete, ready for implementation

**Documentation:**
- `design/CONFIG-FORMAT-SPEC.md` - Detailed specification
- `design/CONFIG-FORMAT-REAL-EXAMPLE.md` - Real-world example
- `examples/azure-landing-zone/exp-alz/.tfpipboy/backends.yaml` - Implemented backend config
- `examples/azure-landing-zone/exp-alz/.tfpipboy/variables-complete.yaml` - Complete variables example

**Next:** Begin implementation of YAML parser and variable interpolation engine
