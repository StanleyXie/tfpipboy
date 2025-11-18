// Package terraform provides utilities for interacting with Terraform state,
// configuration, and backend information.
package terraform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// BackendConfig represents Terraform backend configuration
type BackendConfig struct {
	Type   string
	Config map[string]interface{}
}

// GetBackend detects the Terraform backend configuration
func GetBackend(path string) (*BackendConfig, error) {
	if !isTerraformDirectory(path) {
		return nil, nil
	}

	// Try reading from .terraform/terraform.tfstate first (more reliable)
	backend, err := getBackendFromTfState(path)
	if err == nil && backend != nil {
		return backend, nil
	}

	// Fallback to parsing .tf files
	return getBackendFromTfFiles(path)
}

// getBackendFromTfState reads backend config from .terraform/terraform.tfstate
func getBackendFromTfState(path string) (*BackendConfig, error) {
	stateFile := filepath.Join(path, ".terraform", "terraform.tfstate")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state struct {
		Backend struct {
			Type   string                 `json:"type"`
			Config map[string]interface{} `json:"config"`
		} `json:"backend"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	if state.Backend.Type == "" {
		return &BackendConfig{
			Type:   "local",
			Config: make(map[string]interface{}),
		}, nil
	}

	return &BackendConfig{
		Type:   state.Backend.Type,
		Config: state.Backend.Config,
	}, nil
}

// getBackendFromTfFiles parses backend configuration from .tf files
func getBackendFromTfFiles(path string) (*BackendConfig, error) {
	files, err := filepath.Glob(filepath.Join(path, "*.tf"))
	if err != nil {
		return nil, err
	}

	backendRegex := regexp.MustCompile(`(?s)backend\s+"([^"]+)"\s*\{([^}]*)\}`)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		content := string(data)
		matches := backendRegex.FindStringSubmatch(content)

		if len(matches) >= 2 {
			backendType := matches[1]
			configBlock := ""
			if len(matches) >= 3 {
				configBlock = matches[2]
			}

			config := parseBackendConfig(configBlock)

			return &BackendConfig{
				Type:   backendType,
				Config: config,
			}, nil
		}
	}

	// No backend block found, assume local
	return &BackendConfig{
		Type:   "local",
		Config: make(map[string]interface{}),
	}, nil
}

// parseBackendConfig parses backend configuration block
func parseBackendConfig(configBlock string) map[string]interface{} {
	config := make(map[string]interface{})

	// Simple key = "value" parser
	kvRegex := regexp.MustCompile(`(\w+)\s*=\s*"([^"]*)"`)
	matches := kvRegex.FindAllStringSubmatch(configBlock, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			key := match[1]
			value := match[2]
			config[key] = value
		}
	}

	return config
}

// GetBackendSummary returns a human-readable backend summary
func GetBackendSummary(backend *BackendConfig) string {
	if backend == nil {
		return "Unknown"
	}

	switch backend.Type {
	case "local":
		return "Local"

	case "s3":
		bucket := getConfigValue(backend.Config, "bucket")
		key := getConfigValue(backend.Config, "key")
		if bucket != "" && key != "" {
			return "S3: " + bucket + "/" + key
		}
		return "S3"

	case "azurerm":
		storageAccount := getConfigValue(backend.Config, "storage_account_name")
		container := getConfigValue(backend.Config, "container_name")
		if storageAccount != "" && container != "" {
			return "Azure: " + storageAccount + "/" + container
		}
		return "Azure Storage"

	case "gcs":
		bucket := getConfigValue(backend.Config, "bucket")
		if bucket != "" {
			return "GCS: " + bucket
		}
		return "Google Cloud Storage"

	case "remote":
		org := getConfigValue(backend.Config, "organization")
		if org != "" {
			return "Terraform Cloud: " + org
		}
		return "Terraform Cloud"

	case "consul":
		path := getConfigValue(backend.Config, "path")
		if path != "" {
			return "Consul: " + path
		}
		return "Consul"

	case "etcd", "etcdv3":
		return "etcd"

	case "http":
		address := getConfigValue(backend.Config, "address")
		if address != "" {
			return "HTTP: " + address
		}
		return "HTTP"

	default:
		// Capitalize first letter of backend type
		if len(backend.Type) == 0 {
			return backend.Type
		}
		return strings.ToUpper(backend.Type[:1]) + backend.Type[1:]
	}
}

// getConfigValue safely retrieves a string value from config map
func getConfigValue(config map[string]interface{}, key string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
