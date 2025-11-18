package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TerraformExecutor handles execution of Terraform commands in isolated workspaces
type TerraformExecutor struct {
	workspaceManager    WorkspaceManager
	logger              Logger
	timeout             time.Duration
	dryRun              bool
	progressDisplay     *ProgressDisplay
	outputParser        *TerraformOutputParser
	suppressProgressBar bool                     // Suppress individual progress bars during parallel execution
	eventChan           chan<- StatusUpdateEvent // Channel for sending status update events (for LiveBoard)
	eventBus            *ExecutorEventBus        // Event bus for async communication (no locks needed!)
	currentJobID        string                   // Current job ID for routing messages
	jobPlanSummaries    map[string]string        // Per-job plan summaries (to avoid shared parser race)
	mu                  sync.Mutex               // Mutex ONLY for jobPlanSummaries (local data structure)
}

// NewTerraformExecutor creates a new Terraform executor
func NewTerraformExecutor(workspaceManager WorkspaceManager, logger Logger) *TerraformExecutor {
	display := NewProgressDisplay()
	return &TerraformExecutor{
		workspaceManager: workspaceManager,
		logger:           logger,
		timeout:          30 * time.Minute,
		dryRun:           false,
		progressDisplay:  display,
		outputParser:     NewTerraformOutputParser(display),
		jobPlanSummaries: make(map[string]string),
	}
}

// SetTimeout sets the execution timeout
func (e *TerraformExecutor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

// SetDryRun sets dry run mode
func (e *TerraformExecutor) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
}

// SetEventBus sets the event bus for async communication
func (e *TerraformExecutor) SetEventBus(eventBus *ExecutorEventBus) {
	e.eventBus = eventBus
}

// ExecuteJob executes a single Terraform job
func (e *TerraformExecutor) ExecuteJob(ctx context.Context, job *ExecutionJob, suppressOutput bool) error {
	if !suppressOutput {
		e.logger.Info("Starting job execution", "job_id", job.ID, "module", job.ModuleName, "operation", job.Operation)
	}

	// Update job status
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartTime = &now

	// Get or create workspace
	workspace, err := e.workspaceManager.GetWorkspace(job.ID)
	if err != nil {
		// Try to create workspace if it doesn't exist
		module, instance, err := e.getModuleAndInstance(job)
		if err != nil {
			return e.failJob(job, fmt.Errorf("failed to get module configuration: %w", err))
		}

		workspace, err = e.workspaceManager.CreateWorkspace(job.ID, module, instance)
		if err != nil {
			return e.failJob(job, fmt.Errorf("failed to create workspace: %w", err))
		}
	}

	// Execute the operation
	switch TerraformOperation(job.Operation) {
	case OpInit:
		err = e.executeInit(ctx, job, workspace, suppressOutput)
	case OpPlan:
		err = e.executePlan(ctx, job, workspace, suppressOutput)
	case OpApply:
		err = e.executeApply(ctx, job, workspace, suppressOutput)
	case OpDestroy:
		err = e.executeDestroy(ctx, job, workspace, suppressOutput)
	case OpValidate:
		err = e.executeValidate(ctx, job, workspace, suppressOutput)
	case OpRefresh:
		err = e.executeRefresh(ctx, job, workspace, suppressOutput)
	case OpOutput:
		err = e.executeOutput(ctx, job, workspace, suppressOutput)
	default:
		err = fmt.Errorf("unsupported operation: %s", job.Operation)
	}

	// Update job completion
	endTime := time.Now()
	job.EndTime = &endTime
	job.Duration = job.EndTime.Sub(*job.StartTime)

	if err != nil {
		// Save metadata even on failure
		e.saveJobMetadata(job, workspace, 1)
		return e.failJob(job, err)
	}

	job.Status = JobStatusCompleted
	if !suppressOutput {
		e.logger.Info("Job completed successfully", "job_id", job.ID, "duration", job.Duration)
	}

	// Save job metadata as artifact
	e.saveJobMetadata(job, workspace, 0)

	return nil
}

// executeInit runs terraform init
func (e *TerraformExecutor) executeInit(ctx context.Context, job *ExecutionJob, workspace *Workspace, suppressOutput bool) error {
	// Validate backend authentication before attempting init
	if workspace.Backend != nil && workspace.Backend.Type != "" && workspace.Backend.Type != "local" {
		e.logger.Debug("Validating backend authentication",
			"job_id", job.ID,
			"backend_type", workspace.Backend.Type)

		authStatus := ValidateBackendAuth(ctx, workspace.Backend.Type)

		if !authStatus.Authenticated {
			// Authentication failed - provide helpful error message
			errMsg := fmt.Sprintf("Backend authentication failed for '%s' backend.\n\n", workspace.Backend.Type)
			errMsg += fmt.Sprintf("Status: %s\n", authStatus.Message)
			if authStatus.Error != "" {
				errMsg += fmt.Sprintf("Details: %s\n", authStatus.Error)
			}
			errMsg += "\nPlease authenticate before running Terraform:\n"
			authCmd := GetAuthenticationCommand(workspace.Backend.Type)
			if authCmd != "" {
				errMsg += fmt.Sprintf("  $ %s\n", authCmd)
			}

			e.logger.Error("Backend authentication check failed",
				"job_id", job.ID,
				"backend_type", workspace.Backend.Type,
				"status", authStatus.Message)

			return fmt.Errorf("%s", errMsg)
		}

		// Authentication successful
		if !suppressOutput {
			e.logger.Info("Backend authentication validated",
				"job_id", job.ID,
				"backend_type", workspace.Backend.Type,
				"details", authStatus.Details)
		}
	}

	args := []string{"init"}

	// Disable interactive input
	args = append(args, "-input=false")

	// Add backend configuration
	if workspace.Backend != nil {
		// Check if there's a backend HCL file (external config)
		backendHCL := filepath.Join(workspace.Path, "backend.hcl")
		if _, err := os.Stat(backendHCL); err == nil {
			args = append(args, fmt.Sprintf("-backend-config=%s", backendHCL))
		}
		// Otherwise the backend.tf file generated by workspace manager will be used
	}

	// Add upgrade flag for safety
	args = append(args, "-upgrade")

	// Start operation display - use instance ID directly as the display name
	displayName := job.InstanceName // Instance ID (e.g., "seed-dev-gwc")
	if displayName == "" {
		displayName = job.ModuleName
	}
	if !suppressOutput {
		e.progressDisplay.StartOperation(job.ID, displayName, OpInit)
	}

	// Send step update event
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "step",
		StepName:  "Init",
	})
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "progress",
		Progress:  "Running terraform init...",
	})

	e.outputParser.Reset()

	// Execute terraform init
	err := e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)

	// Stop operation display with result
	status := StatusSuccess
	if err != nil {
		status = StatusFailed
	}
	if !suppressOutput {
		e.progressDisplay.StopOperation(status, "")
	}

	return err
}

