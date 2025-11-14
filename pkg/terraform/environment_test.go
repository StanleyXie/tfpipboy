package terraform

import (
	"os"
	"testing"
)

func TestGetEnvironmentVars(t *testing.T) {
	// Set some test environment variables
	os.Setenv("TF_DATA_DIR", "/tmp/terraform")
	os.Setenv("TF_LOG", "DEBUG")
	os.Setenv("TF_VAR_region", "us-west-2")
	os.Setenv("TF_VAR_instance_type", "t2.micro")
	defer func() {
		os.Unsetenv("TF_DATA_DIR")
		os.Unsetenv("TF_LOG")
		os.Unsetenv("TF_VAR_region")
		os.Unsetenv("TF_VAR_instance_type")
	}()

	env := GetEnvironmentVars()

	// Check core variables
	if env.DataDir != "/tmp/terraform" {
		t.Errorf("Expected DataDir '/tmp/terraform', got '%s'", env.DataDir)
	}

	if env.LogLevel != "DEBUG" {
		t.Errorf("Expected LogLevel 'DEBUG', got '%s'", env.LogLevel)
	}

	// Check TF_VAR variables
	if env.Variables["region"] != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got '%s'", env.Variables["region"])
	}

	if env.Variables["instance_type"] != "t2.micro" {
		t.Errorf("Expected instance_type 't2.micro', got '%s'", env.Variables["instance_type"])
	}
}

func TestIsTerraformRelated(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"TF_LOG", true},
		{"TERRAFORM_VERSION", true},
		{"AWS_ACCESS_KEY_ID", true},
		{"AZURE_CLIENT_ID", true},
		{"ARM_SUBSCRIPTION_ID", true},
		{"GOOGLE_CREDENTIALS", true},
		{"KUBERNETES_CONFIG", true},
		{"VAULT_ADDR", true},
		{"RANDOM_VAR", false},
		{"HOME", false},
		{"PATH", false},
	}

	for _, tt := range tests {
		result := isTerraformRelated(tt.key)
		if result != tt.expected {
			t.Errorf("isTerraformRelated(%s) = %v, expected %v", tt.key, result, tt.expected)
		}
	}
}

func TestGetTerraformEnvVars(t *testing.T) {
	// Set test variables
	os.Setenv("TF_LOG", "DEBUG")
	os.Setenv("TF_DATA_DIR", "/tmp/tf")
	os.Setenv("TERRAFORM_VERSION", "1.5.0")
	os.Setenv("AWS_ACCESS_KEY_ID", "test") // Should not be included
	defer func() {
		os.Unsetenv("TF_LOG")
		os.Unsetenv("TF_DATA_DIR")
		os.Unsetenv("TERRAFORM_VERSION")
		os.Unsetenv("AWS_ACCESS_KEY_ID")
	}()

	vars := GetTerraformEnvVars()

	// Check TF_ variables are included
	if vars["TF_LOG"] != "DEBUG" {
		t.Error("Expected TF_LOG to be included")
	}

	if vars["TF_DATA_DIR"] != "/tmp/tf" {
		t.Error("Expected TF_DATA_DIR to be included")
	}

	if vars["TERRAFORM_VERSION"] != "1.5.0" {
		t.Error("Expected TERRAFORM_VERSION to be included")
	}

	// Check non-TF variables are excluded
	if _, exists := vars["AWS_ACCESS_KEY_ID"]; exists {
		t.Error("Expected AWS_ACCESS_KEY_ID to be excluded")
	}
}

