package orchestrator

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TerraformOutputFilter filters and formats terraform output for display
type TerraformOutputFilter struct {
	instanceID string
	operation  TerraformOperation

	// Extracted information
	backend     string
	providers   []string
	initResult  string
	planChanges *PlanChanges
	applyResult string
	errors      []string
}

// PlanChanges represents the changes from terraform plan
type PlanChanges struct {
	ToAdd     int
	ToChange  int
	ToDestroy int
	Summary   string
	Details   []string
}

// NewTerraformOutputFilter creates a new output filter
func NewTerraformOutputFilter(instanceID string, operation TerraformOperation) *TerraformOutputFilter {
	return &TerraformOutputFilter{
		instanceID: instanceID,
		operation:  operation,
		providers:  make([]string, 0),
		errors:     make([]string, 0),
	}
}

// ProcessLine processes a single line of terraform output
func (f *TerraformOutputFilter) ProcessLine(line string) {
	// Debug: write lines to file to see what's actually being processed
	debugFile, err := os.OpenFile("/tmp/tfpipboy-filter-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		fmt.Fprintf(debugFile, "[%s] LINE: %q\n", f.instanceID, line)
		debugFile.Close()
	}

	// Extract backend information
	if strings.Contains(line, "Initializing the backend") {
		f.backend = "Remote backend"
		if debugFile, _ := os.OpenFile("/tmp/tfpipboy-filter-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); debugFile != nil {
			fmt.Fprintf(debugFile, "[%s] MATCHED: backend\n", f.instanceID)
			debugFile.Close()
		}
	} else if strings.Contains(line, "terraform.tfstate") {
		f.backend = "Local backend"
	}

	// Extract provider information - look for lines with "- Using previously-installed" or "- Installing"
	if strings.Contains(line, "- Using previously-installed") || strings.Contains(line, "- Installing") {
		provider := extractProvider(line)
		if provider != "" && !containsString(f.providers, provider) {
			f.providers = append(f.providers, provider)
			if debugFile, _ := os.OpenFile("/tmp/tfpipboy-filter-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); debugFile != nil {
				fmt.Fprintf(debugFile, "[%s] MATCHED: provider=%s\n", f.instanceID, provider)
				debugFile.Close()
			}
		}
	}

	// Extract initialization result
	if strings.Contains(line, "Terraform has been successfully initialized") {
		f.initResult = "✓ Successfully initialized"
		if debugFile, _ := os.OpenFile("/tmp/tfpipboy-filter-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); debugFile != nil {
			fmt.Fprintf(debugFile, "[%s] MATCHED: init success\n", f.instanceID)
			debugFile.Close()
		}
	}

	// Extract plan changes
	if f.planChanges == nil {
		f.planChanges = &PlanChanges{Details: make([]string, 0)}
	}

	// Match: Plan: 5 to add, 2 to change, 1 to destroy
	planRegex := regexp.MustCompile(`Plan: (\d+) to add, (\d+) to change, (\d+) to destroy`)
	if matches := planRegex.FindStringSubmatch(line); matches != nil {
		f.planChanges.Summary = line
		// Parse numbers if needed
	}

	// Extract "No changes" message (match with or without ".")
	if strings.Contains(line, "No changes") && strings.Contains(line, "infrastructure matches") {
		f.planChanges.Summary = "No changes"
		if debugFile, _ := os.OpenFile("/tmp/tfpipboy-filter-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); debugFile != nil {
			fmt.Fprintf(debugFile, "[%s] MATCHED: no changes\n", f.instanceID)
			debugFile.Close()
		}
	}

	// Extract resource changes from terraform plan output
	// Format: "  # module.path.resource_type.name will be created/updated/destroyed"
	trimmed := strings.TrimSpace(line)

	// Match the comment lines that show what will happen to each resource
	// Examples:
	//   # module.baseline.azurerm_resource_group.main will be created
	//   # azurerm_virtual_network.example will be updated in-place
	//   # aws_instance.example will be destroyed
	if strings.HasPrefix(trimmed, "#") &&
		(strings.Contains(line, " will be created") ||
			strings.Contains(line, " will be updated") ||
			strings.Contains(line, " will be destroyed") ||
			strings.Contains(line, " will be replaced") ||
			strings.Contains(line, " must be replaced")) {
		// Extract just the resource identifier and action
		// Remove the leading "# " and keep the rest
		resourceLine := strings.TrimPrefix(trimmed, "#")
		resourceLine = strings.TrimSpace(resourceLine)

		// Determine the action symbol
		var symbol string
		if strings.Contains(line, "will be created") {
			symbol = "+"
		} else if strings.Contains(line, "will be destroyed") {
			symbol = "-"
		} else {
			symbol = "~"
		}

		// Format: "+ module.baseline.azurerm_resource_group.main"
		formatted := fmt.Sprintf("%s %s", symbol, resourceLine)
		f.planChanges.Details = append(f.planChanges.Details, formatted)

		if debugFile, _ := os.OpenFile("/tmp/tfpipboy-filter-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); debugFile != nil {
			fmt.Fprintf(debugFile, "[%s] MATCHED RESOURCE CHANGE: %s\n", f.instanceID, formatted)
			debugFile.Close()
		}
	}

	// Extract errors
	if strings.Contains(line, "Error:") || strings.Contains(line, "│ Error:") {
		f.errors = append(f.errors, line)
	}
}

