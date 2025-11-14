# tf-pipboy

**Terraform orchestration tool with parallel execution, dependency management, and real-time monitoring.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/StanleyXie/tf-pipboy)](go.mod)
[![Release](https://img.shields.io/github/v/release/StanleyXie/tf-pipboy)](https://github.com/StanleyXie/tf-pipboy/releases)

---

## Overview

tf-pipboy is a powerful Terraform orchestration tool that simplifies managing complex, multi-module infrastructure deployments. It provides dependency-aware execution, parallel processing, and real-time monitoring through a beautiful terminal interface.

### Key Features

- 🚀 **Parallel Execution** - Run independent modules concurrently
- 🔗 **Dependency Management** - Automatic execution ordering
- 🔐 **Authentication Monitoring** - Real-time status for AWS, Azure, GCP, GitHub
- 📊 **Live Board** - Beautiful TUI with real-time job status
- 🏗️ **Isolated Workspaces** - Each module runs in its own environment
- 📋 **Pipeline Support** - Define reusable deployment workflows
- ✅ **Validation** - Comprehensive configuration validation

---

## Quick Start

### Installation

#### Homebrew (macOS/Linux)
```bash
brew tap stanleyxie/tap
brew install tfpipboy
```

#### Go Install
```bash
go install github.com/StanleyXie/tf-pipboy/cmd/tfpipboy@latest
```

#### From Source
```bash
git clone https://github.com/StanleyXie/tf-pipboy.git
cd tf-pipboy
make build
make install
```

### Basic Usage

1. **Create configuration**:
```bash
mkdir .tfpipboy
```

2. **Define your infrastructure** in `.tfpipboy/tfproject.yaml`:
```yaml
version: "1.0"

modules:
  - name: networking
    path: terraform/modules/networking
    
  - name: database
    path: terraform/modules/database
    depends_on: [networking]
    
  - name: application
    path: terraform/modules/application
    depends_on: [networking, database]

instances:
  - module: networking
    workspace: "prod-network"
    
  - module: database
    workspace: "prod-db"
    
  - module: application
    workspace: "prod-app"
```

3. **Execute**:
```bash
# Plan all modules
tfpipboy --operation plan --targets-all

# Apply infrastructure
tfpipboy --operation apply --targets-all

# Destroy specific modules
tfpipboy --operation destroy --targets networking,database
```

---

## Status

**Current Version**: v0.6.0  
**Status**: Core features complete, preparing for public release

### What's Implemented ✅

- ✅ Multi-module Terraform orchestration
- ✅ Parallel execution with dependency management
- ✅ Authentication monitoring (AWS, Azure, GCP, GitHub)
- ✅ Live board TUI with real-time status
- ✅ Configuration validation
- ✅ Pipeline support
- ✅ Isolated workspace management

### Roadmap 🗺️

See [ROADMAP.md](ROADMAP.md) for detailed project history and future plans.

- **v0.7.0**: Documentation & stability
- **v0.8.0**: Enhanced user experience
- **v1.0.0**: Production release with Homebrew distribution

---

## Documentation

### Getting Started
- **[Getting Started Guide](docs/getting-started.md)** - Quick start and basic concepts
- **[User Guide](docs/user-guide.md)** - Complete usage documentation
- **[Configuration Reference](docs/configuration.md)** - Detailed configuration options

### Project Information
- **[ROADMAP](ROADMAP.md)** - Project evolution and future plans
- **[CHANGELOG](CHANGELOG.md)** - Version history
- **[CONTRIBUTING](CONTRIBUTING.md)** - How to contribute
- **[SECURITY](SECURITY.md)** - Security policy

### Additional Resources
- **[Documentation Index](docs/README.md)** - Complete documentation overview
- **[Design History](docs/design-archive/)** - Historical design documents
- **[Examples](examples/)** - Real-world configuration examples *(private examples directory)*

---

## Core Concepts

### Modules
Terraform configurations that manage related resources.

### Instances  
Specific executions of modules with their own workspaces and variables.

### Dependencies
Define execution order - modules run in the correct sequence automatically.

### Pipelines
Reusable deployment workflows with multiple stages.

---

## Features in Detail

### Parallel Execution

tf-pipboy automatically runs independent modules in parallel:

```
Stage 1: networking (runs alone)
Stage 2: database-primary, database-replica (run in parallel)
Stage 3: application (runs after database-primary)
```

Control concurrency:
```bash
tfpipboy --operation apply --targets-all --concurrent 10
```

### Authentication Monitoring

Real-time authentication status:

```
╭─────────────────────── Authentication ────────────────────────╮
│ ✓ AWS     (account: 123456789012)                            │
│ ✓ Azure   (subscription: prod-subscription)                   │
│ ✗ GCP     (not authenticated)                                │
│ ✓ GitHub  (user: yourname)                                   │
╰────────────────────────────────────────────────────────────────╯
```

### Live Board Interface

Beautiful TUI showing real-time progress:

```
╭─────────────────────── Execution Progress ────────────────────╮
│ [✓] networking            Completed    45s                    │
│ [→] database-primary      Running      23s                    │
│ [→] database-replica      Running      23s                    │
│ [⧖] application           Waiting                             │
╰────────────────────────────────────────────────────────────────╯
```

### Multi-Region Support

Deploy to multiple regions easily:

```yaml
instances:
  - module: regional-app
    workspace: "prod-us-east-1-app"
    variables:
      region: us-east-1
      
  - module: regional-app
    workspace: "prod-eu-west-1-app"
    variables:
      region: eu-west-1
```

---

## Example Commands

```bash
# Plan all modules
tfpipboy --operation plan --targets-all

# Apply specific modules
tfpipboy --operation apply --targets networking,database

# Execute a pipeline
tfpipboy --pipeline deploy-all --env production

# Dry run (show execution plan)
tfpipboy --operation apply --targets-all --dry-run

# High concurrency
tfpipboy --operation apply --targets-all --concurrent 10

# Verbose logging
tfpipboy --operation plan --targets-all --verbose
```

---

## Project Structure

```
tf-pipboy/
├── cmd/
│   └── tfpipboy/          # Main application
├── pkg/
│   ├── orchestrator/      # Orchestration engine
│   ├── auth/              # Authentication detection
│   ├── terraform/         # Terraform integration
│   └── cli/               # CLI interface
├── docs/                  # User documentation
│   ├── getting-started.md
│   ├── user-guide.md
│   ├── configuration.md
│   ├── design-archive/    # Historical design docs
│   └── archive/           # Historical status reports
├── ROADMAP.md            # Project roadmap
├── SECURITY.md           # Security policy
└── CHANGELOG.md          # Version history
```

---

## Development

### Prerequisites

- Go 1.21 or later
- Terraform 1.0.0 or later
- Make (optional but recommended)

### Setup

```bash
# Clone repository
git clone https://github.com/StanleyXie/tf-pipboy.git
cd tf-pipboy

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint
```

### Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development workflow
- Code style guidelines  
- Testing requirements
- Pull request process

---

## Technology Stack

- **Language**: Go 1.21+
- **CLI Framework**: Cobra
- **TUI Framework**: Bubble Tea
- **Styling**: Lip Gloss
- **Build**: GoReleaser

---

## Architecture

tf-pipboy uses an async event-driven architecture:

- **Orchestrator**: Coordinates module execution
- **Dependency Graph**: Determines execution order
- **Parallel Executor**: Manages concurrent jobs
- **Live Board**: Real-time TUI updates
- **Auth Monitor**: Tracks cloud provider authentication

See [Design Archive](docs/design-archive/) for complete architecture history.

---

## Security

tf-pipboy follows security best practices:

- No credential storage
- Read-only authentication checks via official CLI tools
- State management through Terraform (no direct manipulation)
- Comprehensive input validation

See [SECURITY.md](SECURITY.md) for our security policy and reporting vulnerabilities.

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Excellent TUI framework
- [Cobra](https://github.com/spf13/cobra) - Powerful CLI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- Inspired by [Terragrunt](https://github.com/gruntwork-io/terragrunt)

---

## Links

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/StanleyXie/tf-pipboy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/StanleyXie/tf-pipboy/discussions)
- **Releases**: [GitHub Releases](https://github.com/StanleyXie/tf-pipboy/releases)

---

**Ready to orchestrate?** Get started with the [Quick Start Guide](docs/getting-started.md)!
