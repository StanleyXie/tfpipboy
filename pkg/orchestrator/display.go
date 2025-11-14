package orchestrator

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Spinner frames for visual progress indicator
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// ANSI regex for stripping colors from log files
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// OperationStatus represents the status of an operation
type OperationStatus string

const (
	StatusRunning OperationStatus = "running"
	StatusSuccess OperationStatus = "success"
	StatusFailed  OperationStatus = "failed"
)

// ProgressDisplay manages enhanced terminal output
type ProgressDisplay struct {
	mu           sync.Mutex
	currentLine  string
	spinnerIndex int
	isActive     bool
	startTime    time.Time
	lastUpdate   time.Time
	stdout       io.Writer
	useColors    bool
	refreshCount map[string]int     // Track terraform refresh operations
	currentOp    TerraformOperation // Current operation
	jobID        string             // Current job ID
	moduleName   string             // Current module name
}

// NewProgressDisplay creates a new progress display
func NewProgressDisplay() *ProgressDisplay {
	return &ProgressDisplay{
		stdout:       os.Stdout,
		useColors:    true,
		refreshCount: make(map[string]int),
	}
}

// StartOperation begins tracking a specific terraform operation with animation
func (p *ProgressDisplay) StartOperation(jobID, moduleName string, op TerraformOperation) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.isActive = true
	p.startTime = time.Now()
	p.lastUpdate = time.Now()
	p.spinnerIndex = 0
	p.currentOp = op
	p.jobID = jobID
	p.moduleName = moduleName

	// Print operation start without newline (for cursor animation)
	timestamp := time.Now().Format("15:04:05")
	opLabel := p.formatOperationLabel(op, StatusRunning)

	if p.useColors {
		fmt.Fprintf(p.stdout, "[%s] %s %s... ", colorGray+timestamp+colorReset, opLabel, moduleName)
	} else {
		fmt.Fprintf(p.stdout, "[%s] %s %s... ", timestamp, opLabel, moduleName)
	}

	go p.animateOperation()
}

// StopOperation ends the operation with a result status
func (p *ProgressDisplay) StopOperation(status OperationStatus, details string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.isActive = false
	p.clearLine()

	timestamp := time.Now().Format("15:04:05")
	elapsed := time.Since(p.startTime).Round(time.Millisecond)

	// Format the result line
	opLabel := p.formatOperationLabel(p.currentOp, status)
	statusIcon := p.getStatusIcon(status)

	if p.useColors {
		fmt.Fprintf(p.stdout, "[%s] %s %s %s (%s)%s\n",
			colorGray+timestamp+colorReset,
			statusIcon,
			opLabel,
			p.moduleName,
			elapsed,
			p.formatDetails(details))
	} else {
		fmt.Fprintf(p.stdout, "[%s] %s %s %s (%s)%s\n",
			timestamp,
			statusIcon,
			opLabel,
			p.moduleName,
			elapsed,
			p.formatDetails(details))
	}
}

// formatOperationLabel returns a colored, formatted label for the operation
func (p *ProgressDisplay) formatOperationLabel(op TerraformOperation, status OperationStatus) string {
	var color string
	var label string

	switch op {
	case OpInit:
		label = "terraform init"
		color = colorCyan
	case OpPlan:
		label = "terraform plan"
		color = colorBlue
	case OpApply:
		label = "terraform apply"
		color = colorYellow
	case OpDestroy:
		label = "terraform destroy"
		color = colorRed
	case OpValidate:
		label = "terraform validate"
		color = colorBlue
	default:
		label = string(op)
		color = colorGray
	}

	// Adjust color based on status
	if status == StatusSuccess {
		color = colorGreen
	} else if status == StatusFailed {
		color = colorRed
	}

	if p.useColors {
		return color + label + colorReset
	}
	return label
}

// getStatusIcon returns an icon based on status
func (p *ProgressDisplay) getStatusIcon(status OperationStatus) string {
	if !p.useColors {
		switch status {
		case StatusSuccess:
			return "✓"
		case StatusFailed:
			return "✗"
		default:
			return "→"
		}
	}

	switch status {
	case StatusSuccess:
		return colorGreen + "✓" + colorReset
	case StatusFailed:
		return colorRed + "✗" + colorReset
	case StatusRunning:
		return colorBlue + "→" + colorReset
	default:
		return colorGray + "·" + colorReset
	}
}