// sendEvent sends a status update event to the event channel if available
func (e *TerraformExecutor) sendEvent(event StatusUpdateEvent) {
	e.mu.Lock()
	eventChan := e.eventChan
	e.mu.Unlock()

	if eventChan != nil {
		select {
		case eventChan <- event:
			// Event sent successfully
		default:
			// Channel full or closed - event dropped
			// This is acceptable for non-critical status updates
		}
	}
}

// executePlan runs terraform plan
func (e *TerraformExecutor) executePlan(ctx context.Context, job *ExecutionJob, workspace *Workspace, suppressOutput bool) error {
	// Ensure init is run first
	if err := e.executeInit(ctx, job, workspace, suppressOutput); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	// Create plan file path in workspace (will be saved as artifact)
	planFile := filepath.Join(workspace.Path, "terraform.tfplan")
	planJSONFile := filepath.Join(workspace.Path, "terraform.tfplan.json")
	planTextFile := filepath.Join(workspace.Path, "terraform.tfplan.txt")

	args := []string{"plan"}

	// Disable interactive input
	args = append(args, "-input=false")

	// Add variables
	if err := e.addVariableArgs(&args, workspace); err != nil {
		return fmt.Errorf("failed to add variables: %w", err)
	}

	// Save plan to file for later apply
	args = append(args, fmt.Sprintf("-out=%s", planFile))

	// Add detailed exitcode
	args = append(args, "-detailed-exitcode")

	// Start operation display
	displayName := job.InstanceName
	if displayName == "" {
		displayName = job.ModuleName
	}
	if !suppressOutput {
		e.progressDisplay.StartOperation(job.ID, displayName, OpPlan)
	}

	// Send step update event
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "step",
		StepName:  "Plan",
	})
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "progress",
		Progress:  "Running terraform plan...",
	})

	e.outputParser.Reset()

	// Execute terraform plan
	err := e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)

	// Get plan summary from per-job capture (bypasses shared parser race)
	e.mu.Lock()
	summary := e.jobPlanSummaries[job.ID]
	delete(e.jobPlanSummaries, job.ID) // Clean up
	e.mu.Unlock()

	// Fallback to parser if capture missed it (shouldn't happen but defensive)
	if summary == "" {
		summary = e.outputParser.GetSummary()
	}

	// Send plan result event and save to job
	if summary != "" {
		planResult := formatPlanSummaryForLiveBoard(summary)
		job.PlanResult = planResult // Save to job for execution summary

		e.sendEvent(StatusUpdateEvent{
			JobID:      job.ID,
			EventType:  "plan_result",
			PlanResult: planResult,
		})

		// Publish plan result event (async, no locks!)
		if e.eventBus != nil {
			e.eventBus.Publish(&ExecutorEvent{
				Type:      EventPlanResult,
				JobID:     job.ID,
				Timestamp: time.Now().Unix(),
				Data: &PlanResultEvent{
					Summary:          summary,
					FormattedSummary: planResult,
				},
			})
		}
	}

	// If plan succeeded, save JSON and human-readable outputs
	if err == nil {
		// Generate JSON plan output
		if planErr := e.savePlanJSON(ctx, job, workspace, planFile, planJSONFile); planErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save plan JSON: %v\n", planErr)
		}

		// Generate human-readable plan output
		if planErr := e.savePlanText(ctx, job, workspace, planFile, planTextFile, suppressOutput); planErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save plan text: %v\n", planErr)
		}

		// Display plan summary with artifacts
		if !suppressOutput {
			DisplayPlanSummary(job.ID, displayName, planTextFile)
		}
	}

	// Stop operation display with result
	status := StatusSuccess
	if err != nil {
		status = StatusFailed
	}
	if !suppressOutput {
		e.progressDisplay.StopOperation(status, summary)
	}

	return err
}

// executeApply runs terraform apply
func (e *TerraformExecutor) executeApply(ctx context.Context, job *ExecutionJob, workspace *Workspace, suppressOutput bool) error {
	if e.dryRun {
		e.logger.Info("Dry run mode - skipping apply", "job_id", job.ID)
		return e.executePlan(ctx, job, workspace, suppressOutput)
	}

	// Ensure init is run first
	if err := e.executeInit(ctx, job, workspace, suppressOutput); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	// Check if we have a saved plan file from previous plan operation
	planFile := filepath.Join(workspace.Path, "terraform.tfplan")
	useSavedPlan := false
	if _, err := os.Stat(planFile); err == nil {
		useSavedPlan = true
		e.logger.Info("Using saved plan file for apply", "job_id", job.ID, "plan_file", planFile)
	}

	args := []string{"apply"}

	if useSavedPlan {
		// Apply the saved plan file (no need for -auto-approve with plan file)
		args = append(args, planFile)
	} else {
		// No saved plan, apply directly with variables
		e.logger.Warn("No saved plan file found, running apply without plan", "job_id", job.ID)

		// Add auto-approve for non-interactive execution
		args = append(args, "-auto-approve")

		// Disable interactive input
		args = append(args, "-input=false")

		// Add variables
		if err := e.addVariableArgs(&args, workspace); err != nil {
			return fmt.Errorf("failed to add variables: %w", err)
		}
	}

	// Start operation display
	displayName := job.InstanceName
	if displayName == "" {
		displayName = job.ModuleName
	}
	if !suppressOutput {
		e.progressDisplay.StartOperation(job.ID, displayName, OpApply)
	}

	// Send step update event
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "step",
		StepName:  "Apply",
	})
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "progress",
		Progress:  "Running terraform apply...",
	})

	e.outputParser.Reset()

	// Execute terraform apply
	err := e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)

	// Stop operation display with result
	status := StatusSuccess
	if err != nil {
		status = StatusFailed
	}
	e.progressDisplay.StopOperation(status, "")

	return err
}

