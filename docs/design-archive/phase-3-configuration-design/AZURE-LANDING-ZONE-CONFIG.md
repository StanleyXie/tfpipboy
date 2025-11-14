# Azure Landing Zone Configuration Analysis & Design
**Document No.**: AZ-LZ-001  
**Date**: 2025-10-19  
**Version**: 1.0

## Executive Summary

This document analyzes the existing Azure Landing Zone (ALZ) configuration from `.terraform-repo/source/root_modules/.env` and designs a tfpipboy configuration format that addresses the real-world complexity while maintaining simplicity and DRY principles.

---

## Analysis of Existing Configuration

### Current File Structure

```
.env/
├── backend.tfvars              # Backend configs per module
├── global.tfvars               # Global Azure settings
├── deployment.tfvars           # Module deployment definitions
├── core.tfvars                 # Core module variables
└── orchestrate/
    ├── pipeline.json           # Pipeline definition (JSON)
    ├── modules.tfvars          # Module catalog
    ├── deployment.tfvars       # Deployment configs
    ├── variables.tfvars        # Azure tenant variables
    └── backend-configs.tfvars  # Backend configurations
```

### Key Patterns Identified

#### 1. **Module Reuse Pattern**
Same module deployed multiple times with different configs:

```hcl
# vending module used twice
vending_conn = {
  module_name = "vending"
  module_path = "./plz/bootstrap/vending"
  deploy_config = {
    landing_zone_name = "connectivity"
  }
}

vending_mgmt = {
  module_name = "vending"
  module_path = "./plz/bootstrap/vending"
  deploy_config = {
    landing_zone_name = "management"
  }
}
```

**Challenge**: Need to support module instances, not just module names.

#### 2. **Complex Dependency Chain**

```
core → vending_conn → baseline_conn → connectivity → firewall_rules_conn → dns_conn
    ↓
    → vending_mgmt → baseline_mgmt → management
                  ↗ (also depends on connectivity)
```

**Challenge**: Complex multi-branch dependencies.

#### 3. **Per-Module Backend Configuration**

Each module has its own backend with variable interpolation:

```hcl
backend_config = {
  tenant_id            = "$TF_VAR_tf_backend_tenant_id"
  subscription_id      = "$TF_VAR_tf_backend_subscription_id_bootstrap"
  resource_group_name  = "$TF_VAR_tf_backend_resource_group_name_bootstrap"
  storage_account_name = "$TF_VAR_tf_backend_storage_account_name_bootstrap"
  container_name       = "$TF_VAR_tf_backend_container_name_bootstrap"
  key                  = "$TF_VAR_tf_backend_key_core"
  use_azuread_auth     = true
}
```

**Challenge**: Verbose backend config with many variables.

#### 4. **Deploy Config (Module-Specific Variables)**

Each module instance has deployment-specific variables:

```hcl
deploy_config = {
  landing_zone_name                         = "connectivity"
  environment                               = "dev"
  org_name                                  = "explz"
  tf_backend_subscription_id_bootstrap      = "xxx"
  tf_backend_resource_group_name_bootstrap  = "yyy"
  tf_backend_storage_account_name_bootstrap = "zzz"
  # ... many more
}
```

**Challenge**: Mix of business logic and infrastructure references.

#### 5. **Variable Interpolation in Terraform Format**

Uses `$TF_VAR_` and `$TF_SECRET_` prefixes:

```hcl
tenant_id = "$TF_VAR_tf_backend_tenant_id"
```

**Challenge**: Need to support environment variable expansion.

#### 6. **Landing Zone Pattern**

Two main landing zones with different configurations:
- **Connectivity**: Hub networking infrastructure
- **Management**: Management services

**Challenge**: Landing zone as a concept needs representation.

---

## Identified Requirements

### Must-Have Features

