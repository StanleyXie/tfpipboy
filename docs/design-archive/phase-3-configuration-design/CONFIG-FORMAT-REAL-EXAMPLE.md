# Real-World Configuration Example for Azure Landing Zone

This document demonstrates the complete tf-pipboy configuration format using real Azure Landing Zone deployment data from `examples/azure-landing-zone/exp-alz`.

## Overview

The configuration is split across 5 YAML files:

1. **variables.yaml** - Global variables and environment settings
2. **backends.yaml** - Backend templates (already exists, 58% reduction achieved)
3. **landing-zones.yaml** - Landing zone definitions
4. **modules.yaml** - Module catalog with dependencies
5. **pipelines.yaml** - Deployment workflows

## 1. variables.yaml

```yaml
version: "1.0"

variables:
  # Azure Tenant
  tenant_id: "c3310862-ab75-4bc6-b8b6-26a14539537c"
  
  # Project Metadata
  project: "atlz-platform"
  root_id: "exp"
  root_name: "ExperimentDev-Stanley"
  
  # GitHub Configuration
  github:
    enabled: true
    org_name: "WITT-AZURE-PLATFORM"
    repo_name: "atlz-platform"
  
  # Azure Region
  location: "germanywestcentral"
  location_slug: "gc"
  
  # Billing
  billing:
    account_id: "bca92218-bdfe-407e-b2c0-02ef02b3b6f8:333013a0-57ae-4229-82b3-b3cd70239eb2_2019-05-31"
    profile_id: "XU5N-S6UC-BG7-PGB"
    use_profile_scope: true
    role_definition_id: "/providers/Microsoft.Billing/billingAccounts/..."
  
  # Organization
  org_name: "explz"
  
  # Bootstrap Backend (exp environment)
  bootstrap:
    subscription_id: "4b05673b-62ce-4723-99fa-c1030624561e"
    resource_group: "rg-exp-bootstrap-tfbackend"
    storage_account: "stexptfbackendbootstrap"
    container_name: "platform"

environments:
  exp:
    environment: "exp"
    root_id: "exp"
    root_name: "ExperimentDev-Stanley"
    org_name: "explz"
    
    bootstrap:
      subscription_id: "4b05673b-62ce-4723-99fa-c1030624561e"
      resource_group: "rg-exp-bootstrap-tfbackend"
      storage_account: "stexptfbackendbootstrap"
      container_name: "platform"
    
    terraform:
      auto_approve: false
      parallelism: 10
    
    tags:
      environment: "exp"
      cost_center: "platform-engineering"
  
  main:
    environment: "main"
    root_id: "atlz"
    root_name: "AzureLandingZone-Main"
    org_name: "atlz"
    
    bootstrap:
      subscription_id: "9fd0d299-f872-47d2-b400-91d27060b5d1"
      resource_group: "rg-atlz-bs-main-gc-base"
      storage_account: "statlzbsmaingcbase"
      container_name: "tfstate"
    
    terraform:
      auto_approve: false
      parallelism: 5
    
    tags:
      environment: "main"
      compliance: "required"
```

## 2. backends.yaml

Already exists at `examples/azure-landing-zone/exp-alz/.tfpipboy/backends.yaml`

Key achievements:
- **58% reduction** from original HCL backend configs
- 4 reusable templates (seed, bootstrap, bootstrap_main, landing_zone)
- Support for module instances
- Variable interpolation

## 3. landing-zones.yaml

