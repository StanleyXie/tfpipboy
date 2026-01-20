// Package main provides the tfpipboy CLI tool for orchestrating Terraform modules
// with enhanced workspace context awareness and real-time authentication monitoring.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/StanleyXie/tfpipboy/pkg/orchestrator"
	"github.com/StanleyXie/tfpipboy/pkg/terraform"
	"github.com/StanleyXie/tfpipboy/pkg/version"
)

func main() {
	var (
		configPath      = flag.String("config", ".tfpipboy", "Path to configuration directory")
		operation       = flag.String("operation", "plan", "Terraform operation (plan, apply, destroy, validate)")
		targets         = flag.String("targets", "", "Comma-separated list of modules or groups to target")
		targetsAll      = flag.Bool("targets-all", false, "Target all instances in the configuration")
		pipeline        = flag.String("pipeline", "", "Pipeline name to execute")
		environment     = flag.String("env", "default", "Environment name")
		concurrentLimit = flag.Int("concurrent", 3, "Maximum number of concurrent executions")
		parallelAll     = flag.Bool("parallel-all", false, "Execute all targeted instances in parallel (ignoring dependencies)")
		timeout         = flag.Duration("timeout", 30*time.Minute, "Execution timeout")
		dryRun          = flag.Bool("dry-run", false, "Show what would be executed without running")
		autoConfirm     = flag.Bool("auto-confirm", false, "Skip confirmation prompt before execution")
		verbose         = flag.Bool("verbose", false, "Enable verbose logging (DEBUG level)")
		trace           = flag.Bool("trace", false, "Enable trace logging (TRACE level, most verbose)")
		outputMode      = flag.String("output-mode", "liveboard_only", "Console output mode: liveboard_only, liveboard_details, all_messages, quiet")
		listModules     = flag.Bool("list-modules", false, "List all available modules")
		listPipelines   = flag.Bool("list-pipelines", false, "List all available pipelines")
		listGroups      = flag.Bool("list-groups", false, "List all available groups")
		cleanup         = flag.Bool("cleanup", false, "Cleanup workspace artifacts")
		cleanupAll      = flag.Bool("all", false, "Remove all artifacts (use with --cleanup)")
		olderThan       = flag.String("older-than", "", "Remove artifacts older than duration (e.g., 7d, 24h)")
		shell           = flag.Bool("shell", false, "Open interactive shell in instance workspace")
		execCmd         = flag.String("exec", "", "Execute command in instance workspace")
		discover        = flag.String("discover", "", "Discover Terraform modules in specified path and generate configuration")
		discoverOutput  = flag.String("discover-output", "", "Output file for discovered configuration (default: stdout)")
		initWorkdir     = flag.Bool("init-workdir", false, "Initialize tfpipboy working directory with default structure")
		stateTrack      = flag.Bool("state-track", false, "Manually trigger state tracking for targets")
		showHelp        = flag.Bool("help", false, "Show help message")
		showVersion     = flag.Bool("version", false, "Show version")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("tfpipboy version %s\n", version.Version)
		return
	}

	if *showHelp {
		printHelp()
		return
	}

	// Handle workdir initialization
	if *initWorkdir {
		handleWorkdirInit(*configPath)
		return
	}

	// Handle module discovery
	if *discover != "" {
		handleModuleDiscovery(*discover, *discoverOutput)
		return
	}

	// Set up logger
	logLevel := orchestrator.LogLevelInfo
	if *verbose {
		logLevel = orchestrator.LogLevelDebug
	}
	var logger orchestrator.Logger = orchestrator.NewDefaultLogger(logLevel)

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to get working directory: %v\n", err)
		logger.Error("Failed to get working directory", "error", err)
		os.Exit(1)
	}

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(wd, logger)

	// Configure message router output modes
	if *trace {
		orch.SetTraceMode(true)
	} else if *verbose {
		orch.SetVerboseMode(true)
	}

	// Parse and set output mode
	var mode orchestrator.OutputMode
	switch *outputMode {
	case "liveboard_only":
		mode = orchestrator.OutputModeLiveBoardOnly
	case "liveboard_details":
		mode = orchestrator.OutputModeLiveBoardDetails
	case "all_messages":
		mode = orchestrator.OutputModeAllMessages
	case "quiet":
		mode = orchestrator.OutputModeQuiet
	default:
		logger.Warn("Unknown output mode, using default", "mode", *outputMode)
		mode = orchestrator.OutputModeLiveBoardOnly
	}
	orch.SetOutputMode(mode)

	// Suppress console logger output when LiveBoard has exclusive control
	// In liveboard_only and liveboard_details modes, only LiveBoard should write to console
	if mode == orchestrator.OutputModeLiveBoardOnly || mode == orchestrator.OutputModeLiveBoardDetails {
		nopLogger := orchestrator.NewNopLogger()
		logger = nopLogger
		orch.SetLogger(nopLogger)
	}

	// Handle cleanup command
	if *cleanup {
		handleCleanup(wd, logger, *cleanupAll, *olderThan)
		return
	}

	// Handle shell/exec command
	if *shell || *execCmd != "" {
		if *targets == "" {
			logger.Error("--targets must be specified with --shell or --exec")
			fmt.Println("\nUsage:")
			fmt.Println("  tfpipboy --config . --targets <instance> --shell")
			fmt.Println("  tfpipboy --config . --targets <instance> --exec \"terraform plan\"")
			os.Exit(1)
		}
		handleWorkspaceCommand(wd, *configPath, *targets, *shell, *execCmd, logger)
		return
	}

	// Configure orchestrator
	orch.SetDryRun(*dryRun)
	orch.SetTimeout(*timeout)
	orch.SetParallelLimit(*concurrentLimit)

	// Add event handler for progress reporting
	orch.AddEventHandler(func(event *orchestrator.Event) {
		switch event.Type {
		case orchestrator.EventJobStarted:
			logger.Info("Starting job", "job_id", event.JobID, "module", event.Module)
		case orchestrator.EventJobCompleted:
			logger.Info("Job completed", "job_id", event.JobID, "module", event.Module)
		case orchestrator.EventJobFailed:
			logger.Error("Job failed", "job_id", event.JobID, "module", event.Module, "error", event.Message)
		case orchestrator.EventStageStarted:
			logger.Info("Starting stage", "stage", event.Stage)
		case orchestrator.EventStageCompleted:
			logger.Info("Stage completed", "stage", event.Stage)
		}
	})

	// Resolve config path
	configDir := *configPath
	if !filepath.IsAbs(configDir) {
		configDir = filepath.Join(wd, configDir)
	}

	// Load configuration
	logger.Info("Loading configuration", "path", configDir)
	if err := orch.LoadConfig(configDir); err != nil {
		// Always print critical errors to stderr, even with NopLogger
		fmt.Fprintf(os.Stderr, "ERROR: Failed to load configuration from %s: %v\n", configDir, err)
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Handle list commands
	if *listModules {
		modules := orch.GetModuleList()
		fmt.Println("Available modules:")
		for _, module := range modules {
			fmt.Printf("  - %s\n", module)
		}
		return
	}

	if *listPipelines {
		pipelines := orch.GetPipelineList()
		fmt.Println("Available pipelines:")
		for _, p := range pipelines {
			fmt.Printf("  - %s\n", p)
		}
		return
	}

	if *listGroups {
		groups := orch.GetGroupList()
		fmt.Println("Available groups:")
		for _, group := range groups {
			fmt.Printf("  - %s\n", group)
		}
		return
	}

	// Validate configuration
	if err := orch.ValidateConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Configuration validation failed: %v\n", err)
		logger.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

	// Execute based on command
	var result *orchestrator.ExecutionResult

	if *pipeline != "" {
		// Execute pipeline
		logger.Info("Executing pipeline", "pipeline", *pipeline, "environment", *environment)
		result, err = orch.ExecutePipeline(*pipeline, *environment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Execution failed: %v\n", err)
			logger.Error("Execution failed", "error", err)
			os.Exit(1)
		}
	} else if *targets != "" || *targetsAll {
		// Determine target list
		var targetList []string
		if *targetsAll {
			// Get all instances from all modules
			config := orch.GetConfig()
			for _, module := range config.Modules {
				if len(module.Instances) > 0 {
					// Module has instances - add all instance IDs
					for instanceID := range module.Instances {
						targetList = append(targetList, instanceID)
					}
				} else {
					// Module has no instances - add module name itself
					targetList = append(targetList, module.Name)
				}
			}
			logger.Info("Targeting all instances", "count", len(targetList))
		} else {
			// Execute specific targets
			targetList = strings.Split(*targets, ",")
			for i, target := range targetList {
				targetList[i] = strings.TrimSpace(target)
			}
		}

		op := orchestrator.TerraformOperation(*operation)
		if *stateTrack {
			op = orchestrator.OpStateTrack
		}
		logger.Info("Executing operation", "operation", op, "targets", targetList, "environment", *environment)

		plan, err := orch.PlanExecution(op, targetList, *environment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to create execution plan: %v\n", err)
			logger.Error("Failed to create execution plan", "error", err)
			os.Exit(1)
		}

		// Apply parallel-all flag: force all jobs into a single parallel stage
		if *parallelAll {
			applyParallelAll(plan)
		}

		if *dryRun {
			printExecutionPlan(plan)
			return
		}

		// Collect backend types from plan
		backendTypes := collectBackendTypes(plan)

		// Check authentication status
		ctx := context.Background()
		authStatus := orchestrator.CheckRequiredAuth(ctx, backendTypes)

		// Show enhanced execution plan preview with auth status
		if !*autoConfirm {
			// Display plan preview
			preview := orchestrator.ExecutionPlanPreview(plan, authStatus)
			fmt.Print(preview)

			// Validate required authentication - block execution if any required auth is missing
			hasAuthFailures := false
			var failedProviders []string
			for _, status := range authStatus {
				if !status.Authenticated {
					hasAuthFailures = true
					failedProviders = append(failedProviders, status.Provider)
				}
			}

			if hasAuthFailures {
				fmt.Println()
				fmt.Println("================================================================================")
				fmt.Println("   ⚠️  AUTHENTICATION REQUIRED")
				fmt.Println("================================================================================")
				fmt.Printf("Cannot proceed: The following providers require authentication:\n")
				for _, provider := range failedProviders {
					status := authStatus[strings.ToLower(provider)]
					fmt.Printf("  • %s: %s\n", provider, status.Error)
				}
				fmt.Println()
				fmt.Println("Please authenticate and try again:")
				if contains(failedProviders, "Azure") {
					fmt.Println("  Azure:  az login")
				}
				if contains(failedProviders, "AWS") {
					fmt.Println("  AWS:    aws sso login --profile <profile>")
				}
				if contains(failedProviders, "GCP") {
					fmt.Println("  GCP:    gcloud auth application-default login")
				}
				if contains(failedProviders, "GitHub") {
					fmt.Println("  GitHub: gh auth login")
				}
				fmt.Println("================================================================================")
				os.Exit(1)
			}

			// Prompt for confirmation
			fmt.Printf("\nDo you want to proceed with this execution plan? (yes/no): ")
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
				os.Exit(1)
			}
			response = strings.ToLower(strings.TrimSpace(response))

			if response != "yes" && response != "y" {
				fmt.Println("Execution cancelled.")
				os.Exit(0)
			}
			fmt.Println()
		} else {
			// Even with --auto-confirm, validate required authentication
			hasAuthFailures := false
			var failedProviders []string
			for _, status := range authStatus {
				if !status.Authenticated {
					hasAuthFailures = true
					failedProviders = append(failedProviders, status.Provider)
				}
			}

			if hasAuthFailures {
				logger.Error("Authentication required", "failed_providers", failedProviders)
				fmt.Println()
				fmt.Println("ERROR: Required authentication is missing or expired.")
				fmt.Printf("Failed providers: %s\n", strings.Join(failedProviders, ", "))
				fmt.Println("Please authenticate before using --auto-confirm")
				os.Exit(1)
			}
		}

		result, err = orch.ExecutePlan(plan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Execution failed: %v\n", err)
			logger.Error("Execution failed", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Error("Either --targets, --targets-all, or --pipeline must be specified")
		printUsage()
		os.Exit(1)
	}

	// Stop liveboard and display filtered output if in liveboard_details mode
	messageRouter := orch.GetMessageRouter()
	if messageRouter != nil {
		messageRouter.StopLiveBoard()
	}

	// Print enhanced execution summary with instance details
	summary := orchestrator.ExecutionSummary(
		result,
		!*dryRun, // Show artifacts info only if not dry run
	)
	fmt.Print(summary)

	// Exit with appropriate code
	if !result.Success {
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`tfpipboy - Terraform Module Orchestrator

USAGE:
    tfpipboy [OPTIONS]

OPTIONS:
    --config PATH          Path to configuration directory (default: .tfpipboy)
    --operation OP         Terraform operation: plan, apply, destroy, validate (default: plan)
    --targets MODULES      Comma-separated list of modules or groups to target
    --targets-all          Target all instances in the configuration
    --pipeline NAME        Pipeline name to execute
    --env NAME             Environment name (default: default)
    --concurrent N         Maximum number of concurrent executions (default: 3)
    --parallel-all         Execute all instances in parallel, ignoring dependencies
    --auto-confirm         Skip confirmation prompt before execution
    --timeout DURATION     Execution timeout (default: 30m)
    --dry-run              Show what would be executed without running
    --verbose              Enable verbose logging (DEBUG level)
    --trace                Enable trace logging (TRACE level, most verbose)
    --output-mode MODE     Console output mode (default: liveboard_only)
                           - liveboard_only: LiveBoard exclusive, no instance details
                           - liveboard_details: LiveBoard + instance details after
                           - all_messages: Real-time streaming, no LiveBoard
                           - quiet: Errors only, all messages to log files
    --list-modules         List all available modules
    --list-pipelines       List all available pipelines
    --list-groups          List all available groups
    --cleanup              Cleanup workspace artifacts (interactive)
    --all                  Remove all artifacts (use with --cleanup)
    --older-than DURATION  Remove artifacts older than duration (e.g., 7d, 24h)
    --shell                Open interactive shell in instance workspace (requires --targets)
    --exec COMMAND         Execute command in instance workspace (requires --targets)
    --discover PATH        Discover Terraform modules in specified path
    --discover-output FILE Write discovered configuration to file (default: stdout)
    --help                 Show this help message
    --version              Show version

EXAMPLES:
    # List available modules
    tfpipboy --list-modules

    # Plan all instances in parallel
    tfpipboy --config . --targets-all --operation plan --parallel-all

    # Plan all modules in bootstrap group
    tfpipboy --targets bootstrap --operation plan

    # Apply specific modules
    tfpipboy --targets seed,core --operation apply --env exp

    # Execute a pipeline
    tfpipboy --pipeline deploy-platform-complete --env exp

    # Dry run with verbose output for all instances
    tfpipboy --targets-all --operation apply --dry-run --verbose

    # Plan with instance details displayed after LiveBoard completes
    tfpipboy --targets-all --operation plan --output-mode liveboard_details

    # Real-time streaming output (no LiveBoard, for debugging)
    tfpipboy --targets seed,core --operation plan --output-mode all_messages

    # Quiet mode - only errors to console, full logs in files
    tfpipboy --targets-all --operation plan --output-mode quiet

    # Trace level logging for maximum detail
    tfpipboy --targets seed --operation plan --trace

    # Open interactive shell in workspace for manual debugging
    tfpipboy --config .tfpipboy --targets core --shell

    # Execute single command in workspace
    tfpipboy --config .tfpipboy --targets core --exec "terraform plan"

    # Discover Terraform modules in a directory
    tfpipboy --discover ./terraform

    # Discover modules and save configuration to file
    tfpipboy --discover ./terraform --discover-output .tfpipboy/modules.yaml

    # Run manual Terraform commands
    tfpipboy --targets baseline.connectivity --exec "terraform state list"

ARTIFACT CLEANUP:
    # Interactive cleanup (prompts for confirmation)
    tfpipboy --cleanup

    # Remove all artifacts without prompt
    tfpipboy --cleanup --all

    # Remove artifacts older than 7 days
    tfpipboy --cleanup --older-than 168h

    # Remove artifacts older than 30 days (720 hours)
    tfpipboy --cleanup --older-than 720h

NOTE:
    Workspaces and logs are preserved as artifacts after each execution
    (similar to GitHub Actions). Use --cleanup to manage old artifacts.

CONFIGURATION:
    tfpipboy looks for configuration files in the --config directory:
    - tfproject.yaml    - Main project configuration
    - modules.yaml      - Module definitions and dependencies
    - pipelines.yaml    - Pipeline definitions

    See the examples/ directory for sample configurations.
`)
}

func printUsage() {
	fmt.Printf(`Usage: tfpipboy [OPTIONS]

Use --help for detailed help information.
`)
}

func printExecutionPlan(plan *orchestrator.ExecutionPlan) {
	// Use the enhanced preview function with auth check
	ctx := context.Background()
	backendTypes := collectBackendTypes(plan)
	authStatus := orchestrator.CheckRequiredAuth(ctx, backendTypes)

	preview := orchestrator.ExecutionPlanPreview(plan, authStatus)
	fmt.Print(preview)
}

func printExecutionResult(result *orchestrator.ExecutionResult) {
	fmt.Printf("\nExecution Result:\n")
	fmt.Printf("  Success: %t\n", result.Success)
	fmt.Printf("  Total Jobs: %d\n", result.TotalJobs)
	fmt.Printf("  Completed: %d\n", result.Completed)
	fmt.Printf("  Failed: %d\n", result.Failed)
	fmt.Printf("  Skipped: %d\n", result.Skipped)
	fmt.Printf("  Duration: %s\n", result.Duration)
	fmt.Printf("  Started: %s\n", result.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Ended: %s\n\n", result.EndTime.Format("2006-01-02 15:04:05"))

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for i, err := range result.Errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
		fmt.Println()
	}

	// Print job details if there were failures
	if result.Failed > 0 {
		fmt.Printf("Failed Jobs:\n")
		for _, job := range result.Jobs {
			if job.Status == orchestrator.JobStatusFailed {
				moduleRef := job.ModuleName
				if job.InstanceName != "" {
					moduleRef = fmt.Sprintf("%s.%s", job.ModuleName, job.InstanceName)
				}

				fmt.Printf("  - %s (%s): %s\n", moduleRef, job.Operation, job.Error)
				if job.Duration > 0 {
					fmt.Printf("    Duration: %s\n", job.Duration)
				}
			}
		}
		fmt.Println()
	}

	// Print outputs if available
	if len(result.Outputs) > 0 {
		fmt.Printf("Outputs:\n")
		for key, value := range result.Outputs {
			fmt.Printf("  %s: %v\n", key, value)
		}
		fmt.Println()
	}
}

// handleCleanup handles workspace and log cleanup with retention policies
func handleCleanup(baseDir string, logger orchestrator.Logger, cleanupAll bool, olderThan string) {
	tfpipboyDir := filepath.Join(baseDir, ".tfpipboy")
	workspacesDir := filepath.Join(tfpipboyDir, "workspaces")
	logsDir := filepath.Join(tfpipboyDir, "logs")

	// Parse retention duration if specified
	var retentionDuration time.Duration
	if olderThan != "" {
		var err error
		retentionDuration, err = time.ParseDuration(olderThan)
		if err != nil {
			logger.Error("Invalid duration format", "older-than", olderThan, "error", err)
			fmt.Printf("Error: Invalid duration format '%s'. Use formats like: 24h, 7d, 168h\n", olderThan)
			os.Exit(1)
		}
	}

	// Collect artifacts to clean
	type Artifact struct {
		JobID      string
		Path       string
		Type       string
		ModifiedAt time.Time
		SizeBytes  int64
	}

	var artifacts []Artifact
	cutoffTime := time.Now().Add(-retentionDuration)

	// Scan workspaces
	if entries, err := os.ReadDir(workspacesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				workspacePath := filepath.Join(workspacesDir, entry.Name())
				info, err := entry.Info()
				if err != nil {
					continue
				}

				// Check if should be included based on retention policy
				if cleanupAll || olderThan == "" || info.ModTime().Before(cutoffTime) {
					size := getDirSize(workspacePath)
					artifacts = append(artifacts, Artifact{
						JobID:      entry.Name(),
						Path:       workspacePath,
						Type:       "workspace",
						ModifiedAt: info.ModTime(),
						SizeBytes:  size,
					})
				}
			}
		}
	}

	// Scan logs
	if entries, err := os.ReadDir(logsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				logPath := filepath.Join(logsDir, entry.Name())
				info, err := entry.Info()
				if err != nil {
					continue
				}

				if cleanupAll || olderThan == "" || info.ModTime().Before(cutoffTime) {
					size := getDirSize(logPath)
					artifacts = append(artifacts, Artifact{
						JobID:      entry.Name(),
						Path:       logPath,
						Type:       "log",
						ModifiedAt: info.ModTime(),
						SizeBytes:  size,
					})
				}
			}
		}
	}

	if len(artifacts) == 0 {
		fmt.Printf("No artifacts to cleanup.\n")
		return
	}

	// Display artifacts to be removed
	fmt.Printf("\n")
	fmt.Printf("================================================================================\n")
	fmt.Printf("   ARTIFACTS TO BE REMOVED\n")
	fmt.Printf("================================================================================\n")

	totalSize := int64(0)
	jobsMap := make(map[string][]Artifact)

	for _, artifact := range artifacts {
		jobsMap[artifact.JobID] = append(jobsMap[artifact.JobID], artifact)
		totalSize += artifact.SizeBytes
	}

	for jobID, jobArtifacts := range jobsMap {
		jobSize := int64(0)
		for _, a := range jobArtifacts {
			jobSize += a.SizeBytes
		}
		fmt.Printf("  %s (%s, modified: %s)\n",
			jobID,
			formatSize(jobSize),
			jobArtifacts[0].ModifiedAt.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\n")
	fmt.Printf("Total: %d job(s), %s\n", len(jobsMap), formatSize(totalSize))
	fmt.Printf("================================================================================\n")

	// Prompt for confirmation unless --all is specified
	if !cleanupAll && olderThan == "" {
		fmt.Printf("\nProceed with cleanup? (y/N): ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			return
		}
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Printf("Cleanup cancelled.\n")
			return
		}
	}

	// Perform cleanup
	logger.Info("Cleaning up artifacts...")
	removedCount := 0
	var errors []string

	for _, artifact := range artifacts {
		if err := os.RemoveAll(artifact.Path); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", artifact.JobID, err))
		} else {
			removedCount++
		}
	}

	fmt.Printf("\n")
	if len(errors) > 0 {
		logger.Warn("Cleanup completed with errors", "removed", removedCount, "errors", len(errors))
		fmt.Printf("Removed %d artifacts with %d errors:\n", removedCount, len(errors))
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
	} else {
		logger.Info("Cleanup completed successfully", "removed", removedCount)
		fmt.Printf("Successfully removed %d job artifact(s)\n", removedCount)
	}
}

