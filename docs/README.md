# tfpipboy Documentation

Welcome to the tfpipboy documentation! This directory contains comprehensive guides, design documents, and release information.

## Quick Start

- **[Getting Started](getting-started.md)** - Installation and first steps
- **[User Guide](user-guide.md)** - Complete usage guide
- **[Configuration](configuration.md)** - Configuration file reference

## Guides

- **[Adding Authentication Providers](guides/adding-authentication-providers.md)** - Extend auth support
- **[Extending TUI](guides/extending-tui.md)** - Customize the terminal UI

## Setup & Configuration

- **[Terraform Setup](TERRAFORM-SETUP.md)** - Terraform integration guide
- **[TUI Implementation Notes](TUI_IMPLEMENTATION_NOTES.md)** - Technical implementation details

## Release Information

- **[Release Notes](releases/)** - Version release notes and changelogs
- **[Manual Release Steps](releases/MANUAL_RELEASE_STEPS.md)** - How to create releases
- **[Local Machine Steps](releases/LOCAL_MACHINE_STEPS.md)** - Local release workflow

## Design Archive

Historical design documents are preserved in [`design-archive/`](design-archive/):

- **Phase 1**: Initial technology stack and architecture decisions
- **Phase 2**: Architecture evolution (Orchestrator-Runner pattern)
- **Phase 3**: Configuration format design
- **Phase 4**: Feature implementations (LiveBoard, parallel execution)

See [design-archive/README.md](design-archive/README.md) for the complete design document index.

## Security

- **[Git History Security](GIT-HISTORY-SECURITY.md)** - Best practices for secure git history

## Archive

Older documentation and completed implementation reports are in [`archive/`](archive/).

## Contributing

See the main [CONTRIBUTING.md](../CONTRIBUTING.md) in the repository root.

## Project Structure

```
docs/
├── README.md                    # This file
├── getting-started.md           # Quick start guide
├── user-guide.md                # Complete user guide  
├── configuration.md             # Configuration reference
├── TERRAFORM-SETUP.md           # Terraform integration
├── TUI_IMPLEMENTATION_NOTES.md  # TUI technical details
├── GIT-HISTORY-SECURITY.md      # Security best practices
├── guides/                      # How-to guides
├── releases/                    # Release documentation
├── design-archive/              # Historical design docs
└── archive/                     # Old documentation
```

## Questions?

- Check the [main README](../README.md)
- Review the [ROADMAP](../ROADMAP.md)
- See [CHANGELOG](../CHANGELOG.md) for version history
