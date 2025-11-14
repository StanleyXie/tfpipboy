# tf-pipboy MVP Development Plan

**Version**: 1.0  
**Date**: 2025-10-10  
**Target**: 2-Week MVP Sprint  
**Technology**: Go + Bubble Tea TUI Framework

---

## MVP Scope

Build a TUI (Text User Interface) application that provides real-time context awareness for Terraform operations with focus on:
- Azure CLI authentication status
- GitHub CLI (gh) authentication status
- Terraform context (workspace, backend, module)
- OS environment variables (TF_VAR_*)
- Automatic context switching when changing directories

---

## Feature List

### F1: TUI Application Shell
**Priority**: P0 (Critical)  
**Effort**: 2 days

Basic Bubble Tea application with multi-pane layout:
- Top pane: Status bar (auto-refresh)
- Middle pane: Command input
- Bottom pane: Output display
- Footer: Help text and shortcuts

---

### F2: Azure CLI Authentication Detection
**Priority**: P0 (Critical)  
**Effort**: 1 day

Detect and display Azure CLI authentication status:
- Execute `az account show` to check auth
- Parse account name, subscription, user
- Display auth status (✓/✗)
- Cache result (60s TTL)
- Handle errors gracefully (CLI not installed, not logged in)

---

### F3: GitHub CLI Authentication Detection
**Priority**: P0 (Critical)  
**Effort**: 1 day

Detect and display GitHub CLI authentication status:
- Execute `gh auth status` to check auth
- Parse username, host
- Display auth status (✓/✗)
- Cache result (60s TTL)
- Handle errors gracefully

---

### F4: Terraform Context Detection
**Priority**: P0 (Critical)  
**Effort**: 2 days

Detect current Terraform configuration:
- **Workspace**: Read `.terraform/environment` file
- **Backend**: Parse `.terraform/terraform.tfstate` (local state pointer)
- **Module Path**: Current working directory
- **Terraform Version**: Execute `terraform version`
- **Initialization Status**: Check if `.terraform/` directory exists
- Display all context in status bar

---

### F5: OS Environment Variable Extraction
**Priority**: P0 (Critical)  
**Effort**: 1 day

Extract and display Terraform-related environment variables:
- Filter `TF_VAR_*` variables
- Filter `TF_*` variables (TF_LOG, TF_INPUT, etc.)
- Display in status bar or expandable section
- Show count if many variables

---

### F6: Real-Time Status Bar Refresh
**Priority**: P0 (Critical)  
**Effort**: 1 day

Auto-refresh status bar without blocking:
- Background goroutine for context detection
- Configurable refresh interval (default: 5s)
- Non-blocking updates
- Visual indicator during refresh
- Handle errors without crashing

---

### F7: Directory Change Detection
**Priority**: P0 (Critical)  
**Effort**: 2 days

Detect when user changes directory and update context:
- Monitor current working directory
- Detect `cd` equivalent (directory change)
- Re-run context detection when directory changes
- Update status bar immediately
- Clear old context, show new context

---

### F8: Terraform Command Execution
**Priority**: P1 (High)  
**Effort**: 2 days

Execute terraform commands with real-time output:
- Accept user input for terraform commands
- Execute `terraform <command>` as subprocess
- Stream stdout/stderr to output pane in real-time
- Preserve colors and formatting
- Handle Ctrl+C interruption
- Return terraform exit code

---

### F9: Basic Error Handling
**Priority**: P1 (High)  
**Effort**: 1 day

Gracefully handle common errors:
- Terraform not installed
- CLI tools not installed (az, gh)
- Not in terraform directory
- Permission errors
- Network timeouts
- Display helpful error messages

---

### F10: Configuration File Support
**Priority**: P2 (Nice to have)  
**Effort**: 1 day

Support basic configuration:
- Location: `~/.tfpipboy/config.yaml`
- Settings:
  - Refresh interval
  - Enabled auth providers
  - Display preferences
- Load on startup
- Provide defaults if file missing

---

## Out of Scope (Post-MVP)

These features are explicitly **NOT** in MVP:
- ❌ AWS authentication detection
- ❌ GCP authentication detection
- ❌ Multi-session orchestration
- ❌ Session history
- ❌ Command history
- ❌ Advanced visualizations (graphs, charts)
- ❌ Tabs or multiple workspaces
- ❌ Search functionality
- ❌ Export/logging features
- ❌ Remote/SSH support
- ❌ Plugin system

---

## User Stories

### Epic 1: Basic TUI Application

#### US-001: Launch TUI Application
**As a** DevOps engineer  
**I want to** launch tf-pipboy TUI application  
**So that** I can see my Terraform context in a persistent interface

