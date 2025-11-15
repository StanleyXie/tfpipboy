package orchestrator

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// Global mutex for stdout writes to prevent interleaving
var stdoutMutex sync.Mutex

// StatusUpdateEvent represents a status update event from a job
type StatusUpdateEvent struct {
	JobID      string
	EventType  string // "status", "step", "progress", "error", "plan_result"
	Status     JobStatus
	StepName   string
	Progress   string
	Error      string
	PlanResult string
}

// LiveBoard provides a real-time dashboard view for parallel job execution
type LiveBoard struct {
	mu          sync.RWMutex
	jobs        map[string]*LiveJobStatus
	jobOrder    []string // Maintain insertion order for consistent display
	stdout      io.Writer
	isActive    bool
	startTime   time.Time
	refreshRate time.Duration
	stopChan    chan struct{}
	eventChan   chan StatusUpdateEvent // Channel for async status updates
	maxNameLen  int
	useColors   bool
	firstRender bool          // Track if this is the first render
	refreshDone chan struct{} // Signal when refresh loop has stopped
	isTTY       bool          // Whether stdout is a terminal
	lastRenderLines int       // Number of lines in last render (for TTY clearing)
}

// WorkflowStep represents a step in the execution workflow
type WorkflowStep struct {
	Name      string
	Status    string // "pending", "running", "completed", "failed"
	StartTime time.Time
	EndTime   time.Time
}

// LiveJobStatus tracks real-time status of a job execution
type LiveJobStatus struct {
	ID           string
	ModuleName   string
	InstanceName string
	Category     string // Module category for grouping
	ConfigOrder  int    // Original order in configuration (index in array)
	Stage        int    // Execution stage based on dependencies (seed=0, core=1, vending=2, etc.)
	Status       JobStatus
	Operation    TerraformOperation
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Progress     string // Current progress message (e.g., "Initializing...", "Planning...")
	Error        string
	SpinnerIndex int
	Steps        []WorkflowStep // Workflow steps for this job
	CurrentStep  int            // Index of current step
	PlanResult   string         // Plan result summary (e.g., "No changes", "3 to add, 2 to change")
}

// NewLiveBoard creates a new live board display
func NewLiveBoard(useColors bool) *LiveBoard {
	// Detect if stdout is a terminal
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))

	// If not a TTY, use slower refresh rate to reduce output spam
	refreshRate := 200 * time.Millisecond
	if !isTTY {
		refreshRate = 2 * time.Second // Much slower for non-TTY
	}

	return &LiveBoard{
		jobs:        make(map[string]*LiveJobStatus),
		jobOrder:    []string{},
		stdout:      os.Stdout,
		refreshRate: refreshRate,
		stopChan:    make(chan struct{}),
		eventChan:   make(chan StatusUpdateEvent, 100), // Buffered channel for async events
		refreshDone: make(chan struct{}),
		useColors:   useColors,
		isTTY:       isTTY,
	}
}

// GetEventChannel returns the event channel for sending status updates
func (lb *LiveBoard) GetEventChannel() chan<- StatusUpdateEvent {
	return lb.eventChan
}

// GetJobStatuses returns a copy of all job statuses from the liveboard
func (lb *LiveBoard) GetJobStatuses() map[string]*LiveJobStatus {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	statuses := make(map[string]*LiveJobStatus)
	for id, job := range lb.jobs {
		statuses[id] = job
	}
	return statuses
}