```yaml
version: "1.0"

landing_zones:
  connectivity:
    description: "Hub networking infrastructure"
    org_name: "explz"
    abbreviation: "conn"
    archetype: "connectivity"
    workload_type: "Production"
    repo_name: "atlz-platform"
    
    # Subscriptions per environment
    subscriptions:
      exp: "e786a45d-030c-4dfb-a827-fd18444ed496"
      dev: "e786a45d-030c-4dfb-a827-fd18444ed496"
      main: "f4c8307d-44ec-415d-81fe-6767e3679d29"
      prod: "f4c8307d-44ec-415d-81fe-6767e3679d29"
    
    # Backend for connectivity modules
    backend:
      resource_group: "rg-${org_name}-conn-${environment}-${location_slug}-base"
      storage_account: "st${org_name}conn${environment}${location_slug}base"
      # exp: stexplzconnexpgcbase
      # dev: stexplzconndevgcbase
    
    # Network configuration
    network:
      enable_resources: false  # Baseline creates minimal network
      enable_peering: false    # Hub doesn't peer to itself
      address_spaces:
        exp: ["10.200.208.0/23"]
        dev: ["10.200.208.0/23"]
        main: ["10.180.208.0/23"]
        prod: ["10.180.208.0/23"]
    
    # VPN configuration per environment
    vpn_config:
      exp:
        onprem_gateway_address: "83.135.59.10"
        azure_bgp_peering_address: "169.254.21.2"
        onprem_bgp_peering_address: "169.254.21.1"
        azure_vgw_sku: "VpnGw1"
        azure_asn: 65515
        onprem_asn: 65001
      prod:
        azure_vgw_sku: "VpnGw2"  # Larger for production
    
    # Firewall configuration
    firewall_config:
      exp:
        sku_tier: "Standard"
      prod:
        sku_tier: "Premium"
    
    # Public IP prefix
    public_ip_prefix:
      vpn: "31"
    
    tags:
      BEE360_ID: "SE1070"
      archetype: "connectivity"
      workload_type: "Production"
  
  management:
    description: "Management services and monitoring"
    org_name: "explz"
    abbreviation: "mgmt"
    archetype: "management"
    workload_type: "Production"
    repo_name: "atlz-platform"
    
    subscriptions:
      exp: "57ddcee2-0c0c-4b3f-90be-b3b222fd8d9a"
      dev: "57ddcee2-0c0c-4b3f-90be-b3b222fd8d9a"
      main: "4c89f399-7da6-4a53-baf5-3bffe373d6c0"
      prod: "4c89f399-7da6-4a53-baf5-3bffe373d6c0"
    
    backend:
      resource_group: "rg-${org_name}-mgmt-${environment}-${location_slug}-base"
      storage_account: "st${org_name}mgmt${environment}${location_slug}base"
    
    network:
      enable_resources: true      # Create VNet
      enable_default_subnet: true
      enable_peering: true        # Peer to connectivity hub
      address_spaces:
        exp: ["10.200.212.0/25"]
        dev: ["10.200.212.0/25"]
        main: ["10.180.212.0/25"]
        prod: ["10.180.213.0/25"]
      hub_vnet_id:
        exp: "/subscriptions/e786a45d-030c-4dfb-a827-fd18444ed496/resourceGroups/rg-explz-conn-dev-gc-hub/providers/Microsoft.Network/virtualNetworks/vnet-explz-conn-dev-gc-hub"
        # ... other environments
    
    storage_account_settings:
      replication_type: "LRS"
      shared_access_key_enabled: false
      public_network_access_enabled: false
      blob_versioning_enabled: false
      access_tier: "cool"
      network_rules:
        default_action: "Allow"
        enable_blob_private_endpoint: true
        suffix: "log"
    
    tags:
      BEE360_ID: "SE1070"
      archetype: "management"
      workload_type: "Production"

landing_zone_groups:
  platform:
    - connectivity
    - management
  
  hubs:
    - connectivity
  
  spokes:
    - management
```

## 4. modules.yaml (COMPLETE)

