package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ModuleType represents the type of a Terraform module
type ModuleType string

const (
	// ModuleTypeRoot represents a root module that can be directly deployed
	ModuleTypeRoot ModuleType = "root"
	// ModuleTypeSource represents a source module that must be referenced
	ModuleTypeSource ModuleType = "source"
)

// DiscoveredModule represents a discovered Terraform module with its metadata
type DiscoveredModule struct {
	Name         string              `yaml:"name"`
	Path         string              `yaml:"path"`
	RelativePath string              `yaml:"relative_path"`
	Type         ModuleType          `yaml:"type"`
	Description  string              `yaml:"description,omitempty"`
	Variables    []VariableMetadata  `yaml:"variables,omitempty"`
	Outputs      []OutputMetadata    `yaml:"outputs,omitempty"`
	Dependencies []string            `yaml:"dependencies,omitempty"` // Module sources referenced
	Providers    []string            `yaml:"providers,omitempty"`
	Resources    []ResourceMetadata  `yaml:"resources,omitempty"`
}

// VariableMetadata represents metadata about a Terraform variable
type VariableMetadata struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type,omitempty"`
	Description string `yaml:"description,omitempty"`
	Default     string `yaml:"default,omitempty"`
	Required    bool   `yaml:"required"`
}

// OutputMetadata represents metadata about a Terraform output
type OutputMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Sensitive   bool   `yaml:"sensitive,omitempty"`
}

// ResourceMetadata represents basic metadata about a resource
type ResourceMetadata struct {
	Type string `yaml:"type"`
	Name string `yaml:"name"`
}

// TfvarsFile represents a discovered tfvars file
type TfvarsFile struct {
	Path         string `yaml:"path"`
	RelativePath string `yaml:"relative_path"`
	Name         string `yaml:"name"`
}

// DiscoveryResult represents the complete result of module discovery
type DiscoveryResult struct {
	RootPath      string              `yaml:"root_path"`
	Modules       []*DiscoveredModule `yaml:"modules"`
	RootModules   []*DiscoveredModule `yaml:"root_modules"`
	SourceModules []*DiscoveredModule `yaml:"source_modules"`
	TfvarsFiles   []*TfvarsFile       `yaml:"tfvars_files"`
	TreeStructure *TreeNode           `yaml:"tree_structure,omitempty"`
	Summary       DiscoverySummary    `yaml:"summary"`
}

// TreeNode represents a node in the directory tree structure
type TreeNode struct {
	Name         string      `yaml:"name"`
	Path         string      `yaml:"path"`
	RelativePath string      `yaml:"relative_path"`
	IsModule     bool        `yaml:"is_module"`
	ModuleType   ModuleType  `yaml:"module_type,omitempty"`
	Children     []*TreeNode `yaml:"children,omitempty"`
	Depth        int         `yaml:"depth"`
}

// DiscoverySummary provides a summary of the discovery results
type DiscoverySummary struct {
	TotalModules       int `yaml:"total_modules"`
	RootModules        int `yaml:"root_modules"`
	SourceModules      int `yaml:"source_modules"`
	TfvarsFilesFound   int `yaml:"tfvars_files_found"`
}

// DiscoverModules scans the specified path recursively to discover all Terraform modules
func DiscoverModules(rootPath string) (*DiscoveryResult, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	result := &DiscoveryResult{
		RootPath:      absPath,
		Modules:       make([]*DiscoveredModule, 0),
		RootModules:   make([]*DiscoveredModule, 0),
		SourceModules: make([]*DiscoveredModule, 0),
		TfvarsFiles:   make([]*TfvarsFile, 0),
	}

	// Scan for modules
	if err := scanDirectory(absPath, absPath, result); err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Discover tfvars files
	if err := discoverTfvarsFiles(absPath, absPath, result); err != nil {
		return nil, fmt.Errorf("failed to discover tfvars files: %w", err)
	}

	// Build tree structure
	result.TreeStructure = buildTreeStructure(absPath, result)

	// Sort modules by hierarchical order (tree structure)
	sortModulesByHierarchy(result)

	// Update summary
	result.Summary.TotalModules = len(result.Modules)
	result.Summary.RootModules = len(result.RootModules)
	result.Summary.SourceModules = len(result.SourceModules)
	result.Summary.TfvarsFilesFound = len(result.TfvarsFiles)

	return result, nil
}

