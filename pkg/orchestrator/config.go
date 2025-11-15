package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ValidationError represents a single validation error
type ValidationError struct {
	Category string // "module", "pipeline", "backend", "dependency", etc.
	Item     string // Name of the item (module name, pipeline name, etc.)
	Field    string // Specific field with issue (optional)
	Message  string // Error message
}

// ValidationResult holds all validation errors and warnings
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
}

// HasErrors returns true if there are any validation errors
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// AddError adds a validation error
func (vr *ValidationResult) AddError(category, item, field, message string) {
	vr.Errors = append(vr.Errors, ValidationError{
		Category: category,
		Item:     item,
		Field:    field,
		Message:  message,
	})
}

// AddWarning adds a validation warning
func (vr *ValidationResult) AddWarning(category, item, field, message string) {
	vr.Warnings = append(vr.Warnings, ValidationError{
		Category: category,
		Item:     item,
		Field:    field,
		Message:  message,
	})
}

// FormatReport formats validation results as a user-friendly report
func (vr *ValidationResult) FormatReport() string {
	if !vr.HasErrors() && len(vr.Warnings) == 0 {
		return ""
	}

	var sb strings.Builder

	if vr.HasErrors() {
		sb.WriteString("\n")
		sb.WriteString("================================================================================\n")
		sb.WriteString("   ❌ CONFIGURATION VALIDATION ERRORS\n")
		sb.WriteString("================================================================================\n")
		sb.WriteString(fmt.Sprintf("Found %d error(s) that must be fixed:\n\n", len(vr.Errors)))

		// Group errors by category
		errorsByCategory := make(map[string][]ValidationError)
		for _, err := range vr.Errors {
			errorsByCategory[err.Category] = append(errorsByCategory[err.Category], err)
		}

		categories := make([]string, 0, len(errorsByCategory))
		for cat := range errorsByCategory {
			categories = append(categories, cat)
		}
		sort.Strings(categories)

		for _, cat := range categories {
			sb.WriteString(fmt.Sprintf("▶ %s:\n", strings.ToUpper(cat)))
			for _, err := range errorsByCategory[cat] {
				if err.Field != "" {
					sb.WriteString(fmt.Sprintf("  • %s [%s]: %s\n", err.Item, err.Field, err.Message))
				} else {
					sb.WriteString(fmt.Sprintf("  • %s: %s\n", err.Item, err.Message))
				}
			}
			sb.WriteString("\n")
		}
		sb.WriteString("================================================================================\n")
	}

	if len(vr.Warnings) > 0 {
		sb.WriteString("\n")
		sb.WriteString("⚠️  WARNINGS:\n")
		for _, warn := range vr.Warnings {
			if warn.Field != "" {
				sb.WriteString(fmt.Sprintf("  • %s [%s]: %s\n", warn.Item, warn.Field, warn.Message))
			} else {
				sb.WriteString(fmt.Sprintf("  • %s: %s\n", warn.Item, warn.Message))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ConfigParser handles parsing of tfpipboy configuration files
type ConfigParser struct {
	basePath string
}

// NewConfigParser creates a new configuration parser
func NewConfigParser(basePath string) *ConfigParser {
	return &ConfigParser{
		basePath: basePath,
	}
}

// LoadConfig loads and parses the configuration from the specified directory
func (p *ConfigParser) LoadConfig(configDir string) (*Config, error) {
	config := &Config{
		Modules:   make(map[string]*Module),
		Groups:    make(map[string][]string),
		Pipelines: make(map[string]*Pipeline),
	}

	// Try to load from different possible files
	configFiles := []string{
		filepath.Join(configDir, "tfproject.yaml"),
		filepath.Join(configDir, "modules.yaml"),
		filepath.Join(configDir, "pipelines.yaml"),
	}

	var loadedFiles []string
	for _, file := range configFiles {
		if _, err := os.Stat(file); err == nil {
			if err := p.loadConfigFile(file, config); err != nil {
				return nil, fmt.Errorf("failed to load config file %s: %w", file, err)
			}
			loadedFiles = append(loadedFiles, file)
		}
	}

	if len(loadedFiles) == 0 {
		return nil, fmt.Errorf("no configuration files found in %s", configDir)
	}

	// Post-process configuration
	if err := p.postProcessConfig(config); err != nil {
		return nil, fmt.Errorf("failed to post-process config: %w", err)
	}

	return config, nil
}

// loadConfigFile loads a single configuration file
func (p *ConfigParser) loadConfigFile(filename string, config *Config) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create a temporary config to parse into
	tempConfig := &Config{
		Modules:   make(map[string]*Module),
		Groups:    make(map[string][]string),
		Pipelines: make(map[string]*Pipeline),
	}

	if err := yaml.Unmarshal(data, tempConfig); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Merge the temporary config into the main config
	p.mergeConfigs(config, tempConfig)

	return nil
}

// mergeConfigs merges source config into target config
func (p *ConfigParser) mergeConfigs(target, source *Config) {
	// Set version if not already set
	if target.Version == "" && source.Version != "" {
		target.Version = source.Version
	}

	// Merge modules
	for name, module := range source.Modules {
		module.Name = name
		target.Modules[name] = module
	}

	// Merge groups
	for name, modules := range source.Groups {
		target.Groups[name] = modules
	}

	// Merge pipelines
	for name, pipeline := range source.Pipelines {
		pipeline.Name = name
		target.Pipelines[name] = pipeline
	}
}

// postProcessConfig performs post-processing on the loaded configuration
func (p *ConfigParser) postProcessConfig(config *Config) error {
	// Process modules
	for name, module := range config.Modules {
		if err := p.processModule(name, module); err != nil {
			return fmt.Errorf("failed to process module %s: %w", name, err)
		}
	}

	// Process pipelines
	for name, pipeline := range config.Pipelines {
		if err := p.processPipeline(name, pipeline); err != nil {
			return fmt.Errorf("failed to process pipeline %s: %w", name, err)
		}
	}

	return nil
}

// processModule processes a single module configuration
func (p *ConfigParser) processModule(name string, module *Module) error {
	module.Name = name

	// Set default enabled state
	if module.Enabled == false && module.Path != "" {
		module.Enabled = true
	}

	// Merge backend-config into backend if backend-config is used
	if module.BackendCfg != nil && module.Backend == nil {
		module.Backend = module.BackendCfg
	}

	// Process module path - make it absolute if relative
	if module.Path != "" && !filepath.IsAbs(module.Path) {
		module.Path = filepath.Join(p.basePath, module.Path)
	}

	// Process instances
	for instanceName, instance := range module.Instances {
		instance.Name = instanceName
		if instance.Enabled == false {
			instance.Enabled = true
		}

		// Merge backend-config into backend for instance
		if instance.BackendCfg != nil && instance.Backend == nil {
			instance.Backend = instance.BackendCfg
		}

		// Inherit from parent module if not specified
		if instance.Backend == nil {
			instance.Backend = module.Backend
		}
		if instance.VarConfig == nil {
			instance.VarConfig = module.VarConfig
		}
		if len(instance.Variables) == 0 {
			instance.Variables = module.Variables
		}
	}

	// Process backend configuration
	if module.Backend != nil {
		if err := p.processBackendConfig(module.Backend); err != nil {
			return fmt.Errorf("failed to process backend config: %w", err)
		}
	}

	// Process variable configuration
	if module.VarConfig != nil {
		if err := p.processVariableConfig(module.VarConfig); err != nil {
			return fmt.Errorf("failed to process variable config: %w", err)
		}
	}

	return nil
}

// processBackendConfig processes backend configuration
func (p *ConfigParser) processBackendConfig(backend *BackendConfig) error {
	// Set default backend type
	if backend.Type == "" {
		if backend.SubscriptionID != "" || backend.StorageAccountName != "" {
			backend.Type = "azurerm"
		} else {
			backend.Type = "local"
		}
	}

	// Process backend file path
	if backend.File != "" && !filepath.IsAbs(backend.File) {
		backend.File = filepath.Join(p.basePath, backend.File)
	}

	return nil
}

// processVariableConfig processes variable configuration
func (p *ConfigParser) processVariableConfig(varConfig *VariableConfig) error {
	// Process single variable file path (backward compatible)
	if varConfig.File != "" && !filepath.IsAbs(varConfig.File) {
		varConfig.File = filepath.Join(p.basePath, varConfig.File)
	}

	// Process multiple variable file paths
	for i, file := range varConfig.Files {
		if file != "" && !filepath.IsAbs(file) {
			varConfig.Files[i] = filepath.Join(p.basePath, file)
		}
	}

	return nil
}

// processPipeline processes a single pipeline configuration
func (p *ConfigParser) processPipeline(name string, pipeline *Pipeline) error {
	pipeline.Name = name

	// Set default operation
	if pipeline.DefaultOperation == "" {
		pipeline.DefaultOperation = "plan"
	}

	// Process pipeline settings
	if pipeline.Settings != nil {
		p.processPipelineSettings(pipeline.Settings)
	}

	// Process stages
	for _, stage := range pipeline.Stages {
		if stage.Settings != nil {
			p.processPipelineSettings(stage.Settings)
		}
	}

	// Process environment configurations
	for _, envConfig := range pipeline.Environments {
		if envConfig.Settings != nil {
			p.processPipelineSettings(envConfig.Settings)
		}
	}

	return nil
}

// processPipelineSettings processes pipeline settings
func (p *ConfigParser) processPipelineSettings(settings *PipelineSettings) {
	// Set default parallel limit
	if settings.ParallelLimit <= 0 {
		settings.ParallelLimit = 1
	}

	// Set default timeout
	if settings.Timeout == 0 {
		settings.Timeout = 30 * time.Minute
	}

	// Set default retry count
	if settings.RetryCount <= 0 {
		settings.RetryCount = 0
	}

	// Set default retry delay
	if settings.RetryDelay == 0 {
		settings.RetryDelay = 30 * time.Second
	}
}

// ValidateConfig validates the loaded configuration and collects all errors
func (p *ConfigParser) ValidateConfig(config *Config) error {
	result := &ValidationResult{}

	// Validate modules
	for name, module := range config.Modules {
		p.validateModule(name, module, config, result)
	}

	// Check for duplicate instance names across modules
	p.validateUniqueInstanceNames(config, result)

	// Validate groups
	for name, modules := range config.Groups {
		p.validateGroup(name, modules, config, result)
	}

	// Validate pipelines
	for name, pipeline := range config.Pipelines {
		p.validatePipeline(name, pipeline, config, result)
	}

	// Validate dependencies (checks for cycles)
	p.validateDependencies(config, result)

	// If there are errors or warnings, format and return them
	if result.HasErrors() || len(result.Warnings) > 0 {
		report := result.FormatReport()
		if result.HasErrors() {
			return fmt.Errorf("%s", report)
		}
		// Just warnings, print them but don't fail
		fmt.Print(report)
	}

	return nil
}

// validateUniqueInstanceNames checks for duplicate instance names across modules
func (p *ConfigParser) validateUniqueInstanceNames(config *Config, result *ValidationResult) {
	instanceToModules := make(map[string][]string)

	for moduleName, module := range config.Modules {
		for instanceName := range module.Instances {
			instanceToModules[instanceName] = append(instanceToModules[instanceName], moduleName)
		}
	}

	// Add warnings for duplicates
	for instanceName, modules := range instanceToModules {
		if len(modules) > 1 {
			// Sort modules for deterministic warning message
			sort.Strings(modules)
			result.AddWarning("instance", instanceName, "",
				fmt.Sprintf("Duplicated across modules %v - module '%s' will be selected (alphabetically first)", modules, modules[0]))
		}
	}
}

// validateModule validates a single module
func (p *ConfigParser) validateModule(name string, module *Module, config *Config, result *ValidationResult) {
	if module.Path == "" {
		result.AddError("module", name, "path", "module path is required")
		return
	}

	// Check if path exists
	if _, err := os.Stat(module.Path); err != nil {
		result.AddError("module", name, "path", fmt.Sprintf("module path does not exist: %s", module.Path))
	}

	// Validate backend configuration if present
	if module.Backend != nil {
		p.validateBackend(name, module.Backend, result)
	}

	// Validate dependencies exist
	for _, dep := range module.DependsOn {
		if err := p.validateDependency(dep, config); err != nil {
			result.AddError("module", name, "depends_on", fmt.Sprintf("invalid dependency '%s': %v", dep, err))
		}
	}

	// Validate instances
	for instanceName, instance := range module.Instances {
		// Validate instance backend if present
		if instance.Backend != nil {
			p.validateBackend(fmt.Sprintf("%s.%s", name, instanceName), instance.Backend, result)
		}

		// Validate instance dependencies
		for _, dep := range instance.DependsOn {
			if err := p.validateDependency(dep, config); err != nil {
				result.AddError("instance", fmt.Sprintf("%s.%s", name, instanceName), "depends_on",
					fmt.Sprintf("invalid dependency '%s': %v", dep, err))
			}
		}
	}
}

// validateBackend validates backend configuration
func (p *ConfigParser) validateBackend(item string, backend *BackendConfig, result *ValidationResult) {
	// If no type is specified, skip validation
	// This allows for partial backend configurations or inheritance
	if backend.Type == "" {
		return
	}

	// Validate based on backend type
	switch backend.Type {
	case "azurerm":
		if backend.ResourceGroupName == "" {
			result.AddError("backend", item, "resource_group_name", "resource_group_name is required for azurerm backend")
		}
		if backend.StorageAccountName == "" {
			result.AddError("backend", item, "storage_account_name", "storage_account_name is required for azurerm backend")
		}
		if backend.ContainerName == "" {
			result.AddError("backend", item, "container_name", "container_name is required for azurerm backend")
		}
		if backend.Key == "" {
			result.AddError("backend", item, "key", "key is required for azurerm backend")
		}
	case "s3":
		// Check AdditionalConfig for s3-specific fields
		if backend.AdditionalConfig["bucket"] == "" {
			result.AddError("backend", item, "bucket", "bucket is required for s3 backend")
		}
		if backend.Key == "" && backend.AdditionalConfig["key"] == "" {
			result.AddError("backend", item, "key", "key is required for s3 backend")
		}
		if backend.AdditionalConfig["region"] == "" {
			result.AddError("backend", item, "region", "region is required for s3 backend")
		}
	case "gcs":
		// Check AdditionalConfig for gcs-specific fields
		if backend.AdditionalConfig["bucket"] == "" {
			result.AddError("backend", item, "bucket", "bucket is required for gcs backend")
		}
		if backend.AdditionalConfig["prefix"] == "" {
			result.AddError("backend", item, "prefix", "prefix is required for gcs backend")
		}
	case "local":
		if backend.AdditionalConfig["path"] == "" {
			result.AddError("backend", item, "path", "path is required for local backend")
		}
	default:
		result.AddWarning("backend", item, "type", fmt.Sprintf("unknown backend type '%s' - validation skipped", backend.Type))
	}
}

// validateGroup validates a module group
func (p *ConfigParser) validateGroup(name string, modules []string, config *Config, result *ValidationResult) {
	if len(modules) == 0 {
		result.AddError("group", name, "", "group must contain at least one module")
		return
	}

	for _, moduleName := range modules {
		if err := p.validateDependency(moduleName, config); err != nil {
			result.AddError("group", name, "modules", fmt.Sprintf("invalid module '%s': %v", moduleName, err))
		}
	}
}

// validatePipeline validates a pipeline
func (p *ConfigParser) validatePipeline(name string, pipeline *Pipeline, config *Config, result *ValidationResult) {
	if len(pipeline.Stages) == 0 {
		result.AddError("pipeline", name, "stages", "pipeline must contain at least one stage")
		return
	}

	// Validate stages
	for i, stage := range pipeline.Stages {
		stageDesc := fmt.Sprintf("stage %d", i+1)

		// Check that stage has modules or groups
		if len(stage.Modules) == 0 && len(stage.Groups) == 0 {
			result.AddError("pipeline", name, stageDesc, "stage must contain at least one module or group")
		}

		// Validate modules in stage
		for _, moduleName := range stage.Modules {
			if err := p.validateDependency(moduleName, config); err != nil {
				result.AddError("pipeline", name, stageDesc, fmt.Sprintf("invalid module '%s': %v", moduleName, err))
			}
		}

		// Validate groups in stage
		for _, groupName := range stage.Groups {
			if _, exists := config.Groups[groupName]; !exists {
				result.AddError("pipeline", name, stageDesc, fmt.Sprintf("group '%s' not found", groupName))
			}
		}
	}
}

// validateDependency validates that a dependency reference is valid
func (p *ConfigParser) validateDependency(dep string, config *Config) error {
	// First, try to find as instance name across all modules
	for moduleName, module := range config.Modules {
		if _, exists := module.Instances[dep]; exists {
			return nil // Found as instance name
		}
		// Also check if the dep matches the module name (for backward compatibility)
		if moduleName == dep {
			return nil
		}
	}

	// If not found as instance name, try module.instance format (backward compatibility)
	parts := strings.Split(dep, ".")
	if len(parts) > 1 {
		moduleName := parts[0]
		instanceName := parts[1]

		module, exists := config.Modules[moduleName]
		if !exists {
			return fmt.Errorf("module %s not found", moduleName)
		}

		if _, exists := module.Instances[instanceName]; !exists {
			return fmt.Errorf("instance %s not found in module %s", instanceName, moduleName)
		}
		return nil
	}

	// Not found in any format
	return fmt.Errorf("dependency %s not found (not a valid instance name or module reference)", dep)
}

// validateDependencies validates that there are no circular dependencies
func (p *ConfigParser) validateDependencies(config *Config, result *ValidationResult) {
	// Build dependency graph
	graph := make(map[string][]string)

	// Add module dependencies
	for name, module := range config.Modules {
		graph[name] = module.DependsOn

		// Add instance dependencies
		for instanceName, instance := range module.Instances {
			instanceKey := fmt.Sprintf("%s.%s", name, instanceName)
			graph[instanceKey] = instance.DependsOn
		}
	}

	// Check for circular dependencies using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range graph {
		if !visited[node] {
			cycle := p.findCycle(node, graph, visited, recStack, []string{})
			if len(cycle) > 0 {
				cycleStr := strings.Join(cycle, " → ")
				result.AddError("dependency", node, "", fmt.Sprintf("circular dependency: %s", cycleStr))
			}
		}
	}
}

// findCycle finds and returns the cycle path if one exists
func (p *ConfigParser) findCycle(node string, graph map[string][]string, visited, recStack map[string]bool, path []string) []string {
	visited[node] = true
	recStack[node] = true
	path = append(path, node)

	for _, dep := range graph[node] {
		if !visited[dep] {
			if cycle := p.findCycle(dep, graph, visited, recStack, path); len(cycle) > 0 {
				return cycle
			}
		} else if recStack[dep] {
			// Found a cycle, return the path
			cycleStart := -1
			for i, n := range path {
				if n == dep {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				return append(path[cycleStart:], dep)
			}
			return append(path, dep)
		}
	}

	recStack[node] = false
	return nil
}

// SaveConfig saves the configuration to a file
func (p *ConfigParser) SaveConfig(config *Config, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetModuleList returns a list of all modules (including instances)
func (p *ConfigParser) GetModuleList(config *Config) []string {
	var modules []string

	for name, module := range config.Modules {
		if len(module.Instances) > 0 {
			for instanceName := range module.Instances {
				modules = append(modules, fmt.Sprintf("%s.%s", name, instanceName))
			}
		} else {
			modules = append(modules, name)
		}
	}

	return modules
}

// ExpandGroup expands a group name to a list of modules
func (p *ConfigParser) ExpandGroup(config *Config, groupName string) ([]string, error) {
	group, exists := config.Groups[groupName]
	if !exists {
		return nil, fmt.Errorf("group %s not found", groupName)
	}

	var expanded []string
	for _, item := range group {
		// Check if item is another group
		if _, isGroup := config.Groups[item]; isGroup {
			subExpanded, err := p.ExpandGroup(config, item)
			if err != nil {
				return nil, err
			}
			expanded = append(expanded, subExpanded...)
		} else {
			expanded = append(expanded, item)
		}
	}

	return expanded, nil
}
