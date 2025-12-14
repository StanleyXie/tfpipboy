package orchestrator

import (
	"time"
)

// Config represents the complete tfpipboy configuration
type Config struct {
	Version   string               `yaml:"version"`
	Modules   map[string]*Module   `yaml:"modules"`
	Groups    map[string][]string  `yaml:"groups"`
	Pipelines map[string]*Pipeline `yaml:"pipelines"`
}

// Module represents a Terraform module configuration
type Module struct {
	Name        string                 `yaml:"-"` // Set during parsing
	Description string                 `yaml:"description"`
	Category    string                 `yaml:"category"`
	Owner       string                 `yaml:"owner"`
	Path        string                 `yaml:"path"`
	DependsOn   []string               `yaml:"depends_on"`
	Backend     *BackendConfig         `yaml:"backend"`
	BackendCfg  *BackendConfig         `yaml:"backend-config"` // Alternative field name
	VarConfig   *VariableConfig        `yaml:"var-config"`
	Variables   map[string]interface{} `yaml:"variables"`
	Outputs     []string               `yaml:"outputs"`
	Instances   map[string]*Instance   `yaml:"instances"`
	Enabled     bool                   `yaml:"enabled"`
}

// Instance represents a module instance when a module has multiple instances
type Instance struct {
	Name        string                 `yaml:"-"` // Set during parsing (instance ID)
	Description string                 `yaml:"description"`
	Environment string                 `yaml:"environment"` // First-class field for environment
	Region      string                 `yaml:"region"`      // First-class field for region
	DependsOn   []string               `yaml:"depends_on"`
	Backend     *BackendConfig         `yaml:"backend"`
	BackendCfg  *BackendConfig         `yaml:"backend-config"` // Alternative field name
	VarConfig   *VariableConfig        `yaml:"var-config"`
	Variables   map[string]interface{} `yaml:"variables"`
	Outputs     []string               `yaml:"outputs"`
	Enabled     bool                   `yaml:"enabled"`
	Parallel    *bool                  `yaml:"parallel"` // true=parallel, false=sequential, nil=inherit from dependencies
}

// BackendConfig represents Terraform backend configuration
type BackendConfig struct {
	Type               string            `yaml:"type"`
	File               string            `yaml:"file"`
	Template           string            `yaml:"template"`
	LandingZone        string            `yaml:"landing_zone"`
	TenantID           string            `yaml:"tenant_id"`
	SubscriptionID     string            `yaml:"subscription_id"`
	ResourceGroupName  string            `yaml:"resource_group_name"`
	StorageAccountName string            `yaml:"storage_account_name"`
	ContainerName      string            `yaml:"container_name"`
	Key                string            `yaml:"key"`
	UseAzureADAuth     bool              `yaml:"use_azuread_auth"`
	AdditionalConfig   map[string]string `yaml:",inline"`
}

// VariableConfig represents variable configuration for a module
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

// Pipeline represents an orchestration pipeline
type Pipeline struct {
	Name             string                `yaml:"-"` // Set during parsing
	Description      string                `yaml:"description"`
	DefaultOperation string                `yaml:"default_operation"`
	Settings         *PipelineSettings     `yaml:"settings"`
	Before           []*Hook               `yaml:"before"`
	Stages           []*Stage              `yaml:"stages"`
	After            []*Hook               `yaml:"after"`
	OnFailure        []*Hook               `yaml:"on_failure"`
	RequireConfirm   bool                  `yaml:"require_confirmation"`
	ConfirmMessage   string                `yaml:"confirmation_message"`
	Environments     map[string]*EnvConfig `yaml:"environments"`
}

// Stage represents a pipeline stage
type Stage struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Modules     []string          `yaml:"modules"`
	Groups      []string          `yaml:"groups"`
	Parallel    bool              `yaml:"parallel"`
	Settings    *PipelineSettings `yaml:"settings"`
	Before      []*Hook           `yaml:"before"`
	After       []*Hook           `yaml:"after"`
	OnFailure   []*Hook           `yaml:"on_failure"`
}

// Hook represents a command hook (before/after/on_failure)
type Hook struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}

