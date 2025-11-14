package orchestrator

import (
	"fmt"
	"regexp"
	"strings"
)

// TerraformErrorType represents different categories of Terraform errors
type TerraformErrorType string

const (
	ErrorTypeConfigFile    TerraformErrorType = "CONFIG_FILE"
	ErrorTypePathNotFound  TerraformErrorType = "PATH_NOT_FOUND"
	ErrorTypeVariableError TerraformErrorType = "VARIABLE_ERROR"
	ErrorTypeBackendError  TerraformErrorType = "BACKEND_ERROR"
	ErrorTypeProviderError TerraformErrorType = "PROVIDER_ERROR"
	ErrorTypeAuthError     TerraformErrorType = "AUTH_ERROR"
	ErrorTypeResourceError TerraformErrorType = "RESOURCE_ERROR"
	ErrorTypeStateError    TerraformErrorType = "STATE_ERROR"
	ErrorTypeSyntaxError   TerraformErrorType = "SYNTAX_ERROR"
	ErrorTypeModuleError   TerraformErrorType = "MODULE_ERROR"
	ErrorTypeVersionError  TerraformErrorType = "VERSION_ERROR"
	ErrorTypeUnknown       TerraformErrorType = "UNKNOWN"
)

// TerraformError represents a categorized Terraform error with context
type TerraformError struct {
	Type         TerraformErrorType
	Category     string // User-friendly category name
	ErrorMessage string // Original error message
	Details      string // Extracted error details
	Resolution   string // Suggested resolution steps
	FilePath     string // Related file path if available
	LineNumber   int    // Line number if available
}

// ErrorPattern represents a pattern to match and categorize errors
type ErrorPattern struct {
	Pattern    *regexp.Regexp
	Type       TerraformErrorType
	Category   string
	Resolution string
}

