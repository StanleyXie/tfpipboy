package orchestrator

import (
	"fmt"
	"sort"
	"strings"
)

// DependencyGraphBuilder builds and manages dependency graphs for modules
type DependencyGraphBuilder struct {
	config *Config
}

// NewDependencyGraphBuilder creates a new dependency graph builder
func NewDependencyGraphBuilder(config *Config) *DependencyGraphBuilder {
	return &DependencyGraphBuilder{
		config: config,
	}
}

// BuildGraph builds a dependency graph for the specified modules
func (b *DependencyGraphBuilder) BuildGraph(moduleNames []string) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Nodes: make(map[string]*GraphNode),
		Edges: make(map[string][]string),
	}

	// First, collect all modules to include (including dependencies)
	allModules, err := b.collectAllModules(moduleNames)
	if err != nil {
		return nil, fmt.Errorf("failed to collect modules: %w", err)
	}

	// Create nodes for all modules
	for _, moduleRef := range allModules {
		if err := b.addModuleToGraph(graph, moduleRef); err != nil {
			return nil, fmt.Errorf("failed to add module %s to graph: %w", moduleRef, err)
		}
	}

	// Build edges (dependencies)
	for _, moduleRef := range allModules {
		if err := b.addDependenciesToGraph(graph, moduleRef); err != nil {
			return nil, fmt.Errorf("failed to add dependencies for module %s: %w", moduleRef, err)
		}
	}

	// Calculate stages (topological ordering)
	if err := b.calculateStages(graph); err != nil {
		return nil, fmt.Errorf("failed to calculate stages: %w", err)
	}

	return graph, nil
}

