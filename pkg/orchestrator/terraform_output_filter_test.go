package orchestrator

import (
	"strings"
	"testing"
)

func TestTerraformOutputFilter_Init(t *testing.T) {
	filter := NewTerraformOutputFilter("test-instance", OpInit)

	// Simulate terraform init output
	lines := []string{
		"Initializing the backend...",
		"Initializing provider plugins...",
		"- Finding hashicorp/azurerm versions matching \">= 4.16.0\"...",
		"- Using previously-installed hashicorp/azurerm v4.52.0",
		"- Using previously-installed hashicorp/random v3.7.2",
		"",
		"Terraform has been successfully initialized!",
		"",
		"You may now begin working with Terraform. Try running \"terraform plan\" to see",
	}

	for _, line := range lines {
		filter.ProcessLine(line)
	}

	output := filter.FormatOutput()

	t.Logf("Filter output:\n%s", output)

	// Check that backend was detected
	if !strings.Contains(output, "Backend: Remote backend") {
		t.Errorf("Expected backend to be detected, got: %s", output)
	}

	// Check that providers were detected
	if !strings.Contains(output, "hashicorp/azurerm") {
		t.Errorf("Expected azurerm provider to be detected, got: %s", output)
	}
	if !strings.Contains(output, "hashicorp/random") {
		t.Errorf("Expected random provider to be detected, got: %s", output)
	}

	// Check that init result was detected
	if !strings.Contains(output, "Successfully initialized") {
		t.Errorf("Expected init success message, got: %s", output)
	}
}

func TestTerraformOutputFilter_Plan(t *testing.T) {
	filter := NewTerraformOutputFilter("test-instance", OpPlan)

	// Simulate terraform plan output
	lines := []string{
		"data.terraform_remote_state.connectivity: Reading...",
		"No changes. Your infrastructure matches the configuration.",
	}

	for _, line := range lines {
		filter.ProcessLine(line)
	}

	output := filter.FormatOutput()

	// Check that plan summary was detected
	if !strings.Contains(output, "No changes") {
		t.Errorf("Expected 'No changes' message, got: %s", output)
	}
}