// Common error patterns for detection
var errorPatterns = []ErrorPattern{
	// Config file errors
	{
		Pattern:    regexp.MustCompile(`(?i)no configuration files|no \S+\.tf files found|directory \S+ has no \S+ configuration files`),
		Type:       ErrorTypeConfigFile,
		Category:   "Configuration File Missing",
		Resolution: "Ensure your module directory contains .tf files. Check that the module path in your configuration is correct.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)failed to read file:|error reading \S+:|cannot read file`),
		Type:       ErrorTypeConfigFile,
		Category:   "Configuration File Read Error",
		Resolution: "Check that the configuration files exist and have proper read permissions.",
	},

	// Path not found errors
	{
		Pattern:    regexp.MustCompile(`(?i)no such file or directory|cannot find module|module not found|directory does not exist`),
		Type:       ErrorTypePathNotFound,
		Category:   "Path Not Found",
		Resolution: "Verify that the module path exists and is correctly specified in your configuration.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)failed to read module directory|module directory .* does not exist`),
		Type:       ErrorTypePathNotFound,
		Category:   "Module Directory Not Found",
		Resolution: "Check that the module directory path is correct and accessible.",
	},

	// Variable errors
	{
		Pattern:    regexp.MustCompile(`(?i)required variable .* not set|variable .* is required|no value for required variable`),
		Type:       ErrorTypeVariableError,
		Category:   "Required Variable Missing",
		Resolution: "Add the missing variable to your tfvars file or instance configuration. Check that all required variables are defined.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)invalid value for variable|variable .* has invalid type|type mismatch for variable`),
		Type:       ErrorTypeVariableError,
		Category:   "Invalid Variable Value",
		Resolution: "Check that the variable value matches the expected type (string, number, list, map, etc.).",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)error in function call|invalid function argument`),
		Type:       ErrorTypeVariableError,
		Category:   "Variable Function Error",
		Resolution: "Review the function calls in your configuration files. Check syntax and argument types.",
	},

	// Backend errors
	{
		Pattern:    regexp.MustCompile(`(?i)backend initialization required|backend not initialized|run.*terraform init`),
		Type:       ErrorTypeBackendError,
		Category:   "Backend Not Initialized",
		Resolution: "The backend needs initialization. This should happen automatically, but may have failed.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)backend configuration changed|backend has changed|backend type changed`),
		Type:       ErrorTypeBackendError,
		Category:   "Backend Configuration Changed",
		Resolution: "Backend configuration has changed. You may need to run terraform init with -reconfigure flag.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)error loading state|failed to load state|state file not found|cannot access state`),
		Type:       ErrorTypeBackendError,
		Category:   "State Access Error",
		Resolution: "Check backend configuration and credentials. Verify state file location and access permissions.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)error configuring .* backend|backend configuration error|invalid backend configuration`),
		Type:       ErrorTypeBackendError,
		Category:   "Backend Configuration Error",
		Resolution: "Review your backend configuration in backends.yaml. Check that all required parameters are provided.",
	},

	// Provider errors
	{
		Pattern:    regexp.MustCompile(`(?i)provider .* not found|provider registry error|failed to install provider`),
		Type:       ErrorTypeProviderError,
		Category:   "Provider Installation Failed",
		Resolution: "Check network connectivity and provider registry access. Verify provider version constraints.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)provider configuration error|invalid provider configuration`),
		Type:       ErrorTypeProviderError,
		Category:   "Provider Configuration Error",
		Resolution: "Review provider configuration block in your .tf files. Check required provider settings.",
	},

	// Authentication errors
	{
		Pattern:    regexp.MustCompile(`(?i)Backend authentication failed`),
		Type:       ErrorTypeAuthError,
		Category:   "Backend Authentication Required",
		Resolution: "Backend authentication check failed. Follow the authentication instructions in the error message.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)authentication failed|access denied|unauthorized|credentials.*invalid|permission denied`),
		Type:       ErrorTypeAuthError,
		Category:   "Authentication Failed",
		Resolution: "Verify your cloud provider credentials. Check authentication for your backend provider.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)expired credentials|token expired|session expired|token has expired`),
		Type:       ErrorTypeAuthError,
		Category:   "Credentials Expired",
		Resolution: "Your credentials have expired. Re-authenticate with your cloud provider (aws sso login, az login, gcloud auth login).",
	},

	// Resource errors
	{
		Pattern:    regexp.MustCompile(`(?i)resource .* already exists|duplicate resource|resource name conflict`),
		Type:       ErrorTypeResourceError,
		Category:   "Resource Already Exists",
		Resolution: "A resource with this name already exists. Consider importing existing resources or using different names.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)resource not found|resource .* does not exist`),
		Type:       ErrorTypeResourceError,
		Category:   "Resource Not Found",
		Resolution: "The referenced resource does not exist. Check resource dependencies and names.",
	},

	// State errors
	{
		Pattern:    regexp.MustCompile(`(?i)state lock|state is locked|lock acquisition failed|unable to acquire lock`),
		Type:       ErrorTypeStateError,
		Category:   "State Lock Conflict",
		Resolution: "Another process has locked the state. Wait for other operations to complete or manually release the lock if needed.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)state file corrupted|invalid state file|error parsing state`),
		Type:       ErrorTypeStateError,
		Category:   "State File Corrupted",
		Resolution: "The state file is corrupted or invalid. Restore from backup or check state file integrity.",
	},

	// Syntax errors
	{
		Pattern:    regexp.MustCompile(`(?i)syntax error|invalid syntax|parse error|unexpected token`),
		Type:       ErrorTypeSyntaxError,
		Category:   "Configuration Syntax Error",
		Resolution: "Fix syntax errors in your .tf files. Check brackets, quotes, and HCL syntax.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)invalid block type|unexpected block|block not allowed`),
		Type:       ErrorTypeSyntaxError,
		Category:   "Invalid Configuration Block",
		Resolution: "Review your configuration blocks. Ensure they follow Terraform HCL syntax.",
	},

	// Module errors
	{
		Pattern:    regexp.MustCompile(`(?i)module .* not found|module download failed|failed to load module`),
		Type:       ErrorTypeModuleError,
		Category:   "Module Not Found",
		Resolution: "Check module source path or registry URL. Ensure network access to module sources.",
	},
	{
		Pattern:    regexp.MustCompile(`(?i)incompatible module|module version constraint|module requirements not met`),
		Type:       ErrorTypeModuleError,
		Category:   "Module Version Incompatible",
		Resolution: "Check module version constraints. Update module versions or adjust constraints.",
	},

	// Version errors
	{
		Pattern:    regexp.MustCompile(`(?i)terraform version|version constraint|unsupported version|version mismatch`),
		Type:       ErrorTypeVersionError,
		Category:   "Terraform Version Incompatible",
		Resolution: "Check Terraform version requirements in your configuration. Install compatible Terraform version.",
	},
}

