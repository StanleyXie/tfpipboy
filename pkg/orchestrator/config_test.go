package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewConfigParser tests creating a new config parser
func TestNewConfigParser(t *testing.T) {
	parser := NewConfigParser("/test/path")

	if parser == nil {
		t.Fatal("Expected non-nil parser")
	}

	if parser.basePath != "/test/path" {
		t.Errorf("Expected basePath '/test/path', got '%s'", parser.basePath)
	}
}

// TestLoadConfig_ValidYAML tests loading a valid configuration
func TestLoadConfig_ValidYAML(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create a test configuration file
	configContent := `
version: "1.0"
modules:
  vpc:
    path: ./terraform/vpc
    description: VPC module
    instances:
      vpc-dev:
        environment: dev
        variables:
          cidr: "10.0.0.0/16"
`

	configPath := filepath.Join(tmpDir, "tfproject.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	parser := NewConfigParser(tmpDir)
	config, err := parser.LoadConfig(tmpDir)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", config.Version)
	}

	if len(config.Modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(config.Modules))
	}

	vpc, exists := config.Modules["vpc"]
	if !exists {
		t.Fatal("Expected 'vpc' module to exist")
	}

	if vpc.Description != "VPC module" {
		t.Errorf("Expected description 'VPC module', got '%s'", vpc.Description)
	}

	if len(vpc.Instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(vpc.Instances))
	}

	instance, exists := vpc.Instances["vpc-dev"]
	if !exists {
		t.Fatal("Expected 'vpc-dev' instance to exist")
	}

	if instance.Environment != "dev" {
		t.Errorf("Expected environment 'dev', got '%s'", instance.Environment)
	}
}

// TestLoadConfig_NoConfigFile tests error when no config file exists
func TestLoadConfig_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	parser := NewConfigParser(tmpDir)
	_, err := parser.LoadConfig(tmpDir)

	if err == nil {
		t.Fatal("Expected error when no config file exists")
	}

	if !strings.Contains(err.Error(), "no configuration files found") {
		t.Errorf("Expected 'no configuration files found' error, got: %v", err)
	}
}

// TestLoadConfig_InvalidYAML tests error when YAML is invalid
func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid YAML
	configContent := `
version: "1.0"
modules:
  vpc:
    path: ./terraform/vpc
    invalid: [unclosed array
`

	configPath := filepath.Join(tmpDir, "tfproject.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	parser := NewConfigParser(tmpDir)
	_, err := parser.LoadConfig(tmpDir)

	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}

	if !strings.Contains(err.Error(), "failed to parse YAML") {
		t.Errorf("Expected 'failed to parse YAML' error, got: %v", err)
	}
}

// TestValidateConfig_ValidConfig tests validating a valid configuration
func TestValidateConfig_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create module directories
	vpcPath := filepath.Join(tmpDir, "terraform", "vpc")
	if err := os.MkdirAll(vpcPath, 0755); err != nil {
		t.Fatalf("Failed to create module directory: %v", err)
	}

	config := &Config{
		Version: "1.0",
		Modules: map[string]*Module{
			"vpc": {
				Name: "vpc",
				Path: vpcPath,
				Instances: map[string]*Instance{
					"vpc-dev": {
						Name:        "vpc-dev",
						Environment: "dev",
					},
				},
			},
		},
	}

	parser := NewConfigParser(tmpDir)
	err := parser.ValidateConfig(config)

	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}

// TestValidateConfig_MissingModulePath tests validation error for missing module path
func TestValidateConfig_MissingModulePath(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"vpc": {
				Name: "vpc",
				// Path is missing
			},
		},
	}

	parser := NewConfigParser("/tmp")
	err := parser.ValidateConfig(config)

	if err == nil {
		t.Fatal("Expected error for missing module path")
	}

	if !strings.Contains(err.Error(), "module path is required") {
		t.Errorf("Expected 'module path is required' error, got: %v", err)
	}
}

