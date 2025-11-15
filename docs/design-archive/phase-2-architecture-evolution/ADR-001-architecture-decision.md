# ADR-001: Architecture Decision for tfpipboy

**Status**: Accepted  
**Date**: 2025-10-10  
**Decision Makers**: Stanley Xie, Claude Code (Analysis)  
**Supersedes**: N/A

---

## Context

We need to build **tfpipboy**, a context-aware Terraform CLI tool that provides real-time visibility into:
- Terraform runtime environment (backend, state, workspace, module)
- Cloud provider authentication status (AWS, Azure, GCP, GitHub)
- OS-level environment variables
- Multi-session orchestration capabilities

The tool should enhance developer productivity when working with Terraform by providing persistent context awareness and reducing cognitive load.

---

## Decision Drivers

1. **Time to Market**: Need MVP quickly to validate concept
2. **User Adoption**: Must work in users' existing workflows
3. **Compatibility**: Should work in any terminal, any platform
4. **Complexity**: Manageable by small team (1-2 developers)
5. **Stability**: Production-ready technology
6. **Extensibility**: Easy to add features based on feedback
7. **Performance**: Real-time updates without blocking
8. **Maintenance**: Low ongoing burden

---

## Options Considered

### Option 1: CLI Wrapper
**Description**: Simple command wrapper `tfpipboy plan` that shows context before executing `terraform plan`

**Pros**:
- Simplest to implement (1-2 weeks)
- Universal compatibility
- Easy adoption (alias tf=tfpipboy)

**Cons**:
- No persistent status display
- Cannot track session history
- Manual invocation required

**Score**: 3.5/5

---

### Option 2: Shell Hooks Integration
**Description**: Integrate with shell (bash/zsh/fish) using preexec/precmd hooks for automatic command interception

**Pros**:
- Automatic command tracking
- Persistent status in prompt
- Works in any terminal

**Cons**:
- Shell-specific code
- More complex than CLI wrapper
- Limited to shell environments

**Score**: 4.0/5

---

### Option 3: TUI Framework Application
**Description**: Full TUI application using Go + Bubble Tea or Python + Textual with split panes for status and terminal

**Pros**:
- Professional, polished interface
- Persistent status bar with auto-refresh
- Multi-session orchestration
- Rich interactivity
- Universal terminal compatibility

**Cons**:
- More complex than CLI wrapper
- Takes over terminal window
- Learning curve for TUI framework

**Score**: 4.8/5

---

### Option 4: Terminal Emulator Wrapper (Fork Ghostty)
**Description**: Build custom terminal by forking Ghostty or embedding libghostty

**Pros**:
- Deepest integration possible
- Native performance (GPU-accelerated)
- Complete control over UX

**Cons**:
- Extremely complex (5,000-20,000 LOC)
- 6-12 months development time
- Ghostty-only (forces terminal switch)
- Unstable APIs (libghostty not ready)
- High maintenance burden
- Very high risk

**Score**: 2.0/5

---

### Option 5: Ghostty Plugin/Extension
**Description**: Build plugin for Ghostty using scripting API

**Status**: **Not Available**
- Ghostty has no plugin system (as of 2025)
- Scripting API only in discussion, not committed
- Timeline unknown

**Score**: N/A (not viable)

---

## Decision

**CHOSEN: Option 3 - TUI Framework Application**

**Implementation**: Go + Bubble Tea

---

## Rationale

### Why TUI Framework?

**1. Best Balance of Capability and Complexity**
- Professional UX with persistent status bar
- Multi-session orchestration
- Real-time updates
- Only 500-1,500 lines of code
- 2-4 weeks to MVP

**2. Universal Compatibility**
- Works in ANY terminal (iTerm, Alacritty, Ghostty, Terminal.app, etc.)
- Works over SSH
- Cross-platform (macOS, Linux, Windows)
- No forced tool switching

**3. Production-Ready Technology**
- Bubble Tea: Battle-tested (used by kubectl, Glow, etc.)
- Stable API (no breaking changes expected)
- Excellent documentation and community
- Proven in production

**4. User Adoption**
- Low barrier to entry
- Can coexist with normal terraform workflow
- Optional TUI mode vs CLI mode
- Easy to try

**5. Low Risk**
- Predictable timeline
- Known technology
- Easy to maintain
- No vendor lock-in

---

### Why Go + Bubble Tea?

**Language Choice: Go**
- Single binary distribution (no runtime needed)
- Fast startup (<100ms)
- Low memory footprint (~20MB)
- Strong concurrency (goroutines for multi-session)
- Standard in DevOps tooling (Terraform, kubectl, Docker use Go)
- Cross-compilation support

**Framework Choice: Bubble Tea**
- Production-proven (Glow: 20k+ stars, used in kubectl)
- Elm architecture (clean, predictable state management)
- Excellent performance
- Active community
- Rich ecosystem (Lip Gloss for styling, Bubbles for components)

---

### Why NOT Ghostty Integration?

**Critical Blockers**:
1. **Not Available**: No plugin system, no stable API (as of 2025)
2. **Extreme Complexity**: Would require forking entire terminal (10,000+ LOC)
3. **Long Timeline**: 6-12 months minimum vs 2-4 weeks for TUI
4. **Ghostty-Only**: Forces users to switch terminals (low adoption)
5. **High Risk**: Unstable APIs, maintenance burden, vendor lock-in
6. **Overkill**: GPU acceleration not needed for terraform monitoring

**Research Findings**:
- libghostty API: "not yet a stable API and has not been released"
- Scripting API: "exploring multiple approaches... not yet committed to implementation"
- Would need to fork and maintain entire terminal emulator