// formatDetails formats additional details (like plan summary)
func (p *ProgressDisplay) formatDetails(details string) string {
	if details == "" {
		return ""
	}
	if p.useColors {
		return " " + colorGray + "- " + details + colorReset
	}
	return " - " + details
}

// animateOperation runs the spinner animation at cursor position
func (p *ProgressDisplay) animateOperation() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		if !p.isActive {
			p.mu.Unlock()
			return
		}

		// Move cursor back, print spinner and elapsed time, but don't newline
		spinner := spinnerFrames[p.spinnerIndex]
		p.spinnerIndex = (p.spinnerIndex + 1) % len(spinnerFrames)

		elapsed := time.Since(p.startTime).Round(time.Second)

		// Overwrite the ellipsis with spinner and time
		if p.useColors {
			fmt.Fprintf(p.stdout, "\r[%s] %s %s... %s%s%s (%s)",
				colorGray+time.Now().Format("15:04:05")+colorReset,
				p.formatOperationLabel(p.currentOp, StatusRunning),
				p.moduleName,
				colorBlue,
				spinner,
				colorReset,
				elapsed)
		} else {
			fmt.Fprintf(p.stdout, "\r[%s] %s %s... %s (%s)",
				time.Now().Format("15:04:05"),
				p.formatOperationLabel(p.currentOp, StatusRunning),
				p.moduleName,
				spinner,
				elapsed)
		}
		p.mu.Unlock()
	}
}

// Update changes the current status message (for legacy compatibility)
func (p *ProgressDisplay) Update(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.currentLine = message
	p.lastUpdate = time.Now()
}

// clearLine clears the current terminal line
func (p *ProgressDisplay) clearLine() {
	fmt.Fprint(p.stdout, "\r\033[K")
}

// TerraformOutputParser parses and deduplicates terraform output
type TerraformOutputParser struct {
	mu              sync.Mutex
	refreshCount    int
	lastRefreshMsg  string
	resourceChanges []string
	planSummary     string
	display         *ProgressDisplay
}

// NewTerraformOutputParser creates a new parser
func NewTerraformOutputParser(display *ProgressDisplay) *TerraformOutputParser {
	return &TerraformOutputParser{
		display: display,
	}
}

// ParseLine processes a line of terraform output
func (p *TerraformOutputParser) ParseLine(line string) (shouldPrint bool, formatted string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Strip ANSI codes for analysis
	cleanLine := StripANSI(line)

	// Detect refresh operations
	if strings.Contains(cleanLine, "Refreshing state...") || strings.Contains(cleanLine, "Reading...") {
		p.refreshCount++
		p.lastRefreshMsg = cleanLine
		return false, "" // Don't print individual refresh lines
	}

	// Detect planning phase
	if strings.Contains(cleanLine, "Planning...") {
		return false, "" // Suppress verbose planning messages
	}

	// Detect resource changes
	if strings.HasPrefix(strings.TrimSpace(cleanLine), "+") ||
		strings.HasPrefix(strings.TrimSpace(cleanLine), "~") ||
		strings.HasPrefix(strings.TrimSpace(cleanLine), "-") {
		// Extract resource name
		parts := strings.Fields(cleanLine)
		if len(parts) >= 2 {
			p.resourceChanges = append(p.resourceChanges, parts[1])
		}
		return false, "" // Don't print individual resource lines during operation
	}

	// Detect plan summary
	if strings.Contains(cleanLine, "Plan:") {
		p.planSummary = cleanLine
		// Debug
		if debugFile, _ := os.OpenFile("/tmp/tfpipboy-parser-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); debugFile != nil {
			fmt.Fprintf(debugFile, "[ParseLine] Captured plan summary: %s\n", cleanLine)
			debugFile.Close()
		}
		return false, "" // Don't print here, will show in StopOperation
	}

	// Detect "No changes" message
	if strings.Contains(cleanLine, "No changes.") && strings.Contains(cleanLine, "infrastructure matches") {
		p.planSummary = "No changes. Your infrastructure matches the configuration."
		// Debug
		if debugFile, _ := os.OpenFile("/tmp/tfpipboy-parser-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); debugFile != nil {
			fmt.Fprintf(debugFile, "[ParseLine] Captured no changes\n")
			debugFile.Close()
		}
		return false, ""
	}

	// Print errors and warnings immediately
	if strings.Contains(cleanLine, "Error:") || strings.Contains(cleanLine, "Warning:") {
		return true, line
	}

	// Skip verbose/redundant lines
	if strings.Contains(cleanLine, "Terraform has been successfully initialized") ||
		strings.Contains(cleanLine, "Terraform will perform the following actions") ||
		strings.Contains(cleanLine, "unless you have made equivalent changes") ||
		strings.Contains(cleanLine, "Initializing the backend...") ||
		strings.Contains(cleanLine, "Initializing provider plugins...") {
		return false, ""
	}

	// Suppress most other output during operations
	return false, ""
}

