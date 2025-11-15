# Ghostty Integration Strategy for tfpipboy

**Date**: 2025-10-10  
**Purpose**: Define how tfpipboy can leverage Ghostty terminal emulator features

## What is Ghostty?

**Ghostty** is a modern, fast, GPU-accelerated terminal emulator that:
- Uses platform-native UI (Swift/AppKit on macOS, GTK4 on Linux)
- Provides extensive terminal protocol support (Kitty graphics, synchronized rendering)
- Offers rich shell integration capabilities
- Is built on **libghostty**, a C-compatible embeddable library
- Emphasizes standards compliance and performance

**Website**: https://ghostty.org/  
**Repository**: https://github.com/ghostty-org/ghostty  
**License**: MIT

---

## Integration Opportunities

### Phase 1: Shell Integration (Available Now)

Ghostty provides automatic shell integration for bash, zsh, fish, and elvish with features we can leverage immediately.

#### 1.1 Prompt Marking (OSC 133)

**What it is**: Ghostty implements OSC 133 semantic markup to mark prompt boundaries and command execution phases.

**How tfpipboy can use it:**
```bash
# Emit OSC 133 sequences to mark tfpipboy context display
printf '\e]133;A\e\\'  # Mark prompt start
echo "tfpipboy context: production @ aws/vpc"
printf '\e]133;B\e\\'  # Mark prompt end

# Then run terraform
terraform plan
```

**Benefits:**
- Allows users to jump between prompts with keybindings
- Ghostty can visually distinguish tfpipboy output from terraform output
- Better scrollback navigation

**Implementation Priority**: ⭐⭐⭐ Medium (nice to have)

---

#### 1.2 Working Directory Reporting

**What it is**: Ghostty tracks the current working directory through shell integration.

**How tfpipboy can use it:**
- Ghostty already knows the current directory
- tfpipboy can use this for Terraform module detection
- Potential for future integration with Ghostty's directory tracking

**Benefits:**
- Consistent with Ghostty's native directory awareness
- Could enable Ghostty UI features (title bar, tabs) to show Terraform context

**Implementation Priority**: ⭐⭐ Low (Ghostty handles this automatically)

---

#### 1.3 Custom Escape Sequences

**What it is**: Ghostty supports various terminal escape sequences for customization.

**How tfpipboy can use it:**

```bash
# Set terminal title to show Terraform context
printf '\e]0;tfpipboy: production @ aws/vpc\e\\'

# Change cursor color based on auth status
# Green cursor = all auth OK, red = missing auth
printf '\e]12;#00ff00\e\\'  # Green cursor
```

**Benefits:**
- Visual feedback in terminal title bar
- Cursor color as quick auth status indicator
- Works in Ghostty and many other terminals

**Implementation Priority**: ⭐⭐⭐⭐ High (easy to implement, useful)

---

#### 1.4 Status Bar / Terminal Title

**What it is**: Update terminal title to show tfpipboy context persistently.

**Implementation:**
```python
def set_terminal_title(context: TerraformContext, auth: AuthStatus):
    """Set terminal title with current context"""
    title = f"tfpipboy: {context.workspace} @ {context.module}"
    
    if not auth.all_ok():
        title += " ⚠️ Auth Issues"
    
    # ANSI escape sequence for terminal title
    print(f"\033]0;{title}\007", end='')
```

**Example Terminal Title:**
```
tfpipboy: production @ aws/vpc ✓
```

**Benefits:**
- Always visible (even when scrolled up)
- Works in Ghostty and most modern terminals
- Low overhead, high visibility

**Implementation Priority**: ⭐⭐⭐⭐⭐ Very High (easy, highly visible)

---

### Phase 2: Enhanced Display (Ghostty-Specific)

These features work best in Ghostty but degrade gracefully in other terminals.

#### 2.1 Synchronized Rendering

**What it is**: Ghostty supports synchronized rendering (DEC mode 2026) to prevent tearing during updates.

**How tfpipboy can use it:**
```python
def display_context_with_sync():
    # Begin synchronized update
    print("\033[?2026h", end='')
    
    # Render entire context display
    render_context_header()
    
    # End synchronized update
    print("\033[?2026l", end='')
```

**Benefits:**
- Smooth, tear-free context display updates
- Professional appearance
- Fallback: Just works without sync in other terminals

**Implementation Priority**: ⭐⭐⭐ Medium (polish feature)

---

#### 2.2 Hyperlinks (OSC 8)

**What it is**: Ghostty supports clickable hyperlinks in terminal output.

**How tfpipboy can use it:**
```python
def make_link(url: str, text: str) -> str:
    """Create clickable terminal link"""
    return f"\033]8;;{url}\033\\{text}\033]8;;\033\\"

# Link to Terraform Cloud workspace
ws_url = f"https://app.terraform.io/app/org/workspaces/{workspace}"
print(f"Workspace: {make_link(ws_url, workspace)}")

# Link to AWS Console
console_url = f"https://console.aws.amazon.com/?region={region}"
print(f"AWS: {make_link(console_url, region)}")
```

