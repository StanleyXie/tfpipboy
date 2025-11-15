# Variable Configuration Enhancement - Technical Summary

## Overview

Enhanced the `var-config` section in tfpipboy to support multiple variable files and improved variable handling, while maintaining backward compatibility with existing configurations.

## Changes Made

### 1. Updated Data Structures

#### `types.go` - VariableConfig struct

**Before:**
```go
type VariableConfig struct {
	File string                 `yaml:"file"`
	JSON map[string]interface{} `yaml:"json"`
}
```

**After:**
```go
type VariableConfig struct {
	// Single file (backward compatible)
	File string `yaml:"file"`
	// Multiple files for -var-file arguments
	Files []string `yaml:"files"`
	// Individual variables as key-value pairs for -var arguments (backward compatible)
	JSON map[string]interface{} `yaml:"json"`
	// List of individual variables for -var arguments
	Vars map[string]interface{} `yaml:"vars"`
}
```

#### `types.go` - Workspace struct

**Added field:**
```go
type Workspace struct {
	// ... existing fields ...
	VarFiles []string `yaml:"var_files"` // List of variable file paths to use with -var-file
	// ... existing fields ...
}
```

### 2. Updated Variable Processing

#### `workspace.go` - setupVariables()

**Key Changes:**
- Initialize `workspace.VarFiles = []string{}`
- Process single `file` field (backward compatible)
- Process multiple `files` field (new feature)
- Process `json` field (backward compatible)
- Process `vars` field (new feature)
- Updated `copyVariablesFile()` to accept an index and return the destination path

**Logic:**
```go
// Single file (backward compatible)
if varConfig.File != "" {
	copiedFile, err := wm.copyVariablesFile(varFilePath, workspace.Path, 0)
	workspace.VarFiles = append(workspace.VarFiles, copiedFile)
}

// Multiple files (new)
for idx, file := range varConfig.Files {
	copiedFile, err := wm.copyVariablesFile(varFilePath, workspace.Path, idx+1)
	workspace.VarFiles = append(workspace.VarFiles, copiedFile)
}

// JSON variables (backward compatible)
for key, value := range varConfig.JSON {
	workspace.Variables[key] = fmt.Sprintf("%v", value)
}

// Vars variables (new)
for key, value := range varConfig.Vars {
	workspace.Variables[key] = fmt.Sprintf("%v", value)
}
```

#### `workspace.go` - copyVariablesFile()

**Signature Change:**
```go
// Before
func (wm *DefaultWorkspaceManager) copyVariablesFile(srcFile, destDir string) error

// After
func (wm *DefaultWorkspaceManager) copyVariablesFile(srcFile, destDir string, idx int) (string, error)
```

**File Naming Logic:**
- `idx == 0`: First file uses standard naming (`terraform.tfvars`, `terraform.tfvars.json`)
- `idx > 0`: Additional files use indexed naming (`vars-1-filename.tfvars`, `vars-2-filename.tfvars`)

### 3. Updated Terraform Command Building

#### `executor.go` - addVariableArgs()

**Before:**
```go
// Add tfvars file if it exists
tfvarsFile := filepath.Join(workspace.Path, "terraform.tfvars")
if _, err := os.Stat(tfvarsFile); err == nil {
	*args = append(*args, fmt.Sprintf("-var-file=%s", tfvarsFile))
}
```

**After:**
```go
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
```

### 4. Updated Configuration Processing

#### `config.go` - processVariableConfig()

**Before:**
```go
func (p *ConfigParser) processVariableConfig(varConfig *VariableConfig) error {
	// Process variable file path
	if varConfig.File != "" && !filepath.IsAbs(varConfig.File) {
		varConfig.File = filepath.Join(p.basePath, varConfig.File)
	}
	return nil
}
```

**After:**
```go
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
```

## Backward Compatibility

### Guaranteed Compatibility

All existing configurations continue to work without modification:

**Old Config:**
```yaml
var-config:
  file: "variables/seed.tfvars"
  json:
    environment: "production"
```

**Behavior:** Unchanged - works exactly as before

### New Features