// CategorizeError analyzes Terraform error output and categorizes it
func CategorizeError(output string) *TerraformError {
	tfError := &TerraformError{
		Type:         ErrorTypeUnknown,
		Category:     "Terraform Error",
		ErrorMessage: output,
		Details:      extractErrorDetails(output),
	}

	// Try to match against known patterns
	for _, pattern := range errorPatterns {
		if pattern.Pattern.MatchString(output) {
			tfError.Type = pattern.Type
			tfError.Category = pattern.Category
			tfError.Resolution = pattern.Resolution

			// Try to extract file path and line number
			tfError.FilePath, tfError.LineNumber = extractFileLocation(output)

			return tfError
		}
	}

	// Fallback: generic error with best-effort extraction
	tfError.Resolution = "Review the error details below and Terraform logs for more information."
	tfError.FilePath, tfError.LineNumber = extractFileLocation(output)

	return tfError
}

// extractErrorDetails extracts the main error message from Terraform output
func extractErrorDetails(output string) string {
	var errorLines []string
	var inErrorBlock bool

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Detect error markers
		cleanLine := StripANSI(line)

		if strings.Contains(cleanLine, "Error:") || strings.Contains(cleanLine, "error:") ||
			strings.Contains(cleanLine, "ERROR:") || strings.HasPrefix(cleanLine, "│ Error") ||
			strings.HasPrefix(cleanLine, "╷") {
			inErrorBlock = true
		}

		// Collect error lines
		if inErrorBlock {
			// Skip box drawing characters for cleaner output
			if !strings.HasPrefix(cleanLine, "│") &&
				!strings.HasPrefix(cleanLine, "╷") &&
				!strings.HasPrefix(cleanLine, "╵") &&
				cleanLine != "" {
				errorLines = append(errorLines, cleanLine)
			}

			// End of error block
			if strings.HasPrefix(cleanLine, "╵") {
				inErrorBlock = false
			}
		}
	}

	if len(errorLines) == 0 {
		// Try to get last non-empty lines as a fallback
		for i := len(lines) - 1; i >= 0 && len(errorLines) < 10; i-- {
			cleanLine := strings.TrimSpace(StripANSI(lines[i]))
			if cleanLine != "" {
				errorLines = append([]string{cleanLine}, errorLines...)
			}
		}
	}

	result := strings.Join(errorLines, "\n")
	if result == "" {
		return "No specific error details could be extracted."
	}

	return result
}