// scanDirectory recursively scans a directory for Terraform modules
func scanDirectory(rootPath, currentPath string, result *DiscoveryResult) error {
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return err
	}

	// Check if current directory is a Terraform module
	if isTerraformDirectory(currentPath) {
		module, err := extractModuleMetadata(rootPath, currentPath)
		if err != nil {
			// Log error but continue scanning
			fmt.Fprintf(os.Stderr, "Warning: failed to extract metadata from %s: %v\n", currentPath, err)
		} else if module != nil {
			result.Modules = append(result.Modules, module)

			// Categorize by type
			if module.Type == ModuleTypeRoot {
				result.RootModules = append(result.RootModules, module)
			} else {
				result.SourceModules = append(result.SourceModules, module)
			}
		}
	}

	// Recursively scan subdirectories
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden directories, .terraform, and common non-module directories
		if strings.HasPrefix(name, ".") ||
		   name == "node_modules" ||
		   name == "vendor" ||
		   name == ".git" {
			continue
		}

		subPath := filepath.Join(currentPath, name)
		if err := scanDirectory(rootPath, subPath, result); err != nil {
			// Log error but continue scanning
			fmt.Fprintf(os.Stderr, "Warning: failed to scan %s: %v\n", subPath, err)
		}
	}

	return nil
}

// extractModuleMetadata extracts metadata from a Terraform module
func extractModuleMetadata(rootPath, modulePath string) (*DiscoveredModule, error) {
	relPath, err := filepath.Rel(rootPath, modulePath)
	if err != nil {
		relPath = modulePath
	}

	module := &DiscoveredModule{
		Name:         filepath.Base(modulePath),
		Path:         modulePath,
		RelativePath: relPath,
		Type:         determineModuleType(modulePath),
		Variables:    make([]VariableMetadata, 0),
		Outputs:      make([]OutputMetadata, 0),
		Dependencies: make([]string, 0),
		Providers:    make([]string, 0),
		Resources:    make([]ResourceMetadata, 0),
	}

	// Parse .tf files to extract metadata
	tfFiles, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return module, nil
	}

	for _, tfFile := range tfFiles {
		content, err := os.ReadFile(tfFile)
		if err != nil {
			continue
		}

		fileContent := string(content)

		// Extract variables
		vars := parseVariables(fileContent)
		module.Variables = append(module.Variables, vars...)

		// Extract outputs
		outputs := parseOutputs(fileContent)
		module.Outputs = append(module.Outputs, outputs...)

		// Extract module dependencies
		deps := parseModuleDependencies(fileContent)
		module.Dependencies = append(module.Dependencies, deps...)

		// Extract providers
		providers := parseProviders(fileContent)
		module.Providers = append(module.Providers, providers...)

		// Extract resources
		resources := parseResources(fileContent)
		module.Resources = append(module.Resources, resources...)
	}

	// Remove duplicate dependencies
	module.Dependencies = removeDuplicates(module.Dependencies)
	module.Providers = removeDuplicates(module.Providers)

	return module, nil
}

// determineModuleType determines whether a module is a root module or source module
func determineModuleType(modulePath string) ModuleType {
	// Check if the module is in a typical source module location
	// Common source module directories: modules/, terraform-modules/, etc.
	pathParts := strings.Split(filepath.ToSlash(modulePath), "/")
	for _, part := range pathParts {
		// If any part of the path is "modules", "terraform-modules", or similar,
		// it's likely a source module
		lowerPart := strings.ToLower(part)
		if lowerPart == "modules" ||
		   lowerPart == "terraform-modules" ||
		   lowerPart == "tf-modules" ||
		   strings.HasPrefix(lowerPart, "module-") {
			return ModuleTypeSource
		}
	}

	// Check if parent directory has .tf files (nested module)
	parent := filepath.Dir(modulePath)
	if parent != modulePath {
		// Check all ancestor directories for .tf files
		current := parent
		for current != filepath.Dir(current) {
			matches, err := filepath.Glob(filepath.Join(current, "*.tf"))
			if err == nil && len(matches) > 0 {
				// Parent has .tf files, this is a source module
				return ModuleTypeSource
			}
			current = filepath.Dir(current)
		}
	}

	// Use the existing isRootModule function as final check
	if isRootModule(modulePath) {
		return ModuleTypeRoot
	}

	return ModuleTypeSource
}