**Acceptance Criteria**:
- Given I run `tfpipboy tui`
- When the application starts
- Then I see a multi-pane TUI with status bar, input, and output sections
- And the application is responsive to keyboard input
- And I can quit with 'q' or Ctrl+C

**Tasks**:
- [ ] Initialize Go project with Bubble Tea
- [ ] Create main.go with basic Bubble Tea model
- [ ] Implement View() with lipgloss styling
- [ ] Add keyboard event handling
- [ ] Test on macOS and Linux

**Estimate**: 2 days

---

### Epic 2: Authentication Status Detection

#### US-002: View Azure Authentication Status
**As a** DevOps engineer  
**I want to** see my Azure CLI authentication status  
**So that** I know if I'm logged in before running terraform

**Acceptance Criteria**:
- Given I am logged into Azure CLI
- When I launch tf-pipboy
- Then I see "✓ Azure: user@example.com (Subscription: Production)"
- And the status refreshes every 5 seconds
- Given I am NOT logged into Azure CLI
- Then I see "✗ Azure: Not authenticated"

**Tasks**:
- [ ] Create `auth/azure.go` package
- [ ] Implement `CheckAzureAuth()` function
- [ ] Execute `az account show --output json`
- [ ] Parse JSON response
- [ ] Handle errors (not installed, not logged in)
- [ ] Add caching with 60s TTL
- [ ] Display in status bar
- [ ] Write unit tests

**Estimate**: 1 day

---

#### US-003: View GitHub Authentication Status
**As a** DevOps engineer  
**I want to** see my GitHub CLI authentication status  
**So that** I know if I can access GitHub resources

**Acceptance Criteria**:
- Given I am logged into GitHub CLI
- When I launch tf-pipboy
- Then I see "✓ GitHub: octocat"
- Given I am NOT logged into GitHub CLI
- Then I see "✗ GitHub: Not authenticated"

**Tasks**:
- [ ] Create `auth/github.go` package
- [ ] Implement `CheckGitHubAuth()` function
- [ ] Execute `gh auth status`
- [ ] Parse output for username
- [ ] Handle errors
- [ ] Add caching with 60s TTL
- [ ] Display in status bar
- [ ] Write unit tests

**Estimate**: 1 day

---

### Epic 3: Terraform Context Awareness

#### US-004: View Current Terraform Workspace
**As a** DevOps engineer  
**I want to** see my current Terraform workspace  
**So that** I don't accidentally apply changes to the wrong environment

**Acceptance Criteria**:
- Given I am in a Terraform directory with workspace "production"
- When I launch tf-pipboy
- Then I see "Workspace: production" in the status bar
- Given I am not in a Terraform directory
- Then I see "Workspace: N/A"

**Tasks**:
- [ ] Create `terraform/context.go` package
- [ ] Implement `GetWorkspace()` function
- [ ] Read `.terraform/environment` file
- [ ] Handle file not existing (default workspace)
- [ ] Display in status bar
- [ ] Write unit tests

**Estimate**: 0.5 days

---

#### US-005: View Current Terraform Backend
**As a** DevOps engineer  
**I want to** see my Terraform backend configuration  
**So that** I know where my state is stored

**Acceptance Criteria**:
- Given I am in a Terraform directory with S3 backend
- When I launch tf-pipboy
- Then I see "Backend: s3 (my-bucket/terraform.tfstate)"
- Given backend is local
- Then I see "Backend: local"

**Tasks**:
- [ ] Implement `GetBackend()` function
- [ ] Parse `.terraform/terraform.tfstate` (local state pointer)
- [ ] Extract backend type and location
- [ ] Handle different backend types (s3, azurerm, gcs, local)
- [ ] Display in status bar
- [ ] Write unit tests

**Estimate**: 1 day

---

#### US-006: View Current Module Path
**As a** DevOps engineer  
**I want to** see my current Terraform module path  
**So that** I know which module I'm working in

**Acceptance Criteria**:
- Given I am in `/home/user/infra/azure/aks`
- When I launch tf-pipboy
- Then I see "Module: azure/aks" or full path based on config

**Tasks**:
- [ ] Implement `GetModulePath()` function
- [ ] Get current working directory
- [ ] Optionally shorten path (show relative to repo root)
- [ ] Display in status bar
- [ ] Write unit tests

**Estimate**: 0.5 days

---

### Epic 4: Environment Variables

#### US-007: View Terraform Environment Variables
**As a** DevOps engineer  
**I want to** see Terraform-related environment variables  
**So that** I understand what configuration is being used

**Acceptance Criteria**:
- Given I have `TF_VAR_environment=staging` set
- When I launch tf-pipboy
- Then I see environment variables section showing "TF_VAR_environment=staging"
- And I can expand/collapse the section

