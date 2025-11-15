package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultWorkspaceManager implements the WorkspaceManager interface
type DefaultWorkspaceManager struct {
	baseDir    string
	tempDir    string
	workspaces map[string]*Workspace
}

// NewDefaultWorkspaceManager creates a new default workspace manager
func NewDefaultWorkspaceManager(baseDir string) *DefaultWorkspaceManager {
	tempDir := filepath.Join(baseDir, ".tfpipboy", "workspaces")
	return &DefaultWorkspaceManager{
		baseDir:    baseDir,
		tempDir:    tempDir,
		workspaces: make(map[string]*Workspace),
	}
}

// CreateWorkspace creates an isolated workspace for a module execution
func (wm *DefaultWorkspaceManager) CreateWorkspace(jobID string, module *Module, instance *Instance) (*Workspace, error) {
	// Create unique workspace directory
	workspaceDir := filepath.Join(wm.tempDir, jobID)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Create persistent logs directory (outside workspace, so it persists after cleanup)
	logsBaseDir := filepath.Join(filepath.Dir(wm.tempDir), "logs")
	logDir := filepath.Join(logsBaseDir, jobID)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Determine the actual module path
	modulePath := module.Path
	if !filepath.IsAbs(modulePath) {
		modulePath = filepath.Join(wm.baseDir, modulePath)
	}

	// Create workspace
	workspace := &Workspace{
		ID:          jobID,
		Path:        workspaceDir,
		ModulePath:  modulePath,
		LogDir:      logDir,
		Environment: make(map[string]string),
		TempDir:     workspaceDir,
		CreatedAt:   time.Now(),
	}

	// Set up backend configuration
	if err := wm.setupBackend(workspace, module, instance); err != nil {
		wm.CleanupWorkspace(workspace)
		return nil, fmt.Errorf("failed to setup backend: %w", err)
	}

	// Set up variables
	if err := wm.setupVariables(workspace, module, instance); err != nil {
		wm.CleanupWorkspace(workspace)
		return nil, fmt.Errorf("failed to setup variables: %w", err)
	}

	// Set up environment variables
	if err := wm.setupEnvironment(workspace, module, instance); err != nil {
		wm.CleanupWorkspace(workspace)
		return nil, fmt.Errorf("failed to setup environment: %w", err)
	}

	// Copy module files to workspace if needed (for safety)
	if err := wm.setupModuleFiles(workspace, module); err != nil {
		wm.CleanupWorkspace(workspace)
		return nil, fmt.Errorf("failed to setup module files: %w", err)
	}

	// Store workspace
	wm.workspaces[jobID] = workspace

	return workspace, nil
}

// setupBackend configures the Terraform backend for the workspace
func (wm *DefaultWorkspaceManager) setupBackend(workspace *Workspace, module *Module, instance *Instance) error {
	// Determine which backend config to use
	var backendConfig *BackendConfig
	if instance != nil {
		// Check both Backend and BackendCfg (backend-config in YAML)
		if instance.Backend != nil {
			backendConfig = instance.Backend
		} else if instance.BackendCfg != nil {
			backendConfig = instance.BackendCfg
		}
	}
	if backendConfig == nil && module.Backend != nil {
		backendConfig = module.Backend
	}
	if backendConfig == nil {
		// No backend config, use local state
		return nil
	}

	workspace.Backend = backendConfig

	// If external backend file is specified, copy it and detect type from file
	if backendConfig.File != "" {
		// Resolve relative paths from base directory
		backendFilePath := backendConfig.File
		if !filepath.IsAbs(backendFilePath) {
			backendFilePath = filepath.Join(wm.baseDir, backendFilePath)
		}
		if err := wm.copyBackendFile(backendFilePath, workspace.Path); err != nil {
			return fmt.Errorf("failed to copy backend file: %w", err)
		}

		// Detect backend type from the file contents if not already set
		if backendConfig.Type == "" {
			backendConfig.Type = wm.detectBackendTypeFromFile(backendFilePath)
		}

		return nil
	}

	// Auto-detect backend type from inline config fields if not specified
	if backendConfig.Type == "" {
		backendConfig.Type = wm.detectBackendType(backendConfig)
	}

	// Generate backend configuration file for inline config
	// Write as backend.hcl (partial config) so it can be passed with -backend-config flag
	backendFile := filepath.Join(workspace.Path, "backend.hcl")

	var backendContent string
	switch backendConfig.Type {
	case "azurerm":
		backendContent = wm.generateAzureBackendHCL(backendConfig)
	case "s3":
		backendContent = wm.generateS3BackendHCL(backendConfig)
	case "gcs":
		backendContent = wm.generateGCSBackendHCL(backendConfig)
	default:
		// Default to local backend - no config needed
		return nil
	}

	if err := os.WriteFile(backendFile, []byte(backendContent), 0644); err != nil {
		return fmt.Errorf("failed to write backend config: %w", err)
	}

	return nil
}