// collectAllModules collects all modules including their transitive dependencies
func (b *DependencyGraphBuilder) collectAllModules(moduleNames []string) ([]string, error) {
	visited := make(map[string]bool)
	var result []string

	for _, moduleName := range moduleNames {
		if err := b.collectModuleDependencies(moduleName, visited, &result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// collectModuleDependencies recursively collects module dependencies
func (b *DependencyGraphBuilder) collectModuleDependencies(moduleRef string, visited map[string]bool, result *[]string) error {
	if visited[moduleRef] {
		return nil
	}

	visited[moduleRef] = true

	// Parse module reference
	moduleName, instanceName := b.parseModuleReference(moduleRef)

	// Check if module exists
	module, exists := b.config.Modules[moduleName]
	if !exists {
		return formatTargetNotFoundError(moduleRef, b.config)
	}

	// Get dependencies
	var dependencies []string
	if instanceName != "" {
		// Instance-specific dependencies
		instance, exists := module.Instances[instanceName]
		if !exists {
			return formatTargetNotFoundError(moduleRef, b.config)
		}
		dependencies = instance.DependsOn
	} else {
		// Module-level dependencies
		dependencies = module.DependsOn
	}

	// Recursively collect dependencies
	for _, dep := range dependencies {
		if err := b.collectModuleDependencies(dep, visited, result); err != nil {
			return err
		}
	}

	// Add current module to result
	*result = append(*result, moduleRef)

	return nil
}

// addModuleToGraph adds a module node to the graph
func (b *DependencyGraphBuilder) addModuleToGraph(graph *DependencyGraph, moduleRef string) error {
	moduleName, instanceName := b.parseModuleReference(moduleRef)

	node := &GraphNode{
		ID:           moduleRef,
		ModuleName:   moduleName,
		InstanceName: instanceName,
		Dependencies: []string{},
		Dependents:   []string{},
		Stage:        0, // Will be calculated later
	}

	graph.Nodes[moduleRef] = node
	graph.Edges[moduleRef] = []string{}

	return nil
}

// addDependenciesToGraph adds dependency edges to the graph
func (b *DependencyGraphBuilder) addDependenciesToGraph(graph *DependencyGraph, moduleRef string) error {
	moduleName, instanceName := b.parseModuleReference(moduleRef)

	// Get module
	module, exists := b.config.Modules[moduleName]
	if !exists {
		return formatTargetNotFoundError(moduleRef, b.config)
	}

	// Get dependencies
	var dependencies []string
	if instanceName != "" {
		instance, exists := module.Instances[instanceName]
		if !exists {
			return formatTargetNotFoundError(moduleRef, b.config)
		}
		dependencies = instance.DependsOn
	} else {
		dependencies = module.DependsOn
	}

	// Add edges for each dependency
	for _, dep := range dependencies {
		// Verify dependency exists in graph
		if _, exists := graph.Nodes[dep]; !exists {
			return fmt.Errorf("dependency %s not found in graph", dep)
		}

		// Add edge from dependency to current module
		graph.Edges[dep] = append(graph.Edges[dep], moduleRef)

		// Update node information
		graph.Nodes[moduleRef].Dependencies = append(graph.Nodes[moduleRef].Dependencies, dep)
		graph.Nodes[dep].Dependents = append(graph.Nodes[dep].Dependents, moduleRef)
	}

	return nil
}

// calculateStages calculates the stage (execution order) for each node
func (b *DependencyGraphBuilder) calculateStages(graph *DependencyGraph) error {
	// Use topological sort with Kahn's algorithm
	inDegree := make(map[string]int)
	queue := []string{}

	// Calculate in-degrees
	for nodeID := range graph.Nodes {
		inDegree[nodeID] = len(graph.Nodes[nodeID].Dependencies)
		if inDegree[nodeID] == 0 {
			queue = append(queue, nodeID)
		}
	}

	stage := 0
	processed := 0

	for len(queue) > 0 {
		// Process all nodes at current stage
		currentStage := queue
		queue = []string{}

		for _, nodeID := range currentStage {
			graph.Nodes[nodeID].Stage = stage
			processed++

			// Reduce in-degree for dependents
			for _, dependent := range graph.Edges[nodeID] {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					queue = append(queue, dependent)
				}
			}
		}

		stage++
	}

	// Check for cycles
	if processed != len(graph.Nodes) {
		return fmt.Errorf("circular dependency detected in graph")
	}

	return nil
}

// parseModuleReference parses a module reference (instance name or module.instance format)
func (b *DependencyGraphBuilder) parseModuleReference(moduleRef string) (string, string) {
	// First, try to find as instance name across all modules
	// Use sorted module names for deterministic behavior
	var moduleNames []string
	for moduleName := range b.config.Modules {
		moduleNames = append(moduleNames, moduleName)
	}
	sort.Strings(moduleNames)

	for _, moduleName := range moduleNames {
		module := b.config.Modules[moduleName]
		if _, exists := module.Instances[moduleRef]; exists {
			return moduleName, moduleRef // Found as instance name
		}
	}

	// If not found as instance, try module.instance format (backward compatibility)
	parts := strings.Split(moduleRef, ".")
	if len(parts) == 1 {
		// Just module name (no instance)
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// GetExecutionOrder returns the modules grouped by execution stage
func (b *DependencyGraphBuilder) GetExecutionOrder(graph *DependencyGraph) [][]string {
	// Group nodes by stage
	stageMap := make(map[int][]string)
	maxStage := 0

	for nodeID, node := range graph.Nodes {
		stage := node.Stage
		stageMap[stage] = append(stageMap[stage], nodeID)
		if stage > maxStage {
			maxStage = stage
		}
	}

	// Convert to ordered slice
	result := make([][]string, maxStage+1)
	for stage := 0; stage <= maxStage; stage++ {
		modules := stageMap[stage]
		sort.Strings(modules) // Sort for consistent ordering
		result[stage] = modules
	}

	return result
}

// GetRootNodes returns nodes with no dependencies
func (b *DependencyGraphBuilder) GetRootNodes(graph *DependencyGraph) []string {
	var roots []string
	for nodeID, node := range graph.Nodes {
		if len(node.Dependencies) == 0 {
			roots = append(roots, nodeID)
		}
	}
	sort.Strings(roots)
	return roots
}

// GetLeafNodes returns nodes with no dependents
func (b *DependencyGraphBuilder) GetLeafNodes(graph *DependencyGraph) []string {
	var leaves []string
	for nodeID, node := range graph.Nodes {
		if len(node.Dependents) == 0 {
			leaves = append(leaves, nodeID)
		}
	}
	sort.Strings(leaves)
	return leaves
}

// ValidateGraph validates the dependency graph
func (b *DependencyGraphBuilder) ValidateGraph(graph *DependencyGraph) error {
	// Check that all referenced dependencies exist
	for nodeID, node := range graph.Nodes {
		for _, dep := range node.Dependencies {
			if _, exists := graph.Nodes[dep]; !exists {
				return fmt.Errorf("node %s references non-existent dependency %s", nodeID, dep)
			}
		}

		for _, dependent := range node.Dependents {
			if _, exists := graph.Nodes[dependent]; !exists {
				return fmt.Errorf("node %s references non-existent dependent %s", nodeID, dependent)
			}
		}
	}

	// Check edge consistency
	for from, tos := range graph.Edges {
		for _, to := range tos {
			// Verify the reverse relationship exists
			toNode := graph.Nodes[to]
			if !contains(toNode.Dependencies, from) {
				return fmt.Errorf("edge %s -> %s exists but %s doesn't list %s as dependency", from, to, to, from)
			}

			fromNode := graph.Nodes[from]
			if !contains(fromNode.Dependents, to) {
				return fmt.Errorf("edge %s -> %s exists but %s doesn't list %s as dependent", from, to, from, to)
			}
		}
	}

	return nil
}

// GetSubgraph returns a subgraph containing only the specified nodes and their relationships
func (b *DependencyGraphBuilder) GetSubgraph(graph *DependencyGraph, nodeIDs []string) *DependencyGraph {
	subgraph := &DependencyGraph{
		Nodes: make(map[string]*GraphNode),
		Edges: make(map[string][]string),
	}

	nodeSet := make(map[string]bool)
	for _, nodeID := range nodeIDs {
		nodeSet[nodeID] = true
	}

	// Copy nodes
	for _, nodeID := range nodeIDs {
		if node, exists := graph.Nodes[nodeID]; exists {
			newNode := *node // Copy the node
			// Filter dependencies and dependents to only include nodes in the subgraph
			newNode.Dependencies = filterStringSlice(node.Dependencies, nodeSet)
			newNode.Dependents = filterStringSlice(node.Dependents, nodeSet)
			subgraph.Nodes[nodeID] = &newNode
			subgraph.Edges[nodeID] = []string{}
		}
	}

	// Copy relevant edges
	for nodeID := range subgraph.Nodes {
		if edges, exists := graph.Edges[nodeID]; exists {
			for _, target := range edges {
				if nodeSet[target] {
					subgraph.Edges[nodeID] = append(subgraph.Edges[nodeID], target)
				}
			}
		}
	}

	// Recalculate stages for the subgraph
	b.calculateStages(subgraph)

	return subgraph
}

// GetUpstreamDependencies returns all upstream dependencies of a node
func (b *DependencyGraphBuilder) GetUpstreamDependencies(graph *DependencyGraph, nodeID string) []string {
	visited := make(map[string]bool)
	var result []string

	b.collectUpstreamDependencies(graph, nodeID, visited, &result)

	// Remove the original node if it was included
	filtered := make([]string, 0, len(result))
	for _, dep := range result {
		if dep != nodeID {
			filtered = append(filtered, dep)
		}
	}

	sort.Strings(filtered)
	return filtered
}

// collectUpstreamDependencies recursively collects upstream dependencies
func (b *DependencyGraphBuilder) collectUpstreamDependencies(graph *DependencyGraph, nodeID string, visited map[string]bool, result *[]string) {
	if visited[nodeID] {
		return
	}

	visited[nodeID] = true
	*result = append(*result, nodeID)

	node := graph.Nodes[nodeID]
	for _, dep := range node.Dependencies {
		b.collectUpstreamDependencies(graph, dep, visited, result)
	}
}

// GetDownstreamDependents returns all downstream dependents of a node
func (b *DependencyGraphBuilder) GetDownstreamDependents(graph *DependencyGraph, nodeID string) []string {
	visited := make(map[string]bool)
	var result []string

	b.collectDownstreamDependents(graph, nodeID, visited, &result)

	// Remove the original node if it was included
	filtered := make([]string, 0, len(result))
	for _, dep := range result {
		if dep != nodeID {
			filtered = append(filtered, dep)
		}
	}

	sort.Strings(filtered)
	return filtered
}

// collectDownstreamDependents recursively collects downstream dependents
func (b *DependencyGraphBuilder) collectDownstreamDependents(graph *DependencyGraph, nodeID string, visited map[string]bool, result *[]string) {
	if visited[nodeID] {
		return
	}

	visited[nodeID] = true
	*result = append(*result, nodeID)

	node := graph.Nodes[nodeID]
	for _, dependent := range node.Dependents {
		b.collectDownstreamDependents(graph, dependent, visited, result)
	}
}

// Helper functions

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// filterStringSlice filters a string slice based on a set
func filterStringSlice(slice []string, set map[string]bool) []string {
	var result []string
	for _, item := range slice {
		if set[item] {
			result = append(result, item)
		}
	}
	return result
}