// parseVariables extracts variable definitions from Terraform content
func parseVariables(content string) []VariableMetadata {
	vars := make([]VariableMetadata, 0)
	lines := strings.Split(content, "\n")

	var currentVar *VariableMetadata
	inVarBlock := false
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Count braces to track nested blocks
		braceCount += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

		// Check for variable block start
		if strings.HasPrefix(trimmed, "variable") && strings.Contains(trimmed, "{") {
			inVarBlock = true
			braceCount = strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

			// Extract variable name
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := strings.Trim(parts[1], `"`)
				currentVar = &VariableMetadata{
					Name:     name,
					Required: true, // Default to required
				}
			}
			continue
		}

		if inVarBlock && currentVar != nil {
			// Look for type
			if strings.HasPrefix(trimmed, "type") && strings.Contains(trimmed, "=") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					typeVal := strings.TrimSpace(parts[1])
					currentVar.Type = strings.Trim(typeVal, `"`)
				}
			}

			// Look for description
			if strings.HasPrefix(trimmed, "description") && strings.Contains(trimmed, "=") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					desc := strings.TrimSpace(parts[1])
					currentVar.Description = strings.Trim(desc, `"`)
				}
			}

			// Look for default
			if strings.HasPrefix(trimmed, "default") && strings.Contains(trimmed, "=") {
				currentVar.Required = false
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					defaultVal := strings.TrimSpace(parts[1])
					currentVar.Default = strings.Trim(defaultVal, `"`)
				}
			}

			// Check if we've closed the variable block
			if braceCount == 0 && strings.HasPrefix(trimmed, "}") {
				vars = append(vars, *currentVar)
				currentVar = nil
				inVarBlock = false
			}
		}
	}

	return vars
}

// parseOutputs extracts output definitions from Terraform content
func parseOutputs(content string) []OutputMetadata {
	outputs := make([]OutputMetadata, 0)
	lines := strings.Split(content, "\n")

	var currentOutput *OutputMetadata
	inOutputBlock := false
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Count braces
		braceCount += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

		// Check for output block start
		if strings.HasPrefix(trimmed, "output") && strings.Contains(trimmed, "{") {
			inOutputBlock = true
			braceCount = strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

			// Extract output name
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := strings.Trim(parts[1], `"`)
				currentOutput = &OutputMetadata{
					Name: name,
				}
			}
			continue
		}

		if inOutputBlock && currentOutput != nil {
			// Look for description
			if strings.HasPrefix(trimmed, "description") && strings.Contains(trimmed, "=") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					desc := strings.TrimSpace(parts[1])
					currentOutput.Description = strings.Trim(desc, `"`)
				}
			}

			// Look for sensitive
			if strings.HasPrefix(trimmed, "sensitive") && strings.Contains(trimmed, "=") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					sensitive := strings.TrimSpace(parts[1])
					currentOutput.Sensitive = sensitive == "true"
				}
			}

			// Check if we've closed the output block
			if braceCount == 0 && strings.HasPrefix(trimmed, "}") {
				outputs = append(outputs, *currentOutput)
				currentOutput = nil
				inOutputBlock = false
			}
		}
	}

	return outputs
}

// parseModuleDependencies extracts module source references
func parseModuleDependencies(content string) []string {
	deps := make([]string, 0)
	moduleSources := parseModuleSources(content)

	for _, ms := range moduleSources {
		if ms.Source != "" {
			deps = append(deps, ms.Source)
		}
	}

	return deps
}

// parseProviders extracts provider configurations
func parseProviders(content string) []string {
	providers := make([]string, 0)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for provider blocks
		if strings.HasPrefix(trimmed, "provider") && strings.Contains(trimmed, "{") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				provider := strings.Trim(parts[1], `"`)
				providers = append(providers, provider)
			}
		}

		// Also look for required_providers in terraform block
		if strings.HasPrefix(trimmed, "required_providers") {
			// This is a simplified extraction - a full HCL parser would be better
			continue
		}
	}

	return providers
}

// parseResources extracts resource declarations
func parseResources(content string) []ResourceMetadata {
	resources := make([]ResourceMetadata, 0)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for resource blocks: resource "type" "name" {
		if strings.HasPrefix(trimmed, "resource") && strings.Contains(trimmed, "{") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				resourceType := strings.Trim(parts[1], `"`)
				resourceName := strings.Trim(parts[2], `"{`)
				resources = append(resources, ResourceMetadata{
					Type: resourceType,
					Name: resourceName,
				})
			}
		}
	}

	return resources
}