// detectBackendType auto-detects the backend type from the fields present in BackendConfig
func (wm *DefaultWorkspaceManager) detectBackendType(config *BackendConfig) string {
	// Check for Azure-specific fields
	if config.ResourceGroupName != "" || config.StorageAccountName != "" || config.UseAzureADAuth {
		return "azurerm"
	}

	// Check for S3-specific fields (bucket would be in AdditionalConfig typically)
	if config.AdditionalConfig != nil {
		if _, hasBucket := config.AdditionalConfig["bucket"]; hasBucket {
			return "s3"
		}
		if _, hasProject := config.AdditionalConfig["project"]; hasProject {
			return "gcs"
		}
	}

	// Default to local backend
	return "local"
}

// detectBackendTypeFromFile detects backend type by parsing the backend.hcl file contents
func (wm *DefaultWorkspaceManager) detectBackendTypeFromFile(filePath string) string {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "unknown"
	}

	contentStr := string(content)

	// Check for Azure backend fields
	if strings.Contains(contentStr, "storage_account_name") ||
		strings.Contains(contentStr, "resource_group_name") ||
		strings.Contains(contentStr, "use_azuread_auth") {
		return "azurerm"
	}

	// Check for S3 backend fields
	if strings.Contains(contentStr, "bucket") &&
		(strings.Contains(contentStr, "region") || strings.Contains(contentStr, "dynamodb_table")) {
		return "s3"
	}

	// Check for GCS backend fields
	if strings.Contains(contentStr, "bucket") && strings.Contains(contentStr, "prefix") {
		return "gcs"
	}

	// Default to local
	return "local"
}

// generateAzureBackendHCL generates Azure backend partial configuration (for -backend-config flag)
func (wm *DefaultWorkspaceManager) generateAzureBackendHCL(config *BackendConfig) string {
	var builder strings.Builder

	if config.SubscriptionID != "" {
		builder.WriteString(fmt.Sprintf("subscription_id      = \"%s\"\n", config.SubscriptionID))
	}
	if config.ResourceGroupName != "" {
		builder.WriteString(fmt.Sprintf("resource_group_name  = \"%s\"\n", config.ResourceGroupName))
	}
	if config.StorageAccountName != "" {
		builder.WriteString(fmt.Sprintf("storage_account_name = \"%s\"\n", config.StorageAccountName))
	}
	if config.ContainerName != "" {
		builder.WriteString(fmt.Sprintf("container_name       = \"%s\"\n", config.ContainerName))
	}
	if config.Key != "" {
		builder.WriteString(fmt.Sprintf("key                  = \"%s\"\n", config.Key))
	}
	if config.TenantID != "" {
		builder.WriteString(fmt.Sprintf("tenant_id            = \"%s\"\n", config.TenantID))
	}
	if config.UseAzureADAuth {
		builder.WriteString("use_azuread_auth     = true\n")
	}

	// Add any additional configuration
	for key, value := range config.AdditionalConfig {
		builder.WriteString(fmt.Sprintf("%s = \"%s\"\n", key, value))
	}

	return builder.String()
}

// generateS3BackendHCL generates S3 backend partial configuration (for -backend-config flag)
func (wm *DefaultWorkspaceManager) generateS3BackendHCL(config *BackendConfig) string {
	var builder strings.Builder

	if config.Key != "" {
		builder.WriteString(fmt.Sprintf("key = \"%s\"\n", config.Key))
	}

	// Add any additional configuration
	for key, value := range config.AdditionalConfig {
		builder.WriteString(fmt.Sprintf("%s = \"%s\"\n", key, value))
	}

	return builder.String()
}