// executeDestroy runs terraform destroy
func (e *TerraformExecutor) executeDestroy(ctx context.Context, job *ExecutionJob, workspace *Workspace, suppressOutput bool) error {
	if e.dryRun {
		e.logger.Info("Dry run mode - skipping destroy", "job_id", job.ID)
		// Run plan with destroy flag instead
		return e.executeDestroyPlan(ctx, job, workspace, suppressOutput)
	}

	// Ensure init is run first
	if err := e.executeInit(ctx, job, workspace, suppressOutput); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	args := []string{"destroy"}

	// Add auto-approve for non-interactive execution
	args = append(args, "-auto-approve")

	// Disable interactive input
	args = append(args, "-input=false")

	// Add variables
	if err := e.addVariableArgs(&args, workspace); err != nil {
		return fmt.Errorf("failed to add variables: %w", err)
	}

	// Start operation display
	displayName := job.InstanceName
	if displayName == "" {
		displayName = job.ModuleName
	}
	if !suppressOutput {
		e.progressDisplay.StartOperation(job.ID, displayName, OpDestroy)
	}

	// Send step update event
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "step",
		StepName:  "Destroy",
	})
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "progress",
		Progress:  "Running terraform destroy...",
	})

	e.outputParser.Reset()

	// Execute terraform destroy
	err := e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)

	// Stop operation display with result
	status := StatusSuccess
	if err != nil {
		status = StatusFailed
	}
	e.progressDisplay.StopOperation(status, "")

	return err
}

// executeDestroyPlan runs terraform plan -destroy
func (e *TerraformExecutor) executeDestroyPlan(ctx context.Context, job *ExecutionJob, workspace *Workspace, suppressOutput bool) error {
	// Ensure init is run first
	if err := e.executeInit(ctx, job, workspace, suppressOutput); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	args := []string{"plan", "-destroy"}

	// Add variables
	if err := e.addVariableArgs(&args, workspace); err != nil {
		return fmt.Errorf("failed to add variables: %w", err)
	}

	return e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)
}

// executeValidate runs terraform validate
func (e *TerraformExecutor) executeValidate(ctx context.Context, job *ExecutionJob, workspace *Workspace, suppressOutput bool) error {
	// Ensure init is run first
	if err := e.executeInit(ctx, job, workspace, suppressOutput); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	args := []string{"validate"}
	args = append(args, "-json") // Get JSON output for better parsing

	// Send step update event
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "step",
		StepName:  "Validate",
	})
	e.sendEvent(StatusUpdateEvent{
		JobID:     job.ID,
		EventType: "progress",
		Progress:  "Running terraform validate...",
	})

	return e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)
}

// executeRefresh runs terraform refresh
func (e *TerraformExecutor) executeRefresh(ctx context.Context, job *ExecutionJob, workspace *Workspace, suppressOutput bool) error {
	// Ensure init is run first
	if err := e.executeInit(ctx, job, workspace, suppressOutput); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	args := []string{"refresh"}

	// Add variables
	if err := e.addVariableArgs(&args, workspace); err != nil {
		return fmt.Errorf("failed to add variables: %w", err)
	}

	return e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)
}

// executeOutput runs terraform output
func (e *TerraformExecutor) executeOutput(ctx context.Context, job *ExecutionJob, workspace *Workspace, suppressOutput bool) error {
	args := []string{"output", "-json"}

	return e.runTerraformCommand(ctx, job, workspace, args, suppressOutput)
}

// addVariableArgs adds variable arguments to terraform command
func (e *TerraformExecutor) addVariableArgs(args *[]string, workspace *Workspace) error {
	// Add all variable files from workspace.VarFiles (new approach)
	if len(workspace.VarFiles) > 0 {
		for _, varFile := range workspace.VarFiles {
			*args = append(*args, fmt.Sprintf("-var-file=%s", varFile))
		}
	} else {
		// Backward compatibility: check for terraform.tfvars file
		tfvarsFile := filepath.Join(workspace.Path, "terraform.tfvars")
		if _, err := os.Stat(tfvarsFile); err == nil {
			*args = append(*args, fmt.Sprintf("-var-file=%s", tfvarsFile))
		}
	}

	// Add individual variables (from json/vars config or inline variables)
	// Terraform requires: -var 'key=value' (two separate arguments)
	for key, value := range workspace.Variables {
		*args = append(*args, "-var", fmt.Sprintf("%s=%s", key, value))
	}

	return nil
}

// runTerraformCommand executes a terraform command with proper isolation
func (e *TerraformExecutor) runTerraformCommand(ctx context.Context, job *ExecutionJob, workspace *Workspace, args []string, suppressOutput bool) error {
	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Create the command
	cmd := exec.CommandContext(cmdCtx, "terraform", args...)

	// Set working directory to the module path
	cmd.Dir = workspace.ModulePath

	// Set environment variables
	env := e.buildEnvironment(workspace)
	cmd.Env = env

	// Create log file for this job in persistent logs directory
	logFilePath := filepath.Join(workspace.LogDir, fmt.Sprintf("%s.log", job.ID))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		e.logger.Warn("Failed to create log file", "job_id", job.ID, "path", logFilePath, "error", err)
		logFile = nil // Continue without log file
	}
	if logFile != nil {
		defer func() {
			if err := logFile.Close(); err != nil {
				e.logger.Warn("Failed to close log file", "error", err)
			}
		}()

		// Write header to log file
		header := "=== Terraform Command Log ===\n"
		header += fmt.Sprintf("Instance ID: %s\n", job.ID)
		header += fmt.Sprintf("Module: %s\n", job.ModuleName)
		if job.InstanceName != "" {
			header += fmt.Sprintf("Instance: %s\n", job.InstanceName)
		}
		header += fmt.Sprintf("Operation: %s\n", job.Operation)
		header += fmt.Sprintf("Command: terraform %s\n", strings.Join(args, " "))
		header += fmt.Sprintf("Working Dir: %s\n", cmd.Dir)
		header += fmt.Sprintf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		header += "=============================\n\n"
		if _, err := logFile.WriteString(header); err != nil {
			e.logger.Warn("Failed to write log header", "error", err)
		}
	}

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Format command with multi-line arguments for better readability
	formattedCmd := formatTerraformCommand(args)

	if !suppressOutput {
		e.logger.Info("Executing terraform command",
			"job_id", job.ID,
			"command", formattedCmd,
			"working_dir", cmd.Dir,
			"log_file", logFilePath,
		)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start terraform command: %w", err)
	}

	// Capture output
	var outputBuilder strings.Builder
	var wg sync.WaitGroup

	// Create channels for output monitoring and hang detection
	lastOutputTime := time.Now()
	outputActivity := make(chan struct{}, 100)

	// Set hang timeout based on operation
	// Init should fail fast if hung, but plan/apply/destroy can take long without output
	var hangTimeout time.Duration
	operationName := ""
	if len(args) > 0 {
		operationName = args[0]
	}

	switch operationName {
	case "init":
		// Init should complete quickly - 2 minutes of no output means it's hung
		hangTimeout = 2 * time.Minute
	case "plan", "apply", "destroy":
		// These operations can legitimately take long without output
		// Only detect hang if truly stuck (e.g., 10 minutes no output)
		hangTimeout = 10 * time.Minute
	default:
		hangTimeout = 5 * time.Minute
	}

	// Read stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.streamOutputWithMonitoring(stdout, &outputBuilder, "STDOUT", job.ID, logFile, suppressOutput, outputActivity)
	}()

	// Read stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.streamOutputWithMonitoring(stderr, &outputBuilder, "STDERR", job.ID, logFile, suppressOutput, outputActivity)
	}()

	// Monitor for hangs and interactive prompts
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for command to complete or detect issues
	var cmdErr error
	hangDetected := false

	// Create a timer for hang detection
	hangTimer := time.NewTimer(hangTimeout)
	defer hangTimer.Stop()