1. ✅ **Module Instances**: Deploy same module multiple times with different names
2. ✅ **Complex Dependencies**: Multi-branch dependency graphs
3. ✅ **Backend Templates**: Reusable backend configurations with variable substitution
4. ✅ **Environment Variables**: Support `$TF_VAR_*` and `$TF_SECRET_*` expansion
5. ✅ **Landing Zone Abstraction**: First-class support for landing zone concept
6. ✅ **Module Variants**: Same module path, different configurations

### Nice-to-Have Features

7. ⭐ **Backend Inheritance**: Common backend config with per-module overrides
8. ⭐ **Variable Layering**: Global → Landing Zone → Module → Instance
9. ⭐ **State Dependencies**: Reference outputs from other modules via state
10. ⭐ **Naming Conventions**: Automatic naming based on patterns

---

## Proposed tfpipboy Configuration Design

### File Structure

```
.tfpipboy/
├── variables.yaml              # Global variables
├── backends.yaml               # Backend templates
├── landing-zones.yaml          # Landing zone definitions
├── modules.yaml                # Module catalog
└── pipelines.yaml              # Deployment workflows
```

### Layer 1: Variables (`variables.yaml`)

```yaml
version: "1.0"

# Global Azure configuration
variables:
  # Project metadata
  project: "atlz-platform"
  root_id: "exp"
  root_name: "ExperimentDev-Stanley"
  environment: "dev"
  
  # Azure tenant
  tenant_id: "c3310862-ab75-4bc6-b8b6-26a14539537c"
  
  # Location settings
  location: "germanywestcentral"
  location_slug: "gc"
  
  # Organization
  org_name: "explz"
  github_org_name: "WITT-AZURE-PLATFORM"
  
  # Billing
  billing_account_id: "bca92218-bdfe-407e-b2c0-02ef02b3b6f8:333013a0-57ae-4229-82b3-b3cd70239eb2_2019-05-31"
  billing_profile_id: "XU5N-S6UC-BG7-PGB"
  
  # Bootstrap subscription (for state storage)
  bootstrap:
    subscription_id: "4b05673b-62ce-4723-99fa-c1030624561e"
    resource_group: "rg-exp-bootstrap-tfbackend"
    storage_account: "stexptfbackendbootstrap"
  
  # Naming templates
  naming:
    resource_group: "rg-${org_name}-${landing_zone}-${environment}-${location_slug}-${suffix}"
    storage_account: "st${org_name}${landing_zone}${environment}${location_slug}${suffix}"
    vnet: "vnet-${org_name}-${landing_zone}-${environment}-${location_slug}-${suffix}"

# Environment-specific overrides
environments:
  dev:
    environment: "dev"
  
  prod:
    environment: "prod"
    org_name: "prodlz"
```

### Layer 2: Backend Templates (`backends.yaml`)

```yaml
version: "1.0"

# Backend templates for DRY
backend_templates:
  # Bootstrap backend (for core and vending)
  bootstrap:
    type: "azurerm"
    tenant_id: "${tenant_id}"
    subscription_id: "${bootstrap.subscription_id}"
    resource_group_name: "${bootstrap.resource_group}"
    storage_account_name: "${bootstrap.storage_account}"
    container_name: "platform"
    use_azuread_auth: true
    # Key is per-module
  
  # Landing zone backend (for deployed resources)
  landing_zone:
    type: "azurerm"
    tenant_id: "${tenant_id}"
    subscription_id: "${landing_zones.${landing_zone}.subscription_id}"
    resource_group_name: "${naming.resource_group}"
    storage_account_name: "${naming.storage_account}"
    container_name: "tfstate"
    use_azuread_auth: true
```

### Layer 3: Landing Zones (`landing-zones.yaml`)

