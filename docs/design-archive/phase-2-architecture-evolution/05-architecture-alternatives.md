# Architecture Alternatives: CLI Wrapper vs Terminal Emulator

**Date**: 2025-10-10  
**Purpose**: Analyze two fundamentally different architectural approaches for tf-pipboy

## The Question

**Should tf-pipboy be:**
1. A **CLI wrapper** that wraps `terraform` commands, OR
2. A **terminal emulator wrapper** that intercepts all commands in a custom terminal?

---

## Approach 1: CLI Wrapper (Current Design)

### Architecture
```
User types: tfpipboy plan
         ↓
   CLI Wrapper (tf-pipboy)
         ↓
   Detect Context (read files, check auth)
         ↓
   Display Status
         ↓
   Execute: terraform plan
```

### How It Works
- User replaces `terraform` with `tfpipboy` in their commands
- tf-pipboy detects context before each command
- Passes through to real terraform
- Works in any terminal emulator

### Pros ✅

1. **Simple to Implement**
   - Just a CLI tool (weeks to build)
   - Uses existing terminal
   - No complex terminal emulation

2. **Universal Compatibility**
   - Works in ANY terminal (iTerm, Alacritty, Ghostty, etc.)
   - No special terminal required
   - Users keep their preferred terminal

3. **Easy Distribution**
   - Single binary
   - `brew install tfpipboy`
   - No configuration needed

4. **Minimal User Change**
   - Alias: `alias tf=tfpipboy`
   - Or just type `tfpipboy` instead of `terraform`

5. **Focused Scope**
   - Only Terraform-aware
   - Clear, single purpose
   - Easy to maintain

### Cons ❌

1. **Manual Invocation**
   - User must remember to use `tfpipboy`
   - Doesn't catch plain `terraform` commands
   - Can't track commands run without wrapper

2. **Limited Context**
   - Only sees context at command time
   - Can't track command history
   - Doesn't know what happened before

3. **No Persistent State**
   - Context display only during command
   - No always-visible status bar
   - Need to run command to see context

4. **Can't Track Other Tools**
   - Only knows about terraform commands
   - Misses: `aws`, `az`, `kubectl`, etc.
   - No holistic environment view

---

## Approach 2: Terminal Emulator Wrapper

### Architecture
```
User starts: tf-pipboy-terminal
         ↓
   Custom Terminal Emulator
   (based on Ghostty/libghostty)
         ↓
   Command Interceptor
         ↓
   Parse ALL commands → Update State
         ↓
   Execute Command → Track Output
         ↓
   Display Context (persistent status bar)
```

### How It Works
- User launches tf-pipboy as their terminal
- Terminal intercepts every command typed
- Builds state by watching commands and output
- Persistent context display (status bar)
- No need to wrap individual commands

### Pros ✅

1. **Automatic Context Tracking**
   - Sees ALL commands (`terraform`, `aws`, `cd`, etc.)
   - No user action required
   - Tracks entire session history

2. **Persistent State Display**
   - Always-visible status bar
   - Real-time updates
   - No need to run command to see context

3. **Holistic Environment View**
   - Track: terraform, aws, az, gcloud, kubectl, cd, etc.
   - Know: current directory, last commands, errors
   - Context from multiple tools

4. **Smarter Context Detection**
   - Watch `terraform init` → know backend changed
   - Watch `terraform workspace select` → know workspace changed
   - Watch `cd` → know module changed
   - Parse output for errors/warnings

5. **Better UX**
   - No command prefix needed
   - Just type `terraform plan` naturally
   - Terminal handles everything

6. **Future Possibilities**
   - Command suggestions
   - Error highlighting
   - Interactive resource browser
   - Command history with context

### Cons ❌

1. **Much More Complex**
   - Need to build/embed terminal emulator
   - Months of development (not weeks)
   - Complex state management

2. **Requires Custom Terminal**
   - User must switch from their terminal
   - Can't use iTerm/Alacritty/etc. features
   - Lock-in to your terminal

3. **Command Parsing Challenges**
   - Need to parse output of every tool
   - Brittle: breaks when tools change output
   - Complex: handle aliases, functions, scripts

4. **Distribution Complexity**
   - Full application (not just CLI tool)
   - macOS/Linux/Windows builds
   - Larger binary, more dependencies

5. **Maintenance Burden**
   - Keep up with Ghostty updates
   - Handle terminal emulator bugs
   - Support diverse terminal features

6. **User Adoption Barrier**
   - "Yet another terminal to try"
   - Learning curve for new terminal
   - Reluctance to switch terminals

7. **libghostty Not Stable Yet**
   - As of 2025, libghostty API not stable
   - Would need to track Ghostty internals
   - Risk of breaking changes

---

## Technical Implementation Comparison

### CLI Wrapper Implementation

**Technology**: Go with Cobra
**Complexity**: Low
**Timeline**: 2-4 weeks for MVP