// PipelineSettings represents execution settings for pipelines and stages
type PipelineSettings struct {
	ParallelLimit int           `yaml:"parallel_limit"`
	Timeout       time.Duration `yaml:"timeout"`
	AutoApprove   bool          `yaml:"auto_approve"`
	RetryCount    int           `yaml:"retry_count"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
}

// EnvConfig represents environment-specific pipeline configuration
type EnvConfig struct {
	Settings       *PipelineSettings `yaml:"settings"`
	RequireConfirm bool              `yaml:"require_confirmation"`
	ConfirmMessage string            `yaml:"confirmation_message"`
	Variables      map[string]string `yaml:"variables"`
}

// ExecutionPlan represents a planned execution sequence
type ExecutionPlan struct {
	Operation   string          `yaml:"operation"`
	Environment string          `yaml:"environment"`
	Modules     []*ExecutionJob `yaml:"modules"`
	TotalJobs   int             `yaml:"total_jobs"`
	CreatedAt   time.Time       `yaml:"created_at"`
}

// ExecutionJob represents a single module execution job
type ExecutionJob struct {
	ID              string                 `yaml:"id"` // Now uses instance ID (e.g., "seed-dev-gwc") instead of "job-1"
	ModuleName      string                 `yaml:"module_name"`
	InstanceName    string                 `yaml:"instance_name,omitempty"` // Same as ID for consistency
	Category        string                 `yaml:"category,omitempty"`      // Module category for grouping
	Environment     string                 `yaml:"environment,omitempty"`   // Instance environment
	Region          string                 `yaml:"region,omitempty"`        // Instance region
	Path            string                 `yaml:"path"`
	Operation       string                 `yaml:"operation"`
	DependsOn       []string               `yaml:"depends_on"`
	DependencyDepth int                    `yaml:"dependency_depth,omitempty"` // Calculated depth based on depends_on (0=no deps, 1=depends on 0, etc.)
	Variables       map[string]interface{} `yaml:"variables"`
	Backend         *BackendConfig         `yaml:"backend"`
	Stage           int                    `yaml:"stage"`
	CanParallel     bool                   `yaml:"can_parallel"`
	Status          JobStatus              `yaml:"status"`
	StartTime       *time.Time             `yaml:"start_time,omitempty"`
	EndTime         *time.Time             `yaml:"end_time,omitempty"`
	Duration        time.Duration          `yaml:"duration"`
	Output          string                 `yaml:"output,omitempty"`
	Error           string                 `yaml:"error,omitempty"`
	// Artifacts tracks generated artifacts (e.g. state keys) during execution
	Artifacts  map[string]string `yaml:"artifacts,omitempty"`
	PlanResult string            `yaml:"plan_result,omitempty"` // Plan result summary (e.g., "No changes", "+3 ~2 -1")
}

// JobStatus represents the status of an execution job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusSkipped   JobStatus = "skipped"
	JobStatusCancelled JobStatus = "cancelled"
)

// ExecutionResult represents the result of an execution
type ExecutionResult struct {
	Success   bool                   `yaml:"success"`
	TotalJobs int                    `yaml:"total_jobs"`
	Completed int                    `yaml:"completed"`
	Failed    int                    `yaml:"failed"`
	Skipped   int                    `yaml:"skipped"`
	Duration  time.Duration          `yaml:"duration"`
	Jobs      []*ExecutionJob        `yaml:"jobs"`
	Outputs   map[string]interface{} `yaml:"outputs"`
	Errors    []string               `yaml:"errors"`
	StartTime time.Time              `yaml:"start_time"`
	EndTime   time.Time              `yaml:"end_time"`
}

// TerraformOperation represents supported Terraform operations
type TerraformOperation string

const (
	OpPlan       TerraformOperation = "plan"
	OpApply      TerraformOperation = "apply"
	OpDestroy    TerraformOperation = "destroy"
	OpValidate   TerraformOperation = "validate"
	OpInit       TerraformOperation = "init"
	OpRefresh    TerraformOperation = "refresh"
	OpOutput     TerraformOperation = "output"
	OpStateTrack TerraformOperation = "state-track"
)

// DependencyGraph represents a module dependency graph
type DependencyGraph struct {
	Nodes map[string]*GraphNode `yaml:"nodes"`
	Edges map[string][]string   `yaml:"edges"`
}

// GraphNode represents a node in the dependency graph
type GraphNode struct {
	ID           string   `yaml:"id"`
	ModuleName   string   `yaml:"module_name"`
	InstanceName string   `yaml:"instance_name,omitempty"`
	Dependencies []string `yaml:"dependencies"`
	Dependents   []string `yaml:"dependents"`
	Stage        int      `yaml:"stage"`
}

// WorkspaceManager interface for managing isolated workspaces
type WorkspaceManager interface {
	CreateWorkspace(jobID string, module *Module, instance *Instance) (*Workspace, error)
	CleanupWorkspace(workspace *Workspace) error
	GetWorkspace(jobID string) (*Workspace, error)
	CleanupAllWorkspaces() error
}

// Workspace represents an isolated execution workspace
type Workspace struct {
	ID          string            `yaml:"id"`
	Path        string            `yaml:"path"`
	ModulePath  string            `yaml:"module_path"`
	LogDir      string            `yaml:"log_dir"`
	Environment map[string]string `yaml:"environment"`
	Backend     *BackendConfig    `yaml:"backend"`
	Variables   map[string]string `yaml:"variables"`
	VarFiles    []string          `yaml:"var_files"` // List of variable file paths to use with -var-file
	TempDir     string            `yaml:"temp_dir"`
	CreatedAt   time.Time         `yaml:"created_at"`
}

// JobMetadata represents metadata for a job execution (artifact tracking)
type JobMetadata struct {
	InstanceID       string                 `json:"instance_id"` // Renamed from JobID, uses instance name
	ModuleName       string                 `json:"module_name"`
	InstanceName     string                 `json:"instance_name,omitempty"`        // Same as InstanceID
	InstanceEnv      string                 `json:"instance_environment,omitempty"` // Instance environment
	InstanceRegion   string                 `json:"instance_region,omitempty"`      // Instance region
	Operation        string                 `json:"operation"`
	Status           string                 `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	DurationSeconds  float64                `json:"duration_seconds,omitempty"`
	ExitCode         int                    `json:"exit_code"`
	TerraformVersion string                 `json:"terraform_version,omitempty"`
	ModulePath       string                 `json:"module_path"`
	BackendType      string                 `json:"backend_type,omitempty"`
	BackendConfig    *BackendConfig         `json:"backend_config,omitempty"`
	WorkspaceSize    int64                  `json:"workspace_size_bytes,omitempty"`
	Artifacts        map[string]string      `json:"artifacts"`
	Environment      map[string]interface{} `json:"environment,omitempty"`
}