**Benefits:**
- Click to open Terraform Cloud workspace
- Click to open AWS/Azure/GCP console
- Enhances workflow efficiency

**Implementation Priority**: ⭐⭐⭐⭐ High (great UX improvement)

---

#### 2.3 Kitty Graphics Protocol

**What it is**: Ghostty supports the Kitty graphics protocol for displaying images in terminal.

**How tfpipboy could use it (future):**
- Display infrastructure diagrams
- Show resource graphs
- Render plan output as visual diff

**Example Use Case:**
```bash
# After terraform plan, show resource graph
tfpipboy plan --with-graph
# → Displays PNG image of planned changes inline
```

**Benefits:**
- Visual representation of complex infrastructure
- Better understanding of dependencies
- Modern terminal capabilities

**Implementation Priority**: ⭐ Very Low (future enhancement)

---

### Phase 3: libghostty Embedding (Future)

**Status**: libghostty is not yet stable for standalone use (as of Ghostty 1.0)

**Long-term Vision**: When libghostty stabilizes, tfpipboy could embed terminal emulation directly.

#### 3.1 Embedded Terminal View

**Concept**: tfpipboy becomes a full application with embedded Ghostty terminal.

```
┌─────────────────────────────────────────────────────────────┐
│ tfpipboy - Context-Aware Terraform Environment             │
├─────────────────────────────────────────────────────────────┤
│ Context Panel                                               │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Module:      aws/vpc                 Workspace: prod    │ │
│ │ Backend:     s3://my-state/vpc                          │ │
│ │ Auth Status: ✓ AWS  ✓ Azure  ✗ GCP  ✓ GitHub          │ │
│ └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│ Terminal (powered by libghostty)                            │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ $ terraform plan                                        │ │
│ │                                                         │ │
│ │ Terraform will perform the following actions:          │ │
│ │   # aws_vpc.main will be created                       │ │
│ │   + resource "aws_vpc" "main" {                        │ │
│ │       + cidr_block = "10.0.0.0/16"                     │ │
│ │     }                                                   │ │
│ │                                                         │ │
│ │ Plan: 1 to add, 0 to change, 0 to destroy.            │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

**Benefits:**
- Always-visible context panel
- Integrated experience
- Custom rendering of Terraform output
- Enhanced interactivity

**Challenges:**
- Requires libghostty to be stable
- Significant development effort
- Need to build UI wrapper (Swift/GTK)
- May conflict with user's terminal preference

**Implementation Priority**: ⭐⭐ Low (wait for libghostty stability, Phase 3+)

---

#### 3.2 Custom Rendering

**Concept**: Use libghostty-vt to parse and re-render Terraform output with enhancements.

**Potential Features:**
- Syntax highlighting for Terraform plan output
- Collapsible sections for large plans
- Inline resource documentation
- Click to open resource in cloud console

**Implementation Priority**: ⭐ Very Low (Phase 3+, experimental)

---

## Ghostty Feature Detection

**Strategy**: Detect if running in Ghostty and enable enhanced features.

```python
def is_ghostty() -> bool:
    """Detect if running in Ghostty terminal"""
    term_program = os.environ.get('TERM_PROGRAM', '')
    return 'ghostty' in term_program.lower()

def get_terminal_capabilities() -> TerminalCapabilities:
    """Detect terminal capabilities"""
    return TerminalCapabilities(
        hyperlinks=supports_osc8(),
        true_color=supports_true_color(),
        synchronized_rendering=is_ghostty(),  # Conservative
        graphics=is_ghostty() and check_kitty_graphics(),
    )

# Use capabilities to enhance display
caps = get_terminal_capabilities()
if caps.hyperlinks:
    display_with_links(context)
else:
    display_plain(context)
```

---

## Recommended Implementation Roadmap

### MVP (Phase 1a) - Basic Terminal Integration
**Timeframe**: Week 1-2

✅ **Must Have:**
1. Terminal title updates with Terraform context
2. Color-coded status display (works in all terminals)
3. Basic context header before terraform commands

**Ghostty-Specific:** None required, but works great in Ghostty

---

### v1.0 (Phase 1b) - Enhanced Terminal Features
**Timeframe**: Month 1

✅ **Should Have:**
1. Hyperlinks to cloud consoles (OSC 8)
2. Cursor color indicator for auth status
3. OSC 133 prompt marking for navigation
4. Synchronized rendering for smooth updates

**Ghostty-Specific:** Graceful fallback for non-Ghostty terminals

---

### v2.0 (Phase 2) - Advanced Features
**Timeframe**: Month 3-6

✅ **Nice to Have:**
1. Ghostty-optimized display modes
2. Shell integration hooks
3. Custom keybindings (via Ghostty config)
4. Notification integration

**Ghostty-Specific:** Enhanced experience in Ghostty, still works elsewhere

---

### v3.0+ (Phase 3) - libghostty Integration
**Timeframe**: Year 1+ (wait for libghostty stability)

🔮 **Future:**
1. Embedded terminal view
2. Custom Terraform output rendering
3. Interactive plan inspection
4. Full TUI interface option

**Ghostty-Specific:** Requires libghostty, Ghostty-only features

---

## Configuration for Ghostty Users

### Recommended Ghostty Config

**File**: `~/.config/ghostty/config`

```conf
# tfpipboy optimizations
shell-integration = detect
shell-integration-features = cursor,sudo,title

