# Integration Tests for tfpipboy

This directory contains integration tests that validate tfpipboy's orchestration capabilities with real Terraform modules.

## Overview

The integration tests use example Terraform modules to verify:

- **Dependency graph building**: Correctly ordering modules based on dependencies
- **Parallel execution**: Running independent modules concurrently
- **Pipeline orchestration**: Multi-stage deployments with proper sequencing
- **Multi-environment support**: Managing dev, staging, and production configurations
- **Error handling**: Graceful handling of configuration errors and missing dependencies

## Directory Structure

```
test/integration/
├── README.md                      # This file
├── tfproject.yaml                 # Orchestration configuration
├── run-integration-tests.sh       # Test runner script
└── terraform-modules/             # Example Terraform modules
    ├── vpc/                       # VPC module (foundation)
    │   └── main.tf
    ├── subnet/                    # Subnet module (depends on VPC)
    │   └── main.tf
    ├── application/               # Application module (depends on subnet)
    │   └── main.tf
    └── database/                  # Database module (depends on subnet)
        └── main.tf
```

## Test Modules

### VPC Module

**Purpose**: Foundation networking layer

**Dependencies**: None

**Instances**:
- `vpc-dev`: Development VPC (10.0.0.0/16)
- `vpc-staging`: Staging VPC (10.1.0.0/16)
- `vpc-prod`: Production VPC (10.2.0.0/16)

**Execution**: Can run in parallel across environments

### Subnet Module

**Purpose**: Network segmentation within VPC

**Dependencies**: Requires VPC

**Instances**:
- `subnet-dev-public`: Public subnet in dev VPC
- `subnet-dev-private`: Private subnet in dev VPC
- `subnet-staging-public`: Public subnet in staging VPC
- `subnet-staging-private`: Private subnet in staging VPC

**Execution**: Public and private subnets can run in parallel within same environment

### Application Module

**Purpose**: Application deployment

**Dependencies**: Requires subnet

**Instances**:
- `app-dev-frontend`: Frontend app in dev public subnet
- `app-dev-backend`: Backend app in dev private subnet
- `app-staging-frontend`: Frontend app in staging public subnet
- `app-staging-backend`: Backend app in staging private subnet

**Execution**: Frontend and backend can run in parallel

### Database Module

**Purpose**: Database instance provisioning

**Dependencies**: Requires subnet

**Instances**:
- `db-dev`: Development database
- `db-staging`: Staging database

**Execution**: Sequential per environment

## Dependency Graph

The test configuration creates the following dependency graph:

```
Stage 0: Foundation Layer (Parallel)
├── vpc-dev
├── vpc-staging
└── vpc-prod

Stage 1: Network Layer (Parallel within environment)
├── subnet-dev-public (depends on vpc-dev)
├── subnet-dev-private (depends on vpc-dev)
├── subnet-staging-public (depends on vpc-staging)
└── subnet-staging-private (depends on vpc-staging)

Stage 2: Data Layer (Parallel across environments)
├── db-dev (depends on subnet-dev-private)
└── db-staging (depends on subnet-staging-private)

Stage 3: Application Layer (Parallel)
├── app-dev-frontend (depends on subnet-dev-public)
├── app-dev-backend (depends on subnet-dev-private)
├── app-staging-frontend (depends on subnet-staging-public)
└── app-staging-backend (depends on subnet-staging-private)
```

## Running Integration Tests

### Prerequisites

1. **Install Terraform**:
   ```bash
   # macOS
   brew install terraform

   # Linux
   wget https://releases.hashicorp.com/terraform/1.6.0/terraform_1.6.0_linux_amd64.zip
   unzip terraform_1.6.0_linux_amd64.zip
   sudo mv terraform /usr/local/bin/

   # Verify
   terraform version
   ```

2. **Build tfpipboy**:
   ```bash
   cd ../..  # Go to project root
   make build
   ```

### Run All Tests

```bash
cd test/integration
./run-integration-tests.sh
```

**Expected Output**:
```
[INFO] Starting tfpipboy integration tests...
[INFO] Setting up integration test environment...
[SUCCESS] tfpipboy binary found: /path/to/bin/tfpipboy
[SUCCESS] Terraform found: Terraform v1.6.0
[INFO] Running test: tfpipboy --help
[SUCCESS] Test passed: tfpipboy --help
...
=========================================
Integration Test Summary
=========================================
Tests Run:    12
Tests Passed: 12
Tests Failed: 0
=========================================
[SUCCESS] All integration tests passed!
```

### Run Individual Test Scenarios

**1. Validate Configuration**
```bash
../../bin/tfpipboy validate --config-dir=.
```

**2. List All Modules**
```bash
../../bin/tfpipboy list --config-dir=.
```

**3. Show Dependency Graph**
```bash
# For dev environment
../../bin/tfpipboy graph --config-dir=. dev-environment

# For all modules
../../bin/tfpipboy graph --config-dir=. all-infrastructure
```

**4. Plan Single Module**
```bash
../../bin/tfpipboy plan --config-dir=. vpc-dev
```

**5. Plan with Dependencies**
```bash
# This will plan both vpc-dev and subnet-dev-public
../../bin/tfpipboy plan --config-dir=. subnet-dev-public
```

**6. Plan Module Group**
```bash
../../bin/tfpipboy plan --config-dir=. --group dev-environment
```