// FormatOutput formats the filtered output for display
func (f *TerraformOutputFilter) FormatOutput() string {
	var output strings.Builder

	// Instance header
	output.WriteString(f.instanceID)
	output.WriteString("\n")
	output.WriteString(strings.Repeat("─", 80))
	output.WriteString("\n")

	switch f.operation {
	case OpInit:
		output.WriteString(f.formatInit())
	case OpPlan:
		output.WriteString(f.formatPlan())
	case OpApply:
		output.WriteString(f.formatApply())
	}

	// Errors (if any)
	if len(f.errors) > 0 {
		output.WriteString("\n❌ Errors:\n")
		for _, err := range f.errors {
			output.WriteString("  ")
			output.WriteString(err)
			output.WriteString("\n")
		}
	}

	output.WriteString("\n")
	return output.String()
}

// formatInit formats init operation output
func (f *TerraformOutputFilter) formatInit() string {
	var output strings.Builder

	// Backend
	if f.backend != "" {
		output.WriteString("📦 Backend: ")
		output.WriteString(f.backend)
		output.WriteString("\n")
	}

	// Providers
	if len(f.providers) > 0 {
		output.WriteString("🔌 Providers: ")
		output.WriteString(strings.Join(f.providers, ", "))
		output.WriteString("\n")
	}

	// Result
	if f.initResult != "" {
		output.WriteString(f.initResult)
		output.WriteString("\n")
	}

	return output.String()
}

// formatPlan formats plan operation output
func (f *TerraformOutputFilter) formatPlan() string {
	var output strings.Builder

	// Backend
	if f.backend != "" {
		output.WriteString("📦 Backend: ")
		output.WriteString(f.backend)
		output.WriteString("\n")
	}

	// Providers (from init phase)
	if len(f.providers) > 0 {
		output.WriteString("🔌 Providers: ")
		output.WriteString(strings.Join(f.providers, ", "))
		output.WriteString("\n")
	}

	// Init result (from init phase)
	if f.initResult != "" {
		output.WriteString(f.initResult)
		output.WriteString("\n")
	}

	// Plan summary
	if f.planChanges != nil && f.planChanges.Summary != "" {
		output.WriteString("📋 ")
		output.WriteString(f.planChanges.Summary)
		output.WriteString("\n")

		// Show all changes
		if len(f.planChanges.Details) > 0 {
			output.WriteString("\nKey Changes:\n")
			for _, detail := range f.planChanges.Details {
				output.WriteString("  ")
				output.WriteString(detail)
				output.WriteString("\n")
			}
		}
	}

	return output.String()
}

// formatApply formats apply operation output
func (f *TerraformOutputFilter) formatApply() string {
	var output strings.Builder

	// Backend
	if f.backend != "" {
		output.WriteString("📦 Backend: ")
		output.WriteString(f.backend)
		output.WriteString("\n")
	}

	// Providers (from init phase)
	if len(f.providers) > 0 {
		output.WriteString("🔌 Providers: ")
		output.WriteString(strings.Join(f.providers, ", "))
		output.WriteString("\n")
	}

	// Init result (from init phase)
	if f.initResult != "" {
		output.WriteString(f.initResult)
		output.WriteString("\n")
	}

	// Apply result
	if f.applyResult != "" {
		output.WriteString(f.applyResult)
		output.WriteString("\n")
	}

	return output.String()
}

// Helper functions

func extractProvider(line string) string {
	// Extract provider name from lines like:
	// "- Using previously-installed hashicorp/azurerm v4.52.0"
	// "- Installing hashicorp/azurerm v4.52.0..."

	if strings.Contains(line, "hashicorp/") {
		parts := strings.Split(line, "hashicorp/")
		if len(parts) > 1 {
			provider := strings.Split(parts[1], " ")[0]
			return "hashicorp/" + provider
		}
	}

	if strings.Contains(line, "azure/") {
		parts := strings.Split(line, "azure/")
		if len(parts) > 1 {
			provider := strings.Split(parts[1], " ")[0]
			return "azure/" + provider
		}
	}

	return ""
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