// Start begins the live board display and starts the refresh loop
func (lb *LiveBoard) Start(jobs []*ExecutionJob) {
	lb.mu.Lock()

	lb.isActive = true
	lb.startTime = time.Now()
	lb.firstRender = true

	// Initialize job statuses
	for i, job := range jobs {
		// Initialize workflow steps based on operation
		steps := lb.getWorkflowSteps(TerraformOperation(job.Operation))

		// Use DependencyDepth for display ordering if available, otherwise fall back to Stage
		displayStage := job.DependencyDepth
		if displayStage == 0 && job.Stage > 0 {
			// If no dependency depth calculated (all 0) but we have execution stages, use those
			displayStage = job.Stage
		}

		jobStatus := &LiveJobStatus{
			ID:           job.ID,
			ModuleName:   job.ModuleName,
			InstanceName: job.InstanceName,
			Category:     job.Category,
			ConfigOrder:  i,            // Preserve original array order
			Stage:        displayStage, // Use dependency depth for ordering
			Status:       JobStatusPending,
			Operation:    TerraformOperation(job.Operation),
			Progress:     "Waiting...",
			Steps:        steps,
			CurrentStep:  -1, // Not started yet
		}
		lb.jobs[job.ID] = jobStatus

		// Debug: Print stage information (disabled for console snippet exclusivity)
		// fmt.Fprintf(os.Stderr, "[DEBUG-LIVEBOARD] Job '%s' (module: %s) - Stage: %d, DepDepth: %d, DisplayStage: %d, ConfigOrder: %d, Category: %s\n",
		// 	job.ID, job.ModuleName, job.Stage, job.DependencyDepth, displayStage, i, job.Category)

		// Track max name length for alignment
		nameLen := len(job.ID)
		if nameLen > lb.maxNameLen {
			lb.maxNameLen = nameLen
		}
	}

	// Sort jobs by category first, then by configuration order within category
	lb.sortJobsByCategory()

	// Hide cursor
	fmt.Fprint(lb.stdout, "\033[?25l")

	// Do initial render immediately to clear screen and show initial state
	lb.render()

	// Unlock after initial render completes
	lb.mu.Unlock()

	// Start event processing loop
	go lb.eventProcessingLoop()

	// Start refresh loop
	go lb.refreshLoop()
}

// Stop ends the live board display
func (lb *LiveBoard) Stop() {
	// First, signal stop and wait for refresh loop to finish
	close(lb.stopChan)
	<-lb.refreshDone // Wait for refresh loop to stop

	close(lb.eventChan) // Close event channel to stop event processing

	// Give event processor time to drain remaining events
	time.Sleep(100 * time.Millisecond)

	// Now lock and do final render
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if !lb.isActive {
		return
	}

	lb.isActive = false

	// Final render
	lb.render()

	// Show cursor
	fmt.Fprint(lb.stdout, "\033[?25h\n")
}

// eventProcessingLoop processes status update events asynchronously
func (lb *LiveBoard) eventProcessingLoop() {
	for event := range lb.eventChan {
		lb.mu.Lock()
		job, exists := lb.jobs[event.JobID]
		if !exists {
			lb.mu.Unlock()
			continue
		}

		switch event.EventType {
		case "status":
			job.Status = event.Status
			job.Progress = event.Progress
			if event.Status == JobStatusRunning && job.StartTime.IsZero() {
				job.StartTime = time.Now()
			}
			if event.Status == JobStatusCompleted || event.Status == JobStatusFailed {
				job.EndTime = time.Now()
				if !job.StartTime.IsZero() {
					job.Duration = job.EndTime.Sub(job.StartTime)
				}
				// Mark all workflow steps as completed (or failed)
				finalStatus := "completed"
				if event.Status == JobStatusFailed {
					finalStatus = "failed"
				}
				for i := range job.Steps {
					if job.Steps[i].Status == "pending" || job.Steps[i].Status == "running" {
						job.Steps[i].Status = finalStatus
						if job.Steps[i].StartTime.IsZero() {
							job.Steps[i].StartTime = job.StartTime
						}
						job.Steps[i].EndTime = job.EndTime
					}
				}
			}

		case "step":
			// Find the step by name and mark it as running
			for i, step := range job.Steps {
				if step.Name == event.StepName {
					// Mark previous RUNNING steps as completed
					for j := 0; j < i; j++ {
						if job.Steps[j].Status == "running" {
							job.Steps[j].Status = "completed"
							job.Steps[j].EndTime = time.Now()
						}
					}
					// Mark current step as running
					job.Steps[i].Status = "running"
					job.Steps[i].StartTime = time.Now()
					job.CurrentStep = i
					break
				}
			}

		case "progress":
			job.Progress = event.Progress

		case "error":
			job.Error = event.Error
			job.Status = JobStatusFailed
			job.EndTime = time.Now()
			if !job.StartTime.IsZero() {
				job.Duration = job.EndTime.Sub(job.StartTime)
			}
			// Mark all workflow steps as failed (pending or running)
			for i := range job.Steps {
				if job.Steps[i].Status == "pending" || job.Steps[i].Status == "running" {
					job.Steps[i].Status = "failed"
					if job.Steps[i].StartTime.IsZero() {
						job.Steps[i].StartTime = job.StartTime
					}
					job.Steps[i].EndTime = job.EndTime
				}
			}

		case "plan_result":
			job.PlanResult = event.PlanResult
		}

		lb.mu.Unlock()
	}
}