monitorLoop:
	for {
		select {
		case cmdErr = <-done:
			// Command completed
			break monitorLoop

		case <-outputActivity:
			// Output received, reset the hang timer
			if !hangTimer.Stop() {
				select {
				case <-hangTimer.C:
				default:
				}
			}
			hangTimer.Reset(hangTimeout)
			lastOutputTime = time.Now()

		case <-hangTimer.C:
			// No output for hangTimeout duration
			timeSinceLastOutput := time.Since(lastOutputTime)
			// Check if process is still running and appears hung
			if cmd.Process != nil {
				hangDetected = true
				e.logger.Warn("Terraform process appears hung (no output)",
					"job_id", job.ID,
					"operation", operationName,
					"time_since_output", timeSinceLastOutput.String(),
					"last_output_preview", e.getLastOutputLines(outputBuilder.String(), 3))

				// For init, check for interactive prompts (common cause of hangs)
				// For plan/apply/destroy, be more lenient (they can pause for user review)
				if operationName == "init" {
					lastOutput := e.getLastOutputLines(outputBuilder.String(), 10)
					if e.looksLikeInteractivePrompt(lastOutput) {
						e.logger.Error("Detected interactive prompt - terraform waiting for input",
							"job_id", job.ID,
							"last_output", lastOutput)
						cmdErr = fmt.Errorf("terraform is waiting for interactive input (already using -input=false)")
					} else {
						cmdErr = fmt.Errorf("terraform init hung (no output for %v) - check backend connectivity", timeSinceLastOutput)
					}

					// Kill the hung init process
					if err := cmd.Process.Kill(); err != nil {
						e.logger.Warn("Failed to kill hung process", "error", err)
					}
					break monitorLoop
				}

				// For plan/apply/destroy, just log the warning but don't kill
				// User can still Ctrl+C if needed
				e.logger.Warn("Long running terraform operation with no output",
					"job_id", job.ID,
					"operation", operationName,
					"duration", timeSinceLastOutput.String(),
					"note", "Press Ctrl+C to cancel if stuck")

				// Reset timer to check again later
				hangTimer.Reset(hangTimeout)
			}

		case <-cmdCtx.Done():
			// Context timeout or cancellation
			if cmd.Process != nil {
				e.logger.Warn("Terraform process timeout, killing process", "job_id", job.ID, "timeout", e.timeout)
				if err := cmd.Process.Kill(); err != nil {
					e.logger.Warn("Failed to kill timed out process", "error", err)
				}
			}
			cmdErr = fmt.Errorf("terraform command timed out after %v", e.timeout)
			break monitorLoop
		}
	}

	// Wait for output goroutines to finish
	wg.Wait()

	// Use the error from monitoring if hang was detected
	if hangDetected && cmdErr != nil {
		err = cmdErr
	} else if cmdErr != nil {
		err = cmdErr
	}

	// Store output
	job.Output = outputBuilder.String()

	// Check if this is a plan command with -detailed-exitcode
	// Exit code 2 means "succeeded with changes" for plan operations
	isPlanWithDetailedExitcode := false
	for _, arg := range args {
		if arg == "plan" {
			isPlanWithDetailedExitcode = true
			break
		}
	}

	// Handle exit codes
	if err != nil {
		// For plan with -detailed-exitcode, exit code 2 means success with changes
		if isPlanWithDetailedExitcode {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 2 {
					// Exit code 2 is success for plan (means there are changes)
					err = nil
				}
			}
		}
	}

	// Handle errors with enhanced logging
	if err != nil {
		// Categorize the error using the new error detection system
		tfError := CategorizeError(job.Output)

		// Write footer to log file with error details
		if logFile != nil {
			footer := "\n=============================\n"
			footer += fmt.Sprintf("Completed: %s\n", time.Now().Format("2006-01-02 15:04:05"))
			footer += "Status: FAILED\n"
			footer += fmt.Sprintf("Exit Code: %v\n", err)
			footer += fmt.Sprintf("Error Category: %s\n", tfError.Category)
			footer += "=============================\n"
			footer += "\n=== ERROR SUMMARY ===\n"
			footer += tfError.Details
			footer += "\n=====================\n"
			if _, err := logFile.WriteString(footer); err != nil {
				e.logger.Warn("Failed to write log footer", "error", err)
			}
		}

		// Create error log file in persistent logs directory
		errorLogPath := filepath.Join(workspace.LogDir, fmt.Sprintf("%s-error.log", job.ID))
		if errorFile, errCreate := os.Create(errorLogPath); errCreate == nil {
			defer func() {
				if err := errorFile.Close(); err != nil {
					e.logger.Warn("Failed to close error log file", "error", err)
				}
			}()

			// Write error log header
			errorHeader := "=== TERRAFORM ERROR LOG ===\n"
			errorHeader += fmt.Sprintf("Instance ID: %s\n", job.ID)
			errorHeader += fmt.Sprintf("Module: %s\n", job.ModuleName)
			if job.InstanceName != "" {
				errorHeader += fmt.Sprintf("Instance: %s\n", job.InstanceName)
			}
			errorHeader += fmt.Sprintf("Operation: %s\n", job.Operation)
			errorHeader += fmt.Sprintf("Command: terraform %s\n", strings.Join(args, " "))
			errorHeader += fmt.Sprintf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
			errorHeader += fmt.Sprintf("Error Category: %s (%s)\n", tfError.Category, tfError.Type)
			if tfError.FilePath != "" {
				if tfError.LineNumber > 0 {
					errorHeader += fmt.Sprintf("Location: %s:%d\n", tfError.FilePath, tfError.LineNumber)
				} else {
					errorHeader += fmt.Sprintf("File: %s\n", tfError.FilePath)
				}
			}
			errorHeader += "===========================\n\n"
			if _, err := errorFile.WriteString(errorHeader); err != nil {
				e.logger.Warn("Failed to write error header", "error", err)
			}

			// Write categorized error details
			_, _ = errorFile.WriteString("=== ERROR DETAILS ===\n")
			_, _ = errorFile.WriteString(tfError.Details)
			_, _ = errorFile.WriteString("\n\n")

			// Write resolution suggestion
			if tfError.Resolution != "" {
				_, _ = errorFile.WriteString("=== SUGGESTED RESOLUTION ===\n")
				_, _ = errorFile.WriteString(tfError.Resolution)
				_, _ = errorFile.WriteString("\n\n")
			}

			_, _ = errorFile.WriteString("=== FULL OUTPUT ===\n")
			_, _ = errorFile.WriteString(job.Output)

			// Only log error when liveboard is not active
			if !suppressOutput {
				e.logger.Error("Error log created", "job_id", job.ID, "error_log", errorLogPath, "category", tfError.Category)
			}
		}

		// Print categorized error to console (suppress when liveboard is active)
		if !suppressOutput {
			e.printCategorizedErrorToConsole(job, tfError, args)
		}

		// Check if it's a context cancellation
		if cmdCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("terraform command timed out after %v", e.timeout)
		}
		if cmdCtx.Err() == context.Canceled {
			return fmt.Errorf("terraform command was cancelled")
		}

		return fmt.Errorf("terraform command failed: %w", err)
	}

	// Write success footer to log file
	if logFile != nil {
		footer := "\n=============================\n"
		footer += fmt.Sprintf("Completed: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		footer += "Status: SUCCESS\n"
		footer += "=============================\n"
		if _, err := logFile.WriteString(footer); err != nil {
			e.logger.Warn("Failed to write success footer", "error", err)
		}
	}

	return nil
}