**Multiple Files:**
```yaml
var-config:
  files:
    - "variables/common.tfvars"
    - "variables/seed.tfvars"
    - "variables/overrides.tfvars"
```

**Multiple Variables (using `vars`):**
```yaml
var-config:
  vars:
    environment: "production"
    location: "germanywestcentral"
```

**Combined:**
```yaml
var-config:
  files:
    - "variables/common.tfvars"
    - "variables/seed.tfvars"
  vars:
    environment: "production"
    override_flag: "true"
```

## Terraform Command Output

### Single File (Backward Compatible)
```bash
terraform plan \
  -var-file=/workspace/terraform.tfvars \
  -var=environment=production
```

### Multiple Files (New)
```bash
terraform plan \
  -var-file=/workspace/terraform.tfvars \
  -var-file=/workspace/vars-1-seed.tfvars \
  -var-file=/workspace/vars-2-overrides.tfvars \
  -var=environment=production \
  -var=location=germanywestcentral
```

## Variable Precedence

Variables are applied in Terraform's standard precedence order:

1. **First:** Variable files in order (earlier files first)
   - `file` field → `terraform.tfvars`
   - `files[0]` → `vars-1-*.tfvars`
   - `files[1]` → `vars-2-*.tfvars`
   - etc.

2. **Last (highest precedence):** Individual `-var` flags
   - `json` values
   - `vars` values
   - Inline `variables` field

This matches Terraform's behavior where:
- Later `-var-file` arguments override earlier ones
- `-var` flags override all `-var-file` values

## Testing Recommendations

### Test Case 1: Backward Compatibility
```yaml
var-config:
  file: "test.tfvars"
  json:
    test: "value"
```
**Expected:** Works identically to v0.1.0

### Test Case 2: Multiple Files
```yaml
var-config:
  files:
    - "file1.tfvars"
    - "file2.tfvars"
    - "file3.tfvars"
```
**Expected:** All files copied and added as `-var-file` arguments

### Test Case 3: Mixed Configuration
```yaml
var-config:
  file: "base.tfvars"
  files:
    - "override1.tfvars"
    - "override2.tfvars"
  json:
    var1: "value1"
  vars:
    var2: "value2"
```
**Expected:** All files and variables correctly processed

### Test Case 4: Instance Override
```yaml
modules:
  seed:
    var-config:
      file: "common.tfvars"
    instances:
      seed-dev:
        var-config:
          files:
            - "dev.tfvars"
```
**Expected:** Instance var-config replaces module var-config

## Files Modified

1. `pkg/orchestrator/types.go` - Updated VariableConfig and Workspace structs
2. `pkg/orchestrator/workspace.go` - Updated variable processing and file copying
3. `pkg/orchestrator/executor.go` - Updated Terraform command building
4. `pkg/orchestrator/config.go` - Updated path resolution for multiple files
5. `design/VAR-CONFIG-EXAMPLES.md` - Comprehensive usage examples (NEW)
6. `design/VAR-CONFIG-ENHANCEMENT.md` - Technical summary (NEW)

## Benefits

1. **Flexibility:** Support for multiple variable files enables better organization
2. **DRY Principle:** Share common variables across modules
3. **Backward Compatible:** Existing configurations work without changes
4. **Terraform Native:** Uses standard `-var-file` and `-var` arguments
5. **Clear Precedence:** Follows Terraform's variable precedence rules
6. **Instance Overrides:** Easy to override variables per instance

## Migration Path

### No Migration Required
Existing configurations continue to work as-is.

### Optional Enhancement
Users can incrementally adopt new features:

**Step 1:** Split single file into multiple files
```yaml
# Before
var-config:
  file: "all-vars.tfvars"

# After
var-config:
  files:
    - "common.tfvars"
    - "environment.tfvars"
    - "module-specific.tfvars"
```

**Step 2:** Use `vars` for runtime overrides
```yaml
var-config:
  files:
    - "common.tfvars"
  vars:
    environment: "production"  # Runtime override
```

## Documentation

See `design/VAR-CONFIG-EXAMPLES.md` for comprehensive usage examples including:
- 7 common configuration patterns
- Variable precedence explanation
- Path resolution rules
- 3 real-world use cases
- Migration guide
- Best practices