// UpdateJobStatus updates the status of a specific job
func (lb *LiveBoard) UpdateJobStatus(jobID string, status JobStatus, progress string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if job, exists := lb.jobs[jobID]; exists {
		job.Status = status
		job.Progress = progress

		if status == JobStatusRunning && job.StartTime.IsZero() {
			job.StartTime = time.Now()
		}

		if status == JobStatusCompleted || status == JobStatusFailed {
			job.EndTime = time.Now()
			if !job.StartTime.IsZero() {
				job.Duration = job.EndTime.Sub(job.StartTime)
			}

			// Mark all workflow steps as completed (or failed)
			finalStatus := "completed"
			if status == JobStatusFailed {
				finalStatus = "failed"
			}
			for i := range job.Steps {
				if job.Steps[i].Status == "pending" || job.Steps[i].Status == "running" {
					job.Steps[i].Status = finalStatus
					if job.Steps[i].StartTime.IsZero() {
						job.Steps[i].StartTime = job.StartTime
					}
					job.Steps[i].EndTime = job.EndTime
				}
			}
		}
	}
}

// UpdateJobProgress updates just the progress message for a job
func (lb *LiveBoard) UpdateJobProgress(jobID string, progress string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if job, exists := lb.jobs[jobID]; exists {
		job.Progress = progress
	}
}

// UpdateJobStep updates the current step in the workflow
func (lb *LiveBoard) UpdateJobStep(jobID string, stepName string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Debug: List all known jobs (disabled for console snippet exclusivity)
	// fmt.Fprintf(os.Stderr, "\n[DEBUG-LIVEBOARD] UpdateJobStep called: jobID='%s', step='%s'\n", jobID, stepName)
	// fmt.Fprintf(os.Stderr, "[DEBUG-LIVEBOARD] Known jobs in liveboard: ")
	// for id := range lb.jobs {
	// 	fmt.Fprintf(os.Stderr, "'%s' ", id)
	// }
	// fmt.Fprintf(os.Stderr, "\n")

	job, exists := lb.jobs[jobID]
	if !exists {
		// Job not found - silently return for console snippet exclusivity
		return
	}

	// fmt.Fprintf(os.Stderr, "[DEBUG-LIVEBOARD] Job '%s' found, updating step '%s'\n", jobID, stepName)

	// Find the step by name and mark it as running
	for i, step := range job.Steps {
		if step.Name == stepName {
			// Mark previous RUNNING steps as completed (not failed or pending)
			// This assumes successful completion of running steps before moving to next step
			for j := 0; j < i; j++ {
				if job.Steps[j].Status == "running" {
					job.Steps[j].Status = "completed"
					job.Steps[j].EndTime = time.Now()
				}
			}

			// Mark current step as running
			job.Steps[i].Status = "running"
			job.Steps[i].StartTime = time.Now()
			job.CurrentStep = i
			// fmt.Fprintf(os.Stderr, "[DEBUG-LIVEBOARD] Step '%s' marked as running for job '%s'\n", stepName, jobID)
			break
		}
	}
}

// UpdateJobError updates the error message for a job
func (lb *LiveBoard) UpdateJobError(jobID string, errorMsg string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if job, exists := lb.jobs[jobID]; exists {
		job.Error = errorMsg
		job.Status = JobStatusFailed
		job.EndTime = time.Now()
		if !job.StartTime.IsZero() {
			job.Duration = job.EndTime.Sub(job.StartTime)
		}
		// Mark all workflow steps as failed (pending or running)
		for i := range job.Steps {
			if job.Steps[i].Status == "pending" || job.Steps[i].Status == "running" {
				job.Steps[i].Status = "failed"
				if job.Steps[i].StartTime.IsZero() {
					job.Steps[i].StartTime = job.StartTime
				}
				job.Steps[i].EndTime = job.EndTime
			}
		}
	}
}