// streamOutput streams command output and logs it
func (e *TerraformExecutor) streamOutput(reader io.Reader, builder *strings.Builder, prefix, jobID string, logFile *os.File, suppressOutput bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Write to output builder
		builder.WriteString(line)
		builder.WriteString("\n")

		// Parse line with output parser (for plan summary detection, etc.)
		e.outputParser.ParseLine(line)

		// Strip ANSI codes once for reuse
		cleanLine := StripANSI(line)

		// Capture plan summary DIRECTLY per-job (bypass shared parser race)
		if strings.Contains(cleanLine, "Plan:") && (strings.Contains(cleanLine, "to add") || strings.Contains(cleanLine, "to change") || strings.Contains(cleanLine, "to destroy")) {
			e.mu.Lock()
			e.jobPlanSummaries[jobID] = cleanLine
			e.mu.Unlock()
		} else if strings.Contains(cleanLine, "No changes") && strings.Contains(cleanLine, "infrastructure matches") {
			e.mu.Lock()
			e.jobPlanSummaries[jobID] = "No changes. Your infrastructure matches the configuration."
			e.mu.Unlock()
		}

		// Publish terraform output event (async, no locks!)
		if e.eventBus != nil && jobID != "" {
			e.eventBus.Publish(&ExecutorEvent{
				Type:      EventTerraformOutput,
				JobID:     jobID,
				Timestamp: time.Now().Unix(),
				Data: &TerraformOutputEvent{
					Component: "terraform",
					Line:      line,
				},
			})
		}

		// Write to log file if available (strip ANSI codes for clean logs)
		// NOTE: This is the OLD log file system - will be replaced by message pipeline log
		if logFile != nil {
			logLine := fmt.Sprintf("[%s] %s\n", prefix, cleanLine)
			_, _ = logFile.WriteString(logLine)
		}

		// Log the line (with error highlighting)
		// Suppress console logging when liveboard is active (suppressOutput == true)
		if !suppressOutput {
			if prefix == "STDERR" && (strings.Contains(line, "Error:") || strings.Contains(line, "error:") || strings.Contains(line, "ERROR:")) {
				e.logger.Error("Command error output",
					"job_id", jobID,
					"stream", prefix,
					"line", line,
				)
			} else {
				e.logger.Debug("Command output",
					"job_id", jobID,
					"stream", prefix,
					"line", line,
				)
			}
		}
	}
}

// printCategorizedErrorToConsole prints categorized error information to console
func (e *TerraformExecutor) printCategorizedErrorToConsole(job *ExecutionJob, tfError *TerraformError, args []string) {
	// Use the formatted error from the errors module
	formattedError := FormatCategorizedError(job, tfError, args)
	fmt.Print(formattedError)
}