func TestGetProviderEnvVars(t *testing.T) {
	// Set provider-specific variables
	os.Setenv("AWS_ACCESS_KEY_ID", "test-aws-key")
	os.Setenv("AZURE_CLIENT_ID", "test-azure-id")
	os.Setenv("ARM_SUBSCRIPTION_ID", "test-arm-sub")
	os.Setenv("GOOGLE_CREDENTIALS", "test-gcp-creds")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AZURE_CLIENT_ID")
		os.Unsetenv("ARM_SUBSCRIPTION_ID")
		os.Unsetenv("GOOGLE_CREDENTIALS")
	}()

	providers := GetProviderEnvVars()

	// Check AWS
	if awsVars, ok := providers["aws"]; !ok {
		t.Error("Expected aws provider variables")
	} else if awsVars["AWS_ACCESS_KEY_ID"] != "test-aws-key" {
		t.Error("Expected AWS_ACCESS_KEY_ID in aws provider vars")
	}

	// Check Azure
	if azureVars, ok := providers["azure"]; !ok {
		t.Error("Expected azure provider variables")
	} else {
		if azureVars["AZURE_CLIENT_ID"] != "test-azure-id" {
			t.Error("Expected AZURE_CLIENT_ID in azure provider vars")
		}
		if azureVars["ARM_SUBSCRIPTION_ID"] != "test-arm-sub" {
			t.Error("Expected ARM_SUBSCRIPTION_ID in azure provider vars")
		}
	}

	// Check Google
	if gcpVars, ok := providers["google"]; !ok {
		t.Error("Expected google provider variables")
	} else if gcpVars["GOOGLE_CREDENTIALS"] != "test-gcp-creds" {
		t.Error("Expected GOOGLE_CREDENTIALS in google provider vars")
	}
}

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{"AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE", "AKIA****"},
		{"GITHUB_TOKEN", "ghp_1234567890abcdef", "ghp_****"}, // gitleaks:allow - test fixture
		{"PASSWORD", "secret123", "secr****"},
		{"API_KEY", "abc", "****"},                // Short value
		{"REGION", "us-west-2", "us-west-2"},      // Not sensitive
		{"WORKSPACE", "production", "production"}, // Not sensitive
	}

	for _, tt := range tests {
		result := MaskSensitiveValue(tt.key, tt.value)
		if result != tt.expected {
			t.Errorf("MaskSensitiveValue(%s, %s) = %s, expected %s",
				tt.key, tt.value, result, tt.expected)
		}
	}
}

func TestGetSensitiveVars(t *testing.T) {
	// Set various environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("TF_VAR_password", "secret")
	os.Setenv("TF_LOG", "DEBUG")         // Not sensitive
	os.Setenv("AWS_REGION", "us-west-2") // Not sensitive
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("GITHUB_TOKEN")
		os.Unsetenv("TF_VAR_password")
		os.Unsetenv("TF_LOG")
		os.Unsetenv("AWS_REGION")
	}()

	sensitive := GetSensitiveVars()

	// Build a map for easier checking
	sensitiveMap := make(map[string]bool)
	for _, key := range sensitive {
		sensitiveMap[key] = true
	}

	// Check sensitive vars are included
	if !sensitiveMap["AWS_ACCESS_KEY_ID"] {
		t.Error("Expected AWS_ACCESS_KEY_ID to be marked as sensitive")
	}

	// Check non-sensitive vars are excluded
	if sensitiveMap["TF_LOG"] {
		t.Error("Expected TF_LOG to NOT be marked as sensitive")
	}

	if sensitiveMap["AWS_REGION"] {
		t.Error("Expected AWS_REGION to NOT be marked as sensitive")
	}
}

func TestGetEnvironmentSummary(t *testing.T) {
	// Set test variables
	os.Setenv("TF_LOG", "DEBUG")
	os.Setenv("TF_VAR_region", "us-west-2")
	os.Setenv("TF_VAR_env", "prod")
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	defer func() {
		os.Unsetenv("TF_LOG")
		os.Unsetenv("TF_VAR_region")
		os.Unsetenv("TF_VAR_env")
		os.Unsetenv("AWS_ACCESS_KEY_ID")
	}()

	summary := GetEnvironmentSummary()

	// Check log level
	if logLevel, ok := summary["log_level"]; !ok || logLevel != "DEBUG" {
		t.Error("Expected log_level in summary")
	}

	// Check variables count
	if varCount, ok := summary["variables_count"]; !ok || varCount != 2 {
		t.Errorf("Expected variables_count to be 2, got %v", varCount)
	}

	// Check providers
	if providers, ok := summary["providers"]; ok {
		if providerMap, ok := providers.(map[string]int); ok {
			if awsCount, ok := providerMap["aws"]; !ok || awsCount < 1 {
				t.Error("Expected aws provider in summary")
			}
		}
	}
}
