# Open Source Components Analysis

**Date**: 2025-10-10  
**Purpose**: Identify and evaluate open-source libraries for tf-pipboy implementation

## Component Categories

1. **CLI Framework**: Command parsing and structure
2. **TUI/Display**: Terminal UI and status rendering
3. **Subprocess Management**: Execute and capture external commands
4. **Configuration Parsing**: Read Terraform and cloud provider configs
5. **Terminal Integration**: Shell integration and advanced terminal features

---

## 1. CLI Framework Options

### Python: Click

**Repository**: https://github.com/pallets/click  
**License**: BSD-3-Clause  
**Stars**: 15.5k+

**Features:**
- Decorator-based command definition
- Automatic help page generation
- Nested command groups
- Parameter validation
- Shell completion

**Example:**
```python
import click

@click.command()
@click.option('--count', default=1)
@click.argument('name')
def hello(count, name):
    for _ in range(count):
        click.echo(f'Hello {name}!')
```

**Pros:**
- Battle-tested, stable API
- Excellent documentation
- Large community
- Easy to learn

**Cons:**
- Decorator style not everyone's preference
- Some advanced features require boilerplate

---

### Python: Typer

**Repository**: https://github.com/tiangolo/typer  
**License**: MIT  
**Stars**: 15k+

**Features:**
- Built on Click
- Type hints for automatic validation
- Auto-completion out of the box
- FastAPI-like developer experience
- Progressive disclosure (simple → complex)

**Example:**
```python
import typer

app = typer.Typer()

@app.command()
def hello(name: str, count: int = 1):
    for _ in range(count):
        typer.echo(f"Hello {name}!")
```

**Pros:**
- Modern Python (type hints)
- Great developer experience
- Automatic help from type hints
- Less boilerplate than Click

**Cons:**
- Newer than Click (less battle-tested)
- Smaller community

---

### Go: Cobra

**Repository**: https://github.com/spf13/cobra  
**License**: Apache 2.0  
**Stars**: 38k+

**Features:**
- Used by Kubernetes, Docker, GitHub CLI, Hugo
- Nested subcommands
- Persistent and local flags
- Automatic help generation
- Shell completion (bash, zsh, fish, powershell)
- Integration with Viper (configuration)

**Example:**
```go
var rootCmd = &cobra.Command{
    Use:   "tfpipboy",
    Short: "Context-aware Terraform CLI",
    Run: func(cmd *cobra.Command, args []string) {
        // Run terraform command
    },
}

func main() {
    rootCmd.Execute()
}
```

**Pros:**
- Industry standard for Go CLIs
- Excellent documentation
- Active maintenance
- Generator tool (cobra-cli)

**Cons:**
- Verbose compared to Python options
- Requires more setup code

---

### Rust: Clap

**Repository**: https://github.com/clap-rs/clap  
**License**: MIT/Apache 2.0  
**Stars**: 14k+

**Features:**
- Derive macros for declarative CLIs
- Builder pattern alternative
- Automatic help and version flags
- Shell completion generation
- Argument validation

**Example:**
```rust
use clap::Parser;

#[derive(Parser)]
#[command(name = "tfpipboy")]
struct Cli {
    #[arg(short, long)]
    verbose: bool,
    
    command: String,
}
```

**Pros:**
- Very powerful and flexible
- Compile-time validation
- Great error messages

**Cons:**
- Rust learning curve
- More complex than needed

---

## 2. Terminal UI / Display Libraries

### Python: Rich

**Repository**: https://github.com/Textualize/rich  
**License**: MIT  
**Stars**: 49k+

**Features:**
- Beautiful terminal output
- Progress bars with custom columns
- Live display updates (status, tables)
- Syntax highlighting
- Markdown rendering
- Tree structures
- Automatic terminal capability detection

**Key Features for tf-pipboy:**
```python
from rich.console import Console
from rich.live import Live
from rich.table import Table

console = Console()

# Status indicator
with console.status("[bold green]Checking auth status..."):
    check_auth()

# Live updating table
with Live(generate_table(), refresh_per_second=4) as live:
    while True:
        live.update(generate_table())
```

