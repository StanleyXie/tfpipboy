package terraform

import (
	"os"
	"strings"
)

// EnvironmentVars represents Terraform-related environment variables
type EnvironmentVars struct {
	// Terraform Core
	TerraformVersion string
	DataDir          string
	WorkspaceDir     string
	LogLevel         string
	LogPath          string

	// CLI Configuration
	CliConfigFile  string
	PluginCacheDir string

	// Provider Configuration
	Providers map[string]string

	// Variable Values
	Variables map[string]string

	// Other
	InAutomation bool
	Other        map[string]string
}

// GetEnvironmentVars extracts all Terraform-related environment variables
func GetEnvironmentVars() *EnvironmentVars {
	env := &EnvironmentVars{
		Providers: make(map[string]string),
		Variables: make(map[string]string),
		Other:     make(map[string]string),
	}

	// Get all environment variables
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		value := pair[1]

		// Terraform core variables
		switch key {
		case "TF_DATA_DIR":
			env.DataDir = value
		case "TF_WORKSPACE":
			env.WorkspaceDir = value
		case "TF_LOG":
			env.LogLevel = value
		case "TF_LOG_PATH":
			env.LogPath = value
		case "TF_CLI_CONFIG_FILE":
			env.CliConfigFile = value
		case "TF_PLUGIN_CACHE_DIR":
			env.PluginCacheDir = value
		case "TF_IN_AUTOMATION":
			env.InAutomation = value == "1" || strings.ToLower(value) == "true"
		}

		// Provider-specific variables
		if strings.HasPrefix(key, "TF_VAR_") {
			varName := strings.TrimPrefix(key, "TF_VAR_")
			env.Variables[varName] = value
		}

		// Provider credentials and config
		if isTerraformRelated(key) {
			env.Other[key] = value
		}
	}

	return env
}

// isTerraformRelated checks if an environment variable is Terraform-related
func isTerraformRelated(key string) bool {
	prefixes := []string{
		"TF_",
		"TERRAFORM_",
		"AWS_",          // AWS provider
		"AZURE_",        // Azure provider
		"ARM_",          // Azure provider (alternative)
		"GOOGLE_",       // GCP provider
		"GCLOUD_",       // GCP provider
		"GCP_",          // GCP provider
		"DIGITALOCEAN_", // DigitalOcean provider
		"LINODE_",       // Linode provider
		"HCLOUD_",       // Hetzner Cloud provider
		"CLOUDFLARE_",   // Cloudflare provider
		"KUBERNETES_",   // Kubernetes provider
		"KUBE_",         // Kubernetes provider
		"VAULT_",        // Vault provider
		"CONSUL_",       // Consul provider
		"NOMAD_",        // Nomad provider
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

// GetTerraformEnvVars returns only TF_* environment variables
func GetTerraformEnvVars() map[string]string {
	vars := make(map[string]string)

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		value := pair[1]

		if strings.HasPrefix(key, "TF_") || strings.HasPrefix(key, "TERRAFORM_") {
			vars[key] = value
		}
	}

	return vars
}

// GetProviderEnvVars returns provider-specific environment variables
func GetProviderEnvVars() map[string]map[string]string {
	providers := map[string]map[string]string{
		"aws":          make(map[string]string),
		"azure":        make(map[string]string),
		"google":       make(map[string]string),
		"kubernetes":   make(map[string]string),
		"digitalocean": make(map[string]string),
		"cloudflare":   make(map[string]string),
		"vault":        make(map[string]string),
	}

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		value := pair[1]

		// AWS
		if strings.HasPrefix(key, "AWS_") {
			providers["aws"][key] = value
		}

		// Azure
		if strings.HasPrefix(key, "AZURE_") || strings.HasPrefix(key, "ARM_") {
			providers["azure"][key] = value
		}

		// Google Cloud
		if strings.HasPrefix(key, "GOOGLE_") || strings.HasPrefix(key, "GCLOUD_") || strings.HasPrefix(key, "GCP_") {
			providers["google"][key] = value
		}

		// Kubernetes
		if strings.HasPrefix(key, "KUBERNETES_") || strings.HasPrefix(key, "KUBE_") {
			providers["kubernetes"][key] = value
		}

		// DigitalOcean
		if strings.HasPrefix(key, "DIGITALOCEAN_") {
			providers["digitalocean"][key] = value
		}

		// Cloudflare
		if strings.HasPrefix(key, "CLOUDFLARE_") {
			providers["cloudflare"][key] = value
		}

		// Vault
		if strings.HasPrefix(key, "VAULT_") {
			providers["vault"][key] = value
		}
	}

	// Remove empty provider maps
	for name, vars := range providers {
		if len(vars) == 0 {
			delete(providers, name)
		}
	}

	return providers
}

// GetSensitiveVars returns a list of environment variable keys that contain sensitive data
func GetSensitiveVars() []string {
	sensitivePatterns := []string{
		"TOKEN",
		"SECRET",
		"KEY",
		"PASSWORD",
		"CREDENTIAL",
		"ACCESS",
		"AUTH",
		"API",
	}

	var sensitive []string

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := pair[0]

		if !isTerraformRelated(key) {
			continue
		}

		keyUpper := strings.ToUpper(key)
		for _, pattern := range sensitivePatterns {
			if strings.Contains(keyUpper, pattern) {
				sensitive = append(sensitive, key)
				break
			}
		}
	}

	return sensitive
}

// MaskSensitiveValue masks sensitive values for display
func MaskSensitiveValue(key, value string) string {
	sensitivePatterns := []string{
		"TOKEN",
		"SECRET",
		"KEY",
		"PASSWORD",
		"CREDENTIAL",
	}

	keyUpper := strings.ToUpper(key)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(keyUpper, pattern) {
			if len(value) <= 4 {
				return "****"
			}
			return value[:4] + "****"
		}
	}

	return value
}

// GetEnvironmentSummary returns a summary of Terraform environment configuration
func GetEnvironmentSummary() map[string]interface{} {
	env := GetEnvironmentVars()
	providerEnv := GetProviderEnvVars()

	summary := make(map[string]interface{})

	// Core Terraform settings
	if env.LogLevel != "" {
		summary["log_level"] = env.LogLevel
	}
	if env.DataDir != "" {
		summary["data_dir"] = env.DataDir
	}
	if env.InAutomation {
		summary["automation"] = true
	}

	// Variable count
	if len(env.Variables) > 0 {
		summary["variables_count"] = len(env.Variables)
	}

	// Provider configuration count
	providerCounts := make(map[string]int)
	for provider, vars := range providerEnv {
		providerCounts[provider] = len(vars)
	}
	if len(providerCounts) > 0 {
		summary["providers"] = providerCounts
	}

	return summary
}
