package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultOrchestrator implements the Orchestrator interface
type DefaultOrchestrator struct {
	config           *Config
	parser           *ConfigParser
	graphBuilder     *DependencyGraphBuilder
	workspaceManager WorkspaceManager
	executor         *TerraformExecutor
	parallelExecutor *ParallelExecutor
	logger           Logger
	messageRouter    *MessageRouter
	eventHandlers    []EventHandler
	basePath         string
	mu               sync.RWMutex
	currentExecution *ExecutionResult
	cancelled        bool
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(basePath string, logger Logger) *DefaultOrchestrator {
	parser := NewConfigParser(basePath)
	workspaceManager := NewDefaultWorkspaceManager(basePath)
	executor := NewTerraformExecutor(workspaceManager, logger)
	parallelExecutor := NewParallelExecutor(executor, 3, logger)

	// Initialize message router with default configuration
	routerConfig := &MessageRouterConfig{
		BaseLogDir:       filepath.Join(basePath, ".tfpipboy", "logs"),
		ConsoleMode:      OutputModeLiveBoardOnly, // Default mode
		VerboseMode:      false,
		TraceMode:        false,
		StripANSIInLogs:  false, // Keep ANSI codes in logs
		StripANSIConsole: false, // Keep ANSI codes in console
		BufferSize:       1000,
	}
	messageRouter := NewMessageRouter(routerConfig)

	// Initialize orchestrator pipeline
	if err := messageRouter.InitializeOrchestrator(); err != nil {
		logger.Error("Failed to initialize message router", "error", err)
	}

	// Create event bus for async communication between executor and message router
	eventBus := NewExecutorEventBus(10000) // Large buffer to prevent blocking

	// Set event bus on executor
	executor.SetEventBus(eventBus)

	// Start event subscriber in message router
	messageRouter.StartEventSubscriber(eventBus)

	// Pass message router to parallel executor
	parallelExecutor.SetMessageRouter(messageRouter)

	orchestrator := &DefaultOrchestrator{
		parser:           parser,
		workspaceManager: workspaceManager,
		executor:         executor,
		parallelExecutor: parallelExecutor,
		logger:           logger,
		messageRouter:    messageRouter,
		basePath:         basePath,
		eventHandlers:    []EventHandler{},
	}

	return orchestrator
}

// LoadConfig loads configuration from the specified path
func (o *DefaultOrchestrator) LoadConfig(configPath string) error {
	o.logger.Info("Loading configuration", "path", configPath)

	config, err := o.parser.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := o.parser.ValidateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	o.config = config
	o.graphBuilder = NewDependencyGraphBuilder(config)

	o.logger.Info("Configuration loaded successfully",
		"modules", len(config.Modules),
		"groups", len(config.Groups),
		"pipelines", len(config.Pipelines),
	)

	return nil
}

// GetConfig returns the current configuration
func (o *DefaultOrchestrator) GetConfig() *Config {
	return o.config
}

// GetMessageRouter returns the message router
func (o *DefaultOrchestrator) GetMessageRouter() *MessageRouter {
	return o.messageRouter
}

// SetOutputMode configures the console output mode
func (o *DefaultOrchestrator) SetOutputMode(mode OutputMode) {
	if o.messageRouter != nil {
		o.messageRouter.config.ConsoleMode = mode
	}
}

// SetVerboseMode enables or disables verbose (DEBUG level) logging
func (o *DefaultOrchestrator) SetVerboseMode(enabled bool) {
	if o.messageRouter != nil {
		o.messageRouter.config.VerboseMode = enabled
	}
}

// SetTraceMode enables or disables trace (TRACE level) logging
func (o *DefaultOrchestrator) SetTraceMode(enabled bool) {
	if o.messageRouter != nil {
		o.messageRouter.config.TraceMode = enabled
	}
}

// SetLogger replaces the orchestrator's logger
func (o *DefaultOrchestrator) SetLogger(logger Logger) {
	o.logger = logger
	// Also update executor's logger
	if o.executor != nil {
		o.executor.logger = logger
	}
	if o.parallelExecutor != nil && o.parallelExecutor.executor != nil {
		o.parallelExecutor.executor.logger = logger
	}
}

// ValidateConfig validates the current configuration
func (o *DefaultOrchestrator) ValidateConfig() error {
	if o.config == nil {
		return fmt.Errorf("no configuration loaded")
	}
	return o.parser.ValidateConfig(o.config)
}

// PlanExecution creates an execution plan for the specified operation and targets
func (o *DefaultOrchestrator) PlanExecution(operation TerraformOperation, targets []string, env string) (*ExecutionPlan, error) {
	if o.config == nil {
		return nil, fmt.Errorf("no configuration loaded")
	}

	o.logger.Info("Planning execution",
		"operation", operation,
		"targets", targets,
		"environment", env,
	)

	// Expand targets (resolve groups)
	expandedTargets, err := o.expandTargets(targets)
	if err != nil {
		return nil, fmt.Errorf("failed to expand targets: %w", err)
	}

	// Build dependency graph
	graph, err := o.BuildDependencyGraph(expandedTargets)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Get execution order
	stages := o.graphBuilder.GetExecutionOrder(graph)

	// Create execution jobs
	var jobs []*ExecutionJob

	for stageNum, moduleRefs := range stages {
		for _, moduleRef := range moduleRefs {
			moduleName, instanceName := o.parseModuleReference(moduleRef)

			// Use instance name as job ID (e.g., "seed-dev-gwc") instead of "job-1"
			// If no instance name, use module name as fallback
			instanceID := instanceName
			if instanceID == "" {
				instanceID = moduleName
			}

			job := &ExecutionJob{
				ID:           instanceID,
				ModuleName:   moduleName,
				InstanceName: instanceID, // Set for consistency
				Operation:    string(operation),
				Stage:        stageNum,
				Status:       JobStatusPending,
				CanParallel:  len(moduleRefs) > 1, // Can run in parallel if multiple modules in same stage
			}

			// Set module path and instance-specific fields
			if module, exists := o.config.Modules[moduleName]; exists {
				job.Path = module.Path
				job.Category = module.Category // Set category for grouping

				// Set dependencies and configuration from instance (or module if no instance)
				if instanceName != "" {
					if instance, exists := module.Instances[instanceName]; exists {
						// Combine module-level and instance-level dependencies
						// Instance dependencies should ADD to module dependencies, not replace them
						job.DependsOn = make([]string, 0)
						job.DependsOn = append(job.DependsOn, module.DependsOn...)
						job.DependsOn = append(job.DependsOn, instance.DependsOn...)

						job.Variables = instance.Variables

						// Check both Backend and BackendCfg fields
						if instance.Backend != nil {
							job.Backend = instance.Backend
						} else if instance.BackendCfg != nil {
							job.Backend = instance.BackendCfg
						}

						// Detect backend type if not set and has file reference
						if job.Backend != nil && job.Backend.Type == "" {
							if job.Backend.File != "" {
								// Will be detected from file, set type based on common patterns
								job.Backend.Type = o.detectBackendTypeFromConfig(job.Backend)
							} else {
								// Detect from inline fields
								job.Backend.Type = o.detectBackendTypeFromInlineConfig(job.Backend)
							}
						}

						job.Environment = instance.Environment
						job.Region = instance.Region

						// Check instance-level parallel setting
						if instance.Parallel != nil {
							if *instance.Parallel {
								// Force this instance to run in parallel
								job.CanParallel = true
							} else {
								// Force this instance to run sequentially
								job.CanParallel = false
							}
						}
					}
				} else {
					job.DependsOn = module.DependsOn
					job.Variables = module.Variables
					job.Backend = module.Backend

					// Detect backend type for module-level backend too
					if job.Backend != nil && job.Backend.Type == "" {
						job.Backend.Type = o.detectBackendTypeFromInlineConfig(job.Backend)
					}
				}
			}

			jobs = append(jobs, job)
		}
	}

	// Calculate dependency depth for display ordering
	o.calculateDependencyDepth(jobs)

	plan := &ExecutionPlan{
		Operation:   string(operation),
		Environment: env,
		Modules:     jobs,
		TotalJobs:   len(jobs),
		CreatedAt:   time.Now(),
	}

	o.logger.Info("Execution plan created",
		"total_jobs", plan.TotalJobs,
		"stages", len(stages),
	)

	return plan, nil
}

// PlanPipeline creates an execution plan for a pipeline
func (o *DefaultOrchestrator) PlanPipeline(pipelineName string, env string) (*ExecutionPlan, error) {
	if o.config == nil {
		return nil, fmt.Errorf("no configuration loaded")
	}

	pipeline, exists := o.config.Pipelines[pipelineName]
	if !exists {
		return nil, fmt.Errorf("pipeline %s not found", pipelineName)
	}

	o.logger.Info("Planning pipeline execution",
		"pipeline", pipelineName,
		"environment", env,
	)

	var allJobs []*ExecutionJob
	jobID := 1

	// Process each stage in the pipeline
	for stageNum, stage := range pipeline.Stages {
		// Collect all modules for this stage
		var stageModules []string

		// Add modules directly specified in the stage
		stageModules = append(stageModules, stage.Modules...)

		// Add modules from groups
		for _, groupName := range stage.Groups {
			groupModules, err := o.parser.ExpandGroup(o.config, groupName)
			if err != nil {
				return nil, fmt.Errorf("failed to expand group %s: %w", groupName, err)
			}
			stageModules = append(stageModules, groupModules...)
		}

		// Create jobs for stage modules
		for _, moduleRef := range stageModules {
			moduleName, instanceName := o.parseModuleReference(moduleRef)

			job := &ExecutionJob{
				ID:           fmt.Sprintf("job-%d", jobID),
				ModuleName:   moduleName,
				InstanceName: instanceName,
				Operation:    pipeline.DefaultOperation,
				Stage:        stageNum,
				Status:       JobStatusPending,
				CanParallel:  stage.Parallel,
			}

			// Set module configuration
			if module, exists := o.config.Modules[moduleName]; exists {
				job.Path = module.Path
				job.Category = module.Category // Set category for grouping

				if instanceName != "" {
					if instance, exists := module.Instances[instanceName]; exists {
						// Combine module-level and instance-level dependencies
						job.DependsOn = make([]string, 0)
						job.DependsOn = append(job.DependsOn, module.DependsOn...)
						job.DependsOn = append(job.DependsOn, instance.DependsOn...)

						job.Variables = instance.Variables
						job.Backend = instance.Backend
					}
				} else {
					job.DependsOn = module.DependsOn
					job.Variables = module.Variables
					job.Backend = module.Backend
				}
			}

			allJobs = append(allJobs, job)
			jobID++
		}
	}

	// Calculate dependency depth for display ordering
	o.calculateDependencyDepth(allJobs)

	plan := &ExecutionPlan{
		Operation:   pipeline.DefaultOperation,
		Environment: env,
		Modules:     allJobs,
		TotalJobs:   len(allJobs),
		CreatedAt:   time.Now(),
	}

	o.logger.Info("Pipeline execution plan created",
		"pipeline", pipelineName,
		"total_jobs", plan.TotalJobs,
		"stages", len(pipeline.Stages),
	)

	return plan, nil
}

// ExecutePlan executes an execution plan
func (o *DefaultOrchestrator) ExecutePlan(plan *ExecutionPlan) (*ExecutionResult, error) {
	o.logger.Info("Starting plan execution", "total_jobs", plan.TotalJobs)

	o.mu.Lock()
	o.cancelled = false
	o.currentExecution = &ExecutionResult{
		TotalJobs: plan.TotalJobs,
		Jobs:      plan.Modules,
		Outputs:   make(map[string]interface{}),
		StartTime: time.Now(),
	}
	o.mu.Unlock()

	// Group jobs by stage
	stageMap := make(map[int][]*ExecutionJob)
	maxStage := 0

	for _, job := range plan.Modules {
		stage := job.Stage
		stageMap[stage] = append(stageMap[stage], job)
		if stage > maxStage {
			maxStage = stage
		}
	}

	// Execute stages sequentially
	ctx := context.Background()
	for stage := 0; stage <= maxStage; stage++ {
		jobs := stageMap[stage]
		if len(jobs) == 0 {
			continue
		}

		// Print stage header with module names
		moduleNames := make([]string, len(jobs))
		for i, job := range jobs {
			moduleNames[i] = job.ModuleName
		}
		fmt.Print(StageHeader(stage+1, maxStage+1, strings.Join(moduleNames, ", ")))

		o.logger.Info("Executing stage", "stage", stage, "jobs", len(jobs))
		o.publishEvent(EventStageStarted, "", "", fmt.Sprintf("stage-%d", stage), fmt.Sprintf("Starting stage %d with %d jobs", stage, len(jobs)), nil)

		// Check if execution was cancelled
		o.mu.RLock()
		cancelled := o.cancelled
		o.mu.RUnlock()

		if cancelled {
			o.logger.Info("Execution cancelled")
			break
		}

		// Execute jobs in this stage
		if len(jobs) == 1 || !jobs[0].CanParallel {
			// Sequential execution
			// Initialize instance message pipelines
			if o.messageRouter != nil {
				for _, job := range jobs {
					operation := TerraformOperation(job.Operation)
					if err := o.messageRouter.InitializeInstance(job.ID, operation); err != nil {
						o.logger.Error("Failed to initialize instance pipeline", "instance_id", job.ID, "error", err)
					}
				}
			}

			for _, job := range jobs {
				if err := o.executeJobWithModule(ctx, job); err != nil {
					o.logger.Error("Job failed", "job_id", job.ID, "error", err)
					// Continue with other jobs even if one fails
				}
			}
		} else {
			// Parallel execution
			result := o.parallelExecutor.ExecuteJobs(ctx, jobs, o.config.Modules)
			o.updateResultFromParallelExecution(result)
		}

		o.publishEvent(EventStageCompleted, "", "", fmt.Sprintf("stage-%d", stage), fmt.Sprintf("Stage %d completed", stage), nil)
	}

	// Finalize result
	o.mu.Lock()
	result := o.currentExecution
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Calculate final statistics
	for _, job := range result.Jobs {
		switch job.Status {
		case JobStatusCompleted:
			result.Completed++
		case JobStatusFailed:
			result.Failed++
			if job.Error != "" {
				result.Errors = append(result.Errors, job.Error)
			}
		case JobStatusSkipped:
			result.Skipped++
		}
	}

	result.Success = result.Failed == 0
	o.mu.Unlock()

	o.logger.Info("Plan execution completed",
		"success", result.Success,
		"completed", result.Completed,
		"failed", result.Failed,
		"skipped", result.Skipped,
		"duration", result.Duration,
	)

	return result, nil
}

// executeJobWithModule executes a single job with module configuration
func (o *DefaultOrchestrator) executeJobWithModule(ctx context.Context, job *ExecutionJob) error {
	o.publishEvent(EventJobStarted, job.ID, job.ModuleName, "", fmt.Sprintf("Starting job %s", job.ID), nil)

	// Get module configuration
	module, exists := o.config.Modules[job.ModuleName]
	if !exists {
		// Create module reference for better error message
		moduleRef := job.ModuleName
		if job.InstanceName != "" {
			moduleRef = job.InstanceName
		}
		err := formatTargetNotFoundError(moduleRef, o.config)
		job.Status = JobStatusFailed
		job.Error = err.Error()
		o.publishEvent(EventJobFailed, job.ID, job.ModuleName, "", err.Error(), nil)
		return err
	}

	var instance *Instance
	if job.InstanceName != "" {
		instance = module.Instances[job.InstanceName]
		if instance == nil {
			err := formatTargetNotFoundError(job.InstanceName, o.config)
			job.Status = JobStatusFailed
			job.Error = err.Error()
			o.publishEvent(EventJobFailed, job.ID, job.ModuleName, "", err.Error(), nil)
			return err
		}
	}

	// Execute the job
	// For single instance execution, don't suppress output (false)
	if err := o.executor.ExecuteJobWithWorkspace(ctx, job, module, instance, false); err != nil {
		o.publishEvent(EventJobFailed, job.ID, job.ModuleName, "", err.Error(), nil)
		return err
	}

	o.publishEvent(EventJobCompleted, job.ID, job.ModuleName, "", fmt.Sprintf("Job %s completed successfully", job.ID), nil)
	return nil
}

// ExecuteModule executes a single module
func (o *DefaultOrchestrator) ExecuteModule(moduleName string, operation TerraformOperation, env string) (*ExecutionResult, error) {
	plan, err := o.PlanExecution(operation, []string{moduleName}, env)
	if err != nil {
		return nil, fmt.Errorf("failed to plan execution: %w", err)
	}

	return o.ExecutePlan(plan)
}

// ExecutePipeline executes a pipeline
func (o *DefaultOrchestrator) ExecutePipeline(pipelineName string, env string) (*ExecutionResult, error) {
	plan, err := o.PlanPipeline(pipelineName, env)
	if err != nil {
		return nil, fmt.Errorf("failed to plan pipeline: %w", err)
	}

	return o.ExecutePlan(plan)
}

// BuildDependencyGraph builds a dependency graph for the specified modules
func (o *DefaultOrchestrator) BuildDependencyGraph(modules []string) (*DependencyGraph, error) {
	if o.graphBuilder == nil {
		return nil, fmt.Errorf("graph builder not initialized")
	}
	return o.graphBuilder.BuildGraph(modules)
}

// ValidateDependencies validates module dependencies
func (o *DefaultOrchestrator) ValidateDependencies() error {
	if o.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Get all modules
	var allModules []string
	for name, module := range o.config.Modules {
		if len(module.Instances) > 0 {
			for instanceName := range module.Instances {
				allModules = append(allModules, fmt.Sprintf("%s.%s", name, instanceName))
			}
		} else {
			allModules = append(allModules, name)
		}
	}

	// Build graph to validate dependencies
	_, err := o.BuildDependencyGraph(allModules)
	return err
}

// GetExecutionOrder returns the execution order for the specified modules
func (o *DefaultOrchestrator) GetExecutionOrder(modules []string) ([][]string, error) {
	graph, err := o.BuildDependencyGraph(modules)
	if err != nil {
		return nil, err
	}

	return o.graphBuilder.GetExecutionOrder(graph), nil
}

// GetExecutionStatus returns the current execution status
func (o *DefaultOrchestrator) GetExecutionStatus() (*ExecutionResult, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.currentExecution == nil {
		return nil, fmt.Errorf("no execution in progress")
	}

	// Return a copy to avoid race conditions
	result := *o.currentExecution
	return &result, nil
}

// CancelExecution cancels the current execution
func (o *DefaultOrchestrator) CancelExecution() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.currentExecution == nil {
		return fmt.Errorf("no execution in progress")
	}

	o.cancelled = true
	o.logger.Info("Execution cancellation requested")

	return nil
}