// GetSummary returns the parsed summary information for the operation
func (p *TerraformOutputParser) GetSummary() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.planSummary != "" {
		return StripANSI(p.planSummary)
	}

	if p.refreshCount > 0 {
		return fmt.Sprintf("%d resources checked", p.refreshCount)
	}

	return ""
}

// Reset clears the parser state
func (p *TerraformOutputParser) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.refreshCount = 0
	p.lastRefreshMsg = ""
	p.resourceChanges = nil
	p.planSummary = ""
}

// StripANSI removes ANSI escape codes from a string
func StripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

// formatResourceChange formats a resource change line
func formatResourceChange(line string) string {
	cleanLine := StripANSI(line)
	prefix := strings.TrimSpace(cleanLine[:1])

	var color string
	var symbol string
	switch prefix {
	case "+":
		color = colorGreen
		symbol = "+"
	case "~":
		color = colorYellow
		symbol = "~"
	case "-":
		color = colorRed
		symbol = "-"
	default:
		return line
	}

	return fmt.Sprintf("    %s%s%s %s", color, symbol, colorReset, strings.TrimSpace(cleanLine[1:]))
}

// formatPlanSummary formats the plan summary line
func formatPlanSummary(line string) string {
	cleanLine := StripANSI(line)
	return fmt.Sprintf("\n%s%s%s\n", colorBlue, cleanLine, colorReset)
}

// ExecutionSummary formats the final execution summary with instance details table
func ExecutionSummary(result *ExecutionResult, artifacts bool) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("================================================================================\n")
	sb.WriteString("   EXECUTION SUMMARY\n")
	sb.WriteString("================================================================================\n")

	// Status line
	status := "✓ SUCCESS"
	statusColor := colorGreen
	if result.Failed > 0 {
		status = "✗ FAILED"
		statusColor = colorRed
	}
	sb.WriteString(fmt.Sprintf("Status: %s%s%s (%s)\n", statusColor, status, colorReset, result.Duration.Round(time.Second)))

	// Job statistics
	sb.WriteString(fmt.Sprintf("Jobs:   %d total, %d completed, %d failed, %d skipped\n\n", result.TotalJobs, result.Completed, result.Failed, result.Skipped))

	// Instance details table
	if len(result.Jobs) > 0 {
		sb.WriteString("Instance Execution Details:\n")
		sb.WriteString("--------------------------------------------------------------------------------\n")
		sb.WriteString(fmt.Sprintf("%-22s %-8s %-10s %-50s\n", "Instance", "Status", "Duration", "Result"))
		sb.WriteString("--------------------------------------------------------------------------------\n")

		for _, job := range result.Jobs {
			// Status indicator
			statusIndicator := "✓"
			statusColor := colorGreen
			if job.Status == JobStatusFailed {
				statusIndicator = "✗"
				statusColor = colorRed
			} else if job.Status == JobStatusSkipped {
				statusIndicator = "○"
				statusColor = colorYellow
			}

			// Instance name (use instance ID which is the full instance name)
			instanceName := job.ID
			if len(instanceName) > 22 {
				instanceName = instanceName[:19] + "..."
			}

			// Duration
			durationStr := "-"
			if job.Duration > 0 {
				durationStr = job.Duration.Round(time.Second).String()
			}

			// Build result message based on status
			var resultMsg string
			if job.Status == JobStatusFailed {
				// For failed jobs, show error message
				if job.Error != "" {
					// Extract first line of error or categorized error type
					errorLines := strings.Split(job.Error, "\n")
					resultMsg = errorLines[0]
					// Truncate if too long
					if len(resultMsg) > 48 {
						resultMsg = resultMsg[:45] + "..."
					}
					resultMsg = fmt.Sprintf("%s%s%s", colorRed, resultMsg, colorReset)
				} else {
					resultMsg = fmt.Sprintf("%sFailed (no error details)%s", colorRed, colorReset)
				}
			} else if job.PlanResult != "" {
				// For successful plan operations, show plan result
				planResult := job.PlanResult
				if strings.Contains(planResult, "No changes") {
					resultMsg = fmt.Sprintf("%sNo changes%s", colorGreen, colorReset)
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
					resultMsg = strings.Join(coloredParts, " ")
				} else {
					resultMsg = planResult
				}
			} else if job.Status == JobStatusCompleted {
				// Completed without plan result (e.g., init, validate)
				resultMsg = fmt.Sprintf("%sCompleted successfully%s", colorGreen, colorReset)
			} else if job.Status == JobStatusSkipped {
				resultMsg = fmt.Sprintf("%sSkipped%s", colorYellow, colorReset)
			} else {
				resultMsg = "-"
			}

			// Format the row
			sb.WriteString(fmt.Sprintf("%-22s %s%-8s%s %-10s %s\n",
				instanceName,
				statusColor, statusIndicator, colorReset,
				durationStr,
				resultMsg))
		}
		sb.WriteString("--------------------------------------------------------------------------------\n\n")
	}

	if artifacts {
		sb.WriteString("Artifacts preserved:\n")
		sb.WriteString("  • Workspaces: .tfpipboy/workspaces/ (terraform execution environments)\n")
		sb.WriteString("  • Logs:       .tfpipboy/logs/ (detailed execution logs)\n")
		sb.WriteString("\n")
		sb.WriteString("Cleanup commands:\n")
		sb.WriteString("  tfpipboy --cleanup               # Interactive cleanup\n")
		sb.WriteString("  tfpipboy --cleanup --all         # Remove all artifacts\n")
		sb.WriteString("  tfpipboy --cleanup --older-than 7d  # Retention-based cleanup\n")
	}

	sb.WriteString("================================================================================\n")

	return sb.String()
}