**Tasks**:
- [ ] Create `env/variables.go` package
- [ ] Implement `GetTerraformVars()` function
- [ ] Filter for `TF_VAR_*` and `TF_*` prefixes
- [ ] Display in status bar (count) or expandable section
- [ ] Handle many variables gracefully
- [ ] Write unit tests

**Estimate**: 1 day

---

### Epic 5: Real-Time Updates

#### US-008: Auto-Refresh Status Bar
**As a** DevOps engineer  
**I want** the status bar to auto-refresh  
**So that** I always see current authentication and context

**Acceptance Criteria**:
- Given the application is running
- When 5 seconds pass
- Then the status bar updates with fresh context
- And the refresh does not block the UI
- And I see a subtle indicator during refresh (spinner/timestamp)

**Tasks**:
- [ ] Implement background goroutine for refresh
- [ ] Use `time.Ticker` for interval
- [ ] Send refresh message to Bubble Tea Update()
- [ ] Run context detection in background
- [ ] Update UI when detection completes
- [ ] Add visual refresh indicator
- [ ] Make interval configurable
- [ ] Write tests

**Estimate**: 1 day

---

### Epic 6: Directory Change Detection

#### US-009: Detect Directory Changes
**As a** DevOps engineer  
**I want** the context to update when I change directories  
**So that** I always see relevant information for my current module

**Acceptance Criteria**:
- Given I am in `/infra/azure/aks`
- When I change to `/infra/azure/vnet` (via some mechanism)
- Then the status bar updates to show vnet module context
- And workspace/backend update if different

**Note**: This is tricky in TUI - need to detect cwd changes.

**Approach**:
- Poll current working directory every N seconds
- Compare with last known directory
- If changed, re-run context detection

**Tasks**:
- [ ] Implement directory monitoring
- [ ] Poll `os.Getwd()` every 2 seconds
- [ ] Compare with cached directory
- [ ] Trigger context refresh on change
- [ ] Clear old context before showing new
- [ ] Write tests

**Estimate**: 2 days

---

### Epic 7: Terraform Command Execution

#### US-010: Run Terraform Commands
**As a** DevOps engineer  
**I want to** run terraform commands from within the TUI  
**So that** I can work without switching windows

**Acceptance Criteria**:
- Given I am in the TUI
- When I type "terraform plan" and press Enter
- Then terraform plan executes
- And I see real-time output in the output pane
- And colors/formatting are preserved
- When terraform completes
- Then I can enter another command

**Tasks**:
- [ ] Implement command input widget
- [ ] Parse user input
- [ ] Execute terraform as subprocess
- [ ] Stream stdout/stderr to output pane
- [ ] Preserve ANSI colors
- [ ] Handle Ctrl+C (cancel command)
- [ ] Show exit code
- [ ] Write tests

**Estimate**: 2 days

---

### Epic 8: Error Handling

#### US-011: Handle Missing Dependencies
**As a** DevOps engineer  
**I want to** see clear errors when tools are missing  
**So that** I know what to install

**Acceptance Criteria**:
- Given Terraform is not installed
- When I launch tf-pipboy
- Then I see "⚠ Terraform not found. Please install from https://terraform.io"
- Given Azure CLI is not installed
- Then I see "⚠ Azure CLI not found. Auth status unavailable."

**Tasks**:
- [ ] Check for terraform binary on startup
- [ ] Check for az binary when checking auth
- [ ] Check for gh binary when checking auth
- [ ] Display helpful error messages
- [ ] Don't crash on missing tools
- [ ] Provide install instructions
- [ ] Write tests

**Estimate**: 1 day

---

## Development Schedule

### Week 1 (Days 1-5)

**Day 1: Project Setup & Basic TUI**
- [x] Initialize Go project (`go mod init`)
- [ ] Add Bubble Tea dependency
- [ ] Create basic TUI shell (US-001)
- [ ] Multi-pane layout with lipgloss
- [ ] Test launch and quit

**Day 2: Authentication Detection - Azure**
- [ ] Implement Azure auth detection (US-002)
- [ ] Test with logged in/out states
- [ ] Add caching
- [ ] Display in status bar

**Day 3: Authentication Detection - GitHub**
- [ ] Implement GitHub auth detection (US-003)
- [ ] Test with logged in/out states
- [ ] Add caching
- [ ] Display in status bar

**Day 4: Terraform Context (Part 1)**
- [ ] Implement workspace detection (US-004)
- [ ] Implement module path detection (US-006)
- [ ] Test in various directories

**Day 5: Terraform Context (Part 2)**
- [ ] Implement backend detection (US-005)
- [ ] Handle different backend types
- [ ] Test with S3, Azure RM, local backends

---

### Week 2 (Days 6-10)

**Day 6: Environment Variables**
- [ ] Implement env var extraction (US-007)
- [ ] Filter TF_VAR_* and TF_* variables
- [ ] Display in UI