// extractFileLocation tries to extract file path and line number from error output
func extractFileLocation(output string) (string, int) {
	// Pattern: on path/to/file.tf line 123
	fileLinePattern := regexp.MustCompile(`(?:on|in) (\S+\.tf) line (\d+)`)
	if matches := fileLinePattern.FindStringSubmatch(output); len(matches) >= 3 {
		lineNum := 0
		fmt.Sscanf(matches[2], "%d", &lineNum)
		return matches[1], lineNum
	}

	// Pattern: path/to/file.tf:123
	colonPattern := regexp.MustCompile(`(\S+\.tf):(\d+)`)
	if matches := colonPattern.FindStringSubmatch(output); len(matches) >= 3 {
		lineNum := 0
		fmt.Sscanf(matches[2], "%d", &lineNum)
		return matches[1], lineNum
	}

	// Pattern: Error in path/to/file.tf
	fileOnlyPattern := regexp.MustCompile(`(?:Error|error).*?(\S+\.tf)`)
	if matches := fileOnlyPattern.FindStringSubmatch(output); len(matches) >= 2 {
		return matches[1], 0
	}

	return "", 0
}

// FormatCategorizedError formats a categorized error for display
func FormatCategorizedError(job *ExecutionJob, tfError *TerraformError, args []string) string {
	// ANSI color codes
	red := "\033[31m"
	yellow := "\033[33m"
	cyan := "\033[36m"
	bold := "\033[1m"
	reset := "\033[0m"

	var sb strings.Builder
	separator := strings.Repeat("═", 80)

	// Header
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s%s%s\n", red, bold, separator))

	// Error type indicator
	switch tfError.Type {
	case ErrorTypeConfigFile, ErrorTypePathNotFound:
		sb.WriteString(fmt.Sprintf("%s   ⚠️  ORCHESTRATOR ERROR: %s%s\n", yellow, tfError.Category, reset))
	case ErrorTypeVariableError:
		sb.WriteString(fmt.Sprintf("%s   ⚠️  TERRAFORM COMMAND ERROR: %s%s\n", yellow, tfError.Category, reset))
	case ErrorTypeBackendError, ErrorTypeStateError:
		sb.WriteString(fmt.Sprintf("%s   ⚠️  BACKEND/STATE ERROR: %s%s\n", yellow, tfError.Category, reset))
	case ErrorTypeAuthError:
		sb.WriteString(fmt.Sprintf("%s   ⚠️  AUTHENTICATION ERROR: %s%s\n", yellow, tfError.Category, reset))
	default:
		sb.WriteString(fmt.Sprintf("%s   ⚠️  TERRAFORM ERROR: %s%s\n", yellow, tfError.Category, reset))
	}

	sb.WriteString(fmt.Sprintf("%s%s%s\n", red, separator, reset))

	// Instance information
	sb.WriteString(fmt.Sprintf("%sInstance:%s        %s\n", cyan, reset, job.ID))
	sb.WriteString(fmt.Sprintf("%sModule:%s          %s\n", cyan, reset, job.ModuleName))
	if job.InstanceName != "" && job.InstanceName != job.ID {
		sb.WriteString(fmt.Sprintf("%sInstance Name:%s   %s\n", cyan, reset, job.InstanceName))
	}
	sb.WriteString(fmt.Sprintf("%sOperation:%s       %s\n", cyan, reset, job.Operation))
	sb.WriteString(fmt.Sprintf("%sCommand:%s         terraform %s\n", cyan, reset, strings.Join(args, " ")))

	// File location if available
	if tfError.FilePath != "" {
		if tfError.LineNumber > 0 {
			sb.WriteString(fmt.Sprintf("%sLocation:%s        %s:%d\n", cyan, reset, tfError.FilePath, tfError.LineNumber))
		} else {
			sb.WriteString(fmt.Sprintf("%sFile:%s            %s\n", cyan, reset, tfError.FilePath))
		}
	}

	sb.WriteString(fmt.Sprintf("%s%s%s\n", red, separator, reset))

	// Error details
	sb.WriteString(fmt.Sprintf("\n%s%sERROR DETAILS:%s\n", bold, red, reset))
	sb.WriteString(fmt.Sprintf("%s%s%s\n", red, separator, reset))
	sb.WriteString(tfError.Details)
	if !strings.HasSuffix(tfError.Details, "\n") {
		sb.WriteString("\n")
	}

	// Resolution suggestion
	if tfError.Resolution != "" {
		sb.WriteString(fmt.Sprintf("\n%s%s💡 SUGGESTED RESOLUTION:%s\n", bold, cyan, reset))
		sb.WriteString(fmt.Sprintf("%s%s%s\n", cyan, separator, reset))
		sb.WriteString(fmt.Sprintf("%s\n", tfError.Resolution))
	}

	// Additional context based on error type
	sb.WriteString(formatAdditionalContext(tfError, job))

	sb.WriteString(fmt.Sprintf("\n%s%s%s\n", red, separator, reset))
	sb.WriteString(fmt.Sprintf("%sLog Files:%s\n", cyan, reset))
	sb.WriteString(fmt.Sprintf("  • Full log:  .tfpipboy/logs/%s/%s.log\n", job.ID, job.ID))
	sb.WriteString(fmt.Sprintf("  • Error log: .tfpipboy/logs/%s/%s-error.log\n", job.ID, job.ID))
	sb.WriteString(fmt.Sprintf("%s%s%s%s\n", red, bold, separator, reset))
	sb.WriteString("\n")

	return sb.String()
}