// CleanupWorkspaces cleans up all workspaces
func (o *DefaultOrchestrator) CleanupWorkspaces() error {
	return o.workspaceManager.CleanupAllWorkspaces()
}

// AddEventHandler adds an event handler
func (o *DefaultOrchestrator) AddEventHandler(handler EventHandler) {
	o.eventHandlers = append(o.eventHandlers, handler)
}

// publishEvent publishes an event to all registered handlers
func (o *DefaultOrchestrator) publishEvent(eventType EventType, jobID, module, stage, message string, data map[string]interface{}) {
	event := &Event{
		Type:      eventType,
		JobID:     jobID,
		Module:    module,
		Stage:     stage,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}

	for _, handler := range o.eventHandlers {
		go handler(event)
	}
}

// Helper methods

// expandTargets expands target specifications (including groups)
func (o *DefaultOrchestrator) expandTargets(targets []string) ([]string, error) {
	var expanded []string
	seen := make(map[string]bool)

	for _, target := range targets {
		// Check if it's a group
		if _, exists := o.config.Groups[target]; exists {
			groupModules, err := o.parser.ExpandGroup(o.config, target)
			if err != nil {
				return nil, fmt.Errorf("failed to expand group %s: %w", target, err)
			}
			for _, module := range groupModules {
				if !seen[module] {
					expanded = append(expanded, module)
					seen[module] = true
				}
			}
		} else {
			// Regular module
			if !seen[target] {
				expanded = append(expanded, target)
				seen[target] = true
			}
		}
	}

	sort.Strings(expanded)
	return expanded, nil
}