// buildEnvironment builds the environment variables for terraform execution
func (e *TerraformExecutor) buildEnvironment(workspace *Workspace) []string {
	var env []string

	// Get workspace environment
	workspaceEnv := workspace.Environment

	// Add TF_DATA_DIR pointing to workspace
	workspaceEnv["TF_DATA_DIR"] = filepath.Join(workspace.Path, ".terraform")

	// Convert to slice format
	for key, value := range workspaceEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// getModuleAndInstance retrieves module and instance configuration for a job
func (e *TerraformExecutor) getModuleAndInstance(job *ExecutionJob) (*Module, *Instance, error) {
	// This would typically be provided by the orchestrator
	// For now, return nil - the caller should provide this information
	return nil, nil, fmt.Errorf("module configuration not available")
}

// failJob marks a job as failed and logs the error
func (e *TerraformExecutor) failJob(job *ExecutionJob, err error) error {
	job.Status = JobStatusFailed
	job.Error = err.Error()

	endTime := time.Now()
	job.EndTime = &endTime

	if job.StartTime != nil {
		job.Duration = endTime.Sub(*job.StartTime)
	}

	e.logger.Error("Job failed",
		"job_id", job.ID,
		"module", job.ModuleName,
		"operation", job.Operation,
		"error", err.Error(),
		"duration", job.Duration,
	)

	return err
}

// ExecuteJobWithWorkspace executes a job with a pre-configured workspace
func (e *TerraformExecutor) ExecuteJobWithWorkspace(ctx context.Context, job *ExecutionJob, module *Module, instance *Instance, suppressOutput bool) error {
	// Create workspace for this job
	_, err := e.workspaceManager.CreateWorkspace(job.ID, module, instance)
	if err != nil {
		return e.failJob(job, fmt.Errorf("failed to create workspace: %w", err))
	}

	// Execute the job
	return e.ExecuteJob(ctx, job, suppressOutput)
}

// ParallelExecutor handles parallel execution of multiple jobs
type ParallelExecutor struct {
	executor      *TerraformExecutor
	maxConcurrent int
	logger        Logger
	liveBoard     *LiveBoard
	messageRouter *MessageRouter
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(executor *TerraformExecutor, maxConcurrent int, logger Logger) *ParallelExecutor {
	return &ParallelExecutor{
		executor:      executor,
		maxConcurrent: maxConcurrent,
		logger:        logger,
	}
}

// SetMessageRouter sets the message router for the parallel executor
func (pe *ParallelExecutor) SetMessageRouter(router *MessageRouter) {
	pe.messageRouter = router
}

// ExecuteJobs executes multiple jobs in parallel with concurrency control
func (pe *ParallelExecutor) ExecuteJobs(ctx context.Context, jobs []*ExecutionJob, modules map[string]*Module) *ExecutionResult {
	result := &ExecutionResult{
		TotalJobs: len(jobs),
		Jobs:      jobs,
		Outputs:   make(map[string]interface{}),
		StartTime: time.Now(),
	}

	// Create event bus for async communication between Executor and MessageRouter
	eventBus := NewExecutorEventBus(10000)
	pe.executor.SetEventBus(eventBus)

	// Start event subscriber goroutine in MessageRouter
	if pe.messageRouter != nil {
		pe.messageRouter.StartEventSubscriber(eventBus)
	}

	// Initialize instance message pipelines
	if pe.messageRouter != nil {
		for _, job := range jobs {
			operation := TerraformOperation(job.Operation)
			if err := pe.messageRouter.InitializeInstance(job.ID, operation); err != nil {
				pe.logger.Error("Failed to initialize instance pipeline", "instance_id", job.ID, "error", err)
			}
		}
	}

	// Initialize and start live board for parallel execution
	useLiveBoard := len(jobs) > 1
	if useLiveBoard {
		pe.liveBoard = NewLiveBoard(true)

		// Set liveboard in message router (starts batch updater)
		if pe.messageRouter != nil {
			pe.messageRouter.SetLiveBoard(pe.liveBoard)
		}

		pe.liveBoard.Start(jobs)
	}

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, pe.maxConcurrent)
	var wg sync.WaitGroup

	// Execute jobs
	for _, job := range jobs {
		wg.Add(1)
		go func(j *ExecutionJob) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Update live board - job starting
			if useLiveBoard {
				pe.liveBoard.UpdateJobStatus(j.ID, JobStatusRunning, "Starting...")
			}

			// Get module configuration
			module := modules[j.ModuleName]
			var instance *Instance
			if j.InstanceName != "" && module != nil {
				instance = module.Instances[j.InstanceName]
			}

			// Execute job with live board updates
			if err := pe.executeJobWithLiveUpdates(ctx, j, module, instance, useLiveBoard); err != nil {
				pe.logger.Error("Job execution failed", "job_id", j.ID, "error", err)
				if useLiveBoard {
					pe.liveBoard.UpdateJobError(j.ID, err.Error())
				}
			} else {
				if useLiveBoard {
					pe.liveBoard.UpdateJobStatus(j.ID, JobStatusCompleted, "Completed")
				}
			}
		}(job)
	}

	// Wait for all jobs to complete
	wg.Wait()

	// Transfer data from liveboard to ExecutionJob objects before stopping
	if useLiveBoard {
		liveboardStatuses := pe.liveBoard.GetJobStatuses()
		for _, job := range jobs {
			if liveStatus, exists := liveboardStatuses[job.ID]; exists {
				// Transfer PlanResult from liveboard
				if liveStatus.PlanResult != "" {
					job.PlanResult = liveStatus.PlanResult
				}
				// Transfer detailed error from liveboard
				if liveStatus.Error != "" {
					job.Error = liveStatus.Error
				}
				// Also ensure status, duration, and times are synced
				job.Status = liveStatus.Status
				if !liveStatus.StartTime.IsZero() {
					job.StartTime = &liveStatus.StartTime
				}
				if !liveStatus.EndTime.IsZero() {
					job.EndTime = &liveStatus.EndTime
				}
				job.Duration = liveStatus.Duration
			}
		}
	}

	// Close event bus FIRST (stops new events, closes subscriber goroutine)
	// This must happen before shutdown to avoid deadlock
	eventBus.Close()

	// Stop live board and message router
	if useLiveBoard {
		// Stop liveboard FIRST (this clears the screen/displays final box)
		pe.liveBoard.Stop()

		// THEN display filtered output (after liveboard has finished its display)
		if pe.messageRouter != nil {
			pe.messageRouter.StopLiveBoard()
		}
	}

	// Shutdown message router pipelines
	if pe.messageRouter != nil {
		pe.messageRouter.Shutdown()
	}

	// Calculate results
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	for _, job := range jobs {
		switch job.Status {
		case JobStatusCompleted:
			result.Completed++
		case JobStatusFailed:
			result.Failed++
			result.Errors = append(result.Errors, job.Error)
		case JobStatusSkipped:
			result.Skipped++
		}
	}

	result.Success = result.Failed == 0

	pe.logger.Info("Parallel execution completed",
		"total", result.TotalJobs,
		"completed", result.Completed,
		"failed", result.Failed,
		"skipped", result.Skipped,
		"duration", result.Duration,
	)

	return result
}