```yaml
version: "1.0"

# Landing zone definitions
landing_zones:
  connectivity:
    description: "Hub networking infrastructure"
    org_name: "explz"
    abbreviation: "conn"
    archetype: "connectivity"
    workload_type: "Production"
    
    # Network configuration
    network:
      enable_resources: true
      enable_peering: false
      address_spaces:
        dev: ["10.200.208.0/23"]
    
    # Subscription per environment
    subscriptions:
      dev: "e786a45d-030c-4dfb-a827-fd18444ed496"
    
    # Storage settings
    storage:
      replication_type: "LRS"
      access_tier: "cool"
      public_access: false
    
    # Tags
    tags:
      BEE360_ID: "SE1070"
      archetype: "connectivity"
  
  management:
    description: "Management and monitoring services"
    org_name: "explz"
    abbreviation: "mgmt"
    archetype: "management"
    workload_type: "Production"
    
    # Network configuration
    network:
      enable_resources: true
      enable_peering: true
      enable_default_subnet: true
      address_spaces:
        dev: ["10.200.212.0/25"]
      hub_vnet_id: "/subscriptions/e786a45d-030c-4dfb-a827-fd18444ed496/resourceGroups/rg-explz-conn-dev-gc-hub/providers/Microsoft.Network/virtualNetworks/vnet-explz-conn-dev-gc-hub"
    
    # Subscription per environment
    subscriptions:
      dev: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    
    # Storage settings
    storage:
      replication_type: "LRS"
      access_tier: "cool"
      public_access: false
    
    # Tags
    tags:
      BEE360_ID: "SE1070"
      archetype: "management"
```

### Layer 4: Modules (`modules.yaml`)