```yaml
version: "1.0"

modules:
  # SEED MODULE
  seed:
    description: "Bootstrap infrastructure - creates backend storage"
    category: "bootstrap"
    owner: "platform-team"
    path: "../../../.terraform-repo/source/root_modules/seed"
    depends_on: []
    
    backend:
      template: "bootstrap"
      key: "${root_id}-seed.tfstate"
    
    variables:
      tf_backend_subscription_id: "${bootstrap.subscription_id}"
      tf_backend_resource_group_name: "${bootstrap.resource_group}"
      tf_backend_storage_account_name: "${bootstrap.storage_account}"
      tf_backend_container_name: "${bootstrap.container_name}"
      environment: "${environment}"
      location: "${location}"
      location_slug: "${location_slug}"
      org_name: "${org_name}"
      billing_account_id: "${billing.account_id}"
      billing_profile_id: "${billing.profile_id}"
      github_org_name: "${github.org_name}"
      github_repo_name: "${github.repo_name}"
    
    outputs:
      - storage_account_name
      - user_managed_identity_id
  
  # CORE MODULE
  core:
    description: "Core platform - management groups and policies"
    category: "platform"
    path: "../../../.terraform-repo/source/root_modules/plz/core"
    depends_on: [seed]
    
    backend:
      template: "bootstrap"
      key: "${root_id}-core.tfstate"
    
    variables:
      tf_backend_tenant_id: "${tenant_id}"
      tf_backend_subscription_id: "${bootstrap.subscription_id}"
      tf_backend_resource_group_name: "${bootstrap.resource_group}"
      tf_backend_storage_account_name: "${bootstrap.storage_account}"
      tf_backend_container_name: "${bootstrap.container_name}"
      root_id: "${root_id}"
      root_name: "${root_name}"
      location: "${location}"
    
    outputs:
      - root_management_group_id
  
  # VENDING MODULE (with instances)
  vending:
    description: "Subscription vending"
    category: "bootstrap"
    path: "../../../.terraform-repo/source/root_modules/plz/bootstrap/vending"
    
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
          root_id: "${root_id}"
          location: "${location}"
          location_slug: "${location_slug}"
          billing_account_id: "${billing.account_id}"
          billing_profile_id: "${billing.profile_id}"
          github_org_name: "${github.org_name}"
          landing_zones: "${landing_zone_configs.connectivity}"
        
        outputs:
          - subscription_id
      
      management:
        description: "Vend management subscription"
        depends_on: [core]
        
        backend:
          template: "bootstrap"
          key: "${root_id}-vending-mgmt.tfstate"
        
        variables:
          landing_zone_name: "management"
          environment: "${environment}"
          landing_zones: "${landing_zone_configs.management}"
        
        outputs:
          - subscription_id
  
  # BASELINE MODULE (with instances)
  baseline:
    description: "Baseline infrastructure - resource groups, storage, network"
    category: "bootstrap"
    path: "../../../.terraform-repo/source/root_modules/plz/bootstrap/baseline"
    
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
          root_id: "${root_id}"
          location: "${location}"
          # Backend keys for remote state references
          tf_backend_key_vending: "${root_id}-vending-conn.tfstate"
          landing_zones: "${landing_zone_configs.connectivity}"
        
        outputs:
          - vnet_id
          - resource_group_name
      
      management:
        description: "Baseline for management landing zone"
        depends_on: [vending.management, baseline.connectivity]
        
        backend:
          template: "bootstrap"
          key: "${root_id}-baseline-mgmt.tfstate"
        
        variables:
          landing_zone_name: "management"
          environment: "${environment}"
          tf_backend_key_vending: "${root_id}-vending-mgmt.tfstate"
          tf_backend_key_baseline_connectivity: "${root_id}-baseline-conn.tfstate"
          landing_zones: "${landing_zone_configs.management}"
        
        outputs:
          - vnet_id
  
  # CONNECTIVITY MODULE
  connectivity:
    description: "Hub network - VPN, Firewall, Bastion"
    category: "landing_zone"
    path: "../../../.terraform-repo/source/root_modules/plz/connectivity"
    depends_on: [baseline.connectivity]
    
    backend:
      template: "landing_zone"
      landing_zone: "connectivity"
      key: "${environment}-connectivity.tfstate"
    
    variables:
      environment: "${environment}"
      location: "${location}"
      # Remote state references
      tf_backend_key_baseline: "${root_id}-baseline-conn.tfstate"
      # VPN config from landing zones
      vpn_config: "${landing_zones.connectivity.vpn_config.${environment}}"
      firewall_config: "${landing_zones.connectivity.firewall_config.${environment}}"
    
    outputs:
      - hub_vnet_id
      - firewall_private_ip
  
  # FIREWALL RULES MODULE
  firewall_rules:
    description: "Azure Firewall policies and rules"
    category: "landing_zone"
    path: "../../../.terraform-repo/source/root_modules/plz/connectivity/firewall_rules"
    
    instances:
      connectivity:
        description: "Firewall rules for connectivity hub"
        depends_on: [connectivity]
        
        backend:
          template: "landing_zone"
          landing_zone: "connectivity"
          key: "${environment}-firewall-rules-conn.tfstate"
        
        variables:
          environment: "${environment}"
          tf_backend_key_connectivity: "${environment}-connectivity.tfstate"
        
        outputs:
          - firewall_policy_id
  
  # DNS MODULE
  dns:
    description: "Private DNS zones and resolvers"
    category: "landing_zone"
    path: "../../../.terraform-repo/source/root_modules/plz/connectivity/dns"
    
    instances:
      connectivity:
        description: "DNS for connectivity landing zone"
        depends_on: [firewall_rules.connectivity]
        
        backend:
          template: "landing_zone"
          landing_zone: "connectivity"
          key: "${environment}-dns-conn.tfstate"
        
        variables:
          environment: "${environment}"
          tf_backend_key_connectivity: "${environment}-connectivity.tfstate"
        
        outputs:
          - dns_resolver_id
  
  # MANAGEMENT MODULE
  management:
    description: "Management services - Log Analytics, Automation"
    category: "landing_zone"
    path: "../../../.terraform-repo/source/root_modules/plz/management"
    depends_on: [baseline.management, connectivity]
    
    backend:
      template: "landing_zone"
      landing_zone: "management"
      key: "${environment}-management.tfstate"
    
    variables:
      environment: "${environment}"
      location: "${location}"
      tf_backend_key_baseline: "${root_id}-baseline-mgmt.tfstate"
      tf_backend_key_connectivity: "${environment}-connectivity.tfstate"
    
    outputs:
      - log_analytics_workspace_id

# Module groups for batch operations
groups:
  bootstrap:
    - seed
    - core
  
  platform_vending:
    - vending.connectivity
    - vending.management
  
  platform_baseline:
    - baseline.connectivity
    - baseline.management
  
  connectivity_stack:
    - connectivity
    - firewall_rules.connectivity
    - dns.connectivity
  
  landing_zones:
    - connectivity
    - management
  
  all:
    - seed
    - core
    - vending.connectivity
    - vending.management
    - baseline.connectivity
    - baseline.management
    - connectivity
    - firewall_rules.connectivity
    - dns.connectivity
    - management
```

