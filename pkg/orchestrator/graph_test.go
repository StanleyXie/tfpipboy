package orchestrator

import (
	"testing"
)

// TestNewDependencyGraphBuilder tests creating a new dependency graph builder
func TestNewDependencyGraphBuilder(t *testing.T) {
	config := &Config{
		Modules: make(map[string]*Module),
	}

	builder := NewDependencyGraphBuilder(config)

	if builder == nil {
		t.Fatal("Expected non-nil builder")
	}

	if builder.config != config {
		t.Error("Expected builder to have correct config reference")
	}
}

// TestBuildGraph_SimpleLinearDependency tests building a graph with simple linear dependencies
func TestBuildGraph_SimpleLinearDependency(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"instance-a": {Name: "instance-a"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"instance-b": {
						Name:      "instance-b",
						DependsOn: []string{"instance-a"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"instance-b"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 2 nodes (instance-a and instance-b)
	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
	}

	// Verify instance-b depends on instance-a
	nodeB := graph.Nodes["instance-b"]
	if len(nodeB.Dependencies) != 1 || nodeB.Dependencies[0] != "instance-a" {
		t.Errorf("Expected instance-b to depend on instance-a, got: %v", nodeB.Dependencies)
	}

	// Verify instance-a has instance-b as dependent
	nodeA := graph.Nodes["instance-a"]
	if len(nodeA.Dependents) != 1 || nodeA.Dependents[0] != "instance-b" {
		t.Errorf("Expected instance-a to have instance-b as dependent, got: %v", nodeA.Dependents)
	}

	// Verify stages: instance-a should be stage 0, instance-b should be stage 1
	if nodeA.Stage != 0 {
		t.Errorf("Expected instance-a to be stage 0, got %d", nodeA.Stage)
	}
	if nodeB.Stage != 1 {
		t.Errorf("Expected instance-b to be stage 1, got %d", nodeB.Stage)
	}
}

// TestBuildGraph_MultipleParallelDependencies tests building a graph with parallel dependencies
func TestBuildGraph_MultipleParallelDependencies(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"instance-a1": {Name: "instance-a1"},
					"instance-a2": {Name: "instance-a2"},
					"instance-a3": {Name: "instance-a3"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"instance-b": {
						Name:      "instance-b",
						DependsOn: []string{"instance-a1", "instance-a2", "instance-a3"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"instance-b"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 4 nodes
	if len(graph.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(graph.Nodes))
	}

	// All a instances should be stage 0 (parallel)
	for _, name := range []string{"instance-a1", "instance-a2", "instance-a3"} {
		node := graph.Nodes[name]
		if node.Stage != 0 {
			t.Errorf("Expected %s to be stage 0, got %d", name, node.Stage)
		}
	}

	// instance-b should be stage 1
	nodeB := graph.Nodes["instance-b"]
	if nodeB.Stage != 1 {
		t.Errorf("Expected instance-b to be stage 1, got %d", nodeB.Stage)
	}
}

// TestBuildGraph_CircularDependency tests detection of circular dependencies
func TestBuildGraph_CircularDependency(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"instance-a": {
						Name:      "instance-a",
						DependsOn: []string{"instance-b"},
					},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"instance-b": {
						Name:      "instance-b",
						DependsOn: []string{"instance-a"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	_, err := builder.BuildGraph([]string{"instance-b"})

	if err == nil {
		t.Fatal("Expected error for circular dependency, got nil")
	}

	if err.Error() != "failed to calculate stages: circular dependency detected in graph" {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

// TestBuildGraph_ModuleNotFound tests error when module doesn't exist
func TestBuildGraph_ModuleNotFound(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{},
	}

	builder := NewDependencyGraphBuilder(config)
	_, err := builder.BuildGraph([]string{"non-existent"})

	if err == nil {
		t.Fatal("Expected error for non-existent module, got nil")
	}
}

// TestGetExecutionOrder tests getting execution order from graph
func TestGetExecutionOrder(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"instance-a": {Name: "instance-a"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"instance-b1": {
						Name:      "instance-b1",
						DependsOn: []string{"instance-a"},
					},
					"instance-b2": {
						Name:      "instance-b2",
						DependsOn: []string{"instance-a"},
					},
				},
			},
			"module-c": {
				Name: "module-c",
				Path: "/path/to/c",
				Instances: map[string]*Instance{
					"instance-c": {
						Name:      "instance-c",
						DependsOn: []string{"instance-b1", "instance-b2"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"instance-c"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	order := builder.GetExecutionOrder(graph)

	// Should have 3 stages
	if len(order) != 3 {
		t.Fatalf("Expected 3 stages, got %d", len(order))
	}

	// Stage 0: instance-a
	if len(order[0]) != 1 || order[0][0] != "instance-a" {
		t.Errorf("Expected stage 0 to be [instance-a], got %v", order[0])
	}

	// Stage 1: instance-b1, instance-b2 (sorted alphabetically)
	if len(order[1]) != 2 {
		t.Errorf("Expected stage 1 to have 2 instances, got %d", len(order[1]))
	}
	if order[1][0] != "instance-b1" || order[1][1] != "instance-b2" {
		t.Errorf("Expected stage 1 to be [instance-b1, instance-b2], got %v", order[1])
	}

	// Stage 2: instance-c
	if len(order[2]) != 1 || order[2][0] != "instance-c" {
		t.Errorf("Expected stage 2 to be [instance-c], got %v", order[2])
	}
}

// TestGetRootNodes tests getting root nodes (no dependencies)
func TestGetRootNodes(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"root1": {Name: "root1"},
					"root2": {Name: "root2"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"child": {
						Name:      "child",
						DependsOn: []string{"root1"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"child", "root2"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	roots := builder.GetRootNodes(graph)

	// Should have 2 root nodes (sorted)
	if len(roots) != 2 {
		t.Fatalf("Expected 2 root nodes, got %d", len(roots))
	}

	if roots[0] != "root1" || roots[1] != "root2" {
		t.Errorf("Expected roots [root1, root2], got %v", roots)
	}
}

// TestGetLeafNodes tests getting leaf nodes (no dependents)
func TestGetLeafNodes(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"root": {Name: "root"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"leaf1": {
						Name:      "leaf1",
						DependsOn: []string{"root"},
					},
					"leaf2": {
						Name:      "leaf2",
						DependsOn: []string{"root"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"leaf1", "leaf2"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	leaves := builder.GetLeafNodes(graph)

	// Should have 2 leaf nodes (sorted)
	if len(leaves) != 2 {
		t.Fatalf("Expected 2 leaf nodes, got %d", len(leaves))
	}

	if leaves[0] != "leaf1" || leaves[1] != "leaf2" {
		t.Errorf("Expected leaves [leaf1, leaf2], got %v", leaves)
	}
}

// TestValidateGraph tests graph validation
func TestValidateGraph(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"instance-a": {Name: "instance-a"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"instance-b": {
						Name:      "instance-b",
						DependsOn: []string{"instance-a"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"instance-b"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Valid graph should pass validation
	if err := builder.ValidateGraph(graph); err != nil {
		t.Errorf("Expected valid graph, got error: %v", err)
	}
}

// TestGetUpstreamDependencies tests getting all upstream dependencies
func TestGetUpstreamDependencies(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"a": {Name: "a"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"b": {
						Name:      "b",
						DependsOn: []string{"a"},
					},
				},
			},
			"module-c": {
				Name: "module-c",
				Path: "/path/to/c",
				Instances: map[string]*Instance{
					"c": {
						Name:      "c",
						DependsOn: []string{"b"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"c"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	upstream := builder.GetUpstreamDependencies(graph, "c")

	// Should have 2 upstream dependencies: a, b (sorted)
	if len(upstream) != 2 {
		t.Fatalf("Expected 2 upstream dependencies, got %d", len(upstream))
	}

	if upstream[0] != "a" || upstream[1] != "b" {
		t.Errorf("Expected upstream [a, b], got %v", upstream)
	}
}

// TestGetDownstreamDependents tests getting all downstream dependents
func TestGetDownstreamDependents(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"a": {Name: "a"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"b": {
						Name:      "b",
						DependsOn: []string{"a"},
					},
				},
			},
			"module-c": {
				Name: "module-c",
				Path: "/path/to/c",
				Instances: map[string]*Instance{
					"c": {
						Name:      "c",
						DependsOn: []string{"b"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"c"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	downstream := builder.GetDownstreamDependents(graph, "a")

	// Should have 2 downstream dependents: b, c (sorted)
	if len(downstream) != 2 {
		t.Fatalf("Expected 2 downstream dependents, got %d", len(downstream))
	}

	if downstream[0] != "b" || downstream[1] != "c" {
		t.Errorf("Expected downstream [b, c], got %v", downstream)
	}
}

// TestGetSubgraph tests creating a subgraph
func TestGetSubgraph(t *testing.T) {
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"a": {Name: "a"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"b": {
						Name:      "b",
						DependsOn: []string{"a"},
					},
				},
			},
			"module-c": {
				Name: "module-c",
				Path: "/path/to/c",
				Instances: map[string]*Instance{
					"c": {
						Name:      "c",
						DependsOn: []string{"b"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"c"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Get subgraph containing only a and b
	subgraph := builder.GetSubgraph(graph, []string{"a", "b"})

	// Should have 2 nodes
	if len(subgraph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes in subgraph, got %d", len(subgraph.Nodes))
	}

	// Verify nodes exist
	if _, exists := subgraph.Nodes["a"]; !exists {
		t.Error("Expected node 'a' in subgraph")
	}
	if _, exists := subgraph.Nodes["b"]; !exists {
		t.Error("Expected node 'b' in subgraph")
	}

	// Verify c is not in subgraph
	if _, exists := subgraph.Nodes["c"]; exists {
		t.Error("Did not expect node 'c' in subgraph")
	}

	// Verify edge between a and b exists
	edges := subgraph.Edges["a"]
	if len(edges) != 1 || edges[0] != "b" {
		t.Errorf("Expected edge from a to b, got %v", edges)
	}
}

// TestParseModuleReference tests parsing module references
func TestParseModuleReference(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		moduleRef    string
		wantModule   string
		wantInstance string
	}{
		{
			name: "instance name",
			config: &Config{
				Modules: map[string]*Module{
					"module-a": {
						Name: "module-a",
						Path: "/path/to/a",
						Instances: map[string]*Instance{
							"my-instance": {Name: "my-instance"},
						},
					},
				},
			},
			moduleRef:    "my-instance",
			wantModule:   "module-a",
			wantInstance: "my-instance",
		},
		{
			name: "module.instance format",
			config: &Config{
				Modules: map[string]*Module{
					"module-a": {
						Name: "module-a",
						Path: "/path/to/a",
					},
				},
			},
			moduleRef:    "module-a.instance-x",
			wantModule:   "module-a",
			wantInstance: "instance-x",
		},
		{
			name: "module name only",
			config: &Config{
				Modules: map[string]*Module{
					"module-a": {
						Name: "module-a",
						Path: "/path/to/a",
					},
				},
			},
			moduleRef:    "module-a",
			wantModule:   "module-a",
			wantInstance: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewDependencyGraphBuilder(tt.config)
			gotModule, gotInstance := builder.parseModuleReference(tt.moduleRef)

			if gotModule != tt.wantModule {
				t.Errorf("parseModuleReference() module = %v, want %v", gotModule, tt.wantModule)
			}
			if gotInstance != tt.wantInstance {
				t.Errorf("parseModuleReference() instance = %v, want %v", gotInstance, tt.wantInstance)
			}
		})
	}
}

// TestBuildGraph_ComplexDiamond tests diamond dependency pattern
func TestBuildGraph_ComplexDiamond(t *testing.T) {
	//     a
	//    / \
	//   b   c
	//    \ /
	//     d
	config := &Config{
		Modules: map[string]*Module{
			"module-a": {
				Name: "module-a",
				Path: "/path/to/a",
				Instances: map[string]*Instance{
					"a": {Name: "a"},
				},
			},
			"module-b": {
				Name: "module-b",
				Path: "/path/to/b",
				Instances: map[string]*Instance{
					"b": {
						Name:      "b",
						DependsOn: []string{"a"},
					},
				},
			},
			"module-c": {
				Name: "module-c",
				Path: "/path/to/c",
				Instances: map[string]*Instance{
					"c": {
						Name:      "c",
						DependsOn: []string{"a"},
					},
				},
			},
			"module-d": {
				Name: "module-d",
				Path: "/path/to/d",
				Instances: map[string]*Instance{
					"d": {
						Name:      "d",
						DependsOn: []string{"b", "c"},
					},
				},
			},
		},
	}

	builder := NewDependencyGraphBuilder(config)
	graph, err := builder.BuildGraph([]string{"d"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify stages
	if graph.Nodes["a"].Stage != 0 {
		t.Errorf("Expected 'a' to be stage 0, got %d", graph.Nodes["a"].Stage)
	}
	if graph.Nodes["b"].Stage != 1 {
		t.Errorf("Expected 'b' to be stage 1, got %d", graph.Nodes["b"].Stage)
	}
	if graph.Nodes["c"].Stage != 1 {
		t.Errorf("Expected 'c' to be stage 1, got %d", graph.Nodes["c"].Stage)
	}
	if graph.Nodes["d"].Stage != 2 {
		t.Errorf("Expected 'd' to be stage 2, got %d", graph.Nodes["d"].Stage)
	}

	// Verify execution order
	order := builder.GetExecutionOrder(graph)
	if len(order) != 3 {
		t.Fatalf("Expected 3 stages in execution order, got %d", len(order))
	}

	// Stage 0: a
	if len(order[0]) != 1 || order[0][0] != "a" {
		t.Errorf("Expected stage 0 to be [a], got %v", order[0])
	}

	// Stage 1: b, c (can run in parallel)
	if len(order[1]) != 2 {
		t.Errorf("Expected stage 1 to have 2 instances, got %d", len(order[1]))
	}

	// Stage 2: d
	if len(order[2]) != 1 || order[2][0] != "d" {
		t.Errorf("Expected stage 2 to be [d], got %v", order[2])
	}
}