// TestValidateConfig_NonExistentModulePath tests validation error for non-existent path
func TestValidateConfig_NonExistentModulePath(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"vpc": {
				Name: "vpc",
				Path: "/non/existent/path",
			},
		},
	}

	parser := NewConfigParser("/tmp")
	err := parser.ValidateConfig(config)

	if err == nil {
		t.Fatal("Expected error for non-existent module path")
	}

	if !strings.Contains(err.Error(), "module path does not exist") {
		t.Errorf("Expected 'module path does not exist' error, got: %v", err)
	}
}

// TestValidateConfig_InvalidDependency tests validation error for invalid dependency
func TestValidateConfig_InvalidDependency(t *testing.T) {
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "terraform", "app")
	if err := os.MkdirAll(modulePath, 0755); err != nil {
		t.Fatalf("Failed to create module directory: %v", err)
	}

	config := &Config{
		Modules: map[string]*Module{
			"app": {
				Name:      "app",
				Path:      modulePath,
				DependsOn: []string{"non-existent-module"},
			},
		},
	}

	parser := NewConfigParser(tmpDir)
	err := parser.ValidateConfig(config)

	if err == nil {
		t.Fatal("Expected error for invalid dependency")
	}

	if !strings.Contains(err.Error(), "invalid dependency") {
		t.Errorf("Expected 'invalid dependency' error, got: %v", err)
	}
}

// TestValidateConfig_CircularDependency tests detection of circular dependencies
func TestValidateConfig_CircularDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create module directories
	moduleAPath := filepath.Join(tmpDir, "terraform", "module-a")
	moduleBPath := filepath.Join(tmpDir, "terraform", "module-b")
	if err := os.MkdirAll(moduleAPath, 0755); err != nil {
		t.Fatalf("Failed to create module-a directory: %v", err)
	}
	if err := os.MkdirAll(moduleBPath, 0755); err != nil {
		t.Fatalf("Failed to create module-b directory: %v", err)
	}

	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: moduleAPath,
				Instances: map[string]*Instance{
					"instance-a": {
						Name:      "instance-a",
						DependsOn: []string{"instance-b"},
					},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: moduleBPath,
				Instances: map[string]*Instance{
					"instance-b": {
						Name:      "instance-b",
						DependsOn: []string{"instance-a"},
					},
				},
			},
		},
	}

	parser := NewConfigParser(tmpDir)
	err := parser.ValidateConfig(config)

	if err == nil {
		t.Fatal("Expected error for circular dependency")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("Expected 'circular dependency' error, got: %v", err)
	}
}