**Opportunity Cost**: While spending 12 months on Ghostty integration, could ship TUI version, get users, iterate, and build many more features.

---

### Why NOT CLI Wrapper Only?

**Limitations**:
- No persistent status display
- Cannot monitor multiple sessions
- No real-time updates
- Less professional UX

**Decision**: TUI provides significantly better UX for only moderate additional complexity

---

### Why NOT Shell Hooks?

**Limitations**:
- Shell-specific code (bash, zsh, fish)
- Limited display capabilities (constrained to prompt)
- Complex to implement cross-shell

**Decision**: TUI provides richer UI for similar complexity

---

## Implementation Plan

### Phase 1: MVP (Weeks 1-2)
**Language**: Go  
**Framework**: Bubble Tea  
**Features**:
- Basic TUI with status bar
- Azure CLI authentication detection
- GitHub (gh) authentication detection
- Terraform context detection (workspace, backend, module)
- OS environment variable extraction (TF_VAR_*)
- Directory change detection
- Real-time status refresh (5-second interval)

### Phase 2: Enhancement (Weeks 3-4)
- Multi-session monitoring
- Additional auth providers (AWS, GCP)
- Session orchestration
- Enhanced UI (tabs, improved layout)

### Phase 3: Production (Month 2)
- Testing and polish
- Documentation
- Binary releases
- Homebrew formula

---

## Consequences

### Positive

✅ **Fast Time to Market**: MVP in 2-4 weeks  
✅ **Universal Compatibility**: Works in any terminal  
✅ **Professional UX**: Rich, interactive interface  
✅ **Low Risk**: Proven technology, predictable timeline  
✅ **Easy Maintenance**: Stable APIs, clear codebase  
✅ **High Adoption Potential**: Low barrier to entry  
✅ **Extensible**: Easy to add features  
✅ **Multi-Session Support**: Can orchestrate terraform operations  

### Negative

⚠️ **Learning Curve**: Team must learn Bubble Tea framework (medium effort)  
⚠️ **Terminal Takeover**: TUI takes over terminal window (but can still provide CLI mode)  
⚠️ **No GPU Acceleration**: Slightly lower performance than native terminal (but sufficient)  

### Neutral

🔄 **Can evolve later**: If Ghostty APIs stabilize, can add Ghostty-specific enhancements as bonus features while keeping TUI as primary

---

## Alternatives Reconsidered

### Future Possibilities

**If these become available, reconsider**:
- Ghostty stable plugin API → Add Ghostty-specific features
- libghostty stable release → Consider embedding for advanced features
- Large user base requesting specific terminal integration → Evaluate demand

**But**: TUI remains primary implementation, terminal-specific features are bonuses

---

## Validation

### Success Metrics

**Technical**:
- ✅ MVP completed in 2-4 weeks
- ✅ Works in iTerm, Alacritty, Ghostty, Terminal.app
- ✅ Status updates within 5 seconds
- ✅ <100ms startup time
- ✅ <50MB memory usage

**User**:
- ✅ Easy to install (single binary or brew install)
- ✅ Works without configuration
- ✅ Enhances existing workflow
- ✅ Positive user feedback

### Risk Mitigation

**Risk**: Bubble Tea proves inadequate  
**Mitigation**: Can pivot to Textual (Python) with similar architecture

**Risk**: Performance issues with large output  
**Mitigation**: Implement pagination and buffering

**Risk**: Low user adoption  
**Mitigation**: Provide both TUI and CLI modes, make TUI optional

---

## Research References

1. **Technology Stack Decision** (design/01-technology-stack-decision.md)
   - Language comparison matrix: Go (8.4/10) vs Python (8.1/10) vs Rust (7.4/10)
   - Framework analysis: Cobra, Bubble Tea, Textual, Rich

2. **Open Source Components** (design/02-open-source-components.md)
   - Bubble Tea: Production-proven in Glow, kubectl
   - Go ecosystem analysis

3. **Architecture Design** (design/03-architecture-design.md)
   - Complete system architecture
   - Component specifications
   - Performance targets: <200ms overhead

4. **Ghostty Integration Analysis** (design/04-ghostty-integration.md)
   - Research findings: No plugin system, unstable API
   - Integration complexity assessment

5. **Architecture Alternatives** (design/05-architecture-alternatives.md)
   - CLI vs Terminal vs Shell Hooks comparison
   - Complexity and timeline analysis

6. **TUI vs Ghostty Deep Comparison** (design/06-tui-vs-ghostty-deep-comparison.md)
   - Comprehensive analysis across 10 criteria
   - Final scoring: TUI 4.8/5 vs Ghostty 2.0/5
   - Implementation complexity: 500-1,500 LOC vs 5,000-20,000 LOC
   - Timeline: 2-4 weeks vs 6-12 months

---

## Approval

**Decision Made By**: Stanley Xie  
**Date**: 2025-10-10  
**Status**: Accepted  

**Next Steps**:
1. Set up Go project structure
2. Initialize Bubble Tea application
3. Implement MVP features (Phase 1)
4. User testing and feedback
5. Iterate based on feedback

---

## Revision History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2025-10-10 | Initial ADR | Stanley Xie, Claude Code |

---

## Notes

This decision is based on extensive research and analysis conducted over multiple sessions, including:
- Technology stack evaluation
- Open-source component analysis
- Architecture design
- Multiple approach comparisons
- Risk assessment
- Timeline estimation

The decision prioritizes pragmatism over perfection: ship a working, useful product quickly, then iterate based on real user feedback rather than spending months building a theoretically superior but riskier solution.

**Key Principle**: "Perfect is the enemy of good. Ship the TUI version now, enhance later."