// generateGCSBackendHCL generates GCS backend partial configuration (for -backend-config flag)
func (wm *DefaultWorkspaceManager) generateGCSBackendHCL(config *BackendConfig) string {
	var builder strings.Builder

	if config.Key != "" {
		builder.WriteString(fmt.Sprintf("prefix = \"%s\"\n", config.Key))
	}

	// Add any additional configuration
	for key, value := range config.AdditionalConfig {
		builder.WriteString(fmt.Sprintf("%s = \"%s\"\n", key, value))
	}

	return builder.String()
}

// copyBackendFile copies an external backend file to the workspace
func (wm *DefaultWorkspaceManager) copyBackendFile(srcFile, destDir string) error {
	// Read source file
	data, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("failed to read backend file %s: %w", srcFile, err)
	}

	// Write to workspace as backend.hcl (for use with -backend-config flag)
	destFile := filepath.Join(destDir, "backend.hcl")
	if err := os.WriteFile(destFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write backend file: %w", err)
	}

	return nil
}

// setupVariables configures variables for the workspace
func (wm *DefaultWorkspaceManager) setupVariables(workspace *Workspace, module *Module, instance *Instance) error {
	workspace.Variables = make(map[string]string)
	workspace.VarFiles = []string{}

	// Collect variables from module and instance
	var variables map[string]interface{}
	var varConfig *VariableConfig

	if instance != nil {
		if len(instance.Variables) > 0 {
			variables = instance.Variables
		} else {
			variables = module.Variables
		}
		if instance.VarConfig != nil {
			varConfig = instance.VarConfig
		} else {
			varConfig = module.VarConfig
		}
	} else {
		variables = module.Variables
		varConfig = module.VarConfig
	}

	// Process variable configuration
	hasExternalFile := false
	if varConfig != nil {
		// Copy single external variables file if specified (backward compatible)
		if varConfig.File != "" {
			// Resolve relative paths from base directory
			varFilePath := varConfig.File
			if !filepath.IsAbs(varFilePath) {
				varFilePath = filepath.Join(wm.baseDir, varFilePath)
			}
			copiedFile, err := wm.copyVariablesFile(varFilePath, workspace.Path, 0)
			if err != nil {
				return fmt.Errorf("failed to copy variables file: %w", err)
			}
			workspace.VarFiles = append(workspace.VarFiles, copiedFile)
			hasExternalFile = true
		}

		// Copy multiple external variables files if specified
		for idx, file := range varConfig.Files {
			// Resolve relative paths from base directory
			varFilePath := file
			if !filepath.IsAbs(varFilePath) {
				varFilePath = filepath.Join(wm.baseDir, varFilePath)
			}
			copiedFile, err := wm.copyVariablesFile(varFilePath, workspace.Path, idx+1)
			if err != nil {
				return fmt.Errorf("failed to copy variables file %s: %w", file, err)
			}
			workspace.VarFiles = append(workspace.VarFiles, copiedFile)
			hasExternalFile = true
		}

		// Add JSON variables (backward compatible)
		for key, value := range varConfig.JSON {
			workspace.Variables[key] = fmt.Sprintf("%v", value)
		}

		// Add Vars variables
		for key, value := range varConfig.Vars {
			workspace.Variables[key] = fmt.Sprintf("%v", value)
		}
	}

	// Add inline variables
	for key, value := range variables {
		workspace.Variables[key] = fmt.Sprintf("%v", value)
	}

	// Create terraform.tfvars file only if no external file was provided
	// When external file exists, JSON/inline vars will be passed via -var flags
	if !hasExternalFile && len(workspace.Variables) > 0 {
		if err := wm.createTfvarsFile(workspace); err != nil {
			return fmt.Errorf("failed to create tfvars file: %w", err)
		}
	}

	return nil
}