```yaml
version: "1.0"

modules:
  # ============================================================
  # CORE MODULE - Management groups and policies
  # ============================================================
  
  core:
    description: "Core Azure management groups and policies"
    path: "./plz/core"
    depends_on: []
    
    backend:
      template: "bootstrap"
      key: "${root_id}-core.tfstate"
    
    variables:
      root_id: "${root_id}"
      root_name: "${root_name}"
      billing_account_id: "${billing_account_id}"
      billing_profile_id: "${billing_profile_id}"
  
  # ============================================================
  # VENDING MODULE - Subscription provisioning
  # ============================================================
  
  vending:
    description: "Subscription vending machine"
    path: "./plz/bootstrap/vending"
    
    # This is a template - instances defined below
    instances:
      connectivity:
        description: "Vend connectivity subscription"
        depends_on: [core]
        
        backend:
          template: "bootstrap"
          key: "${root_id}-vending-conn.tfstate"
        
        variables:
          landing_zone_name: "connectivity"
          landing_zone_config: "${landing_zones.connectivity}"
      
      management:
        description: "Vend management subscription"
        depends_on: [core]
        
        backend:
          template: "bootstrap"
          key: "${root_id}-vending-mgmt.tfstate"
        
        variables:
          landing_zone_name: "management"
          landing_zone_config: "${landing_zones.management}"
  
  # ============================================================
  # BASELINE MODULE - Base infrastructure per landing zone
  # ============================================================
  
  baseline:
    description: "Baseline infrastructure for landing zones"
    path: "./plz/bootstrap/baseline"
    
    instances:
      connectivity:
        description: "Baseline for connectivity landing zone"
        depends_on: [vending.connectivity]
        
        backend:
          template: "bootstrap"
          key: "${root_id}-baseline-conn.tfstate"
        
        variables:
          landing_zone_name: "connectivity"
          environment: "${environment}"
          # Reference vending output via state data source
          subscription_id: "${modules.vending.connectivity.outputs.subscription_id}"
          
          # Backend reference for state data source
          tf_backend_key_vending: "${root_id}-vending-conn.tfstate"
      
      management:
        description: "Baseline for management landing zone"
        depends_on: [vending.management, connectivity]
        
        backend:
          template: "bootstrap"
          key: "${root_id}-baseline-mgmt.tfstate"
        
        variables:
          landing_zone_name: "management"
          environment: "${environment}"
          subscription_id: "${modules.vending.management.outputs.subscription_id}"
          hub_vnet_id: "${landing_zones.connectivity.network.hub_vnet_id}"
          
          # Backend reference for state data source
          tf_backend_key_vending: "${root_id}-vending-mgmt.tfstate"
  
  # ============================================================
  # CONNECTIVITY MODULE - Hub networking
  # ============================================================
  
  connectivity:
    description: "Hub network infrastructure"
    path: "./plz/connectivity"
    depends_on: [baseline.connectivity]
    
    backend:
      template: "landing_zone"
      landing_zone: "connectivity"
      key: "${root_id}-connectivity.tfstate"
    
    variables:
      landing_zone_name: "connectivity"
      environment: "${environment}"
      org_name: "${org_name}"
      location: "${location}"
      
      # Bootstrap backend references for state data sources
      tf_backend_subscription_id_bootstrap: "${bootstrap.subscription_id}"
      tf_backend_resource_group_name_bootstrap: "${bootstrap.resource_group}"
      tf_backend_storage_account_name_bootstrap: "${bootstrap.storage_account}"
      tf_backend_container_name_bootstrap: "platform"
      tf_backend_key_vending_bootstrap: "${root_id}-vending-conn.tfstate"
      tf_backend_key_baseline_bootstrap: "${root_id}-baseline-conn.tfstate"
  
  # ============================================================
  # FIREWALL RULES - Connectivity firewall configuration
  # ============================================================
  
  firewall_rules:
    description: "Azure Firewall rules for hub"
    path: "./plz/connectivity/firewall_rules"
    
    instances:
      connectivity:
        description: "Firewall rules for connectivity hub"
        depends_on: [connectivity]
        
        backend:
          template: "landing_zone"
          landing_zone: "connectivity"
          key: "${root_id}-firewall-rules-conn.tfstate"
        
        variables:
          landing_zone_name: "connectivity"
          environment: "${environment}"
          org_name: "${org_name}"
          
          # Reference connectivity state
          tf_backend_subscription_id_connectivity: "${landing_zones.connectivity.subscriptions.${environment}}"
          tf_backend_resource_group_name_connectivity: "rg-${org_name}-conn-${environment}-${location_slug}-base"
          tf_backend_storage_account_name_connectivity: "st${org_name}conn${environment}${location_slug}base"
          tf_backend_container_name_connectivity: "tfstate"
          tf_backend_key_connectivity: "${root_id}-connectivity.tfstate"
  
  # ============================================================
  # DNS - Connectivity DNS configuration
  # ============================================================
  
  dns:
    description: "DNS zones and records"
    path: "./plz/connectivity/dns"
    
    instances:
      connectivity:
        description: "DNS for connectivity landing zone"
        depends_on: [firewall_rules.connectivity]
        
        backend:
          template: "landing_zone"
          landing_zone: "connectivity"
          key: "${root_id}-dns-conn.tfstate"
        
        variables:
          landing_zone_name: "connectivity"
          environment: "${environment}"
          org_name: "${org_name}"
          
          # Reference both connectivity and firewall states
          tf_backend_subscription_id_connectivity: "${landing_zones.connectivity.subscriptions.${environment}}"
          tf_backend_resource_group_name_connectivity: "rg-${org_name}-conn-${environment}-${location_slug}-base"
          tf_backend_storage_account_name_connectivity: "st${org_name}conn${environment}${location_slug}base"
          tf_backend_container_name_connectivity: "tfstate"
          tf_backend_key_connectivity: "${root_id}-connectivity.tfstate"
          tf_backend_key_connectivity_firewall_rules: "${root_id}-firewall-rules-conn.tfstate"
  
  # ============================================================
  # MANAGEMENT MODULE - Management services
  # ============================================================
  
  management:
    description: "Management and monitoring services"
    path: "./plz/management"
    depends_on: [baseline.management]
    
    backend:
      template: "landing_zone"
      landing_zone: "management"
      key: "${root_id}-management.tfstate"
    
    variables:
      landing_zone_name: "management"
      environment: "${environment}"
      org_name: "${org_name}"
      project: "${project}"
      
      # Bootstrap backend references
      tf_backend_subscription_id_bootstrap: "${bootstrap.subscription_id}"
      tf_backend_resource_group_name_bootstrap: "${bootstrap.resource_group}"
      tf_backend_storage_account_name_bootstrap: "${bootstrap.storage_account}"
      tf_backend_container_name_bootstrap: "platform"
      tf_backend_key_vending_bootstrap: "${root_id}-vending-mgmt.tfstate"
      tf_backend_key_baseline_bootstrap: "${root_id}-baseline-mgmt.tfstate"

# ============================================================
# MODULE GROUPS
# ============================================================

groups:
  bootstrap:
    - core
    - vending.connectivity
    - vending.management
  
  connectivity_stack:
    - baseline.connectivity
    - connectivity
    - firewall_rules.connectivity
    - dns.connectivity
  
  management_stack:
    - baseline.management
    - management
  
  all:
    - core
    - vending.connectivity
    - vending.management
    - baseline.connectivity
    - connectivity
    - firewall_rules.connectivity
    - dns.connectivity
    - baseline.management
    - management
```