// Orchestrator interface defines the main orchestrator functionality
type Orchestrator interface {
	// Configuration
	LoadConfig(configPath string) error
	GetConfig() *Config
	ValidateConfig() error

	// Planning
	PlanExecution(operation TerraformOperation, targets []string, env string) (*ExecutionPlan, error)
	PlanPipeline(pipelineName string, env string) (*ExecutionPlan, error)

	// Execution
	ExecutePlan(plan *ExecutionPlan) (*ExecutionResult, error)
	ExecuteModule(moduleName string, operation TerraformOperation, env string) (*ExecutionResult, error)
	ExecutePipeline(pipelineName string, env string) (*ExecutionResult, error)

	// Dependency Management
	BuildDependencyGraph(modules []string) (*DependencyGraph, error)
	ValidateDependencies() error
	GetExecutionOrder(modules []string) ([][]string, error)

	// State Management
	GetExecutionStatus() (*ExecutionResult, error)
	CancelExecution() error
	CleanupWorkspaces() error
}

// EventType represents different types of orchestrator events
type EventType string

const (
	EventJobStarted     EventType = "job_started"
	EventJobCompleted   EventType = "job_completed"
	EventJobFailed      EventType = "job_failed"
	EventStageStarted   EventType = "stage_started"
	EventStageCompleted EventType = "stage_completed"
	EventStageFailed    EventType = "stage_failed"
	EventHookStarted    EventType = "hook_started"
	EventHookCompleted  EventType = "hook_completed"
	EventHookFailed     EventType = "hook_failed"
)

// Event represents an orchestrator event
type Event struct {
	Type      EventType              `yaml:"type"`
	JobID     string                 `yaml:"job_id,omitempty"`
	Module    string                 `yaml:"module,omitempty"`
	Stage     string                 `yaml:"stage,omitempty"`
	Message   string                 `yaml:"message"`
	Data      map[string]interface{} `yaml:"data,omitempty"`
	Timestamp time.Time              `yaml:"timestamp"`
}

// EventHandler represents an event handler function
type EventHandler func(event *Event)

// Logger interface for orchestrator logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}
