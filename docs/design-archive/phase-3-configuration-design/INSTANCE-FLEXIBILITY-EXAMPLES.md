# Instance Configuration Flexibility Examples

The instance-based model in tfpipboy is designed to be flexible and customizable for various use cases. Environment and region are first-class fields but are entirely optional.

## Example 1: Multi-Environment, Multi-Region (Azure Landing Zone)

```yaml
modules:
  seed:
    path: "terraform/seed"
    instances:
      seed-prod-eastus:
        environment: "production"
        region: "eastus"
        backend:
          type: "azurerm"
          storage_account_name: "tfstateprodeast"
          key: "seed-prod-eastus.tfstate"
      
      seed-dev-westus:
        environment: "development"
        region: "westus"
        backend:
          type: "azurerm"
          storage_account_name: "tfstatedevwest"
          key: "seed-dev-westus.tfstate"
```

**Summary Display:**
```
Instance             Status     Duration     Backend      Details
--------------------------------------------------------------------------------
seed-prod-eastus     ✓          45s          azurerm      production/eastus
seed-dev-westus      ✓          32s          azurerm      development/westus
```

## Example 2: Multi-Cloud (Different Providers)

```yaml
modules:
  infrastructure:
    path: "terraform/infra"
    instances:
      infra-aws-primary:
        environment: "production"
        region: "us-east-1"
        backend:
          type: "s3"
          bucket: "tfstate-aws"
          key: "infra-aws.tfstate"
      
      infra-gcp-backup:
        environment: "production"
        region: "us-central1"
        backend:
          type: "gcs"
          bucket: "tfstate-gcp"
          prefix: "infra-gcp"
      
      infra-azure-dr:
        environment: "production"
        region: "eastus"
        backend:
          type: "azurerm"
          storage_account_name: "tfstateazure"
          key: "infra-azure.tfstate"
```

**Summary Display:**
```
Instance             Status     Duration     Backend      Details
--------------------------------------------------------------------------------
infra-aws-primary    ✓          1m20s        s3           production/us-east-1
infra-gcp-backup     ✓          58s          gcs          production/us-central1
infra-azure-dr       ✓          1m5s         azurerm      production/eastus
```

## Example 3: Multi-Tenant SaaS (Tenant-based, No Region)

```yaml
modules:
  tenant-infrastructure:
    path: "terraform/tenant"
    instances:
      tenant-acme-corp:
        environment: "tenant-acme"
        # No region field - all tenants in same region
        backend:
          type: "s3"
          bucket: "tfstate-tenants"
          key: "tenant-acme.tfstate"
        variables:
          tenant_id: "acme-corp"
          tenant_name: "Acme Corporation"
      
      tenant-globex:
        environment: "tenant-globex"
        backend:
          type: "s3"
          bucket: "tfstate-tenants"
          key: "tenant-globex.tfstate"
        variables:
          tenant_id: "globex"
          tenant_name: "Globex Corporation"
```

**Summary Display:**
```
Instance             Status     Duration     Backend      Details
--------------------------------------------------------------------------------
tenant-acme-corp     ✓          42s          s3           tenant-acme
tenant-globex        ✓          38s          s3           tenant-globex
```

## Example 4: Feature Branch Deployments (No Environment/Region)

```yaml
modules:
  app:
    path: "terraform/app"
    instances:
      app-main:
        # No environment or region - using branch name
        backend:
          type: "s3"
          bucket: "tfstate-branches"
          key: "main.tfstate"
      
      app-feature-auth:
        backend:
          type: "s3"
          bucket: "tfstate-branches"
          key: "feature-auth.tfstate"
      
      app-feature-payments:
        backend:
          type: "s3"
          bucket: "tfstate-branches"
          key: "feature-payments.tfstate"
```

**Summary Display:**
```
Instance             Status     Duration     Backend      Details
--------------------------------------------------------------------------------
app-main             ✓          55s          s3           
app-feature-auth     ✓          48s          s3           
app-feature-payments ✓          51s          s3           
```

## Example 5: Workspace-Style (Environment Only)

```yaml
modules:
  platform:
    path: "terraform/platform"
    instances:
      platform-production:
        environment: "production"
        # No region - single global deployment per environment
        backend:
          type: "s3"
          bucket: "tfstate-platform"
          key: "production.tfstate"
      
      platform-staging:
        environment: "staging"
        backend:
          type: "s3"
          bucket: "tfstate-platform"
          key: "staging.tfstate"
      
      platform-development:
        environment: "development"
        backend:
          type: "s3"
          bucket: "tfstate-platform"
          key: "development.tfstate"
```

**Summary Display:**
```
Instance             Status     Duration     Backend      Details
--------------------------------------------------------------------------------
platform-production  ✓          1m15s        s3           production
platform-staging     ✓          58s          s3           staging
platform-development ✓          45s          s3           development
```

## Example 6: Custom Metadata (Using Environment Field Creatively)

```yaml
modules:
  kubernetes-cluster:
    path: "terraform/k8s"
    instances:
      k8s-cluster-v1-28:
        environment: "k8s-1.28"  # Using for version tracking
        region: "us-east-1"
        backend:
          type: "s3"
          key: "k8s-v1-28.tfstate"
      
      k8s-cluster-v1-29:
        environment: "k8s-1.29"
        region: "us-west-2"
        backend:
          type: "s3"
          key: "k8s-v1-29.tfstate"
```

**Summary Display:**
```
Instance             Status     Duration     Backend      Details
--------------------------------------------------------------------------------
k8s-cluster-v1-28    ✓          2m30s        s3           k8s-1.28/us-east-1
k8s-cluster-v1-29    ✓          2m45s        s3           k8s-1.29/us-west-2
```

## Key Principles

1. **Instance Names Are Free-Form**: Use any naming convention that makes sense for your use case
2. **Environment and Region Are Optional**: Only include them if they're meaningful for your project
3. **Flexible Interpretation**: Use these fields however makes sense (version, tenant, branch, etc.)
4. **No Validation Rules**: The system doesn't enforce any specific values or patterns
5. **Dependencies Use Instance Names**: `depends_on` references instance names, not module names

## Best Practices

1. **Be Consistent**: Pick a naming convention and stick with it across your project
2. **Make Names Meaningful**: Instance names should clearly identify what's being deployed
3. **Use Metadata Wisely**: If environment/region don't fit your use case, consider custom variables
4. **Document Your Convention**: Add comments in your config explaining your naming scheme