// StageHeader formats a stage header
func StageHeader(stageNum, totalStages int, stageName string) string {
	return fmt.Sprintf("\n%s━━━ Stage %d/%d: %s ━━━%s\n", colorBlue, stageNum, totalStages, stageName, colorReset)
}

// ExecutionPlanPreview displays a detailed execution plan with stages, backend types, and auth status
func ExecutionPlanPreview(plan *ExecutionPlan, authStatus map[string]AuthStatus) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("================================================================================\n")
	sb.WriteString("   EXECUTION PLAN PREVIEW\n")
	sb.WriteString("================================================================================\n")
	sb.WriteString(fmt.Sprintf("Operation:   %s\n", plan.Operation))
	sb.WriteString(fmt.Sprintf("Environment: %s\n", plan.Environment))
	sb.WriteString(fmt.Sprintf("Total Jobs:  %d\n\n", plan.TotalJobs))

	// Authentication Status
	if len(authStatus) > 0 {
		sb.WriteString("Authentication Status:\n")
		for _, status := range authStatus {
			statusIcon := colorRed + "✗" + colorReset
			statusText := "Not authenticated"
			if status.Authenticated {
				statusIcon = colorGreen + "✓" + colorReset
				statusText = status.Identity

				// Show active account for GitHub with multiple accounts
				if status.Provider == "GitHub" && len(status.Accounts) > 1 {
					statusText = fmt.Sprintf("%s (active: %s, available: %d accounts)",
						status.ActiveAccount, status.ActiveAccount, len(status.Accounts))
				} else if status.Provider == "GitHub" && status.ActiveAccount != "" {
					statusText = fmt.Sprintf("%s (active)", status.ActiveAccount)
				}
			} else if status.Error != "" {
				// Use specific error message if available
				statusText = status.Error
			}
			sb.WriteString(fmt.Sprintf("  %s %-8s %s\n", statusIcon, status.Provider+":", statusText))
		}
		sb.WriteString("\n")
	}

	// Group jobs by category, then by dependency depth within each category
	categoryMap := make(map[string][]*ExecutionJob)

	for _, job := range plan.Modules {
		category := job.Category
		if category == "" {
			category = "uncategorized"
		}
		categoryMap[category] = append(categoryMap[category], job)
	}

	// Sort jobs within each category by dependency depth, then by config order
	for _, jobs := range categoryMap {
		sort.Slice(jobs, func(i, j int) bool {
			if jobs[i].DependencyDepth != jobs[j].DependencyDepth {
				return jobs[i].DependencyDepth < jobs[j].DependencyDepth
			}
			return jobs[i].ID < jobs[j].ID
		})
	}

	// Get sorted category list
	var categories []string
	for category := range categoryMap {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	// Execution Plan Table
	sb.WriteString("Execution Sequence:\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("%-6s %-5s %-20s %-12s %-12s %s\n", "Stage", "Depth", "Instance", "Backend", "Details", "Dependencies"))
	sb.WriteString("--------------------------------------------------------------------------------\n")

	stageCounter := 1
	for _, category := range categories {
		jobs := categoryMap[category]

		// Category header
		categoryLabel := strings.ToUpper(category)
		sb.WriteString(fmt.Sprintf("\n%s[%s]%s\n", colorCyan, categoryLabel, colorReset))

		// Track which depth we're currently at for stage numbering
		currentDepth := -1
		stageInDepth := stageCounter

		for _, job := range jobs {
			// Update stage counter when depth changes
			if job.DependencyDepth != currentDepth {
				currentDepth = job.DependencyDepth
				stageInDepth = stageCounter
				stageCounter++
			}

			// Stage number (use stageInDepth for jobs at the same depth)
			stageStr := fmt.Sprintf("%d", stageInDepth)

			// Check if there are other jobs at the same depth in this category
			parallelCount := 0
			for _, otherJob := range jobs {
				if otherJob.DependencyDepth == job.DependencyDepth {
					parallelCount++
				}
			}
			if parallelCount > 1 {
				stageStr += "∥" // Parallel indicator
			}

			// Dependency depth
			depthStr := fmt.Sprintf("%d", job.DependencyDepth)
			if job.DependencyDepth == 0 {
				depthStr = "-"
			}

			// Instance name
			instanceName := job.ID
			if len(instanceName) > 20 {
				instanceName = instanceName[:17] + "..."
			}

			// Backend type
			backendType := "local"
			if job.Backend != nil && job.Backend.Type != "" {
				backendType = job.Backend.Type
			}

			// Details (environment/region)
			details := ""
			if job.Environment != "" && job.Region != "" {
				details = fmt.Sprintf("%s/%s", job.Environment, job.Region)
			} else if job.Environment != "" {
				details = job.Environment
			} else if job.Region != "" {
				details = job.Region
			}

			// Dependencies
			deps := "-"
			if len(job.DependsOn) > 0 {
				deps = strings.Join(job.DependsOn, ", ")
				if len(deps) > 20 {
					deps = deps[:17] + "..."
				}
			}

			sb.WriteString(fmt.Sprintf("%-6s %-5s %-20s %-12s %-12s %s\n",
				stageStr,
				depthStr,
				instanceName,
				backendType,
				details,
				deps))
		}
	}

	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("\n%sNote:%s ∥ indicates parallel execution within the same stage\n", colorYellow, colorReset))
	sb.WriteString("================================================================================\n")

	return sb.String()
}