**Pros:**
- Most feature-rich Python terminal library
- Beautiful output out of the box
- Excellent real-time updates
- Great documentation

**Cons:**
- Requires Python runtime
- Some overhead for simple cases

**Recommendation**: ⭐⭐⭐⭐⭐ Best for Python implementation

---

### Python: Textual

**Repository**: https://github.com/Textualize/textual  
**License**: MIT  
**Stars**: 25k+

**Features:**
- Full TUI framework (by Rich authors)
- CSS-like styling
- Reactive components
- Mouse support
- Widget library

**Note**: More complex than needed for tf-pipboy. Consider only if building full TUI interface.

---

### Go: Bubble Tea

**Repository**: https://github.com/charmbracelet/bubbletea  
**License**: MIT  
**Stars**: 27k+

**Features:**
- Based on Elm architecture (Model-Update-View)
- Composable components
- Built-in keybindings
- Works with Lip Gloss (styling) and Bubbles (components)

**Example:**
```go
type model struct {
    status string
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Update state
}

func (m model) View() string {
    return m.status // Render
}
```

**Pros:**
- Modern, well-designed architecture
- Good for interactive CLIs
- Growing ecosystem (Charm suite)

**Cons:**
- Learning curve (Elm pattern)
- More complex than simple printf
- Less feature-rich than Rich for status displays

**Recommendation**: ⭐⭐⭐⭐ Good for Go, but may be overkill

---

### Go: pterm

**Repository**: https://github.com/pterm/pterm  
**License**: MIT  
**Stars**: 4.8k+

**Features:**
- Simple, beautiful output
- Progress bars, spinners, tables
- Less complex than Bubble Tea
- More styled output than fmt.Printf

**Example:**
```go
pterm.Info.Println("Checking Terraform backend...")
spinner := pterm.DefaultSpinner.Start("Loading...")
// ... do work
spinner.Success("Done!")
```

**Pros:**
- Simpler than Bubble Tea
- Beautiful output
- Good for status updates

**Cons:**
- Less powerful for complex UIs
- Smaller community

**Recommendation**: ⭐⭐⭐⭐ Consider for simpler Go approach

---

### Rust: Ratatui

**Repository**: https://github.com/ratatui-org/ratatui  
**License**: MIT  
**Stars**: 10k+

**Features:**
- Successor to tui-rs
- Full-featured TUI framework
- Widgets: tables, charts, lists, gauges
- Multiple backends
- Active development

**Note**: Excellent but overkill for tf-pipboy. Only consider with Rust.

---

## 3. Subprocess Management

### Python: subprocess + asyncio

**Built-in**: Standard library  
**Features:**
- `subprocess.run()` for synchronous execution
- `asyncio.create_subprocess_exec()` for async
- Capture stdout/stderr
- Set environment variables

**Example:**
```python
import subprocess

result = subprocess.run(
    ['aws', 'sts', 'get-caller-identity'],
    capture_output=True,
    text=True,
    timeout=5
)
auth_status = result.returncode == 0
```

**Recommendation**: ⭐⭐⭐⭐⭐ Perfect for Python, built-in

---

### Go: os/exec

**Built-in**: Standard library  
**Features:**
- Execute commands
- Pipe stdin/stdout/stderr
- Concurrent execution with goroutines

**Example:**
```go
cmd := exec.Command("terraform", "init")
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
err := cmd.Run()
```

**Recommendation**: ⭐⭐⭐⭐⭐ Perfect for Go, built-in

---

## 4. Configuration Parsing

### Python: python-hcl2

**Repository**: https://github.com/amplify-education/python-hcl2  
**License**: MIT

**Features:**
- Parse Terraform .tf files
- Returns Python dict
- HCL2 syntax support

**Example:**
```python
import hcl2

with open('backend.tf', 'r') as file:
    config = hcl2.load(file)
    backend = config['terraform'][0]['backend']
```

**Recommendation**: ⭐⭐⭐⭐ Good for Python, parse HCL if needed

---

