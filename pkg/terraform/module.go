package terraform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ModuleInfo represents information about a Terraform module
type ModuleInfo struct {
	Name     string
	Source   string
	Version  string
	Path     string
	IsRoot   bool
	Children []string
}

// GetModuleInfo detects information about the current Terraform module
func GetModuleInfo(path string) (*ModuleInfo, error) {
	if !isTerraformDirectory(path) {
		return nil, nil
	}

	info := &ModuleInfo{
		Path:   path,
		IsRoot: isRootModule(path),
	}

	// Get module name from directory
	info.Name = filepath.Base(path)

	// Try to read from module manifest
	manifest, err := readModuleManifest(path)
	if err == nil && manifest != nil {
		info.Children = manifest.Modules
	}

	return info, nil
}

// ModuleManifest represents .terraform/modules/modules.json
type ModuleManifest struct {
	Modules []string `json:"Modules"`
}

// readModuleManifest reads the module manifest file
func readModuleManifest(path string) (*ModuleManifest, error) {
	manifestFile := filepath.Join(path, ".terraform", "modules", "modules.json")

	data, err := os.ReadFile(manifestFile)
	if err != nil {
		return nil, err
	}

	var manifest struct {
		Modules []struct {
			Key    string `json:"Key"`
			Source string `json:"Source"`
			Dir    string `json:"Dir"`
		} `json:"Modules"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	var moduleKeys []string
	for _, mod := range manifest.Modules {
		if mod.Key != "" {
			moduleKeys = append(moduleKeys, mod.Key)
		}
	}

	return &ModuleManifest{
		Modules: moduleKeys,
	}, nil
}

// isRootModule checks if the current directory is a root module
func isRootModule(path string) bool {
	// A root module typically has:
	// 1. .tf files in the current directory
	// 2. No parent directory with .tf files (or we're at the git root)

	// Check if current directory has .tf files
	matches, err := filepath.Glob(filepath.Join(path, "*.tf"))
	if err != nil || len(matches) == 0 {
		return false
	}

	// Check if we're at a git repository root
	gitDir := filepath.Join(path, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return true
	}

	// Check if parent has .tf files
	parent := filepath.Dir(path)
	if parent == path {
		// We're at filesystem root
		return true
	}

	parentMatches, err := filepath.Glob(filepath.Join(parent, "*.tf"))
	if err != nil || len(parentMatches) == 0 {
		// Parent doesn't have .tf files, so we're likely a root module
		return true
	}

	// Parent has .tf files, so we're likely a child module
	return false
}

// FindRootModule walks up the directory tree to find the root module
func FindRootModule(path string) (string, error) {
	current := path

	for {
		// Check if current directory is a root module
		if isRootModule(current) {
			return current, nil
		}

		// Move to parent directory
		parent := filepath.Dir(current)

		// Check if we've reached the filesystem root
		if parent == current {
			// Couldn't find root module, return original path
			return path, nil
		}

		current = parent
	}
}

// GetModulePath returns a descriptive path for the module
func GetModulePath(path string) string {
	rootModule, err := FindRootModule(path)
	if err != nil || rootModule == path {
		return filepath.Base(path)
	}

	// Get relative path from root module
	relPath, err := filepath.Rel(rootModule, path)
	if err != nil {
		return filepath.Base(path)
	}

	if relPath == "." {
		return filepath.Base(rootModule)
	}

	return filepath.Base(rootModule) + "/" + relPath
}

// ListModuleSources returns all module sources defined in .tf files
func ListModuleSources(path string) ([]ModuleSource, error) {
	if !isTerraformDirectory(path) {
		return nil, nil
	}

	files, err := filepath.Glob(filepath.Join(path, "*.tf"))
	if err != nil {
		return nil, err
	}

	var sources []ModuleSource

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		content := string(data)
		moduleSources := parseModuleSources(content)
		sources = append(sources, moduleSources...)
	}

	return sources, nil
}

// ModuleSource represents a module source declaration
type ModuleSource struct {
	Name    string
	Source  string
	Version string
}

// parseModuleSources extracts module blocks from Terraform configuration
func parseModuleSources(content string) []ModuleSource {
	var sources []ModuleSource

	// Simple parser - looks for module blocks
	// This is a basic implementation; a proper HCL parser would be more robust
	lines := strings.Split(content, "\n")
	var currentModule *ModuleSource
	inModuleBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for module block start
		if strings.HasPrefix(line, "module") && strings.Contains(line, "{") {
			inModuleBlock = true
			// Extract module name
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := strings.Trim(parts[1], `"`)
				currentModule = &ModuleSource{Name: name}
			}
			continue
		}

		if inModuleBlock && currentModule != nil {
			// Look for source
			if strings.HasPrefix(line, "source") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					source := strings.TrimSpace(parts[1])
					source = strings.Trim(source, `"`)
					currentModule.Source = source
				}
			}

			// Look for version
			if strings.HasPrefix(line, "version") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					version := strings.TrimSpace(parts[1])
					version = strings.Trim(version, `"`)
					currentModule.Version = version
				}
			}

			// Check for block end
			if strings.HasPrefix(line, "}") {
				if currentModule.Source != "" {
					sources = append(sources, *currentModule)
				}
				currentModule = nil
				inModuleBlock = false
			}
		}
	}

	return sources
}