// copyVariablesFile copies an external variables file to the workspace
// Returns the destination file path for use with -var-file
func (wm *DefaultWorkspaceManager) copyVariablesFile(srcFile, destDir string, idx int) (string, error) {
	// Read source file
	data, err := os.ReadFile(srcFile)
	if err != nil {
		return "", fmt.Errorf("failed to read variables file %s: %w", srcFile, err)
	}

	// Determine destination filename based on source extension and index
	srcExt := filepath.Ext(srcFile)
	srcBaseName := filepath.Base(srcFile)
	var destFile string

	if idx == 0 {
		// First file (backward compatible naming)
		switch srcExt {
		case ".tfvars":
			destFile = filepath.Join(destDir, "terraform.tfvars")
		case ".tfvars.json":
			destFile = filepath.Join(destDir, "terraform.tfvars.json")
		default:
			destFile = filepath.Join(destDir, srcBaseName)
		}
	} else {
		// Additional files - preserve original name or add index
		destFile = filepath.Join(destDir, fmt.Sprintf("vars-%d-%s", idx, srcBaseName))
	}

	// Write to workspace
	if err := os.WriteFile(destFile, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write variables file: %w", err)
	}

	return destFile, nil
}

// createTfvarsFile creates a terraform.tfvars file from workspace variables
func (wm *DefaultWorkspaceManager) createTfvarsFile(workspace *Workspace) error {
	var builder strings.Builder

	builder.WriteString("# Generated by tfpipboy\n")
	builder.WriteString("# Workspace: " + workspace.ID + "\n\n")

	for key, value := range workspace.Variables {
		// Simple heuristic to determine if value needs quotes
		if wm.needsQuotes(value) {
			builder.WriteString(fmt.Sprintf("%s = \"%s\"\n", key, value))
		} else {
			builder.WriteString(fmt.Sprintf("%s = %s\n", key, value))
		}
	}

	tfvarsFile := filepath.Join(workspace.Path, "terraform.tfvars")
	if err := os.WriteFile(tfvarsFile, []byte(builder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write tfvars file: %w", err)
	}

	return nil
}

// needsQuotes determines if a variable value needs to be quoted
func (wm *DefaultWorkspaceManager) needsQuotes(value string) bool {
	// Simple heuristic: if it's not a boolean, number, or complex expression, quote it
	switch strings.ToLower(value) {
	case "true", "false":
		return false
	}

	// Check if it's a number
	if strings.ContainsAny(value, "0123456789") && !strings.ContainsAny(value, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return false
	}

	// Check if it looks like a Terraform expression
	if strings.HasPrefix(value, "${") || strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
		return false
	}

	return true
}

// setupEnvironment configures environment variables for the workspace
func (wm *DefaultWorkspaceManager) setupEnvironment(workspace *Workspace, module *Module, instance *Instance) error {
	// Set standard Terraform environment variables
	workspace.Environment["TF_DATA_DIR"] = filepath.Join(workspace.Path, ".terraform")
	workspace.Environment["TF_WORKSPACE"] = "default"

	// Set working directory
	workspace.Environment["TF_WORKING_DIR"] = workspace.Path

	// Add any module-specific environment variables
	if module.Variables != nil {
		for key, value := range module.Variables {
			if strings.HasPrefix(key, "TF_VAR_") {
				workspace.Environment[key] = fmt.Sprintf("%v", value)
			}
		}
	}

	// Add instance-specific environment variables
	if instance != nil && instance.Variables != nil {
		for key, value := range instance.Variables {
			if strings.HasPrefix(key, "TF_VAR_") {
				workspace.Environment[key] = fmt.Sprintf("%v", value)
			}
		}
	}

	// Inherit some environment variables from the parent process
	inheritVars := []string{
		"PATH",
		"HOME",
		"USER",
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_TENANT_ID",
		"AZURE_SUBSCRIPTION_ID",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_REGION",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GOOGLE_PROJECT",
		"GOOGLE_REGION",
	}

	for _, varName := range inheritVars {
		if value := os.Getenv(varName); value != "" {
			workspace.Environment[varName] = value
		}
	}

	// Inject GitHub token from local gh CLI authentication
	// This enables Terraform GitHub provider to use local authentication
	if err := wm.injectGitHubToken(workspace); err != nil {
		// Don't fail the workspace creation, just log a warning
		// GitHub authentication might not be needed for this module
	}

	return nil
}