// detectBackendTypeFromConfig detects backend type from file path or inline config
func (o *DefaultOrchestrator) detectBackendTypeFromConfig(backend *BackendConfig) string {
	if backend.File != "" {
		// Try to detect from file path or read file
		filePath := backend.File
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(o.basePath, filePath)
		}

		// Read file and detect
		content, err := os.ReadFile(filePath)
		if err == nil {
			contentStr := string(content)
			if strings.Contains(contentStr, "storage_account_name") ||
				strings.Contains(contentStr, "resource_group_name") {
				return "azurerm"
			}
			if strings.Contains(contentStr, "bucket") && strings.Contains(contentStr, "region") {
				return "s3"
			}
			if strings.Contains(contentStr, "bucket") && strings.Contains(contentStr, "prefix") {
				return "gcs"
			}
		}
	}

	// Fall back to inline detection
	return o.detectBackendTypeFromInlineConfig(backend)
}

// detectBackendTypeFromInlineConfig detects backend type from inline configuration fields
func (o *DefaultOrchestrator) detectBackendTypeFromInlineConfig(backend *BackendConfig) string {
	if backend.ResourceGroupName != "" || backend.StorageAccountName != "" || backend.UseAzureADAuth {
		return "azurerm"
	}

	if backend.AdditionalConfig != nil {
		if _, hasBucket := backend.AdditionalConfig["bucket"]; hasBucket {
			return "s3"
		}
		if _, hasProject := backend.AdditionalConfig["project"]; hasProject {
			return "gcs"
		}
	}

	return "local"
}