// DisplayPlanSummary shows a formatted plan summary with artifact locations
func DisplayPlanSummary(jobID, moduleName, planTextFile string) {
	fmt.Printf("\n%s━━━ Plan Summary: %s ━━━%s\n", colorBlue, moduleName, colorReset)

	// Read and display key parts of the plan
	if planTextFile != "" {
		if content, err := os.ReadFile(planTextFile); err == nil {
			lines := strings.Split(string(content), "\n")
			inSummary := false

			for _, line := range lines {
				// Look for the Plan: line which has the summary
				if strings.Contains(line, "Plan:") || inSummary {
					inSummary = true
					fmt.Printf("  %s\n", line)

					// Stop after a few lines of summary
					if inSummary && strings.TrimSpace(line) == "" {
						break
					}
				}
			}
		}
	}

	fmt.Printf("\n%sArtifacts saved:%s\n", colorGray, colorReset)
	fmt.Printf("  • Plan file:  %s\n", planTextFile)
	fmt.Printf("  • JSON plan:  %s\n", strings.Replace(planTextFile, ".txt", ".json", 1))
	fmt.Printf("  • Binary plan: %s\n", strings.Replace(planTextFile, ".txt", "", 1))
	fmt.Printf("\n%sTo apply this plan:%s\n", colorGray, colorReset)
	fmt.Printf("  tfpipboy --config . --targets %s --operation apply\n", moduleName)
	fmt.Println()
}