// formatAdditionalContext provides additional context based on error type
func formatAdditionalContext(tfError *TerraformError, job *ExecutionJob) string {
	var sb strings.Builder
	cyan := "\033[36m"
	reset := "\033[0m"

	switch tfError.Type {
	case ErrorTypeConfigFile:
		sb.WriteString(fmt.Sprintf("\n%s📋 TROUBLESHOOTING STEPS:%s\n", cyan, reset))
		sb.WriteString("  1. Check module path in your configuration (.tfpipboy/modules.yaml)\n")
		sb.WriteString("  2. Verify that .tf files exist in the module directory\n")
		sb.WriteString("  3. Ensure the module 'path' field points to the correct directory\n")

	case ErrorTypePathNotFound:
		sb.WriteString(fmt.Sprintf("\n%s📋 TROUBLESHOOTING STEPS:%s\n", cyan, reset))
		sb.WriteString("  1. Verify the module path in your configuration\n")
		sb.WriteString("  2. Check that the directory exists and is accessible\n")
		sb.WriteString("  3. If using relative paths, ensure they're relative to the correct base directory\n")

	case ErrorTypeVariableError:
		sb.WriteString(fmt.Sprintf("\n%s📋 TROUBLESHOOTING STEPS:%s\n", cyan, reset))
		sb.WriteString("  1. Check your instance configuration in .tfpipboy/modules.yaml\n")
		sb.WriteString("  2. Verify all required variables are defined in the 'variables' section\n")
		sb.WriteString("  3. Check variable types match what the module expects\n")
		sb.WriteString("  4. Review the module's variables.tf for required variable definitions\n")

	case ErrorTypeBackendError:
		sb.WriteString(fmt.Sprintf("\n%s📋 TROUBLESHOOTING STEPS:%s\n", cyan, reset))
		sb.WriteString("  1. Check backend configuration in .tfpipboy/backends.yaml\n")
		sb.WriteString("  2. Verify all required backend parameters are provided\n")
		sb.WriteString("  3. Ensure you have access to the backend storage (S3, Azure Storage, etc.)\n")
		sb.WriteString("  4. Verify backend credentials are configured correctly\n")

	case ErrorTypeAuthError:
		sb.WriteString(fmt.Sprintf("\n%s📋 AUTHENTICATION STEPS:%s\n", cyan, reset))
		sb.WriteString("  AWS:   aws sso login --profile <profile> or aws configure\n")
		sb.WriteString("  Azure: az login\n")
		sb.WriteString("  GCP:   gcloud auth application-default login\n")
		sb.WriteString("\n  Run authentication commands and then retry the operation.\n")
	}

	return sb.String()
}