// getDirSize calculates the total size of a directory
func getDirSize(path string) int64 {
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

// formatSize formats bytes into human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// collectBackendTypes collects unique backend types from execution plan
func collectBackendTypes(plan *orchestrator.ExecutionPlan) []string {
	backendTypeMap := make(map[string]bool)
	var backendTypes []string

	for _, job := range plan.Modules {
		if job.Backend != nil && job.Backend.Type != "" {
			if !backendTypeMap[job.Backend.Type] {
				backendTypeMap[job.Backend.Type] = true
				backendTypes = append(backendTypes, job.Backend.Type)
			}
		}
	}

	return backendTypes
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// applyParallelAll modifies the execution plan to run all jobs in parallel
// by placing them all in stage 0 and marking them as parallelizable
func applyParallelAll(plan *orchestrator.ExecutionPlan) {
	// Move all jobs to stage 0 and mark as parallel
	for _, job := range plan.Modules {
		job.Stage = 0
		job.CanParallel = true
		// Clear dependencies since we're running everything in parallel
		job.DependsOn = []string{}
	}
}

// handleWorkspaceCommand handles --shell and --exec commands for manual workspace access
func handleWorkspaceCommand(baseDir, configPath, target string, shell bool, execCmd string, logger orchestrator.Logger) {
	// Load configuration via orchestrator
	orch := orchestrator.NewOrchestrator(baseDir, logger)

	// Resolve config path
	configDir := configPath
	if !filepath.IsAbs(configDir) {
		configDir = filepath.Join(baseDir, configDir)
	}

	if err := orch.LoadConfig(configDir); err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	config := orch.GetConfig()

	// Find the workspace for the target instance
	workspacePath, env, err := findWorkspacePath(baseDir, configDir, config, target)
	if err != nil {
		logger.Error("Failed to find workspace", "error", err, "target", target)
		fmt.Printf("\nTip: Run 'tfpipboy --config %s --targets %s --operation plan' first to create the workspace\n", configPath, target)
		os.Exit(1)
	}

	logger.Info("Found workspace", "path", workspacePath, "target", target)

	// Get the module path (follow the symlink)
	modulePath, err := filepath.EvalSymlinks(filepath.Join(workspacePath, "module"))
	if err != nil {
		logger.Error("Failed to resolve module path", "error", err, "workspace", workspacePath)
		fmt.Printf("\nError: Workspace exists but module symlink is broken: %s\n", workspacePath)
		os.Exit(1)
	}

	// Prepare environment variables
	envVars := prepareWorkspaceEnvironment(workspacePath, modulePath, env)

	if shell {
		// Open interactive shell
		openInteractiveShell(workspacePath, modulePath, envVars, logger)
	} else {
		// Execute single command
		executeSingleCommand(workspacePath, modulePath, execCmd, envVars, logger)
	}
}

// findWorkspacePath finds the workspace directory for a given target
func findWorkspacePath(baseDir, configDir string, config *orchestrator.Config, target string) (string, map[string]string, error) {
	// Parse the target to find module and instance
	var moduleName, instanceName string

	// Try to find target as instance name first
	for mName, module := range config.Modules {
		for iName := range module.Instances {
			if iName == target {
				moduleName = mName
				instanceName = iName
				break
			}
		}
		if instanceName != "" {
			break
		}
	}

	// If not found, try module.instance format
	if instanceName == "" {
		parts := strings.Split(target, ".")
		if len(parts) == 2 {
			moduleName = parts[0]
			instanceName = parts[1]
		} else {
			moduleName = target
		}
	}

	// Check if module exists
	module, exists := config.Modules[moduleName]
	if !exists {
		return "", nil, fmt.Errorf("module %s not found", moduleName)
	}

	// Get instance or use module-level config
	var instance *orchestrator.Instance
	var workspaceID string
	if instanceName != "" {
		instance = module.Instances[instanceName]
		if instance == nil {
			return "", nil, fmt.Errorf("instance %s not found in module %s", instanceName, moduleName)
		}
		workspaceID = instanceName
	} else {
		workspaceID = moduleName
	}

	// Build workspace path
	// configDir is the config directory, workspace is under .tfpipboy subdirectory
	workspacePath := filepath.Join(configDir, ".tfpipboy", "workspaces", workspaceID)

	// Check if workspace exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("workspace does not exist: %s", workspacePath)
	}

	// Collect environment variables from instance
	env := make(map[string]string)
	if instance != nil && instance.Variables != nil {
		for k, v := range instance.Variables {
			if str, ok := v.(string); ok {
				env[k] = str
			}
		}
	}

	return workspacePath, env, nil
}

// prepareWorkspaceEnvironment prepares environment variables for the workspace
func prepareWorkspaceEnvironment(workspacePath, modulePath string, instanceEnv map[string]string) []string {
	// Start with current environment
	env := os.Environ()

	// Add workspace-specific variables
	env = append(env, fmt.Sprintf("TFPIPBOY_WORKSPACE=%s", workspacePath))
	env = append(env, fmt.Sprintf("TFPIPBOY_MODULE=%s", modulePath))

	// Add paths to workspace files for easy Terraform command reference
	backendFile := filepath.Join(workspacePath, "backend.hcl")
	tfvarsFile := filepath.Join(workspacePath, "terraform.tfvars")
	env = append(env, fmt.Sprintf("TFPIPBOY_BACKEND_CONFIG=%s", backendFile))
	env = append(env, fmt.Sprintf("TFPIPBOY_TFVARS=%s", tfvarsFile))

	// CRITICAL: Set TF_DATA_DIR to workspace .terraform directory
	// This tells Terraform to store its working data in the workspace, not the module directory
	// This allows the same module to be used by multiple instances with isolated state
	tfDataDir := filepath.Join(workspacePath, ".terraform")
	env = append(env, fmt.Sprintf("TF_DATA_DIR=%s", tfDataDir))

	// Add instance variables
	for k, v := range instanceEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Add helpful PS1 prompt for shell
	workspaceName := filepath.Base(workspacePath)
	env = append(env, fmt.Sprintf("PS1=(tfpipboy:%s) $ ", workspaceName))

	return env
}

// openInteractiveShell opens an interactive shell in the workspace
func openInteractiveShell(workspacePath, modulePath string, env []string, logger orchestrator.Logger) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	logger.Info("Opening interactive shell", "workspace", workspacePath, "module", modulePath, "shell", shell)

	fmt.Println("================================================================================")
	fmt.Printf("  TFPIPBOY WORKSPACE SHELL\n")
	fmt.Println("================================================================================")
	fmt.Printf("Workspace:   %s\n", workspacePath)
	fmt.Printf("Module:      %s\n", modulePath)
	fmt.Printf("Shell:       %s\n\n", shell)
	fmt.Println("Working directory is set to the module path (where .tf files are).")
	fmt.Println("TF_DATA_DIR is set to workspace/.terraform for isolated state.")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  - $TF_DATA_DIR             (terraform working directory - auto-used)")
	fmt.Println("  - $TFPIPBOY_BACKEND_CONFIG (backend configuration file)")
	fmt.Println("  - $TFPIPBOY_TFVARS         (tfvars file)")
	fmt.Println("  - $TFPIPBOY_WORKSPACE      (workspace directory path)")
	fmt.Println("")
	fmt.Println("Example Terraform commands:")
	fmt.Println("  terraform init -backend-config=$TFPIPBOY_BACKEND_CONFIG")
	fmt.Println("  terraform plan -var-file=$TFPIPBOY_TFVARS")
	fmt.Println("  terraform apply -var-file=$TFPIPBOY_TFVARS")
	fmt.Println("")
	fmt.Println("Type 'exit' to return to the main shell.")
	fmt.Println("================================================================================")
	fmt.Println()

	cmd := exec.Command(shell)
	cmd.Dir = modulePath // Use module path as working directory
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logger.Error("Shell exited with error", "error", err)
		os.Exit(1)
	}
}