// UpdateJobPlanResult updates the plan result summary for a job
func (lb *LiveBoard) UpdateJobPlanResult(jobID string, planResult string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if job, exists := lb.jobs[jobID]; exists {
		job.PlanResult = planResult
	}
}

// refreshLoop continuously updates the display
func (lb *LiveBoard) refreshLoop() {
	ticker := time.NewTicker(lb.refreshRate)
	defer ticker.Stop()
	defer close(lb.refreshDone) // Signal that refresh loop has stopped

	for {
		select {
		case <-lb.stopChan:
			return
		case <-ticker.C:
			lb.mu.Lock()
			if lb.isActive {
				lb.render()
				// Update spinner indices
				for _, job := range lb.jobs {
					if job.Status == JobStatusRunning {
						job.SpinnerIndex = (job.SpinnerIndex + 1) % len(spinnerFrames)
					}
				}
			}
			lb.mu.Unlock()
		}
	}
}

// render draws the live board to the terminal
func (lb *LiveBoard) render() {
	// Build the entire output in a buffer for atomic write
	var buf strings.Builder

	if lb.isTTY {
		// TTY mode: use ANSI codes for in-place updates
		if lb.firstRender {
			lb.firstRender = false
			// First render - clear screen and move to top
			fmt.Fprint(&buf, "\033[2J\033[H")
		} else {
			// Subsequent renders - move cursor up to start of previous render
			if lb.lastRenderLines > 0 {
				fmt.Fprintf(&buf, "\033[%dA", lb.lastRenderLines)
			}
			// Move to beginning of line and clear from cursor to end of screen
			fmt.Fprint(&buf, "\r\033[0J")
		}
	} else {
		// Non-TTY mode: only render on first and last (when all jobs done)
		allDone := true
		for _, job := range lb.jobs {
			if job.Status != JobStatusCompleted && job.Status != JobStatusFailed {
				allDone = false
				break
			}
		}

		// Only render on first time or when all done
		if !lb.firstRender && !allDone {
			return
		}

		if lb.firstRender {
			lb.firstRender = false
		}
	}

	// Render header
	elapsed := time.Since(lb.startTime).Round(time.Second)

	// Check if all jobs are done (completed or failed)
	allDone := true
	for _, job := range lb.jobs {
		if job.Status != JobStatusCompleted && job.Status != JobStatusFailed {
			allDone = false
			break
		}
	}

	// Set header color and title based on completion status
	var headerColor, headerTitle string
	if allDone {
		headerColor = colorGreen
		headerTitle = "PARALLEL EXECUTION COMPLETED"
	} else {
		headerColor = colorBlue
		headerTitle = "PARALLEL EXECUTION IN PROGRESS"
	}

	if !lb.useColors {
		headerColor = ""
	}

	fmt.Fprintf(&buf, "%s╭──────────────────────────────────────────────────────────────────────────────╮%s\n",
		headerColor, colorReset)
	fmt.Fprintf(&buf, "%s│ %-60s Elapsed: %6s │%s\n",
		headerColor, headerTitle, elapsed, colorReset)
	fmt.Fprintf(&buf, "%s╰──────────────────────────────────────────────────────────────────────────────╯%s\n",
		headerColor, colorReset)

	// Render job rows grouped by category (only show headers if categories exist)
	var lastCategory string
	hasCategories := lb.hasAnyCategories()

	for _, jobID := range lb.jobOrder {
		job := lb.jobs[jobID]

		// Display category header when category changes (only if categories are defined)
		if hasCategories {
			category := job.Category
			if category == "" {
				category = "uncategorized"
			}

			if category != lastCategory {
				// Render category separator/header
				lb.renderCategoryHeaderToBuffer(&buf, category)
				lastCategory = category
			}
		}

		lb.renderJobRowToBuffer(&buf, job)
	}

	// Render footer with statistics (aligned to match header width)
	lb.renderFooterToBuffer(&buf)

	// Count lines in the buffer for TTY cursor management
	if lb.isTTY {
		lb.lastRenderLines = strings.Count(buf.String(), "\n")
	}

	// Write the entire buffer atomically with mutex protection
	stdoutMutex.Lock()
	io.WriteString(lb.stdout, buf.String())
	stdoutMutex.Unlock()
}