**7. Execute Pipeline**
```bash
# Plan the entire dev pipeline
../../bin/tfpipboy pipeline plan --config-dir=. deploy-dev

# Apply the entire dev pipeline (dry-run)
../../bin/tfpipboy pipeline apply --config-dir=. deploy-dev --dry-run
```

## Test Scenarios

### Test 1: Configuration Validation
**Purpose**: Verify configuration is valid
**Command**: `tfpipboy validate --config-dir=.`
**Expected**: No errors, all modules validated

### Test 2: Module Listing
**Purpose**: Verify all modules are discovered
**Command**: `tfpipboy list --config-dir=.`
**Expected**: All 12 instance names listed

### Test 3: Dependency Graph
**Purpose**: Verify dependency resolution
**Command**: `tfpipboy graph --config-dir=. dev-environment`
**Expected**: Graph shows correct stages and dependencies

### Test 4: Single Module Plan
**Purpose**: Verify Terraform execution for single module
**Command**: `tfpipboy plan --config-dir=. vpc-dev`
**Expected**: Terraform plan executes successfully

### Test 5: Dependency Cascade
**Purpose**: Verify dependencies are included automatically
**Command**: `tfpipboy plan --config-dir=. subnet-dev-public`
**Expected**: Both vpc-dev and subnet-dev-public are planned

### Test 6: Group Execution
**Purpose**: Verify group expansion and execution
**Command**: `tfpipboy plan --config-dir=. --group dev-environment`
**Expected**: All dev environment modules planned in correct order

### Test 7: Pipeline Planning
**Purpose**: Verify pipeline stage sequencing
**Command**: `tfpipboy pipeline plan --config-dir=. deploy-dev`
**Expected**: 4 stages executed in order

### Test 8: Parallel Detection
**Purpose**: Verify parallel execution detection
**Command**: Check graph output for parallel stages
**Expected**: Subnets show as parallelizable within stage

### Test 9: Error Handling - Non-existent Module
**Purpose**: Verify graceful error handling
**Command**: `tfpipboy plan --config-dir=. non-existent-module`
**Expected**: Clear error message, exit code 1

### Test 10: Multi-Environment
**Purpose**: Verify environment isolation
**Command**: Plan both dev and staging
**Expected**: Separate workspaces, no conflicts

## Customizing Tests

### Adding New Test Modules

1. Create module directory in `terraform-modules/`
2. Add `main.tf` with required Terraform code
3. Add module and instances to `tfproject.yaml`
4. Define dependencies using `depends_on`
5. Add module to appropriate groups
6. Update test script if needed

Example:
```yaml
modules:
  my-module:
    path: "./terraform-modules/my-module"
    instances:
      my-instance:
        depends_on:
          - vpc-dev
        variables:
          key: "value"
```

### Adding New Test Cases

Edit `run-integration-tests.sh` and add:

```bash
run_test \
    "Test Description" \
    "command to run" \
    expected_exit_code
```

## Debugging Failed Tests

### Enable Verbose Output

```bash
# Run with set -x for detailed execution
bash -x ./run-integration-tests.sh
```

### Check Terraform Logs

```bash
# Terraform debug logging
export TF_LOG=DEBUG
../../bin/tfpipboy plan --config-dir=. vpc-dev
```

### Inspect Workspaces

```bash
# Workspaces are created in .tfpipboy/
ls -la .tfpipboy/workspaces/
```

### View Test Logs

```bash
# Integration test logs are saved to /tmp
cat /tmp/tfpipboy-*.log
```

## Continuous Integration

These integration tests run automatically in GitHub Actions:

**Workflow**: `.github/workflows/test-multi-level.yml`

**Jobs**:
- `level3-integration-tests`: Runs full test suite
- `level3-e2e-orchestration`: Runs specific orchestration scenarios

**Artifacts**:
- Integration test logs (retained 7 days)
- E2E test logs (retained 7 days)

## Performance Considerations

### Test Execution Time

| Test Type | Duration | Parallelization |
|-----------|----------|-----------------|
| Single module | ~5s | N/A |
| With dependencies | ~10s | Sequential |
| Full environment | ~30s | Parallel |
| All tests | ~2-3min | Sequential |

### Optimization Tips

1. **Use dry-run**: `--dry-run` skips actual Terraform apply
2. **Test subsets**: Test specific modules instead of full suite
3. **Parallel execution**: Use `--parallel` flag when safe
4. **Local backend**: Tests use local backend for speed

## Troubleshooting

### Common Issues

**1. "terraform: command not found"**
- Install Terraform (see Prerequisites)

**2. "tfpipboy binary not found"**
- Run `make build` from project root
- Check `../../bin/tfpipboy` exists

**3. "Configuration validation failed"**
- Check YAML syntax in `tfproject.yaml`
- Verify module paths are correct
- Check for circular dependencies

**4. "Module not found"**
- Ensure module directory exists
- Check path in `tfproject.yaml`
- Verify `main.tf` exists in module directory

**5. "Dependency not found"**
- Check `depends_on` references valid instance names
- Verify instance exists in configuration
- Check for typos in instance names

## Contributing

When adding integration tests:

1. Keep modules simple (use `null_resource` for simulation)
2. Add meaningful dependencies to test orchestration
3. Include both success and failure test cases
4. Document expected behavior
5. Update this README

## References

- [tfpipboy Documentation](../../README.md)
- [Testing Guide](../../TESTING.md)
- [Terraform Documentation](https://www.terraform.io/docs)