// parseModuleReference parses a module reference into module and instance names
func (o *DefaultOrchestrator) parseModuleReference(moduleRef string) (string, string) {
	// First, try to find as instance name across all modules
	// Use sorted module names for deterministic behavior
	var moduleNames []string
	for moduleName := range o.config.Modules {
		moduleNames = append(moduleNames, moduleName)
	}
	sort.Strings(moduleNames)

	var foundModules []string
	for _, moduleName := range moduleNames {
		module := o.config.Modules[moduleName]
		if _, exists := module.Instances[moduleRef]; exists {
			foundModules = append(foundModules, moduleName)
		}
	}

	// Warn if instance name appears in multiple modules
	if len(foundModules) > 1 {
		o.logger.Warn("Instance name found in multiple modules - using first alphabetically",
			"instance", moduleRef,
			"modules", foundModules,
			"selected", foundModules[0])
	}

	if len(foundModules) > 0 {
		return foundModules[0], moduleRef // Return first (alphabetically) found module
	}

	// If not found as instance, try module.instance format (backward compatibility)
	parts := strings.Split(moduleRef, ".")
	if len(parts) == 1 {
		// Just module name (no instance)
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// updateResultFromParallelExecution updates the current execution result from parallel execution
func (o *DefaultOrchestrator) updateResultFromParallelExecution(parallelResult *ExecutionResult) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.currentExecution != nil {
		o.currentExecution.Completed += parallelResult.Completed
		o.currentExecution.Failed += parallelResult.Failed
		o.currentExecution.Skipped += parallelResult.Skipped
		o.currentExecution.Errors = append(o.currentExecution.Errors, parallelResult.Errors...)

		// Merge outputs
		for key, value := range parallelResult.Outputs {
			o.currentExecution.Outputs[key] = value
		}
	}
}

// SetDryRun sets dry run mode for the executor
func (o *DefaultOrchestrator) SetDryRun(dryRun bool) {
	o.executor.SetDryRun(dryRun)
}

// SetTimeout sets the execution timeout
func (o *DefaultOrchestrator) SetTimeout(timeout time.Duration) {
	o.executor.SetTimeout(timeout)
}

// SetParallelLimit sets the parallel execution limit
func (o *DefaultOrchestrator) SetParallelLimit(limit int) {
	// Preserve the messageRouter from the old parallelExecutor
	var messageRouter *MessageRouter
	if o.parallelExecutor != nil {
		messageRouter = o.parallelExecutor.messageRouter
	}

	// Create new parallel executor
	o.parallelExecutor = NewParallelExecutor(o.executor, limit, o.logger)

	// Restore messageRouter to the new executor
	if messageRouter != nil {
		o.parallelExecutor.SetMessageRouter(messageRouter)
	}
}

// GetModuleList returns a list of all available modules
func (o *DefaultOrchestrator) GetModuleList() []string {
	if o.config == nil {
		return nil
	}
	return o.parser.GetModuleList(o.config)
}

// GetPipelineList returns a list of all available pipelines
func (o *DefaultOrchestrator) GetPipelineList() []string {
	if o.config == nil {
		return nil
	}

	var pipelines []string
	for name := range o.config.Pipelines {
		pipelines = append(pipelines, name)
	}
	sort.Strings(pipelines)
	return pipelines
}

// GetGroupList returns a list of all available groups
func (o *DefaultOrchestrator) GetGroupList() []string {
	if o.config == nil {
		return nil
	}

	var groups []string
	for name := range o.config.Groups {
		groups = append(groups, name)
	}
	sort.Strings(groups)
	return groups
}

// calculateDependencyDepth calculates the dependency depth for each job based on depends_on relationships
// Depth 0 = no dependencies, Depth 1 = depends on depth 0, etc.
func (o *DefaultOrchestrator) calculateDependencyDepth(jobs []*ExecutionJob) {
	// Create maps for quick lookup
	jobMap := make(map[string]*ExecutionJob)         // instance ID -> job
	moduleToInstanceMap := make(map[string][]string) // module name -> instance IDs

	for _, job := range jobs {
		jobMap[job.ID] = job
		moduleToInstanceMap[job.ModuleName] = append(moduleToInstanceMap[job.ModuleName], job.ID)
	}

	// Calculate depth recursively with memoization
	depthCache := make(map[string]int)

	var calculateDepth func(jobID string, visited map[string]bool) int
	calculateDepth = func(jobID string, visited map[string]bool) int {
		// Check cache first
		if depth, exists := depthCache[jobID]; exists {
			return depth
		}

		job, exists := jobMap[jobID]
		if !exists {
			return 0 // Job not found, treat as no dependencies
		}

		// Check for circular dependency
		if visited[jobID] {
			o.logger.Warn("Circular dependency detected", "job_id", jobID)
			return 0
		}

		// If no dependencies, depth is 0
		if len(job.DependsOn) == 0 {
			depthCache[jobID] = 0
			return 0
		}

		// Mark as visited for circular dependency detection
		visited[jobID] = true

		// Calculate depth as 1 + max depth of dependencies
		maxDepDepth := 0
		for _, depModuleName := range job.DependsOn {
			// Resolve module name to instance IDs
			// If depModuleName is an instance ID, use it directly
			// Otherwise, find all instances of that module
			depInstanceIDs := []string{depModuleName}
			if instances, exists := moduleToInstanceMap[depModuleName]; exists {
				depInstanceIDs = instances
			}

			// Calculate max depth across all instances of the dependency module
			for _, depInstanceID := range depInstanceIDs {
				depDepth := calculateDepth(depInstanceID, visited)
				if depDepth > maxDepDepth {
					maxDepDepth = depDepth
				}
			}
		}

		depth := maxDepDepth + 1
		depthCache[jobID] = depth

		// Unmark visited
		delete(visited, jobID)

		return depth
	}

	// Calculate depth for all jobs
	for _, job := range jobs {
		visited := make(map[string]bool)
		job.DependencyDepth = calculateDepth(job.ID, visited)
	}
}
