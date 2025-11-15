# Contributing to tfpipboy

Thank you for your interest in contributing to tfpipboy! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional but recommended)
- golangci-lint (for code linting)

### Setup Steps

1. **Fork and Clone**

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/tfpipboy.git
cd tfpipboy
```

2. **Add Upstream Remote**

```bash
git remote add upstream https://github.com/StanleyXie/tfpipboy.git
```

3. **Setup Development Environment**

```bash
# Checkout develop branch
git checkout develop

# Setup tools and dependencies
make dev-setup
make deps
```

4. **Verify Setup**

```bash
# Run tests
make test

# Build the application
make build

# Run the application
make run
```

## Development Workflow

### Branch Strategy

We follow **Git Flow**:

- `main`: Production releases only
- `develop`: Main development branch (PR target)
- `feature/*`: New features
- `bugfix/*`: Bug fixes
- `release/*`: Release preparation

### Creating a Feature

1. **Sync with develop**

```bash
git checkout develop
git pull upstream develop
```

2. **Create feature branch**

```bash
git checkout -b feature/your-feature-name
```

3. **Make changes**

```bash
# Edit files
vim pkg/auth/azure.go

# Run tests
make test

# Format code
make fmt

# Run linter
make lint
```

4. **Commit changes**

Use [Conventional Commits](https://www.conventionalcommits.org/):

```bash
git add .
git commit -m "feat: add Azure CLI authentication detection"
```

**Commit Types:**
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `style:` Formatting, missing semicolons, etc.
- `refactor:` Code restructuring
- `test:` Adding tests
- `chore:` Maintenance

5. **Push and create PR**

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub targeting the `develop` branch.

## Coding Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting (run `make fmt`)
- Pass `golangci-lint` checks (run `make lint`)

### Code Organization

```
pkg/              # Public packages
  auth/           # Authentication detection
  terraform/      # Terraform context
  env/            # Environment variables
  tui/            # TUI components

internal/         # Private packages
  config/         # Configuration

cmd/              # Application entry points
  tfpipboy/       # Main CLI
```

### Naming Conventions

- **Packages**: lowercase, single word (e.g., `auth`, `terraform`)
- **Files**: lowercase with underscores (e.g., `azure_auth.go`)
- **Functions**: camelCase, exported functions start with uppercase
- **Variables**: camelCase
- **Constants**: CamelCase or SCREAMING_SNAKE_CASE

### Documentation

- Add godoc comments for exported functions and types
- Include usage examples in complex functions
- Update README.md for user-facing changes

**Example:**

```go
// CheckAzureAuth detects Azure CLI authentication status.
// It executes 'az account show' and parses the output.
//
// Returns AuthStatus with authentication details or error.
//
// Example:
//   status, err := CheckAzureAuth()
//   if err != nil {
//       return err
//   }
//   fmt.Printf("Azure: %s\n", status.User)
func CheckAzureAuth() (*AuthStatus, error) {
    // Implementation
}
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test ./pkg/auth/...

# Run specific test
go test -run TestCheckAzureAuth ./pkg/auth/
```

### Writing Tests

- Place tests in `*_test.go` files
- Use table-driven tests when appropriate
- Mock external dependencies
- Aim for >80% coverage for new code

**Example:**

```go
func TestCheckAzureAuth(t *testing.T) {
    tests := []struct {
        name    string
        setup   func()
        want    *AuthStatus
        wantErr bool
    }{
        {
            name: "authenticated user",
            setup: func() {
                // Setup mock
            },
            want: &AuthStatus{
                Authenticated: true,
                User: "test@example.com",
            },
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setup != nil {
                tt.setup()
            }

            got, err := CheckAzureAuth()
            if (err != nil) != tt.wantErr {
                t.Errorf("CheckAzureAuth() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // Assert got == tt.want
        })
    }
}
```

## Pull Request Process

### Before Submitting

1. ✅ Code passes `make test`
2. ✅ Code passes `make lint`
3. ✅ Code is formatted with `make fmt`
4. ✅ Documentation is updated
5. ✅ Commit messages follow convention
6. ✅ Branch is up to date with `develop`

### PR Template

When creating a PR, include:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Manual testing performed
- [ ] All tests pass

## Checklist
- [ ] Code follows project style
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] No new warnings generated
```

### Review Process

1. CI/CD checks must pass
2. At least one maintainer approval required
3. Address review comments
4. Squash commits if requested
5. Maintainer will merge to `develop`

## Issue Reporting

### Bug Reports

Use the bug report template:

```markdown
**Describe the bug**
Clear description of what the bug is.

**To Reproduce**
Steps to reproduce:
1. Run '...'
2. Execute '...'
3. See error

**Expected behavior**
What you expected to happen.

**Environment**
- OS: [e.g., macOS 14.0]
- Go version: [e.g., 1.21.0]
- tfpipboy version: [e.g., v0.1.0]

**Additional context**
Any other relevant information.
```

### Feature Requests

Use the feature request template:

```markdown
**Is your feature request related to a problem?**
Description of the problem.

**Describe the solution you'd like**
Clear description of what you want to happen.

**Describe alternatives you've considered**
Other solutions you've considered.

**Additional context**
Any other relevant information.
```

## Communication

- **GitHub Issues**: Bug reports, feature requests
- **Pull Requests**: Code contributions
- **Discussions**: Questions, ideas, general discussion

## Code of Conduct

### Our Standards

- Be respectful and inclusive
- Welcome newcomers
- Accept constructive criticism
- Focus on what's best for the project
- Show empathy towards other contributors

### Unacceptable Behavior

- Harassment or discriminatory language
- Trolling or insulting comments
- Personal or political attacks
- Publishing others' private information
- Other unprofessional conduct

## Getting Help

- Check existing [documentation](./docs/)
- Search [issues](https://github.com/StanleyXie/tfpipboy/issues)
- Create a new issue with your question
- Join discussions

## Recognition

Contributors will be recognized in:
- GitHub contributors page
- Release notes for their contributions
- CONTRIBUTORS.md file (coming soon)

Thank you for contributing to tfpipboy! 🎉