// renderCategoryHeaderToBuffer renders a category header to the buffer
func (lb *LiveBoard) renderCategoryHeaderToBuffer(buf *strings.Builder, category string) {
	// Don't render header for first category or if no colors
	// Just a subtle visual separator
	categoryColor := colorCyan
	if !lb.useColors {
		categoryColor = ""
	}

	// Simple category label line
	fmt.Fprintf(buf, "%s[%s]%s\n", categoryColor, strings.ToUpper(category), colorReset)
}

// renderJobRowToBuffer renders a single job status row to a buffer
func (lb *LiveBoard) renderJobRowToBuffer(buf *strings.Builder, job *LiveJobStatus) {
	// Status indicator
	statusIcon := lb.getStatusIcon(job.Status)
	statusColor := lb.getStatusColor(job.Status)

	// Spinner for running jobs
	spinner := ""
	if job.Status == JobStatusRunning {
		spinner = spinnerFrames[job.SpinnerIndex]
	} else {
		// Use space as placeholder to maintain alignment
		spinner = " "
	}

	// Duration or elapsed time - fixed width [   32s]
	var durationStr string
	if job.Status == JobStatusCompleted || job.Status == JobStatusFailed {
		durationStr = fmt.Sprintf("[%5s]", job.Duration.Round(time.Second))
	} else if job.Status == JobStatusRunning && !job.StartTime.IsZero() {
		elapsed := time.Since(job.StartTime).Round(time.Second)
		durationStr = fmt.Sprintf("[%5s]", elapsed)
	} else {
		durationStr = "[   0s]"
	}

	// Instance name (truncate if too long)
	instanceName := job.ID
	maxInstanceLen := 20
	if len(instanceName) > maxInstanceLen {
		instanceName = instanceName[:maxInstanceLen-3] + "..."
	}

	// Build workflow visualization
	workflow := lb.buildWorkflowString(job)

	// Build result section: duration + plan result + error (max 32 chars total for result message)
	resultSection := durationStr // Start with duration

	// Plan result (if available) with color coding
	if job.PlanResult != "" {
		planResult := job.PlanResult
		var coloredResult string

		if lb.useColors {
			// Apply Terraform-style colors to plan results
			if strings.Contains(planResult, "No changes") {
				coloredResult = fmt.Sprintf("%sNo changes%s", colorGreen, colorReset)
			} else if strings.HasPrefix(planResult, "+") || strings.HasPrefix(planResult, "~") || strings.HasPrefix(planResult, "-") {
				// Colorize each part: +N (green), ~N (yellow), -N (red)
				parts := strings.Fields(planResult)
				var coloredParts []string
				for _, part := range parts {
					if strings.HasPrefix(part, "+") {
						coloredParts = append(coloredParts, fmt.Sprintf("%s%s%s", colorGreen, part, colorReset))
					} else if strings.HasPrefix(part, "~") {
						coloredParts = append(coloredParts, fmt.Sprintf("%s%s%s", colorYellow, part, colorReset))
					} else if strings.HasPrefix(part, "-") {
						coloredParts = append(coloredParts, fmt.Sprintf("%s%s%s", colorRed, part, colorReset))
					} else {
						coloredParts = append(coloredParts, part)
					}
				}
				coloredResult = strings.Join(coloredParts, " ")
			} else {
				coloredResult = planResult
			}
		} else {
			coloredResult = planResult
		}

		resultSection += " " + coloredResult
	}

	// Error message (truncate to fit within 32 char limit for the entire result section)
	// Account for: [   32s] (7 chars) + plan result + error message
	if job.Error != "" && job.Status == JobStatusFailed {
		errorLines := strings.Split(job.Error, "\n")
		if len(errorLines) > 0 {
			errorMsg := errorLines[0]

			// Calculate remaining space for error message
			// Total budget: 32 chars, minus duration (7), minus plan result length, minus " - " (3)
			planResultLen := len(StripANSI(job.PlanResult))
			if planResultLen > 0 {
				planResultLen += 1 // Account for space before plan result
			}

			maxErrorLen := 32 - 7 - planResultLen - 3 // 32 total - duration - plan - " - "
			if maxErrorLen > 3 {                      // Need at least 3 chars for "..."
				if len(errorMsg) > maxErrorLen {
					errorMsg = errorMsg[:maxErrorLen-3] + "..."
				}
				resultSection += " - " + errorMsg
			} else if maxErrorLen > 0 {
				// Not enough space for truncation marker, just take what we can
				if len(errorMsg) > maxErrorLen {
					errorMsg = errorMsg[:maxErrorLen]
				}
				resultSection += " - " + errorMsg
			}
			// If maxErrorLen <= 0, skip error message entirely (no space)
		}
	}

	// Render the row with consistent spacing
	// Format: "  [icon] [spinner] [name-20chars] [workflow] │ [duration + result (max 32 chars)]"
	if lb.useColors {
		fmt.Fprintf(buf, "  %s%s%s %s %-20s %s │ %s\n",
			statusColor, statusIcon, colorReset,
			spinner,
			instanceName,
			workflow,
			resultSection)
	} else {
		fmt.Fprintf(buf, "  %s %s %-20s %s │ %s\n",
			statusIcon,
			spinner,
			instanceName,
			workflow,
			resultSection)
	}
}