// discoverTfvarsFiles recursively finds all .tfvars and .tfvars.json files
func discoverTfvarsFiles(rootPath, currentPath string, result *DiscoveryResult) error {
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(currentPath, entry.Name())

		if entry.IsDir() {
			// Skip hidden directories and .terraform
			if strings.HasPrefix(entry.Name(), ".") || entry.Name() == ".terraform" {
				continue
			}

			// Recursively search subdirectories
			if err := discoverTfvarsFiles(rootPath, fullPath, result); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to scan %s: %v\n", fullPath, err)
			}
		} else {
			// Check if it's a tfvars file
			name := entry.Name()
			if strings.HasSuffix(name, ".tfvars") || strings.HasSuffix(name, ".tfvars.json") {
				relPath, _ := filepath.Rel(rootPath, fullPath)
				result.TfvarsFiles = append(result.TfvarsFiles, &TfvarsFile{
					Path:         fullPath,
					RelativePath: relPath,
					Name:         name,
				})
			}
		}
	}

	return nil
}

// GenerateYAMLConfig generates a tfpipboy YAML configuration from discovery results
// The configuration is ordered hierarchically based on module path structure
func (r *DiscoveryResult) GenerateYAMLConfig() (string, error) {
	var sb strings.Builder

	// Add header comment with tree structure
	sb.WriteString("# Terraform Module Configuration\n")
	sb.WriteString("# Auto-generated by tfpipboy module discovery\n")
	sb.WriteString("#\n")
	if r.TreeStructure != nil {
		treeLines := strings.Split(r.RenderTree(), "\n")
		for _, line := range treeLines {
			if line != "" {
				sb.WriteString("# " + line + "\n")
			}
		}
	}
	sb.WriteString("\n")

	// Build ordered YAML content
	sb.WriteString("version: \"1.0\"\n\n")

	// Add modules section with hierarchical ordering
	sb.WriteString("modules:\n")

	// Modules are already sorted by hierarchy from sortModulesByHierarchy
	for _, module := range r.Modules {
		// Add a blank line and comment showing the module path structure
		indent := strings.Repeat("  ", 1) // Base indent for modules

		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%s# Path: %s\n", indent, module.RelativePath))

		// Module name (key)
		sb.WriteString(fmt.Sprintf("%s%s:\n", indent, module.Name))

		// Module properties with proper indentation
		propIndent := strings.Repeat("  ", 2)

		sb.WriteString(fmt.Sprintf("%spath: %s\n", propIndent, module.RelativePath))
		sb.WriteString(fmt.Sprintf("%sdescription: Auto-discovered %s module\n", propIndent, module.Type))

		// Add depends_on if module has dependencies (for local references)
		if len(module.Dependencies) > 0 {
			localDeps := make([]string, 0)
			for _, dep := range module.Dependencies {
				// Only include local module references (starting with ./ or ../)
				if strings.HasPrefix(dep, "./") || strings.HasPrefix(dep, "../") {
					localDeps = append(localDeps, dep)
				}
			}
			if len(localDeps) > 0 {
				sb.WriteString(fmt.Sprintf("%sdepends_on:\n", propIndent))
				for _, dep := range localDeps {
					sb.WriteString(fmt.Sprintf("%s  - %s\n", propIndent, dep))
				}
			}
		}

		// Add instances section (empty for user to fill in)
		sb.WriteString(fmt.Sprintf("%sinstances: {}\n", propIndent))

		// Add a comment about required configuration
		var comment string
		if module.Type == ModuleTypeRoot {
			comment = "Root module - can be deployed directly. Add instances below to enable execution."
		} else {
			comment = "Source module - referenced by other modules. Add instances only if deploying independently."
		}
		sb.WriteString(fmt.Sprintf("%s# %s\n", propIndent, comment))
	}

	// Add tfvars reference section
	if len(r.TfvarsFiles) > 0 {
		sb.WriteString("\n# Discovered tfvars files (can be used in instance var-config)\n")
		sb.WriteString("_tfvars_files:\n")
		for _, tfvars := range r.TfvarsFiles {
			sb.WriteString(fmt.Sprintf("  - %s\n", tfvars.RelativePath))
		}
	}

	return sb.String(), nil
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// buildTreeStructure builds a tree representation of the directory structure
func buildTreeStructure(rootPath string, result *DiscoveryResult) *TreeNode {
	// Create a map for quick module lookup
	moduleMap := make(map[string]*DiscoveredModule)
	for _, module := range result.Modules {
		moduleMap[module.Path] = module
	}

	// Build the tree recursively
	rootNode := buildTreeNode(rootPath, rootPath, moduleMap, 0)
	return rootNode
}

// buildTreeNode recursively builds tree nodes
func buildTreeNode(rootPath, currentPath string, moduleMap map[string]*DiscoveredModule, depth int) *TreeNode {
	relPath, _ := filepath.Rel(rootPath, currentPath)
	if relPath == "." {
		relPath = ""
	}

	node := &TreeNode{
		Name:         filepath.Base(currentPath),
		Path:         currentPath,
		RelativePath: relPath,
		Depth:        depth,
		Children:     make([]*TreeNode, 0),
	}

	// Check if this directory is a module
	if module, exists := moduleMap[currentPath]; exists {
		node.IsModule = true
		node.ModuleType = module.Type
	}

	// Read directory contents
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return node
	}

	// Process subdirectories
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden directories, .terraform, and other non-module directories
		if strings.HasPrefix(name, ".") ||
			name == "node_modules" ||
			name == "vendor" {
			continue
		}

		childPath := filepath.Join(currentPath, name)

		// Only include directories that contain modules or have module descendants
		if hasModulesInTree(childPath, moduleMap) {
			childNode := buildTreeNode(rootPath, childPath, moduleMap, depth+1)
			node.Children = append(node.Children, childNode)
		}
	}

	// Sort children by name for consistent ordering
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Name < node.Children[j].Name
	})

	return node
}