### Layer 5: Pipelines (`pipelines.yaml`)

```yaml
version: "1.0"

pipelines:
  # ============================================================
  # PLANNING PIPELINES
  # ============================================================
  
  plan-all:
    description: "Plan all Azure Landing Zone infrastructure"
    default_operation: plan
    
    stages:
      - name: "Bootstrap"
        modules:
          - core
          - vending.connectivity
          - vending.management
      
      - name: "Connectivity Stack"
        modules:
          - baseline.connectivity
          - connectivity
          - firewall_rules.connectivity
          - dns.connectivity
      
      - name: "Management Stack"
        modules:
          - baseline.management
          - management
  
  # ============================================================
  # DEPLOYMENT PIPELINES
  # ============================================================
  
  deploy-bootstrap:
    description: "Deploy core and vending only"
    default_operation: apply
    
    stages:
      - name: "Core Platform"
        modules:
          - core
      
      - name: "Subscription Vending"
        modules:
          - vending.connectivity
          - vending.management
        parallel: true
    
    environments:
      prod:
        require_confirmation: true
        confirmation_message: "Deploy bootstrap to PROD? Type 'yes':"
  
  deploy-connectivity:
    description: "Deploy connectivity landing zone"
    default_operation: apply
    
    before:
      - name: "Validate Azure auth"
        command: "az account show"
    
    stages:
      - name: "Baseline"
        modules:
          - baseline.connectivity
      
      - name: "Hub Network"
        modules:
          - connectivity
      
      - name: "Firewall"
        modules:
          - firewall_rules.connectivity
      
      - name: "DNS"
        modules:
          - dns.connectivity
    
    after:
      - name: "Test connectivity"
        command: "scripts/test-hub-connectivity.sh"
  
  deploy-management:
    description: "Deploy management landing zone"
    default_operation: apply
    
    stages:
      - name: "Baseline"
        modules:
          - baseline.management
      
      - name: "Management Services"
        modules:
          - management
    
    after:
      - name: "Verify monitoring"
        command: "scripts/verify-monitoring.sh"
  
  deploy-all:
    description: "Deploy complete Azure Landing Zone"
    default_operation: apply
    
    settings:
      parallel_limit: 2
      timeout: 120m
      retry_failed: 1
    
    before:
      - name: "Pre-flight checks"
        command: "scripts/preflight-azure.sh"
      - name: "Validate Azure credentials"
        command: "az account show && gh auth status"
    
    stages:
      - name: "Core Bootstrap"
        modules:
          - core
      
      - name: "Subscription Provisioning"
        modules:
          - vending.connectivity
          - vending.management
        parallel: true
      
      - name: "Landing Zone Baselines"
        modules:
          - baseline.connectivity
          - baseline.management
        parallel: false  # Management depends on connectivity
      
      - name: "Connectivity Infrastructure"
        modules:
          - connectivity
          - firewall_rules.connectivity
          - dns.connectivity
      
      - name: "Management Infrastructure"
        modules:
          - management
    
    after:
      - name: "Smoke tests"
        command: "scripts/smoke-tests-alz.sh"
      - name: "Generate documentation"
        command: "scripts/generate-alz-docs.sh"
    
    environments:
      dev:
        settings:
          auto_approve: true
          parallel_limit: 3
      
      prod:
        settings:
          auto_approve: false
          require_approval: true
          parallel_limit: 1
        before:
          - name: "Create backup"
            command: "scripts/backup-alz-state.sh"
          - name: "Notify team"
            command: "scripts/notify-alz-deployment.sh"
  
  # ============================================================
  # MAINTENANCE PIPELINES
  # ============================================================
  
  update-firewall-rules:
    description: "Update firewall rules only"
    default_operation: apply
    
    stages:
      - name: "Update Rules"
        modules:
          - firewall_rules.connectivity
  
  update-dns:
    description: "Update DNS configuration only"
    default_operation: apply
    
    stages:
      - name: "Update DNS"
        modules:
          - dns.connectivity
  
  # ============================================================
  # DESTROY PIPELINES
  # ============================================================
  
  destroy-connectivity:
    description: "Destroy connectivity landing zone"
    default_operation: destroy
    
    require_confirmation: true
    confirmation_message: "⚠️  Destroy connectivity? Type 'destroy-connectivity':"
    
    stages:
      - name: "DNS"
        modules:
          - dns.connectivity
      - name: "Firewall"
        modules:
          - firewall_rules.connectivity
      - name: "Hub Network"
        modules:
          - connectivity
      - name: "Baseline"
        modules:
          - baseline.connectivity
  
  destroy-all:
    description: "Destroy complete Azure Landing Zone"
    default_operation: destroy
    
    require_confirmation: true
    confirmation_message: |
      ⚠️  ⚠️  ⚠️  DESTROY AZURE LANDING ZONE ⚠️  ⚠️  ⚠️
      
      This will destroy:
      - All management services
      - All connectivity infrastructure
      - All subscriptions may need manual cleanup
      
      Type 'destroy-alz-${environment}' to confirm:
    
    stages:
      - name: "Management"
        modules:
          - management
          - baseline.management
      
      - name: "Connectivity"
        modules:
          - dns.connectivity
          - firewall_rules.connectivity
          - connectivity
          - baseline.connectivity
      
      - name: "Bootstrap"
        modules:
          - vending.management
          - vending.connectivity
          - core
```