```go
// Pseudocode
func main() {
    args := os.Args[1:]  // Get terraform args
    
    // Detect context (parallel)
    context := detectContext()
    
    // Display
    displayContext(context)
    
    // Execute terraform
    cmd := exec.Command("terraform", args...)
    cmd.Run()
}
```

**Lines of Code**: ~1,000-2,000

---

### Terminal Emulator Implementation

**Technology**: Zig + libghostty (or fork Ghostty)
**Complexity**: High
**Timeline**: 3-6 months for MVP

```zig
// Pseudocode - much simplified
const Terminal = @import("libghostty");

fn onCommand(command: []const u8) void {
    // Parse command
    if (startsWith(command, "terraform")) {
        parseTerrraformCommand(command);
    } else if (startsWith(command, "cd")) {
        updateWorkingDir(command);
    }
    // ... handle dozens of commands
    
    // Update state
    updateContext();
    
    // Update status bar
    renderStatusBar();
    
    // Execute command
    executeInShell(command);
    
    // Parse output
    parseCommandOutput();
}
```

**Lines of Code**: ~10,000-20,000+

---

## Ghostty Plugin Possibility

**Research Finding**: Ghostty does **NOT** have a plugin system (as of 2025).

### Why No Plugin Route?

Ghostty focuses on:
- Fast, native, feature-rich terminal
- NOT on extensibility via plugins

Unlike Zellij (which has WebAssembly plugins), Ghostty doesn't offer plugin architecture.

### Could You Fork Ghostty?

**Yes, but:**
- Large codebase to maintain
- Would diverge from upstream
- Lose benefit of Ghostty updates
- Significant maintenance burden

**Better Alternative**: Wait for libghostty to stabilize

---

## Shell Integration Alternative

### Approach 2.5: Shell Hooks (Middle Ground)

Instead of full terminal emulator, use **shell integration**:

```bash
# In .bashrc or .zshrc
preexec() {
    # Called before every command
    tfpipboy-hook pre "$1"
}

precmd() {
    # Called before prompt
    tfpipboy-hook post "$?"
}

# Automatic alias
alias terraform="tfpipboy"
```

### How It Works
- Hook into shell (bash/zsh/fish)
- Intercept commands via shell hooks
- Update persistent state file
- Display context in prompt (PS1)

### Pros ✅
- Sees all commands (like terminal approach)
- Works in any terminal
- Much simpler than terminal emulator
- Can track command history

### Cons ❌
- Requires shell configuration
- Only works in supported shells
- Can't intercept non-shell commands
- Limited to prompt customization

---

## Recommendation Matrix

| Criterion | CLI Wrapper | Terminal Emulator | Shell Hooks |
|-----------|-------------|-------------------|-------------|
| **Complexity** | ⭐ Low | ⭐⭐⭐⭐⭐ Very High | ⭐⭐ Medium |
| **Development Time** | 2-4 weeks | 3-6 months | 1-2 weeks |
| **User Adoption** | ⭐⭐⭐⭐ Easy | ⭐⭐ Hard | ⭐⭐⭐ Medium |
| **Context Tracking** | ⭐⭐ Limited | ⭐⭐⭐⭐⭐ Excellent | ⭐⭐⭐⭐ Good |
| **Persistent Display** | ❌ No | ⭐⭐⭐⭐⭐ Yes | ⭐⭐⭐⭐ Yes (in prompt) |
| **Compatibility** | ⭐⭐⭐⭐⭐ Universal | ⭐⭐ Terminal lock-in | ⭐⭐⭐⭐ Shell-dependent |
| **Maintenance** | ⭐⭐⭐⭐⭐ Easy | ⭐⭐ Complex | ⭐⭐⭐⭐ Moderate |
| **Risk** | ⭐⭐⭐⭐⭐ Low | ⭐⭐ High | ⭐⭐⭐ Medium |

---

## Detailed Analysis: Your Proposal

### "Terminal emulator wrapped with environment aware functions"

**You're right that it WOULD be simpler for context tracking**, but:

1. **Context Tracking**: ⭐⭐⭐⭐⭐ Excellent
   - See every command
   - Track state changes
   - Parse outputs

2. **Implementation Complexity**: ⭐ Very Hard
   - Need terminal emulator knowledge
   - Complex state management
   - Parsing many tools' outputs

3. **User Experience**: ⭐⭐⭐ Mixed
   - Great: Automatic, persistent status
   - Bad: Must switch terminals

**The Trade-off**: 
- **Simpler** context tracking logic
- **Much harder** overall implementation
- **Worse** user adoption

---

## Recommended Approach

### Phase 1: CLI Wrapper (Now) ✅

**Why:**
- Fast to build (2-4 weeks)
- Validates concept
- Low risk
- Easy user adoption
- Get feedback quickly

**Goal**: Prove value of context-aware Terraform

---

### Phase 2: Shell Hooks (3-6 months)