// injectGitHubToken injects GITHUB_TOKEN from local gh CLI authentication
func (wm *DefaultWorkspaceManager) injectGitHubToken(workspace *Workspace) error {
	// Check if GITHUB_TOKEN is already set in environment
	if _, exists := workspace.Environment["GITHUB_TOKEN"]; exists {
		return nil // Already set, don't override
	}

	// Try to get token from gh CLI
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("gh auth token failed: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token != "" {
		workspace.Environment["GITHUB_TOKEN"] = token
	}

	return nil
}

// setupModuleFiles sets up module files in the workspace
func (wm *DefaultWorkspaceManager) setupModuleFiles(workspace *Workspace, module *Module) error {
	// For now, we'll use the original module path directly
	// In the future, we could copy files for complete isolation

	// Create a symlink to the module directory for easier access
	moduleSymlink := filepath.Join(workspace.Path, "module")
	if err := os.Symlink(workspace.ModulePath, moduleSymlink); err != nil {
		// If symlink fails, it's not critical - we can still use the original path
		// This might happen on Windows or in environments where symlinks aren't supported
	}

	return nil
}

// GetWorkspace retrieves an existing workspace
func (wm *DefaultWorkspaceManager) GetWorkspace(jobID string) (*Workspace, error) {
	workspace, exists := wm.workspaces[jobID]
	if !exists {
		return nil, fmt.Errorf("workspace %s not found", jobID)
	}
	return workspace, nil
}

// CleanupWorkspace cleans up a workspace and removes its files
func (wm *DefaultWorkspaceManager) CleanupWorkspace(workspace *Workspace) error {
	// Remove from tracking
	delete(wm.workspaces, workspace.ID)

	// Remove workspace directory
	if err := os.RemoveAll(workspace.Path); err != nil {
		return fmt.Errorf("failed to remove workspace directory: %w", err)
	}

	return nil
}

// CleanupAllWorkspaces cleans up all workspaces
func (wm *DefaultWorkspaceManager) CleanupAllWorkspaces() error {
	var errors []string

	// First, clean up tracked workspaces
	for _, workspace := range wm.workspaces {
		if err := wm.CleanupWorkspace(workspace); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Also clean up any workspace directories on disk that might not be tracked
	// (e.g., from previous runs or when cleanup is called separately)
	workspacesDir := wm.tempDir
	if entries, err := os.ReadDir(workspacesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				workspacePath := filepath.Join(workspacesDir, entry.Name())
				// Only remove if it's not already in our tracked workspaces
				isTracked := false
				for _, workspace := range wm.workspaces {
					if workspace.Path == workspacePath {
						isTracked = true
						break
					}
				}
				if !isTracked {
					if err := os.RemoveAll(workspacePath); err != nil {
						errors = append(errors, fmt.Sprintf("failed to remove workspace %s: %v", entry.Name(), err))
					}
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to cleanup some workspaces: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ListWorkspaces returns a list of all active workspaces
func (wm *DefaultWorkspaceManager) ListWorkspaces() []*Workspace {
	var workspaces []*Workspace
	for _, workspace := range wm.workspaces {
		workspaces = append(workspaces, workspace)
	}
	return workspaces
}

// GetWorkspacePath returns the working directory for Terraform commands
func (wm *DefaultWorkspaceManager) GetWorkspacePath(workspace *Workspace) string {
	// For execution, we want to run terraform in the original module directory
	// but with our isolated backend and variables
	return workspace.ModulePath
}

// GetWorkspaceEnvironment returns the environment variables for the workspace
func (wm *DefaultWorkspaceManager) GetWorkspaceEnvironment(workspace *Workspace) map[string]string {
	// Create a copy to avoid modification of the original
	env := make(map[string]string)
	for key, value := range workspace.Environment {
		env[key] = value
	}

	// Add workspace-specific paths
	env["TF_DATA_DIR"] = filepath.Join(workspace.Path, ".terraform")

	// Point to our generated backend and variables files
	if workspace.Backend != nil {
		env["TF_CLI_ARGS_init"] = fmt.Sprintf("-backend-config=%s", filepath.Join(workspace.Path, "backend.tf"))
	}

	return env
}