---

## Key Improvements Over Current Format

### 1. **Module Instances**

**Before** (current):
```hcl
vending_conn = { ... }
vending_mgmt = { ... }
```

**After** (tfpipboy):
```yaml
vending:
  instances:
    connectivity: { ... }
    management: { ... }
```

**Benefits**:
- Clear relationship between instances
- Cleaner naming
- Easier to add new instances

### 2. **Backend Templates**

**Before** (current):
```hcl
# Repeated for every module
backend_config = {
  tenant_id            = "$TF_VAR_tf_backend_tenant_id"
  subscription_id      = "$TF_VAR_tf_backend_subscription_id_bootstrap"
  resource_group_name  = "$TF_VAR_tf_backend_resource_group_name_bootstrap"
  storage_account_name = "$TF_VAR_tf_backend_storage_account_name_bootstrap"
  container_name       = "$TF_VAR_tf_backend_container_name_bootstrap"
  key                  = "$TF_VAR_tf_backend_key_core"
  use_azuread_auth     = true
}
```

**After** (tfpipboy):
```yaml
backend:
  template: "bootstrap"
  key: "${root_id}-core.tfstate"
```

**Benefits**:
- 90% reduction in config verbosity
- DRY principle applied
- Easier to maintain

### 3. **Landing Zone First-Class Support**

**Before** (current):
Landing zone scattered across multiple files

**After** (tfpipboy):
```yaml
landing_zones:
  connectivity:
    network: { ... }
    subscriptions: { ... }
    storage: { ... }
```