// buildWorkflowString creates a visual representation of the workflow steps
func (lb *LiveBoard) buildWorkflowString(job *LiveJobStatus) string {
	if len(job.Steps) == 0 {
		return job.Progress
	}

	var parts []string
	for i, step := range job.Steps {
		var stepStr string

		switch step.Status {
		case "completed":
			if lb.useColors {
				stepStr = fmt.Sprintf("%s✓ %s%s", colorGreen, step.Name, colorReset)
			} else {
				stepStr = fmt.Sprintf("✓ %s", step.Name)
			}
		case "running":
			spinner := spinnerFrames[job.SpinnerIndex]
			if lb.useColors {
				stepStr = fmt.Sprintf("%s%s %s%s", colorBlue, spinner, step.Name, colorReset)
			} else {
				stepStr = fmt.Sprintf("%s %s", spinner, step.Name)
			}
		case "failed":
			if lb.useColors {
				stepStr = fmt.Sprintf("%s✗ %s%s", colorRed, step.Name, colorReset)
			} else {
				stepStr = fmt.Sprintf("✗ %s", step.Name)
			}
		default: // pending
			if lb.useColors {
				stepStr = fmt.Sprintf("%s○ %s%s", colorGray, step.Name, colorReset)
			} else {
				stepStr = fmt.Sprintf("○ %s", step.Name)
			}
		}

		parts = append(parts, stepStr)

		// Add arrow between steps (but not after the last step)
		if i < len(job.Steps)-1 {
			if lb.useColors {
				parts = append(parts, fmt.Sprintf("%s→%s", colorGray, colorReset))
			} else {
				parts = append(parts, "→")
			}
		}
	}

	return strings.Join(parts, " ")
}

// renderFooterToBuffer renders summary statistics to a buffer
func (lb *LiveBoard) renderFooterToBuffer(buf *strings.Builder) {
	completed := 0
	failed := 0
	running := 0
	pending := 0

	for _, job := range lb.jobs {
		switch job.Status {
		case JobStatusCompleted:
			completed++
		case JobStatusFailed:
			failed++
		case JobStatusRunning:
			running++
		case JobStatusPending:
			pending++
		}
	}

	total := len(lb.jobs)

	footerColor := colorGray
	if !lb.useColors {
		footerColor = ""
	}

	// Align footer separator with header (78 chars to match ╰─...─╯)
	fmt.Fprintf(buf, "──────────────────────────────────────────────────────────────────────────────\n")

	// Format statistics line with proper padding to align with header width
	statsLine := fmt.Sprintf("Total: %d  |  Completed: %d  |  Running: %d  |  Failed: %d  |  Pending: %d",
		total, completed, running, failed, pending)

	if lb.useColors {
		fmt.Fprintf(buf, "%s%s%s\n", footerColor, statsLine, colorReset)
	} else {
		fmt.Fprintf(buf, "%s\n", statsLine)
	}
}