// TestValidateBackend_AzureRM tests Azure backend validation
func TestValidateBackend_AzureRM(t *testing.T) {
	tests := []struct {
		name        string
		backend     *BackendConfig
		expectError bool
		errorString string
	}{
		{
			name: "valid azurerm backend",
			backend: &BackendConfig{
				Type:               "azurerm",
				ResourceGroupName:  "rg-tfstate",
				StorageAccountName: "tfstatestorage",
				ContainerName:      "tfstate",
				Key:                "terraform.tfstate",
			},
			expectError: false,
		},
		{
			name: "missing resource group",
			backend: &BackendConfig{
				Type:               "azurerm",
				StorageAccountName: "tfstatestorage",
				ContainerName:      "tfstate",
				Key:                "terraform.tfstate",
			},
			expectError: true,
			errorString: "resource_group_name is required",
		},
		{
			name: "missing storage account",
			backend: &BackendConfig{
				Type:              "azurerm",
				ResourceGroupName: "rg-tfstate",
				ContainerName:     "tfstate",
				Key:               "terraform.tfstate",
			},
			expectError: true,
			errorString: "storage_account_name is required",
		},
		{
			name: "missing container",
			backend: &BackendConfig{
				Type:               "azurerm",
				ResourceGroupName:  "rg-tfstate",
				StorageAccountName: "tfstatestorage",
				Key:                "terraform.tfstate",
			},
			expectError: true,
			errorString: "container_name is required",
		},
		{
			name: "missing key",
			backend: &BackendConfig{
				Type:               "azurerm",
				ResourceGroupName:  "rg-tfstate",
				StorageAccountName: "tfstatestorage",
				ContainerName:      "tfstate",
			},
			expectError: true,
			errorString: "key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			modulePath := filepath.Join(tmpDir, "terraform", "test")
			if err := os.MkdirAll(modulePath, 0755); err != nil {
				t.Fatalf("Failed to create module directory: %v", err)
			}

			config := &Config{
				Modules: map[string]*Module{
					"test": {
						Name:    "test",
						Path:    modulePath,
						Backend: tt.backend,
					},
				},
			}

			parser := NewConfigParser(tmpDir)
			err := parser.ValidateConfig(config)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if !strings.Contains(err.Error(), tt.errorString) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorString, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidatePipeline tests pipeline validation
func TestValidatePipeline(t *testing.T) {
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "terraform", "vpc")
	if err := os.MkdirAll(modulePath, 0755); err != nil {
		t.Fatalf("Failed to create module directory: %v", err)
	}

	tests := []struct {
		name        string
		pipeline    *Pipeline
		config      *Config
		expectError bool
		errorString string
	}{
		{
			name: "valid pipeline",
			pipeline: &Pipeline{
				Name: "deploy",
				Stages: []*Stage{
					{
						Name:    "stage1",
						Modules: []string{"vpc"},
					},
				},
			},
			config: &Config{
				Modules: map[string]*Module{
					"vpc": {
						Name: "vpc",
						Path: modulePath,
					},
				},
			},
			expectError: false,
		},
		{
			name: "pipeline with no stages",
			pipeline: &Pipeline{
				Name:   "deploy",
				Stages: []*Stage{},
			},
			config:      &Config{Modules: make(map[string]*Module)},
			expectError: true,
			errorString: "pipeline must contain at least one stage",
		},
		{
			name: "stage with invalid module",
			pipeline: &Pipeline{
				Name: "deploy",
				Stages: []*Stage{
					{
						Name:    "stage1",
						Modules: []string{"non-existent"},
					},
				},
			},
			config:      &Config{Modules: make(map[string]*Module)},
			expectError: true,
			errorString: "invalid module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.Pipelines = map[string]*Pipeline{
				"deploy": tt.pipeline,
			}

			parser := NewConfigParser(tmpDir)
			err := parser.ValidateConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if !strings.Contains(err.Error(), tt.errorString) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorString, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestProcessModule tests module processing
func TestProcessModule(t *testing.T) {
	tmpDir := t.TempDir()

	module := &Module{
		Path: "relative/path",
		Instances: map[string]*Instance{
			"instance1": {
				Variables: map[string]interface{}{},
			},
		},
	}

	parser := NewConfigParser(tmpDir)
	err := parser.processModule("test-module", module)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Path should be converted to absolute
	expectedPath := filepath.Join(tmpDir, "relative/path")
	if module.Path != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, module.Path)
	}

	// Module name should be set
	if module.Name != "test-module" {
		t.Errorf("Expected name 'test-module', got '%s'", module.Name)
	}

	// Instance name should be set
	instance := module.Instances["instance1"]
	if instance.Name != "instance1" {
		t.Errorf("Expected instance name 'instance1', got '%s'", instance.Name)
	}
}

// TestProcessPipeline tests pipeline processing
func TestProcessPipeline(t *testing.T) {
	pipeline := &Pipeline{
		Settings: &PipelineSettings{},
	}

	parser := NewConfigParser("/tmp")
	err := parser.processPipeline("test-pipeline", pipeline)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Pipeline name should be set
	if pipeline.Name != "test-pipeline" {
		t.Errorf("Expected name 'test-pipeline', got '%s'", pipeline.Name)
	}

	// Default operation should be set
	if pipeline.DefaultOperation != "plan" {
		t.Errorf("Expected default operation 'plan', got '%s'", pipeline.DefaultOperation)
	}

	// Default settings should be applied
	if pipeline.Settings.ParallelLimit != 1 {
		t.Errorf("Expected parallel limit 1, got %d", pipeline.Settings.ParallelLimit)
	}

	if pipeline.Settings.Timeout != 30*time.Minute {
		t.Errorf("Expected timeout 30m, got %v", pipeline.Settings.Timeout)
	}
}

// TestExpandGroup tests group expansion
func TestExpandGroup(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"vpc": {Name: "vpc"},
			"app": {Name: "app"},
		},
		Groups: map[string][]string{
			"infrastructure": {"vpc", "app"},
			"all":            {"infrastructure"},
		},
	}

	parser := NewConfigParser("/tmp")

	// Test direct group expansion
	expanded, err := parser.ExpandGroup(config, "infrastructure")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(expanded) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(expanded))
	}

	// Test nested group expansion
	expanded, err = parser.ExpandGroup(config, "all")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(expanded) != 2 {
		t.Errorf("Expected 2 modules from nested expansion, got %d", len(expanded))
	}

	// Test non-existent group
	_, err = parser.ExpandGroup(config, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent group")
	}
}