### Alternative: JSON Parsing

**Approach**: Use `terraform show -json` instead of parsing .tf files

**Pros:**
- No HCL parsing needed
- Official Terraform output format
- More reliable

**Cons:**
- Requires Terraform to be initialized

**Recommendation**: ⭐⭐⭐⭐⭐ Preferred approach for both Python and Go

---

## 5. Existing Terraform Wrappers (Reference)

### Terragrunt

**Repository**: https://github.com/gruntwork-io/terragrunt  
**License**: MIT  
**Language**: Go

**What We Can Learn:**
- DRY Terraform configurations
- Remote state management
- Dependency management between modules
- Configuration structure

**Relevance**: Different use case (orchestration vs context display)

---

### tflocal (Terraform-Local)

**Repository**: https://github.com/localstack/terraform-local  
**License**: Apache 2.0  
**Language**: Python (wrapper script)

**What We Can Learn:**
- Simple Python wrapper approach
- Override mechanism for providers
- Command passthrough pattern

**Relevance**: Similar architecture (wrapper around terraform)

---

## 6. Ghostty Integration

### libghostty (Future)

**Repository**: https://github.com/ghostty-org/ghostty  
**License**: MIT  
**Language**: Zig with C API  
**Status**: Not yet stable for standalone use

**Potential Features:**
- Embed terminal emulation in custom apps
- C API for cross-language support
- Zero dependencies

**Current Ghostty Features We Can Use:**
1. **Shell Integration**: OSC 133 sequences for prompt marking
2. **Custom Status Display**: Using shell integration features
3. **Working Directory Reporting**: Automatic context tracking

**Integration Strategy:**

#### Phase 1 (Now): Shell Integration
```bash
# Use Ghostty's shell integration for enhanced display
# Configure custom prompt to show tf-pipboy context
export PS1="[tf-pipboy: $TF_WORKSPACE] $PS1"
```

#### Phase 2 (Future): libghostty Embedding
- Wait for libghostty stable release
- Embed terminal emulator in tf-pipboy
- Custom rendering for Terraform status
- Advanced features (clickable resources, inline previews)

**Recommendation**: 
- ⭐⭐⭐ (Now): Use Ghostty's shell integration features
- ⭐⭐⭐⭐⭐ (Future): Monitor libghostty for embedding opportunities

---

## Recommended Component Stack

### Option A: Python (Fast Development)

```
CLI:        Typer (modern, type-safe)
Display:    Rich (beautiful, feature-rich)
Subprocess: asyncio + subprocess (built-in)
Config:     json.loads (terraform show -json)
Ghostty:    Shell integration + custom prompts
```

**Total External Dependencies**: 2 (typer, rich)

---

### Option B: Go (Production Ready)

```
CLI:        Cobra (industry standard)
Display:    pterm (simple, beautiful) OR Bubble Tea (powerful)
Subprocess: os/exec (built-in)
Config:     encoding/json (terraform show -json)
Ghostty:    Shell integration + custom prompts
```

**Total External Dependencies**: 2-3 (cobra, pterm or bubbletea)

---

## Installation & Distribution

### Python: pipx

**Command**: `pipx install tfpipboy`

**Pros:**
- Isolated environment
- Global command availability
- Easy updates

**Cons:**
- Requires Python + pipx installed

---

### Go: Binary Release

**Command**: Download binary from GitHub releases

**Pros:**
- Single file, no runtime
- Works everywhere
- Fast startup

**Cons:**
- Need to build for multiple platforms

---

## Next Steps

1. **Choose Stack**: Based on team preferences and timeline
2. **Setup Project**: Initialize with chosen framework
3. **Create POC**: 
   - Command passthrough
   - One auth checker
   - Simple status display
4. **Iterate**: Add features incrementally

## References

- [Rich Documentation](https://rich.readthedocs.io/)
- [Cobra Documentation](https://cobra.dev/)
- [Typer Documentation](https://typer.tiangolo.com/)
- [Ghostty Shell Integration](https://ghostty.org/docs/features/shell-integration)
- [Bubble Tea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)