**Day 7: Real-Time Refresh**
- [ ] Implement auto-refresh (US-008)
- [ ] Background goroutine with ticker
- [ ] Non-blocking UI updates
- [ ] Add refresh indicator

**Day 8: Directory Change Detection**
- [ ] Implement directory monitoring (US-009)
- [ ] Poll working directory
- [ ] Trigger context refresh on change
- [ ] Test directory switching

**Day 9: Terraform Command Execution**
- [ ] Implement command input (US-010)
- [ ] Execute terraform subprocess
- [ ] Stream output in real-time
- [ ] Handle Ctrl+C

**Day 10: Error Handling & Polish**
- [ ] Implement error handling (US-011)
- [ ] Handle missing dependencies
- [ ] Polish UI and styling
- [ ] Final testing

---

## Testing Strategy

### Unit Tests
- [ ] Test auth detection functions
- [ ] Test terraform context detection
- [ ] Test environment variable extraction
- [ ] Test caching logic
- [ ] Test error handling

### Integration Tests
- [ ] Test full context detection pipeline
- [ ] Test with real Azure/GitHub auth
- [ ] Test directory switching
- [ ] Test terraform command execution

### Manual Tests
- [ ] Test in different terminals (iTerm, Ghostty, Terminal.app)
- [ ] Test with and without auth
- [ ] Test in terraform and non-terraform directories
- [ ] Test with various backend types
- [ ] Test on macOS and Linux

---

## Success Criteria

### Functional
- ✅ Displays Azure auth status correctly
- ✅ Displays GitHub auth status correctly
- ✅ Shows current workspace, backend, module
- ✅ Shows Terraform environment variables
- ✅ Status bar auto-refreshes every 5 seconds
- ✅ Context updates when directory changes
- ✅ Can execute terraform commands
- ✅ Handles errors gracefully

### Non-Functional
- ✅ Startup time < 200ms
- ✅ Memory usage < 50MB
- ✅ Status refresh non-blocking
- ✅ Responsive UI (no lag)
- ✅ Works in iTerm, Terminal.app, Ghostty, Alacritty

### User Experience
- ✅ Clear, readable status display
- ✅ Helpful error messages
- ✅ Intuitive keyboard shortcuts
- ✅ Professional appearance

---

## Risk Management

### Technical Risks

**Risk**: Directory change detection may not work reliably  
**Mitigation**: Implement polling-based approach; if too slow, make it manual (user runs `:cd` command)  
**Fallback**: Remove auto-detect, require manual context refresh

**Risk**: Bubble Tea framework proves inadequate  
**Mitigation**: Early prototype to validate framework; can pivot to Textual (Python) if needed  
**Probability**: Low (Bubble Tea is production-proven)

**Risk**: Authentication detection commands too slow  
**Mitigation**: Aggressive caching (60s), timeout on commands (2s), show cached data  
**Fallback**: Make auth checks optional

### Schedule Risks

**Risk**: Features take longer than estimated  
**Mitigation**: Prioritize P0 features, cut P2 features if needed  
**Buffer**: 2-day buffer in 2-week schedule

**Risk**: Blocked by external dependencies  
**Mitigation**: All dependencies are stable open-source projects  
**Probability**: Low

---

## Post-MVP Roadmap

### v0.2 (Week 3-4)
- AWS authentication detection
- GCP authentication detection
- Multi-session monitoring
- Improved UI with tabs

### v0.3 (Month 2)
- Session orchestration
- Dependency management
- Command history
- Configuration file enhancements

### v1.0 (Month 3)
- Production-ready release
- Documentation
- Binary releases (GitHub)
- Homebrew formula
- User feedback incorporation

---

## Resources

### Documentation
- Bubble Tea: https://github.com/charmbracelet/bubbletea
- Lip Gloss (styling): https://github.com/charmbracelet/lipgloss
- Bubbles (components): https://github.com/charmbracelet/bubbles

### Example Projects
- Glow (Markdown reader): https://github.com/charmbracelet/glow
- Soft Serve (Git server): https://github.com/charmbracelet/soft-serve
- kubectl plugins using Bubble Tea

### Design Documents
- ADR-001: Architecture Decision
- design/03-architecture-design.md: Complete architecture
- design/06-tui-vs-ghostty-deep-comparison.md: Technology justification

---

## Team

**Developer**: Stanley Xie  
**Advisor**: Claude Code  
**Timeline**: 2 weeks  
**Start Date**: TBD  
**Target Release**: MVP in 2 weeks, v1.0 in 3 months

---

## Notes

- Focus on P0 features first
- Validate assumptions early with prototype
- Get user feedback as soon as basic version works
- Iterate based on real usage, not speculation
- Keep scope tight for MVP - we can always add more later

**Philosophy**: Ship early, ship often, iterate based on feedback.