// executeJobWithLiveUpdates executes a job and updates the live board with progress
func (pe *ParallelExecutor) executeJobWithLiveUpdates(ctx context.Context, job *ExecutionJob, module *Module, instance *Instance, useLiveBoard bool) error {
	// Note: No need to set messageRouter per-job anymore!
	// The event bus routes events by jobID automatically (lock-free design)

	// Set event channel for this job
	var originalEventChan chan<- StatusUpdateEvent
	if useLiveBoard {
		pe.executor.mu.Lock()
		originalEventChan = pe.executor.eventChan
		pe.executor.eventChan = pe.liveBoard.GetEventChannel()
		pe.executor.mu.Unlock()

		// Restore event channel after execution
		defer func() {
			pe.executor.mu.Lock()
			pe.executor.eventChan = originalEventChan
			pe.executor.mu.Unlock()
		}()
	}

	// Execute the job (will send events to the channel and messages to pipeline)
	// Pass useLiveBoard as suppressOutput to suppress console output when LiveBoard is active
	err := pe.executor.ExecuteJobWithWorkspace(ctx, job, module, instance, useLiveBoard)

	// Send final status event
	if useLiveBoard {
		if err == nil {
			pe.liveBoard.GetEventChannel() <- StatusUpdateEvent{
				JobID:     job.ID,
				EventType: "status",
				Status:    JobStatusCompleted,
				Progress:  "Completed successfully",
			}
		} else {
			pe.liveBoard.GetEventChannel() <- StatusUpdateEvent{
				JobID:     job.ID,
				EventType: "error",
				Error:     err.Error(),
			}
		}
	}

	return err
}

// saveJobMetadata saves job execution metadata as an artifact
func (e *TerraformExecutor) saveJobMetadata(job *ExecutionJob, workspace *Workspace, exitCode int) {
	metadata := &JobMetadata{
		InstanceID:     job.ID, // Now uses instance ID (e.g., "seed-dev-gwc")
		ModuleName:     job.ModuleName,
		InstanceName:   job.InstanceName,
		InstanceEnv:    job.Environment, // Instance environment
		InstanceRegion: job.Region,      // Instance region
		Operation:      string(job.Operation),
		Status:         string(job.Status),
		CreatedAt:      workspace.CreatedAt,
		StartedAt:      job.StartTime,
		CompletedAt:    job.EndTime,
		ExitCode:       exitCode,
		ModulePath:     workspace.ModulePath,
		Artifacts:      make(map[string]string),
	}

	// Calculate duration
	if job.StartTime != nil && job.EndTime != nil {
		metadata.DurationSeconds = job.EndTime.Sub(*job.StartTime).Seconds()
	}

	// Add backend type if available
	if workspace.Backend != nil {
		metadata.BackendType = workspace.Backend.Type
	}

	// Add artifact paths
	logFile := filepath.Join(workspace.LogDir, fmt.Sprintf("%s.log", job.ID))
	if _, err := os.Stat(logFile); err == nil {
		metadata.Artifacts["log_file"] = logFile
	}

	errorLogFile := filepath.Join(workspace.LogDir, fmt.Sprintf("%s-error.log", job.ID))
	if _, err := os.Stat(errorLogFile); err == nil {
		metadata.Artifacts["error_log"] = errorLogFile
	}

	// Add plan artifacts if this was a plan operation
	if job.Operation == string(OpPlan) {
		planFile := filepath.Join(workspace.Path, "terraform.tfplan")
		if _, err := os.Stat(planFile); err == nil {
			metadata.Artifacts["plan_file"] = planFile
		}

		planJSONFile := filepath.Join(workspace.Path, "terraform.tfplan.json")
		if _, err := os.Stat(planJSONFile); err == nil {
			metadata.Artifacts["plan_json"] = planJSONFile
		}

		planTextFile := filepath.Join(workspace.Path, "terraform.tfplan.txt")
		if _, err := os.Stat(planTextFile); err == nil {
			metadata.Artifacts["plan_text"] = planTextFile
		}
	}

	// Add workspace size
	metadata.WorkspaceSize = e.calculateDirSize(workspace.Path)

	// Add environment info (sanitized - no sensitive data)
	metadata.Environment = map[string]interface{}{
		"terraform_data_dir": workspace.Environment["TF_DATA_DIR"],
		"workspace":          workspace.Environment["TF_WORKSPACE"],
	}

	// Detect Terraform version if available
	if tfVersion := e.detectTerraformVersion(workspace.Path); tfVersion != "" {
		metadata.TerraformVersion = tfVersion
	}

	// Write metadata file (renamed from job-metadata.json to instance-metadata.json)
	metadataFile := filepath.Join(workspace.Path, "instance-metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		e.logger.Warn("Failed to marshal job metadata", "job_id", job.ID, "error", err)
		return
	}

	if err := os.WriteFile(metadataFile, data, 0600); err != nil {
		e.logger.Warn("Failed to write job metadata", "job_id", job.ID, "error", err)
		return
	}

	e.logger.Debug("Job metadata saved", "job_id", job.ID, "file", metadataFile)
}

// calculateDirSize calculates the total size of a directory
func (e *TerraformExecutor) calculateDirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// detectTerraformVersion attempts to detect the Terraform version used
func (e *TerraformExecutor) detectTerraformVersion(workspacePath string) string {
	// Try to read from .terraform directory
	versionFile := filepath.Join(workspacePath, ".terraform", "terraform.tfstate")
	if data, err := os.ReadFile(versionFile); err == nil {
		var state map[string]interface{}
		if json.Unmarshal(data, &state) == nil {
			if version, ok := state["terraform_version"].(string); ok {
				return version
			}
		}
	}

	// Fallback: try to get version from terraform binary
	cmd := exec.Command("terraform", "version", "-json")
	if output, err := cmd.Output(); err == nil {
		var versionInfo map[string]interface{}
		if json.Unmarshal(output, &versionInfo) == nil {
			if version, ok := versionInfo["terraform_version"].(string); ok {
				return version
			}
		}
	}

	return ""
}