## 5. pipelines.yaml (COMPLETE)

```yaml
version: "1.0"

pipelines:
  # COMPLETE PLATFORM DEPLOYMENT
  deploy-platform-complete:
    description: "Deploy complete Azure Landing Zone platform"
    default_operation: apply
    
    settings:
      parallel_limit: 3
      timeout: "120m"
      auto_approve: false
    
    before:
      - name: "Verify Azure authentication"
        command: "az account show"
      - name: "Verify GitHub authentication"
        command: "gh auth status"
    
    stages:
      # Stage 1: Bootstrap foundation
      - name: "Bootstrap Infrastructure"
        modules:
          - seed
          - core
        parallel: false  # Sequential within stage
        
        settings:
          timeout: "30m"
      
      # Stage 2: Subscription vending (can run in parallel)
      - name: "Subscription Provisioning"
        modules:
          - vending.connectivity
          - vending.management
        parallel: true
        
        settings:
          timeout: "30m"
      
      # Stage 3: Baseline infrastructure
      - name: "Baseline Infrastructure"
        modules:
          - baseline.connectivity
          - baseline.management
        parallel: false  # Management depends on connectivity
        
        settings:
          timeout: "30m"
      
      # Stage 4: Connectivity hub
      - name: "Hub Networking"
        modules:
          - connectivity
        
        settings:
          timeout: "45m"
      
      # Stage 5: Connectivity sub-modules
      - name: "Connectivity Services"
        modules:
          - firewall_rules.connectivity
          - dns.connectivity
        parallel: false  # DNS depends on firewall rules
        
        settings:
          timeout: "30m"
      
      # Stage 6: Management spoke
      - name: "Management Services"
        modules:
          - management
        
        settings:
          timeout: "30m"
    
    after:
      - name: "Deployment summary"
        command: "echo '✅ Complete Azure Landing Zone deployed successfully!'"
    
    environments:
      exp:
        settings:
          auto_approve: false
      prod:
        require_confirmation: true
        confirmation_message: "Deploy COMPLETE platform to PRODUCTION? Type 'yes':"
  
  # CONNECTIVITY LANDING ZONE ONLY
  deploy-connectivity:
    description: "Deploy connectivity landing zone (hub)"
    default_operation: apply
    
    settings:
      parallel_limit: 2
      timeout: "90m"
    
    stages:
      - name: "Vending and Baseline"
        modules:
          - vending.connectivity
          - baseline.connectivity
        parallel: false
      
      - name: "Hub Network"
        modules:
          - connectivity
      
      - name: "Connectivity Services"
        modules:
          - firewall_rules.connectivity
          - dns.connectivity
        parallel: false
  
  # MANAGEMENT LANDING ZONE ONLY
  deploy-management:
    description: "Deploy management landing zone (spoke)"
    default_operation: apply
    
    settings:
      timeout: "60m"
    
    before:
      - name: "Verify connectivity hub exists"
        command: "az network vnet show --ids ${landing_zones.connectivity.network.hub_vnet_id.${environment}}"
    
    stages:
      - name: "Vending and Baseline"
        modules:
          - vending.management
          - baseline.management
        parallel: false
      
      - name: "Management Services"
        modules:
          - management
  
  # PLAN ALL
  plan-all:
    description: "Plan all modules"
    default_operation: plan
    
    settings:
      parallel_limit: 5
      timeout: "30m"
    
    stages:
      - name: "Plan All"
        groups:
          - all
        parallel: true
  
  # VALIDATE ALL
  validate-all:
    description: "Validate all modules"
    default_operation: validate
    
    settings:
      parallel_limit: 10
      timeout: "10m"
    
    stages:
      - name: "Validate All"
        groups:
          - all
        parallel: true
```

