package terraform

import (
	"os"
	"sync"
	"time"
)

// Manager manages Terraform context detection
type Manager struct {
	currentPath string
	cache       *Cache
	mu          sync.RWMutex
}

// NewManager creates a new Terraform context manager
func NewManager() *Manager {
	cwd, _ := os.Getwd()
	return &Manager{
		currentPath: cwd,
		cache:       NewCache(5 * time.Second), // 5 second cache
	}
}

// SetPath sets the current working path
func (m *Manager) SetPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentPath != path {
		m.currentPath = path
		m.cache.Clear() // Clear cache when path changes
	}
}

// GetPath returns the current working path
func (m *Manager) GetPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentPath
}

// GetContext retrieves complete Terraform context for current path
func (m *Manager) GetContext() (*Context, error) {
	// Check cache first
	if cached := m.cache.Get(); cached != nil {
		return cached, nil
	}

	path := m.GetPath()
	ctx := &Context{
		CheckedAt:   time.Now(),
		Variables:   make(map[string]string),
		Environment: make(map[string]string),
	}

	// Check if this is a Terraform directory
	if !isTerraformDirectory(path) {
		ctx.Error = nil // Not an error, just not a TF directory
		m.cache.Set(ctx)
		return ctx, nil
	}

	// Get workspace
	workspace, err := GetWorkspace(path)
	if err == nil {
		ctx.Workspace = workspace
	}

	// Get backend
	backend, err := GetBackend(path)
	if err == nil && backend != nil {
		ctx.Backend = GetBackendSummary(backend)
	}

	// Get module info
	moduleInfo, err := GetModuleInfo(path)
	if err == nil && moduleInfo != nil {
		ctx.Module = moduleInfo.Name
		if moduleInfo.IsRoot {
			ctx.RootModule = moduleInfo.Path
		} else {
			rootPath, _ := FindRootModule(path)
			ctx.RootModule = rootPath
		}
	}

	// Get environment variables
	env := GetEnvironmentVars()
	ctx.Variables = env.Variables
	ctx.Environment = GetTerraformEnvVars()

	m.cache.Set(ctx)
	return ctx, nil
}

// GetContextAsync retrieves context asynchronously
func (m *Manager) GetContextAsync() <-chan *Context {
	ch := make(chan *Context, 1)

	go func() {
		ctx, _ := m.GetContext()
		ch <- ctx
		close(ch)
	}()

	return ch
}

// RefreshContext clears cache and fetches fresh context
func (m *Manager) RefreshContext() (*Context, error) {
	m.cache.Clear()
	return m.GetContext()
}

// IsInTerraformDirectory checks if current path is a Terraform directory
func (m *Manager) IsInTerraformDirectory() bool {
	path := m.GetPath()
	return isTerraformDirectory(path)
}

// GetWorkspaceList returns all available workspaces
func (m *Manager) GetWorkspaceList() ([]string, error) {
	path := m.GetPath()
	return ListWorkspaces(path)
}

// GetModuleSources returns all module sources in current directory
func (m *Manager) GetModuleSources() ([]ModuleSource, error) {
	path := m.GetPath()
	return ListModuleSources(path)
}

// GetEnvironmentVarsSummary returns environment configuration summary
func (m *Manager) GetEnvironmentVarsSummary() map[string]interface{} {
	return GetEnvironmentSummary()
}

// ContextSummary provides a concise summary of the context
type ContextSummary struct {
	InTerraformDir bool
	Workspace      string
	Backend        string
	Module         string
	HasVariables   bool
	VariableCount  int
}

// GetSummary returns a concise summary of current context
func (m *Manager) GetSummary() (*ContextSummary, error) {
	ctx, err := m.GetContext()
	if err != nil {
		return nil, err
	}

	summary := &ContextSummary{
		InTerraformDir: m.IsInTerraformDirectory(),
		Workspace:      ctx.Workspace,
		Backend:        ctx.Backend,
		Module:         ctx.Module,
		VariableCount:  len(ctx.Variables),
		HasVariables:   len(ctx.Variables) > 0,
	}

	return summary, nil
}
