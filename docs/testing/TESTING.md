# Testing Guide for tfpipboy

This document describes the comprehensive multi-level testing strategy for tfpipboy.

## Overview

The testing strategy is organized into three levels:

1. **Level 1**: Unit Tests, Linting, and Security Validation
2. **Level 2**: Build Testing and Cross-Platform Validation
3. **Level 3**: Integration and End-to-End Orchestration Tests

## Level 1: Unit Tests, Linting, and Security

### Running Level 1 Tests Locally

#### All Level 1 Tests
```bash
make test
```

This runs:
- Unit tests
- Race detector tests
- Linting (strict mode)
- Security scans (gosec)

#### Individual Test Targets

**Unit Tests**
```bash
make test-unit
```

**Race Detector**
```bash
make test-race
```

**Test Coverage**
```bash
make test-coverage
```

This generates:
- `coverage.out`: Coverage data
- `coverage.html`: HTML coverage report

**Linting**
```bash
make lint
```

Linting is strict and will fail on any issues. To auto-fix issues:
```bash
make lint-fix
```

**Security Scanning**
```bash
make test-security
```

This runs gosec and generates:
- Console output with security findings
- `gosec-report.json`: Detailed JSON report

**Benchmark Tests**
```bash
make test-bench
```

### Unit Test Coverage

Current test coverage by package:

| Package | Coverage | Test Files |
|---------|----------|------------|
| `pkg/orchestrator` | Improving | `graph_test.go`, `config_test.go` |
| `pkg/auth` | 42.7% | `azure_test.go`, `manager_test.go`, `github_test.go` |
| `pkg/cli` | 33.3% | `wrapper_test.go` |
| `pkg/terraform` | 53.9% | `context_test.go`, `environment_test.go`, `workspace_test.go` |

### Writing Unit Tests

Unit tests follow Go testing conventions:

```go
func TestFeatureName(t *testing.T) {
    // Arrange
    input := setupTestData()

    // Act
    result := functionUnderTest(input)

    // Assert
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

**Best Practices:**
- Use table-driven tests for multiple test cases
- Use `t.TempDir()` for temporary file operations
- Use `t.Parallel()` for independent tests
- Mock external dependencies
- Test both success and failure cases

### Linting Rules

The project uses `golangci-lint` with the following linters enabled:

- `bodyclose`: Checks HTTP response bodies are closed
- `errcheck`: Checks for unchecked errors
- `goconst`: Finds repeated strings that could be constants
- `gocyclo`: Checks function complexity (max: 15)
- `gofmt`: Checks code formatting
- `goimports`: Checks import formatting
- `gosec`: Security checks
- `gosimple`: Simplification suggestions
- `govet`: Reports suspicious constructs
- `ineffassign`: Detects ineffectual assignments
- `lll`: Checks line length (max: 140)
- `misspell`: Finds commonly misspelled words
- `staticcheck`: Static analysis
- `typecheck`: Type checks
- `unconvert`: Removes unnecessary type conversions
- `unused`: Checks for unused code

### Security Scanning

Security scanning uses `gosec` to detect:

- SQL injection vulnerabilities
- Command injection
- File traversal issues
- Unsafe use of crypto
- Hardcoded credentials
- Integer overflows
- Race conditions
- And more...

Issues are categorized by severity:
- **HIGH**: Must be fixed before merge
- **MEDIUM**: Should be reviewed and addressed
- **LOW**: Informational, review recommended

## Level 2: Build Testing and Cross-Platform Validation

### Running Level 2 Tests Locally

**Build Binary**
```bash
make build
```

**Test Cross-Compilation**
```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o bin/tfpipboy-linux-amd64 ./cmd/tfpipboy

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o bin/tfpipboy-linux-arm64 ./cmd/tfpipboy

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o bin/tfpipboy-darwin-amd64 ./cmd/tfpipboy

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o bin/tfpipboy-darwin-arm64 ./cmd/tfpipboy

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o bin/tfpipboy-windows-amd64.exe ./cmd/tfpipboy
```

**Verify Binary**
```bash
./bin/tfpipboy --version
./bin/tfpipboy --help
```

### Build Validation Checks

The build validation ensures:

1. **Binary builds successfully** on all target platforms
2. **Binary size is reasonable** (< 50MB)
3. **Binary executes** without errors
4. **Help command works** and displays usage
5. **Cross-compilation succeeds** for all platforms

## Level 3: Integration and End-to-End Tests

### Integration Test Structure

```
test/integration/
├── terraform-modules/          # Example Terraform modules
│   ├── vpc/                   # VPC module (foundation)
│   ├── subnet/                # Subnet module (depends on VPC)
│   ├── application/           # App module (depends on subnet)
│   └── database/              # DB module (depends on subnet)
├── tfproject.yaml             # Orchestration configuration
└── run-integration-tests.sh   # Integration test runner
```

### Running Integration Tests Locally

**Prerequisites:**
- Terraform installed (>= 1.0)
- tfpipboy binary built (`make build`)

**Run all integration tests:**
```bash
cd test/integration
./run-integration-tests.sh
```

**Run specific test scenarios:**

1. **Configuration Validation**
   ```bash
   ./bin/tfpipboy validate --config-dir=test/integration
   ```

2. **Module Listing**
   ```bash
   ./bin/tfpipboy list --config-dir=test/integration
   ```

3. **Dependency Graph**
   ```bash
   ./bin/tfpipboy graph --config-dir=test/integration dev-environment
   ```

4. **Plan Single Module**
   ```bash
   ./bin/tfpipboy plan --config-dir=test/integration vpc-dev
   ```

5. **Plan with Dependencies**
   ```bash
   ./bin/tfpipboy plan --config-dir=test/integration subnet-dev-public
   ```

6. **Plan Entire Environment**
   ```bash
   ./bin/tfpipboy plan --config-dir=test/integration --group dev-environment
   ```

7. **Pipeline Execution**
   ```bash
   ./bin/tfpipboy pipeline plan --config-dir=test/integration deploy-dev
   ```

### Integration Test Scenarios

The integration tests cover:

#### 1. Configuration Validation
- Valid YAML parsing
- Schema validation
- Dependency resolution
- Circular dependency detection

#### 2. Dependency Graph
- Linear dependencies (A → B → C)
- Parallel dependencies (A → B, A → C)
- Diamond dependencies (A → B,C → D)
- Complex multi-layer graphs

#### 3. Module Orchestration
- Single module execution
- Module with dependencies
- Group execution
- Pipeline execution with stages

#### 4. Parallel Execution
- Detecting parallelizable modules
- Correct stage ordering
- Respecting parallel limits

#### 5. Error Handling
- Non-existent modules
- Invalid configuration
- Missing dependencies
- Circular dependencies

### Example Test Modules

**VPC Module** (Foundation Layer)
- Creates virtual network
- No dependencies
- Can run in parallel across environments

**Subnet Module** (Network Layer)
- Creates subnets within VPC
- Depends on VPC
- Can run in parallel (public/private)

**Application Module** (Compute Layer)
- Deploys application instances
- Depends on subnet
- Can run in parallel (frontend/backend)

**Database Module** (Data Layer)
- Creates database instances
- Depends on subnet
- Sequential execution per environment

## GitHub Actions Workflows

### Multi-Level Testing Workflow

File: `.github/workflows/test-multi-level.yml`

**Trigger Conditions:**
- Push to `main`, `develop`, or `claude/*` branches
- Pull requests to `main` or `develop`
- Manual workflow dispatch

**Jobs:**

#### Level 1 Jobs
1. `level1-unit-tests`: Run unit tests on Ubuntu, macOS, Windows
2. `level1-linting`: Run strict linting checks
3. `level1-security`: Run security scans (blocking)

#### Level 2 Jobs
4. `level2-build-validation`: Build and validate on all platforms
5. `level2-cross-compile`: Test cross-compilation

#### Level 3 Jobs
6. `level3-integration-tests`: Run integration test suite
7. `level3-e2e-orchestration`: End-to-end orchestration tests

#### Summary Job
8. `test-summary`: Aggregate results and fail if any job failed

### Viewing Test Results

**GitHub Actions UI:**
1. Go to repository Actions tab
2. Select workflow run
3. View job details and logs
4. Download artifacts (coverage reports, logs)

**Test Summary:**
- Visible in GitHub Actions summary
- Shows pass/fail for each level
- Links to detailed logs

## Test Artifacts

### Generated Artifacts

**Coverage Reports:**
- `coverage.out`: Go coverage data
- `coverage.html`: HTML coverage report
- Uploaded to Codecov (Ubuntu only)

**Security Reports:**
- `gosec-report.json`: Detailed security findings
- `results.sarif`: SARIF format for GitHub Security

**Integration Test Logs:**
- `/tmp/tfpipboy-*.log`: Command execution logs
- `/tmp/e2e-*.log`: End-to-end test logs
- Retained for 7-30 days in GitHub Actions

**Build Artifacts:**
- `bin/tfpipboy-*`: Compiled binaries
- Retained for 7 days in GitHub Actions

## Continuous Integration

### Pre-Merge Requirements

Before merging a pull request, the following must pass:

✅ All unit tests pass on all platforms
✅ Linting passes with no errors
✅ Security scan passes with no HIGH severity issues
✅ Build succeeds on all platforms
✅ Integration tests pass
✅ E2E tests pass

### Performance Benchmarks

Run benchmarks to detect performance regressions:

```bash
make test-bench
```

Compare with baseline:
```bash
# Run benchmark and save
go test -bench=. -benchmem ./... > new.bench

# Compare with old
benchcmp old.bench new.bench
```

## Troubleshooting

### Common Issues

**1. Tests fail locally but pass in CI**
- Check Go version matches CI (`1.23.x`)
- Clean and rebuild: `make clean && make build`
- Clear test cache: `go clean -testcache`

**2. Linting fails**
- Run auto-fix: `make lint-fix`
- Check `.golangci.yml` for configuration
- Update golangci-lint: `brew upgrade golangci-lint`

**3. Integration tests fail**
- Ensure Terraform is installed
- Check binary is built: `make build`
- Run with verbose output: `./run-integration-tests.sh 2>&1 | tee test.log`

**4. Race detector failures**
- Indicates potential race conditions
- Run specific test: `go test -race -run TestName ./pkg/...`
- Fix by adding proper synchronization

### Getting Help

- Check existing issues: [GitHub Issues](https://github.com/StanleyXie/tfpipboy/issues)
- Review CI logs for detailed error messages
- Run tests with `-v` flag for verbose output

## Future Improvements

### Planned Testing Enhancements

- [ ] Increase unit test coverage to 80%+
- [ ] Add mutation testing
- [ ] Add performance regression tests
- [ ] Add chaos engineering tests
- [ ] Add contract testing for CLI interface
- [ ] Add snapshot testing for output formatting
- [ ] Add property-based testing with rapid/gopter

## Contributing

When adding new features:

1. **Write tests first** (TDD approach)
2. **Ensure tests pass** before committing
3. **Update documentation** if behavior changes
4. **Add integration tests** for complex features
5. **Check coverage** doesn't decrease

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [golangci-lint](https://golangci-lint.run/)
- [gosec](https://github.com/securego/gosec)
- [GitHub Actions](https://docs.github.com/en/actions)
- [Terraform Testing](https://www.terraform.io/docs/language/modules/testing-experiment.html)