## Key Improvements

### 1. Configuration Reduction

**Backend configurations:**
- Before: 88 lines (11 modules × 8 lines)
- After: 40 lines (4 templates + references)
- **Reduction: 58%**

**Variable management:**
- Before: Scattered across multiple `.tfvars` files
- After: Centralized in `variables.yaml`
- Environment overrides eliminate duplication

### 2. Module Instance Pattern

**Problem:** Same module deployed multiple times (vending, baseline)

**Solution:**
```yaml
vending:
  instances:
    connectivity:
      backend:
        key: "exp-vending-conn.tfstate"
    management:
      backend:
        key: "exp-vending-mgmt.tfstate"
```

**Benefit:** Define once, deploy many times with different configs

### 3. Landing Zone Abstraction

**Problem:** Landing zone config scattered across backend configs and variables

**Solution:** First-class landing zone definitions
```yaml
landing_zones:
  connectivity:
    subscriptions:
      exp: "e786a45d-..."
    backend:
      storage_account: "st${org_name}conn${environment}${location_slug}base"
```

**Benefit:** Reference via `${landing_zones.connectivity.subscriptions.exp}`

### 4. Dependency Management

**Clear dependency graph:**
```
seed → core → vending → baseline → landing_zone_modules
```

**Explicit in config:**
```yaml
baseline:
  instances:
    management:
      depends_on: [vending.management, baseline.connectivity]
```

### 5. Pipeline Orchestration

**Multi-stage deployment:**
```yaml
stages:
  - name: "Bootstrap"
    modules: [seed, core]
    parallel: false
  
  - name: "Vending"
    modules: [vending.connectivity, vending.management]
    parallel: true
```

**Benefit:** Clear deployment order with parallelization where possible

## Usage Examples

### Deploy complete platform to exp environment:
```bash
tfpipboy pipeline deploy-platform-complete --env exp
```

### Deploy only connectivity landing zone:
```bash
tfpipboy pipeline deploy-connectivity --env exp
```

### Plan all changes:
```bash
tfpipboy pipeline plan-all --env exp
```

### Deploy specific module:
```bash
tfpipboy module apply core --env exp
```

### Deploy module group:
```bash
tfpipboy group apply bootstrap --env exp
```

## Summary

This configuration format provides:

1. **DRY Principle**: Eliminate duplication with templates and variables
2. **Clear Structure**: Separation of concerns across 5 files
3. **Reusability**: Module instances for repeated deployments
4. **Flexibility**: Environment overrides without duplication
5. **Orchestration**: Multi-stage pipelines with dependencies
6. **Maintainability**: Centralized configuration management

**Achieved Results:**
- 58% reduction in backend configuration
- Clear module dependency graph
- Automated deployment workflows
- Environment-specific customization without duplication