// executeSingleCommand executes a single command in the workspace
func executeSingleCommand(workspacePath, modulePath, command string, env []string, logger orchestrator.Logger) {
	logger.Info("Executing command", "workspace", workspacePath, "module", modulePath, "command", command)

	fmt.Println("================================================================================")
	fmt.Printf("  EXECUTING COMMAND IN WORKSPACE\n")
	fmt.Println("================================================================================")
	fmt.Printf("Workspace:   %s\n", workspacePath)
	fmt.Printf("Module:      %s\n", modulePath)
	fmt.Printf("Command:     %s\n", command)
	fmt.Println("================================================================================")
	fmt.Println()

	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Dir = modulePath // Use module path as working directory
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logger.Error("Command failed", "error", err)
		os.Exit(1)
	}
}

// handleWorkdirInit initializes the tfpipboy working directory structure
func handleWorkdirInit(configPath string) {
	fmt.Println("================================================================================")
	fmt.Println("  TFPIPBOY WORKDIR INITIALIZATION")
	fmt.Println("================================================================================")
	fmt.Printf("Initializing working directory: %s\n", configPath)
	fmt.Println("================================================================================")
	fmt.Println()

	// Create main config directory
	if err := os.MkdirAll(configPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create directory %s: %v\n", configPath, err)
		os.Exit(1)
	}
	fmt.Printf("✓ Created directory: %s\n", configPath)

	// Create example config.yaml if it doesn't exist
	configFile := filepath.Join(configPath, "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		exampleConfig := `# tfpipboy Configuration
# Documentation: https://github.com/StanleyXie/tfpipboy

# Modules define Terraform root modules to manage
modules: []
  # Example module configuration:
  # - name: vpc
  #   path: terraform/modules/vpc
  #   enabled: true
  #   tags:
  #     - networking
  #   instances:
  #     dev:
  #       workspace: dev
  #       var_files:
  #         - environments/dev.tfvars

# Pipelines define ordered execution sequences
pipelines: {}
  # Example pipeline:
  # deploy-infra:
  #   environments:
  #     dev:
  #       stages:
  #         - name: networking
  #           modules: [vpc, subnets]
  #         - name: compute
  #           modules: [ec2]
  #           depends_on: [networking]

# Groups allow targeting multiple modules together
groups: {}
  # Example groups:
  # networking: [vpc, subnets, security-groups]
  # compute: [ec2, asg]
`
		if err := os.WriteFile(configFile, []byte(exampleConfig), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to create %s: %v\n", configFile, err)
			os.Exit(1)
		}
		fmt.Printf("✓ Created example configuration: %s\n", configFile)
	} else {
		fmt.Printf("⊗ Configuration already exists: %s\n", configFile)
	}

	// Create .gitignore if it doesn't exist
	gitignoreFile := filepath.Join(configPath, ".gitignore")
	if _, err := os.Stat(gitignoreFile); os.IsNotExist(err) {
		gitignoreContent := `# tfpipboy artifacts
*.log
*.tfplan
*.tfstate
*.tfstate.backup
workspaces/
.terraform/
`
		if err := os.WriteFile(gitignoreFile, []byte(gitignoreContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to create %s: %v\n", gitignoreFile, err)
			os.Exit(1)
		}
		fmt.Printf("✓ Created .gitignore: %s\n", gitignoreFile)
	} else {
		fmt.Printf("⊗ .gitignore already exists: %s\n", gitignoreFile)
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("  INITIALIZATION COMPLETE")
	fmt.Println("================================================================================")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit configuration: %s\n", configFile)
	fmt.Println("  2. Discover modules: tfpipboy --discover <path> --discover-output .tfpipboy/discovered-modules.yaml")
	fmt.Println("  3. Run tfpipboy: tfpipboy --targets-all --operation plan")
	fmt.Println()
}

// handleModuleDiscovery discovers Terraform modules and generates configuration
func handleModuleDiscovery(searchPath, outputFile string) {
	fmt.Println("================================================================================")
	fmt.Println("  TERRAFORM MODULE DISCOVERY")
	fmt.Println("================================================================================")
	fmt.Printf("Scanning path: %s\n", searchPath)
	fmt.Println("================================================================================")
	fmt.Println()

	// Discover modules
	result, err := terraform.DiscoverModules(searchPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to discover modules: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println("Discovery Summary:")
	fmt.Printf("  Total modules found:     %d\n", result.Summary.TotalModules)
	fmt.Printf("  Root modules:            %d\n", result.Summary.RootModules)
	fmt.Printf("  Source modules:          %d\n", result.Summary.SourceModules)
	fmt.Printf("  Tfvars files found:      %d\n", result.Summary.TfvarsFilesFound)
	fmt.Println()

	// Print discovered modules
	if len(result.Modules) > 0 {
		fmt.Println("Discovered Modules:")
		for _, module := range result.Modules {
			fmt.Printf("  [%s] %s\n", module.Type, module.RelativePath)
			if len(module.Variables) > 0 {
				fmt.Printf("      Variables: %d", len(module.Variables))
				requiredCount := 0
				for _, v := range module.Variables {
					if v.Required {
						requiredCount++
					}
				}
				if requiredCount > 0 {
					fmt.Printf(" (%d required)", requiredCount)
				}
				fmt.Println()
			}
			if len(module.Outputs) > 0 {
				fmt.Printf("      Outputs: %d\n", len(module.Outputs))
			}
			if len(module.Dependencies) > 0 {
				fmt.Printf("      Module dependencies: %d\n", len(module.Dependencies))
			}
		}
		fmt.Println()
	}

	// Print tfvars files
	if len(result.TfvarsFiles) > 0 {
		fmt.Println("Discovered Tfvars Files:")
		for _, tfvars := range result.TfvarsFiles {
			fmt.Printf("  %s\n", tfvars.RelativePath)
		}
		fmt.Println()
	}

	// Generate YAML configuration
	yamlConfig, err := result.GenerateYAMLConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to generate YAML configuration: %v\n", err)
		os.Exit(1)
	}

	// Output configuration
	if outputFile != "" {
		// Create directory if it doesn't exist
		dir := filepath.Dir(outputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}

		if err := os.WriteFile(outputFile, []byte(yamlConfig), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to write configuration to %s: %v\n", outputFile, err)
			os.Exit(1)
		}
		fmt.Printf("Configuration written to: %s\n", outputFile)
	} else {
		fmt.Println("================================================================================")
		fmt.Println("  GENERATED CONFIGURATION")
		fmt.Println("================================================================================")
		fmt.Println()
		fmt.Println(yamlConfig)
	}

	fmt.Println("================================================================================")
	fmt.Println("NOTE: The generated configuration template requires you to define instances")
	fmt.Println("for each module before execution. Edit the configuration file and add")
	fmt.Println("instance definitions under the 'instances' section of each module.")
	fmt.Println("================================================================================")
}