**Benefits**:
- Landing zone as cohesive concept
- All LZ config in one place
- Reusable across modules

### 4. **Dependency Notation**

**Before** (current):
```hcl
depends_on = ["core"]
depends_on = ["vending_mgmt", "connectivity"]
```

**After** (tfpipboy):
```yaml
depends_on: [core]
depends_on: [vending.management, connectivity]
```

**Benefits**:
- Consistent instance notation
- Clear module vs instance dependencies

### 5. **Variable Interpolation**

**Before** (current):
```hcl
"$TF_VAR_tf_backend_subscription_id_bootstrap"
```

**After** (tfpipboy):
```yaml
"${bootstrap.subscription_id}"
```

**Benefits**:
- Cleaner syntax
- Path-based access
- Environment variables still supported via `$TF_VAR_*`

---

## CLI Usage Examples

```bash
# Plan complete ALZ deployment
tfpipboy run plan-all --env dev

# Deploy bootstrap only
tfpipboy run deploy-bootstrap --env dev

# Deploy connectivity stack
tfpipboy run deploy-connectivity --env dev

# Deploy everything
tfpipboy run deploy-all --env prod

# Update firewall rules only
tfpipboy run update-firewall-rules --env dev

# Show resolved configuration
tfpipboy config show --env dev

# Validate configuration
tfpipboy config validate

# Show dependency graph
tfpipboy graph show deploy-all
```

---

## Migration Path from Current Config

### Step 1: Extract Variables
```bash
# Convert current tfvars to variables.yaml
tfpipboy migrate extract-variables \
  --from .env/global.tfvars \
  --from .env/orchestrate/variables.tfvars \
  --to .tfpipboy/variables.yaml
```

### Step 2: Define Backend Templates
```bash
# Analyze backend configs and create templates
tfpipboy migrate analyze-backends \
  --from .env/backend.tfvars \
  --to .tfpipboy/backends.yaml
```

### Step 3: Convert Module Definitions
```bash
# Convert deployment.tfvars to modules.yaml
tfpipboy migrate convert-modules \
  --from .env/orchestrate/modules.tfvars \
  --from .env/deployment.tfvars \
  --to .tfpipboy/modules.yaml
```

### Step 4: Create Pipelines
```bash
# Create basic pipelines
tfpipboy migrate create-pipelines \
  --modules .tfpipboy/modules.yaml \
  --to .tfpipboy/pipelines.yaml
```

### Step 5: Validate
```bash
# Validate migrated configuration
tfpipboy config validate
tfpipboy config compare \
  --old .env/ \
  --new .tfpipboy/
```

---

## Configuration Best Practices for Azure Landing Zones

### 1. Use Backend Templates
```yaml
backend_templates:
  bootstrap: { ... }  # For core/vending
  landing_zone: { ... }  # For LZ resources
```

### 2. Define Landing Zones Separately
```yaml
landing_zones:
  connectivity: { ... }
  management: { ... }
  workload: { ... }
```

### 3. Use Module Instances for Reuse
```yaml
vending:
  instances:
    connectivity: { ... }
    management: { ... }
```

### 4. Leverage Variable Interpolation
```yaml
variables:
  naming:
    pattern: "rg-${org}-${lz}-${env}-${location}"
```

### 5. Group Related Modules
```yaml
groups:
  connectivity_stack: [baseline.conn, connectivity, firewall, dns]
```

---

## Next Steps

1. ✅ **Review and approve** this Azure Landing Zone configuration design
2. 🔜 **Implement parser** with module instance support
3. 🔜 **Implement backend templates** with variable interpolation
4. 🔜 **Create migration tools** for existing ALZ configs
5. 🔜 **Build example** Azure Landing Zone project

---

## Appendix: Complete Example

See `examples/azure-landing-zone/` for complete working example.

---

**Document Status**: Draft v1.0  
**Needs Review**: Module instances, backend templates, landing zone abstraction  
**Blockers**: None  
**Next**: Implement parser with instance support