// formatTerraformCommand formats terraform command arguments for multi-line display
func formatTerraformCommand(args []string) string {
	if len(args) == 0 {
		return "terraform"
	}

	var formatted strings.Builder
	formatted.WriteString("terraform")

	// Group arguments by type for better readability
	for i, arg := range args {
		if i == 0 {
			// First arg is usually the command (init, plan, apply, etc.)
			formatted.WriteString(" ")
			formatted.WriteString(arg)
		} else if strings.HasPrefix(arg, "-") {
			// Flag arguments - put on new line with indentation
			formatted.WriteString(" \\\n    ")
			formatted.WriteString(arg)
		} else {
			// Value for previous flag - keep on same line
			formatted.WriteString(" ")
			formatted.WriteString(arg)
		}
	}

	return formatted.String()
}

// savePlanJSON generates and saves JSON representation of the plan
func (e *TerraformExecutor) savePlanJSON(ctx context.Context, job *ExecutionJob, workspace *Workspace, planFile, outputFile string) error {
	// Run terraform show -json on the plan file
	cmd := exec.CommandContext(ctx, "terraform", "show", "-json", planFile)
	cmd.Dir = workspace.ModulePath
	cmd.Env = e.buildEnvironment(workspace)

	output, err := cmd.Output()
	if err != nil {
		e.logger.Warn("Failed to generate plan JSON", "job_id", job.ID, "error", err)
		return err
	}

	// Save JSON output
	if err := os.WriteFile(outputFile, output, 0600); err != nil {
		e.logger.Warn("Failed to write plan JSON file", "job_id", job.ID, "error", err)
		return err
	}

	e.logger.Info("Saved plan JSON", "job_id", job.ID, "file", outputFile)
	return nil
}

// savePlanText generates and saves human-readable representation of the plan
func (e *TerraformExecutor) savePlanText(ctx context.Context, job *ExecutionJob, workspace *Workspace, planFile, outputFile string, suppressOutput bool) error {
	// Run terraform show on the plan file (human-readable)
	cmd := exec.CommandContext(ctx, "terraform", "show", planFile)
	cmd.Dir = workspace.ModulePath
	cmd.Env = e.buildEnvironment(workspace)

	output, err := cmd.Output()
	if err != nil {
		e.logger.Warn("Failed to generate plan text", "job_id", job.ID, "error", err)
		return err
	}

	// Strip ANSI codes for clean text output
	cleanOutput := StripANSI(string(output))

	// Save text output
	if err := os.WriteFile(outputFile, []byte(cleanOutput), 0600); err != nil {
		e.logger.Warn("Failed to write plan text file", "job_id", job.ID, "error", err)
		return err
	}

	if !suppressOutput {
		e.logger.Info("Saved plan text", "job_id", job.ID, "file", outputFile)
	}
	return nil
}

// formatPlanSummaryForLiveBoard is now defined in event_filter.go to avoid duplication

// streamOutputWithMonitoring streams output with hang detection
func (e *TerraformExecutor) streamOutputWithMonitoring(reader io.Reader, builder *strings.Builder, prefix, jobID string, logFile *os.File, suppressOutput bool, activity chan<- struct{}) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Signal activity
		select {
		case activity <- struct{}{}:
		default:
			// Channel full, skip (not critical)
		}

		// Write to output builder
		builder.WriteString(line)
		builder.WriteString("\n")

		// Parse line with output parser (for plan summary detection, etc.)
		e.outputParser.ParseLine(line)

		// Strip ANSI codes once for reuse
		cleanLine := StripANSI(line)

		// Capture plan summary DIRECTLY per-job (bypass shared parser race)
		if strings.Contains(cleanLine, "Plan:") && (strings.Contains(cleanLine, "to add") || strings.Contains(cleanLine, "to change") || strings.Contains(cleanLine, "to destroy")) {
			e.mu.Lock()
			e.jobPlanSummaries[jobID] = cleanLine
			e.mu.Unlock()
		} else if strings.Contains(cleanLine, "No changes") && strings.Contains(cleanLine, "infrastructure matches") {
			e.mu.Lock()
			e.jobPlanSummaries[jobID] = "No changes. Your infrastructure matches the configuration."
			e.mu.Unlock()
		}

		// Publish terraform output event (async, no locks!)
		if e.eventBus != nil && jobID != "" {
			e.eventBus.Publish(&ExecutorEvent{
				Type:      EventTerraformOutput,
				JobID:     jobID,
				Timestamp: time.Now().Unix(),
				Data: &TerraformOutputEvent{
					Component: "terraform",
					Line:      line,
				},
			})
		}

		// Write to log file if available (strip ANSI codes for clean logs)
		if logFile != nil {
			logLine := fmt.Sprintf("[%s] %s\n", prefix, cleanLine)
			_, _ = logFile.WriteString(logLine)
		}

		// Log the line (with error highlighting)
		if !suppressOutput {
			if prefix == "STDERR" && (strings.Contains(line, "Error:") || strings.Contains(line, "error:") || strings.Contains(line, "ERROR:")) {
				e.logger.Error("Terraform error output", "job_id", jobID, "line", cleanLine)
			} else {
				e.logger.Debug("Terraform output", "job_id", jobID, "source", prefix, "line", cleanLine)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		e.logger.Warn("Error reading terraform output", "job_id", jobID, "source", prefix, "error", err)
	}
}

// getLastOutputLines returns the last N lines from output string
func (e *TerraformExecutor) getLastOutputLines(output string, n int) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= n {
		return output
	}
	lastLines := lines[len(lines)-n:]
	return strings.Join(lastLines, "\n")
}

// looksLikeInteractivePrompt detects patterns indicating terraform is waiting for input
func (e *TerraformExecutor) looksLikeInteractivePrompt(output string) bool {
	// Common interactive prompt patterns
	interactivePatterns := []string{
		"Enter a value:",
		"Do you want to perform these actions?",
		"Do you want to copy existing state",
		"Terraform will perform the following actions:",
		"Do you want to migrate all workspaces",
		"Please enter",
		"(yes/no):",
		"[yes/no]:",
		"> ",
	}

	cleanOutput := StripANSI(strings.ToLower(output))

	for _, pattern := range interactivePatterns {
		if strings.Contains(cleanOutput, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check if last line ends with a prompt-like character
	lines := strings.Split(strings.TrimSpace(cleanOutput), "\n")
	if len(lines) > 0 {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if strings.HasSuffix(lastLine, ":") || strings.HasSuffix(lastLine, "?") || strings.HasSuffix(lastLine, ">") {
			// Likely a prompt
			return true
		}
	}

	return false
}