// getStatusIcon returns the icon for a job status
func (lb *LiveBoard) getStatusIcon(status JobStatus) string {
	switch status {
	case JobStatusCompleted:
		return "✓"
	case JobStatusFailed:
		return "✗"
	case JobStatusRunning:
		return "▶"
	case JobStatusPending:
		return "○"
	case JobStatusSkipped:
		return "⊘"
	default:
		return "·"
	}
}

// getWorkflowSteps returns the workflow steps for a given operation
func (lb *LiveBoard) getWorkflowSteps(operation TerraformOperation) []WorkflowStep {
	switch operation {
	case OpInit:
		return []WorkflowStep{
			{Name: "Init", Status: "pending"},
		}
	case OpPlan:
		return []WorkflowStep{
			{Name: "Init", Status: "pending"},
			{Name: "Plan", Status: "pending"},
		}
	case OpApply:
		return []WorkflowStep{
			{Name: "Init", Status: "pending"},
			{Name: "Apply", Status: "pending"},
		}
	case OpDestroy:
		return []WorkflowStep{
			{Name: "Init", Status: "pending"},
			{Name: "Destroy", Status: "pending"},
		}
	case OpValidate:
		return []WorkflowStep{
			{Name: "Init", Status: "pending"},
			{Name: "Validate", Status: "pending"},
		}
	case OpRefresh:
		return []WorkflowStep{
			{Name: "Init", Status: "pending"},
			{Name: "Refresh", Status: "pending"},
		}
	default:
		return []WorkflowStep{
			{Name: string(operation), Status: "pending"},
		}
	}
}

// getStatusColor returns the color for a job status
func (lb *LiveBoard) getStatusColor(status JobStatus) string {
	if !lb.useColors {
		return ""
	}

	switch status {
	case JobStatusCompleted:
		return colorGreen
	case JobStatusFailed:
		return colorRed
	case JobStatusRunning:
		return colorBlue
	case JobStatusPending:
		return colorGray
	case JobStatusSkipped:
		return colorYellow
	default:
		return colorGray
	}
}

// hasAnyCategories checks if any job has a category defined
func (lb *LiveBoard) hasAnyCategories() bool {
	for _, job := range lb.jobs {
		if job.Category != "" {
			return true
		}
	}
	return false
}

// sortJobsByCategory sorts jobs by category first, then by dependency stage within each category
func (lb *LiveBoard) sortJobsByCategory() {
	// Group jobs by category, preserving Stage order within each category
	categoryMap := make(map[string][]*LiveJobStatus)

	// Collect all jobs into category groups
	for _, job := range lb.jobs {
		category := job.Category
		if category == "" {
			category = "uncategorized"
		}
		categoryMap[category] = append(categoryMap[category], job)
	}

	// Sort jobs within each category by Stage (dependency-based execution order)
	// If stages are equal, use ConfigOrder as tiebreaker for stable sorting
	for _, jobs := range categoryMap {
		// Debug output disabled for console snippet exclusivity
		// fmt.Fprintf(os.Stderr, "[DEBUG-LIVEBOARD] Before sorting category '%s':\n", category)
		// for _, job := range jobs {
		// 	fmt.Fprintf(os.Stderr, "  - %s: Stage=%d, ConfigOrder=%d\n", job.ID, job.Stage, job.ConfigOrder)
		// }

		sort.Slice(jobs, func(i, j int) bool {
			if jobs[i].Stage != jobs[j].Stage {
				return jobs[i].Stage < jobs[j].Stage
			}
			return jobs[i].ConfigOrder < jobs[j].ConfigOrder
		})

		// Debug output disabled for console snippet exclusivity
		// fmt.Fprintf(os.Stderr, "[DEBUG-LIVEBOARD] After sorting category '%s':\n", category)
		// for _, job := range jobs {
		// 	fmt.Fprintf(os.Stderr, "  - %s: Stage=%d, ConfigOrder=%d\n", job.ID, job.Stage, job.ConfigOrder)
		// }
	}

	// Get sorted category list
	var categories []string
	for category := range categoryMap {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	// Build final job order: sorted categories, jobs in dependency stage order within each category
	lb.jobOrder = []string{}
	for _, category := range categories {
		for _, job := range categoryMap[category] {
			lb.jobOrder = append(lb.jobOrder, job.ID)
		}
	}
}