// TestGetModuleList tests getting module list
func TestGetModuleList(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"vpc": {
				Name: "vpc",
				Instances: map[string]*Instance{
					"vpc-dev":  {Name: "vpc-dev"},
					"vpc-prod": {Name: "vpc-prod"},
				},
			},
			"app": {
				Name: "app",
			},
		},
	}

	parser := NewConfigParser("/tmp")
	modules := parser.GetModuleList(config)

	// Should have 3 entries: vpc.vpc-dev, vpc.vpc-prod, app
	if len(modules) != 3 {
		t.Errorf("Expected 3 modules, got %d", len(modules))
	}
}

// TestValidateUniqueInstanceNames tests duplicate instance name detection
func TestValidateUniqueInstanceNames(t *testing.T) {
	tmpDir := t.TempDir()
	moduleAPath := filepath.Join(tmpDir, "module-a")
	moduleBPath := filepath.Join(tmpDir, "module-b")
	if err := os.MkdirAll(moduleAPath, 0755); err != nil {
		t.Fatalf("Failed to create module-a directory: %v", err)
	}
	if err := os.MkdirAll(moduleBPath, 0755); err != nil {
		t.Fatalf("Failed to create module-b directory: %v", err)
	}

	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: moduleAPath,
				Instances: map[string]*Instance{
					"shared-instance": {Name: "shared-instance"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: moduleBPath,
				Instances: map[string]*Instance{
					"shared-instance": {Name: "shared-instance"},
				},
			},
		},
	}

	parser := NewConfigParser(tmpDir)
	err := parser.ValidateConfig(config)

	// Should not error, but should have warnings
	// The validation doesn't fail on duplicate names, just warns
	if err != nil {
		// Check if it's just warnings (error message but non-fatal)
		if !strings.Contains(err.Error(), "WARNINGS") && !strings.Contains(err.Error(), "Duplicated") {
			t.Errorf("Unexpected error type: %v", err)
		}
	}
}

// TestSaveConfig tests saving configuration to file
func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Version: "1.0",
		Modules: map[string]*Module{
			"vpc": {
				Name:        "vpc",
				Path:        "/path/to/vpc",
				Description: "VPC module",
			},
		},
	}

	parser := NewConfigParser(tmpDir)
	filename := filepath.Join(tmpDir, "output.yaml")
	err := parser.SaveConfig(config, filename)

	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify file can be loaded back
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	if !strings.Contains(string(data), "version:") {
		t.Error("Saved config doesn't contain version")
	}
}

// TestValidationResult tests ValidationResult functionality
func TestValidationResult(t *testing.T) {
	result := &ValidationResult{}

	if result.HasErrors() {
		t.Error("Expected no errors initially")
	}

	result.AddError("module", "vpc", "path", "path is required")

	if !result.HasErrors() {
		t.Error("Expected errors after adding error")
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	result.AddWarning("backend", "vpc", "type", "unknown backend type")

	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	}

	// Test format report
	report := result.FormatReport()
	if !strings.Contains(report, "CONFIGURATION VALIDATION ERRORS") {
		t.Error("Report should contain error header")
	}
	if !strings.Contains(report, "WARNINGS") {
		t.Error("Report should contain warnings header")
	}
}
