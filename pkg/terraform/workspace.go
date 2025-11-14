package terraform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetWorkspace detects the current Terraform workspace
func GetWorkspace(path string) (string, error) {
	// Check if terraform is available
	if !isCommandAvailable("terraform") {
		return "", nil
	}

	// Check if we're in a Terraform directory
	if !isTerraformDirectory(path) {
		return "", nil
	}

	// Try to get workspace from terraform command
	cmd := exec.Command("terraform", "workspace", "show")
	cmd.Dir = path
	output, err := cmd.Output()

	if err != nil {
		// If command fails, try reading from .terraform/environment file
		return getWorkspaceFromFile(path)
	}

	workspace := strings.TrimSpace(string(output))
	return workspace, nil
}

// getWorkspaceFromFile reads workspace from .terraform/environment file
func getWorkspaceFromFile(path string) (string, error) {
	envFile := filepath.Join(path, ".terraform", "environment")

	data, err := os.ReadFile(envFile)
	if err != nil {
		// If file doesn't exist, assume default workspace
		if os.IsNotExist(err) {
			return "default", nil
		}
		return "", err
	}

	workspace := strings.TrimSpace(string(data))
	if workspace == "" {
		workspace = "default"
	}

	return workspace, nil
}

// ListWorkspaces returns all available workspaces
func ListWorkspaces(path string) ([]string, error) {
	if !isCommandAvailable("terraform") {
		return nil, nil
	}

	if !isTerraformDirectory(path) {
		return nil, nil
	}

	cmd := exec.Command("terraform", "workspace", "list")
	cmd.Dir = path
	output, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	var workspaces []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove the * indicator for current workspace
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)

		if line != "" {
			workspaces = append(workspaces, line)
		}
	}

	return workspaces, nil
}

// isTerraformDirectory checks if the path contains Terraform files
func isTerraformDirectory(path string) bool {
	// Check for .tf files
	matches, err := filepath.Glob(filepath.Join(path, "*.tf"))
	if err != nil {
		return false
	}

	if len(matches) > 0 {
		return true
	}

	// Check for .terraform directory
	tfDir := filepath.Join(path, ".terraform")
	info, err := os.Stat(tfDir)
	if err == nil && info.IsDir() {
		return true
	}

	return false
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