# Enable hyperlinks
hyperlink = true

# Smooth rendering
sync-updates = true

# Custom keybind to show tfpipboy status
keybind = ctrl+shift+t=text:\u001b]133;A\u001b\\tfpipboy status\u001b]133;B\u001b\\\n

# Optional: Set theme that works well with tfpipboy colors
theme = dark:tokyonight
```

### tfpipboy Configuration for Ghostty

**File**: `~/.tfpipboy/config.yaml`

```yaml
# Ghostty integration settings
ghostty:
  enabled: auto  # auto-detect, true, false
  
  features:
    terminal_title: true      # Update terminal title
    hyperlinks: true          # OSC 8 hyperlinks
    cursor_color: true        # Color cursor based on status
    osc133: true              # Prompt marking
    sync_rendering: true      # Synchronized updates
  
  # Fallback behavior for non-Ghostty terminals
  fallback:
    hyperlinks: false         # Most terminals don't support
    cursor_color: false
    osc133: false
```

---

## Testing in Ghostty

### Manual Testing Checklist

- [ ] Terminal title updates when changing directories
- [ ] Hyperlinks are clickable
- [ ] Cursor color changes with auth status
- [ ] Context display renders without tearing
- [ ] OSC 133 navigation works (jump_to_prompt keybind)
- [ ] Works correctly in non-Ghostty terminals (graceful fallback)

### Test Script

```bash
#!/bin/bash
# test-ghostty-integration.sh

echo "Testing Ghostty Integration Features"
echo

# Test 1: Terminal Title
echo "Test 1: Terminal Title"
printf '\e]0;tfpipboy: test-workspace @ test/module\e\\'
echo "✓ Terminal title should show: tfpipboy: test-workspace @ test/module"
sleep 2

# Test 2: Hyperlink
echo
echo "Test 2: Hyperlink (click should open browser)"
printf '\e]8;;https://app.terraform.io/\e\\Terraform Cloud\e]8;;\e\\\n'
sleep 2

# Test 3: Cursor Color
echo
echo "Test 3: Cursor Color Change"
printf '\e]12;#00ff00\e\\'  # Green
echo "✓ Cursor should be green"
sleep 2

# Test 4: OSC 133 Prompt Marking
echo
echo "Test 4: Prompt Marking"
printf '\e]133;A\e\\'
echo "Marked prompt start"
printf '\e]133;B\e\\'
echo "(Try jump_to_prompt keybind in Ghostty)"
sleep 2

# Test 5: Synchronized Rendering
echo
echo "Test 5: Synchronized Rendering"
printf '\e[?2026h'  # Begin sync
for i in {1..10}; do
  echo "Line $i of synchronized update"
done
printf '\e[?2026l'  # End sync
echo "✓ Should have rendered smoothly without tearing"

echo
echo "All tests complete!"
```

---

## Alternative: Terminal Status Bar Libraries

If not using Ghostty or for broader compatibility, consider these libraries that provide status bar functionality:

### tmux Status Bar Integration

```bash
# Update tmux status bar with tfpipboy context
tmux set-option -g status-right "#[fg=green]#{pane_current_path} #[fg=yellow]production"
```

### Starship Prompt Integration

```toml
# ~/.config/starship.toml
[custom.tfpipboy]
command = "tfpipboy status --format=starship"
when = "test -d .terraform"
symbol = "🚀 "
```

---

## Summary & Recommendations

### For tfpipboy v1.0 (MVP)

**Implement Now:**
1. ✅ Terminal title updates (OSC 0) - High impact, easy
2. ✅ Basic color-coded output - Works everywhere
3. ✅ Hyperlinks (OSC 8) with fallback - Great for Ghostty users

**Skip for Now:**
1. ❌ Kitty graphics - Not essential, limited use case
2. ❌ libghostty embedding - Not stable yet
3. ❌ Custom rendering - Too complex for v1.0

### For Ghostty Users

**Value Proposition:**
- Best-in-class terminal experience for tfpipboy
- Clickable links to cloud consoles
- Smooth context updates
- Visual auth status indicators
- Future: Even deeper integration via libghostty

**Compatibility:**
- tfpipboy works great in any terminal
- Ghostty users get enhanced features automatically
- No lock-in: switching terminals doesn't break functionality

---

## Resources

- [Ghostty Documentation](https://ghostty.org/docs)
- [Ghostty Shell Integration](https://ghostty.org/docs/features/shell-integration)
- [Ghostty Terminal API (VT)](https://ghostty.org/docs/vt)
- [libghostty Announcement](https://mitchellh.com/writing/libghostty-is-coming)
- [OSC 133 Specification](https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md)
- [OSC 8 Hyperlinks](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda)
- [Kitty Graphics Protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