// hasModulesInTree checks if a directory or its descendants contain any modules
func hasModulesInTree(path string, moduleMap map[string]*DiscoveredModule) bool {
	// Check if this path is a module
	if _, exists := moduleMap[path]; exists {
		return true
	}

	// Check subdirectories recursively
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, ".") ||
			name == "node_modules" ||
			name == "vendor" {
			continue
		}

		childPath := filepath.Join(path, name)
		if hasModulesInTree(childPath, moduleMap) {
			return true
		}
	}

	return false
}

// sortModulesByHierarchy sorts modules by their path hierarchy (depth-first order)
func sortModulesByHierarchy(result *DiscoveryResult) {
	// Sort all modules
	sort.Slice(result.Modules, func(i, j int) bool {
		return comparePathHierarchy(result.Modules[i].RelativePath, result.Modules[j].RelativePath)
	})

	// Sort root modules
	sort.Slice(result.RootModules, func(i, j int) bool {
		return comparePathHierarchy(result.RootModules[i].RelativePath, result.RootModules[j].RelativePath)
	})

	// Sort source modules
	sort.Slice(result.SourceModules, func(i, j int) bool {
		return comparePathHierarchy(result.SourceModules[i].RelativePath, result.SourceModules[j].RelativePath)
	})
}

// comparePathHierarchy compares two paths for hierarchical ordering
// Returns true if path1 should come before path2
func comparePathHierarchy(path1, path2 string) bool {
	// Split paths into components
	parts1 := strings.Split(filepath.ToSlash(path1), "/")
	parts2 := strings.Split(filepath.ToSlash(path2), "/")

	// Compare component by component
	minLen := len(parts1)
	if len(parts2) < minLen {
		minLen = len(parts2)
	}

	for i := 0; i < minLen; i++ {
		if parts1[i] != parts2[i] {
			return parts1[i] < parts2[i]
		}
	}

	// If all components match, shorter path comes first
	return len(parts1) < len(parts2)
}

// RenderTree renders the tree structure as a string
func (r *DiscoveryResult) RenderTree() string {
	if r.TreeStructure == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Module Tree Structure:\n")
	renderTreeNode(r.TreeStructure, "", true, &sb)
	return sb.String()
}

// renderTreeNode recursively renders a tree node
func renderTreeNode(node *TreeNode, prefix string, isLast bool, sb *strings.Builder) {
	if node.Depth > 0 { // Skip root node in display
		// Determine the tree symbols
		var connector, extension string
		if isLast {
			connector = "└── "
			extension = "    "
		} else {
			connector = "├── "
			extension = "│   "
		}

		// Write the node
		sb.WriteString(prefix)
		sb.WriteString(connector)
		sb.WriteString(node.Name)

		// Add module indicator
		if node.IsModule {
			sb.WriteString(fmt.Sprintf(" [%s module]", node.ModuleType))
		}

		sb.WriteString("\n")

		// Update prefix for children
		prefix = prefix + extension
	}

	// Render children
	for i, child := range node.Children {
		renderTreeNode(child, prefix, i == len(node.Children)-1, sb)
	}
}