**Why:**
- Better context tracking
- Works in any terminal
- Persistent status display
- Not too complex

**Implementation:**
```bash
# User installs
tfpipboy init bash  # or zsh, fish

# Adds to .bashrc:
source ~/.tfpipboy/shell-integration.sh

# Now:
# - Automatic terraform→tfpipboy alias
# - Persistent status in prompt
# - Tracks all commands
```

---

### Phase 3: Consider Terminal Emulator (Year 2+)

**Only if:**
1. libghostty API stabilizes
2. CLI wrapper proves very popular
3. Team grows (3+ developers)
4. Users demand terminal features

**Approach**: Contribute to Ghostty for plugin system
- Propose Ghostty plugin architecture
- Contribute upstream
- Benefit entire Ghostty community

---

## Example: What Others Do

### Similar Tools

| Tool | Approach |
|------|----------|
| **Starship** | Shell hooks (prompt) |
| **Oh My Zsh** | Shell hooks |
| **direnv** | Shell hooks |
| **Terragrunt** | CLI wrapper |
| **kubectl** | CLI (with shell completion) |
| **Warp** | Custom terminal (struggled with adoption) |

**Pattern**: Shell hooks are sweet spot for developer tools

---

## Answer to Your Question

> "Is it will be much simpler to have all context be tracked?"

**Yes, context tracking logic is simpler, BUT:**

❌ Overall implementation is **much harder**
❌ User adoption is **much lower**
❌ Development time is **10x longer**
❌ Maintenance burden is **much higher**

**Better Question**: 
> "What's the simplest way to get persistent context display?"

**Answer**: Shell hooks (Phase 2)

---

## Ghostty Integration Path

### Current Reality (2025)

1. **No plugin system**: Ghostty doesn't support plugins
2. **libghostty not stable**: API still changing
3. **Would need fork**: Major maintenance burden

### Future Path (2026+)

1. **Watch libghostty**: Wait for stable release
2. **Contribute upstream**: Propose Ghostty enhancements
3. **Community approach**: Build ecosystem, not fork

### What You CAN Do Now with Ghostty

Use Ghostty's **existing features**:

```yaml
# ghostty config
shell-integration = true

# Custom keybinding to show tf-pipboy status
keybind = ctrl+shift+t=text:tfpipboy status\n
```

Then in your shell:
```bash
# Add to .zshrc
precmd() {
    # Update terminal title
    print -Pn "\e]0;tf-pipboy: $(tfpipboy status --short)\a"
}
```

**Result**: Best of both worlds
- Keep Ghostty as terminal
- tf-pipboy shows in title bar
- No custom terminal needed

---

## Final Recommendation

### 🎯 Start with Phase 1: CLI Wrapper

**Rationale:**
1. **Validate concept** in 2-4 weeks
2. **Low risk**, high learning
3. **Easy adoption**
4. **Can evolve** to shell hooks later

### 🔄 Evolution Path

```
Phase 1: CLI Wrapper (NOW)
    ↓ (3-6 months)
Phase 2: + Shell Hooks
    ↓ (when libghostty stable)
Phase 3: Consider libghostty embedding
```

### ❌ Do NOT start with terminal emulator

**Unless:**
- You have 6+ months
- You're expert in terminal emulation
- You're willing to maintain terminal emulator
- You can accept low user adoption

---

## Code Reusability

**Good news**: Context detection code is **reusable** across all approaches!

```
Context Detection Logic (same for all)
├── Terraform context (backend, workspace)
├── Auth status (AWS, Azure, GCP, GitHub)
└── System info (git, env vars)
```

**Strategy**: Build context detection library first
- Use in CLI wrapper (Phase 1)
- Reuse in shell hooks (Phase 2)
- Reuse if building terminal (Phase 3)

---

## Conclusion

Your **intuition is correct**: A terminal emulator wrapper WOULD have better context tracking.

**But the trade-offs don't favor it**:
- 10x development time
- Much harder to maintain
- Lower user adoption
- Higher risk of failure

**Better path**: 
1. CLI wrapper (quick win)
2. Shell hooks (better context, still accessible)
3. Watch libghostty (future opportunity)

**You can always evolve later**, but you can't un-spend 6 months building a terminal emulator that nobody adopts.

---

## Decision Framework

Choose **CLI Wrapper** if:
- ✅ Want fast MVP
- ✅ Want to validate concept
- ✅ Limited time/resources
- ✅ Want easy user adoption

Choose **Shell Hooks** if:
- ✅ Want persistent display
- ✅ Can wait 3-6 months
- ✅ OK with shell-specific code

Choose **Terminal Emulator** if:
- ✅ Have 6+ months
- ✅ Expert in terminal emulation
- ✅ Have team of 3+ developers
- ✅ Want to build a terminal product (not just Terraform tool)

**Recommended**: Start with CLI Wrapper, evolve to Shell Hooks
