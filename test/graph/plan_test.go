// Tests for plans.
// Throwaway tests just to verify that plan parses terraform feature
// into graph properly. E.g. created modules generate proper graph,
// deleted modules generate proper graph, etc.
package graph_test

import (
	"encoding/json"
	"os"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/ky3ow/infraview/pkg/graph"
)

func TestSimpleResource(t *testing.T) {
	got := setup(t, "00-simple-resource/plan.json")

	expectedNodes := 1
	if len(got.Nodes) != expectedNodes {
		t.Errorf("Expected number of nodes: %v, got: %v", expectedNodes, len(got.Nodes))
	}

	expectedEdges := 0
	if len(got.Edges) != expectedEdges {
		t.Errorf("Expected number of nodes: %v, got: %v", expectedEdges, len(got.Edges))
	}

	resource, ok := got.Nodes["tfcoremock_simple_resource.simple"]
	if !ok {
		t.Errorf("Simple mock resource did not parse")
	}

	expectedName := "simple"
	if resource.Name != expectedName {
		t.Errorf("Expected name: %v, got: %v", resource.Name, expectedName)
	}

	expectedMode := tfjson.ManagedResourceMode
	if resource.Mode != expectedMode {
		t.Errorf("Expected mode: %v, got: %v", resource.Mode, expectedMode)
	}

	expectedParams := map[string]any{
		"bool":    true,
		"float":   2.0,
		"id":      "id",
		"integer": 3.0, // integers are not integers in terraform
		"number":  nil,
		"string":  "string",
	}

	for _, param := range resource.Parameters {
		if _, ok := expectedParams[param]; !ok {
			t.Errorf("Param %v not found", param)
		}
	}

	for k, v := range resource.Changes.After {
		if expectedParams[k] != v {
			t.Errorf("Expected param(%v) %v, got: %v", k, expectedParams[k], v)
		}
	}
}

func TestComplexResource(t *testing.T) {
	got := setup(t, "01-complex-resource/plan.json")

	expectedNodes := 1
	if len(got.Nodes) != expectedNodes {
		t.Errorf("Expected number of nodes: %v, got: %v", expectedNodes, len(got.Nodes))
	}

	expectedEdges := 0
	if len(got.Edges) != expectedEdges {
		t.Errorf("Expected number of nodes: %v, got: %v", expectedEdges, len(got.Edges))
	}

	// TODO blocks as subresources
}

func TestGreenYellowRed(t *testing.T) {
	got := setup(t, "02-simple-create-update-delete/plan.json")

	// check green yellow and red exist and represent proper action to apply
	green, ok := got.Nodes["tfcoremock_simple_resource.green"]

	if !ok {
		t.Errorf("green is missing")
	}

	if green.Changes == nil || green.State != graph.StateCreate {
		t.Errorf("green should create")
	}

	yellow, ok := got.Nodes["tfcoremock_simple_resource.yellow"]

	if !ok {
		t.Errorf("yellow is missing")
	}

	if yellow.Changes == nil || yellow.State != graph.StateModify {
		t.Errorf("yellow should update")
	}

	red, ok := got.Nodes["tfcoremock_simple_resource.red"]

	if !ok {
		t.Errorf("red is missing")
	}

	if red.Changes == nil || red.State != graph.StateDelete {
		t.Errorf("red should delete")
	}
}

func TestSimpleDependencies(t *testing.T) {
	got := setup(t, "03-simple-dependencies/plan.json")

	expectedEdges, gotEdges := 2, len(got.Edges)
	if gotEdges != expectedEdges {
		t.Errorf("Expected %v edges, got %v", expectedEdges, gotEdges)
	}
	expectEdge(t, got, "tfcoremock_simple_resource.original", "tfcoremock_simple_resource.implicit", "Edge")
	expectEdge(t, got, "tfcoremock_simple_resource.original", "tfcoremock_simple_resource.explicit", "Edge")
}

func TestSimpleVars(t *testing.T) {
	got := setup(t, "04-simple-vars/plan.json")

	expectNode(t, got, "var1", "var.var1", "Root var with default")
	expectNode(t, got, "var2", "var.var2", "Root var without default")

	if len(got.Edges) != 2 {
		t.Errorf("Unexpected edges, want 2, got: %v", got.Edges)
	}

	expectEdge(t, got, "var.var1", "tfcoremock_simple_resource.simple", "Edge to root var")
	expectEdge(t, got, "var.var2", "tfcoremock_simple_resource.simple", "Edge to root var")
}

func TestModuleVars(t *testing.T) {
	got := setup(t, "05-module-vars/plan.json")

	expectedEdges := 6
	if len(got.Edges) != expectedEdges {
		t.Errorf("Expected %d edges, got %v", expectedEdges, got.Edges)
	}

	expectNode(t, got, "var1_module", "module.sample.var.var1_module", "1st level Module var with default")
	expectNode(t, got, "var2_module", "module.sample.var.var2_module", "1st level Module var without default")

	expectEdge(t, got, "module.sample.var.var1_module", "module.sample.tfcoremock_simple_resource.simple", "1st level Module var edge inside")
	expectEdge(t, got, "module.sample.var.var2_module", "module.sample.tfcoremock_simple_resource.simple", "1st level Module var edge inside")

	expectEdge(t, got, "var.var1_root", "module.sample.var.var1_module", "1st level Module var edge outside")
	expectEdge(t, got, "var.var2_root", "module.sample.var.var2_module", "1st level Module var edge outside")

	expectNode(t, got, "var1_nested", "module.sample.module.sample_child.var.var1_nested", "2nd level nested module var")

	expectEdge(t, got, "module.sample.module.sample_child.var.var1_nested", "module.sample.module.sample_child.tfcoremock_simple_resource.simple", "2nd level nested Module var edge inside")
	expectEdge(t, got, "module.sample.var.var1_module", "module.sample.module.sample_child.var.var1_nested", "2nd level nested Module var edge outside")

	if t.Failed() {
		t.Errorf("Nodes: %v", got.Nodes)
		t.Errorf("Edges: %v", got.Edges)
	}
}

func TestSimpleOutputs(t *testing.T) {
	got := setup(t, "06-simple-outputs/plan.json")

	expectNode(t, got, "simple_int", "simple_int", "simple output")
	expectEdge(t, got, "tfcoremock_simple_resource.simple", "simple_int", "simple output")
}

func TestModuleOutputs(t *testing.T) {
	got := setup(t, "07-module-outputs/plan.json")

	expectNode(t, got, "outer", "outer", "simple output")
	expectEdge(t, got, "module.sample.two_passthrough_output", "outer", "simple output")

	expectNode(t, got, "one_output", "module.sample.one_output", "module output")
	expectEdge(t, got, "module.sample.tfcoremock_simple_resource.simple", "module.sample.one_output", "module output")

	expectNode(t, got, "two_passthrough_output", "module.sample.two_passthrough_output", "module passthrough output")
	expectEdge(t, got, "module.sample.module.sample_child.two_output", "module.sample.two_passthrough_output", "module passthrough output")

	expectNode(t, got, "sample_child", "module.sample.module.sample_child", "child module")
	expectEdge(t, got, "module.sample.module.sample_child.tfcoremock_simple_resource.simple", "module.sample.module.sample_child.two_output", "module passthrough output")
}

func TestComplexVars(t *testing.T) {
	got := setup(t, "08-complex-vars/plan.json")

	expectNode(t, got, "name", "tfcoremock_simple_resource.name", "simple resource")
	expectEdge(t, got, "tfcoremock_simple_resource.name", "module.sample.var.var2_module", "root resource to module var")

	expectEdge(t, got, "module.sample.tfcoremock_simple_resource.simple", "module.sample.module.sample_child.var.var1_nested", "module resource to nested module var")
}

func TestNestedRed(t *testing.T) {
	// TODO test tested deletion
}

func TestData(t *testing.T) {
	got := setup(t, "09-data-resources/plan.json")

	expectNode(t, got, "name", "tfcoremock_simple_resource.name", "simple resource")
	expectNode(t, got, "external", "data.tfcoremock_simple_resource.external", "simple data resource")

	expectNode(t, got, "name_module", "module.test.tfcoremock_simple_resource.name_module", "simple resource in module")
	expectNode(t, got, "external_module", "module.test.data.tfcoremock_simple_resource.external_module", "simple data resource in module")

	expectEdge(t, got, "data.tfcoremock_simple_resource.external", "tfcoremock_simple_resource.name", "edges from data resources in root")

	expectEdge(t, got, "module.test.data.tfcoremock_simple_resource.external_module", "module.test.tfcoremock_simple_resource.name_module", "edges from data resources in root")
}

func TestDeletedModule(t *testing.T) {
	// TODO enhance EDGES of deleted nodes INSIDE modules
	got := setup(t, "10-module-deletions/plan.json")
	expectNode(t, got, "sample2", "module.sample2", "alive module")
	expectNode(t, got, "another_simple3", "module.sample2.tfcoremock_simple_resource.another_simple3", "alive resource")
	expectNode(t, got, "another_simple4", "module.sample2.tfcoremock_simple_resource.another_simple4", "dead resource")

	expectEdge(t, got, "module.sample2.tfcoremock_simple_resource.another_simple3", "module.sample2.tfcoremock_simple_resource.another_simple4", "alive resource to dead")

	expectNode(t, got, "simple", "module.sample.tfcoremock_simple_resource.simple", "dead resource from dead module")
	expectNode(t, got, "sample_child", "module.sample.module.sample_child", "dead nested module")

	expectNode(t, got, "simple", "module.sample.module.sample_child.tfcoremock_simple_resource.simple", "dead module dead resource")
}

func TestDeleted(t *testing.T) {
	got := setup(t, "11-deletion-edges/plan.json")

	expectNode(t, got, "gray", "tfcoremock_simple_resource.gray", "dependency")
	expectNode(t, got, "red", "tfcoremock_simple_resource.red", "deleted node")
	expectNode(t, got, "red2", "tfcoremock_simple_resource.red2", "deleted node")

	expectEdge(t, got, "tfcoremock_simple_resource.gray", "tfcoremock_simple_resource.red", "deleted implicit edge")
	expectEdge(t, got, "tfcoremock_simple_resource.gray", "tfcoremock_simple_resource.red2", "deleted explicit edge")
}

func TestCount(t *testing.T) {
	got := setup(t, "12-create-count/plan.json")

	expectNode(t, got, "counted", "tfcoremock_simple_resource.counted", "parent node")
	expectContainer(t, got, "tfcoremock_simple_resource.counted", "parent node is container")
	expectNode(t, got, "dep", "tfcoremock_simple_resource.dep", "dependant")

	expectNode(t, got, "counted[0]", "tfcoremock_simple_resource.counted[0]", "child node")
	expectNode(t, got, "counted[1]", "tfcoremock_simple_resource.counted[1]", "child node")
	expectNode(t, got, "counted[2]", "tfcoremock_simple_resource.counted[2]", "child node")

	expectParent(t, got, "tfcoremock_simple_resource.counted[0]", "tfcoremock_simple_resource.counted", "child node parent")
	expectParent(t, got, "tfcoremock_simple_resource.counted[1]", "tfcoremock_simple_resource.counted", "child node parent")
	expectParent(t, got, "tfcoremock_simple_resource.counted[2]", "tfcoremock_simple_resource.counted", "child node parent")

	expectEdge(t, got, "var.a", "tfcoremock_simple_resource.counted", "var to parent node")
	expectEdge(t, got, "tfcoremock_simple_resource.counted[1]", "tfcoremock_simple_resource.dep", "dependency to instance")
	expectEdge(t, got, "tfcoremock_simple_resource.counted", "tfcoremock_simple_resource.dep", "dependency to parent")
}

func TestForEach(t *testing.T) {
	got := setup(t, "13-create-foreach/plan.json")

	expectNode(t, got, "counted", "tfcoremock_simple_resource.counted", "parent node")
	expectContainer(t, got, "tfcoremock_simple_resource.counted", "parent node is container")
	expectNode(t, got, "dep", "tfcoremock_simple_resource.dep", "dependant")

	expectNode(t, got, "counted[\"one\"]", "tfcoremock_simple_resource.counted[\"one\"]", "child node")
	expectNode(t, got, "counted[\"two\"]", "tfcoremock_simple_resource.counted[\"two\"]", "child node")

	expectParent(t, got, "tfcoremock_simple_resource.counted[\"one\"]", "tfcoremock_simple_resource.counted", "child node parent")
	expectParent(t, got, "tfcoremock_simple_resource.counted[\"two\"]", "tfcoremock_simple_resource.counted", "child node parent")

	expectEdge(t, got, "var.a", "tfcoremock_simple_resource.counted", "var to parent node")
	expectEdge(t, got, "tfcoremock_simple_resource.counted[\"one\"]", "tfcoremock_simple_resource.dep", "dependency to instance")
	expectEdge(t, got, "tfcoremock_simple_resource.counted", "tfcoremock_simple_resource.dep", "dependency to parent")
}

func TestDeleteCount(t *testing.T) {
	got := setup(t, "14-delete-count/plan.json")

	expectNode(t, got, "counted", "tfcoremock_simple_resource.counted", "deleted parent node")
	expectContainer(t, got, "tfcoremock_simple_resource.counted", "deleted parent is container")
	expectState(t, got, "tfcoremock_simple_resource.counted", graph.StateDelete, "fully deleted container")

	expectNode(t, got, "counted[0]", "tfcoremock_simple_resource.counted[0]", "deleted child")
	expectNode(t, got, "counted[1]", "tfcoremock_simple_resource.counted[1]", "deleted child")
	expectNode(t, got, "counted[2]", "tfcoremock_simple_resource.counted[2]", "deleted child")
	expectState(t, got, "tfcoremock_simple_resource.counted[0]", graph.StateDelete, "instance state")
	expectState(t, got, "tfcoremock_simple_resource.counted[1]", graph.StateDelete, "instance state")
	expectState(t, got, "tfcoremock_simple_resource.counted[2]", graph.StateDelete, "instance state")
	expectParent(t, got, "tfcoremock_simple_resource.counted[0]", "tfcoremock_simple_resource.counted", "child parent")
	expectParent(t, got, "tfcoremock_simple_resource.counted[1]", "tfcoremock_simple_resource.counted", "child parent")
	expectParent(t, got, "tfcoremock_simple_resource.counted[2]", "tfcoremock_simple_resource.counted", "child parent")

	expectNode(t, got, "plus", "tfcoremock_simple_resource.plus", "partial delete parent")
	expectContainer(t, got, "tfcoremock_simple_resource.plus", "partial parent is container")
	expectState(t, got, "tfcoremock_simple_resource.plus", graph.StateModify, "partially deleted container")

	expectNode(t, got, "plus[0]", "tfcoremock_simple_resource.plus[0]", "kept instance")
	expectState(t, got, "tfcoremock_simple_resource.plus[0]", graph.StateNoop, "kept instance state")
	expectNode(t, got, "plus[1]", "tfcoremock_simple_resource.plus[1]", "deleted instance")
	expectState(t, got, "tfcoremock_simple_resource.plus[1]", graph.StateDelete, "deleted instance state")
	expectNode(t, got, "plus[2]", "tfcoremock_simple_resource.plus[2]", "deleted instance")
	expectState(t, got, "tfcoremock_simple_resource.plus[2]", graph.StateDelete, "deleted instance state")
	expectNode(t, got, "plus[3]", "tfcoremock_simple_resource.plus[3]", "deleted instance")
	expectState(t, got, "tfcoremock_simple_resource.plus[3]", graph.StateDelete, "deleted instance state")

	expectNode(t, got, "dep", "tfcoremock_simple_resource.dep", "deleted dependant")
	expectState(t, got, "tfcoremock_simple_resource.dep", graph.StateDelete, "dep state")

	expectEdge(t, got, "tfcoremock_simple_resource.counted", "tfcoremock_simple_resource.dep", "deleted container to deleted dep")
}

// Verifies transitive edge collapse on deleted chain.
// Chain: one -> two -> three -> four/also_four
// Chain links should be direct, all shortcuts should stay transitive.
func TestTransitiveEdgeCollapse(t *testing.T) {
	got := setup(t, "15-test-collapse/plan.json")

	// Chain edges - indispensable, should be direct
	expectEdgeOfType(t, got, "tfcoremock_simple_resource.one", "tfcoremock_simple_resource.two", "direct", "direct first dependency")
	expectEdgeOfType(t, got, "tfcoremock_simple_resource.two", "tfcoremock_simple_resource.three", "direct", "direct first dependency")
	expectEdgeOfType(t, got, "tfcoremock_simple_resource.three", "tfcoremock_simple_resource.four", "direct", "direct first dependency")
	expectEdgeOfType(t, got, "tfcoremock_simple_resource.three", "tfcoremock_simple_resource.also_four", "direct", "direct first dependency")

	// Shortcut edges - covered by chain, should stay transitive
	expectEdgeOfType(t, got, "tfcoremock_simple_resource.one", "tfcoremock_simple_resource.three", "transitive", "shortcut one->three should stay transitive")
	expectEdgeOfType(t, got, "tfcoremock_simple_resource.one", "tfcoremock_simple_resource.four", "transitive", "transitive descendant")
	expectEdgeOfType(t, got, "tfcoremock_simple_resource.one", "tfcoremock_simple_resource.also_four", "transitive", "transitive descendant")
	expectEdgeOfType(t, got, "tfcoremock_simple_resource.two", "tfcoremock_simple_resource.four", "transitive", "transitive descendant")
	expectEdgeOfType(t, got, "tfcoremock_simple_resource.two", "tfcoremock_simple_resource.also_four", "transitive", "transitive descendant")
}

func setup(t *testing.T, planPath string) *graph.Graph {
	t.Helper()
	file, err := os.Open(planPath)

	if err != nil {
		t.Errorf("No file: %s", planPath)
	}

	var plan tfjson.Plan

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&plan); err != nil {
		t.Errorf("%v", err)
	}

	got, err := graph.New(&plan)

	if err != nil {
		t.Errorf("%v", err)
	}

	return got
}

func expectNode(t *testing.T, g *graph.Graph, name, address, context string) {
	t.Helper()
	node, ok := g.Nodes[address]

	if !ok {
		t.Errorf("[%v]:: %v not found", context, name)
	}

	if node.Name != name {
		t.Errorf("[%v]:: parsing does not yield proper name, got '%v', expected '%v'", context, node.Name, name)
	}

	if node.Address != address {
		t.Errorf("[%v]:: parsing does not yield proper address, got '%v', expected '%v'", context, node.Address, address)
	}
}

func expectParent(t *testing.T, g *graph.Graph, child, parent, context string) {
	t.Helper()
	node, ok := g.Nodes[child]

	if !ok {
		t.Errorf("[%v]:: %v not found", context, child)
	}

	if node.Parent != parent {
		t.Errorf("[%v]:: parent for %v not found, expected %v", context, child, parent)
	}

}

func expectContainer(t *testing.T, g *graph.Graph, addr, context string) {
	t.Helper()
	node, ok := g.Nodes[addr]

	if !ok {
		t.Errorf("[%v]:: %v not found", context, addr)
	}

	if node.Kind != graph.LoopResourceNode {
		t.Errorf("[%v]:: %v is not container type", context, addr)
	}
}

func expectEdge(t *testing.T, g *graph.Graph, a, b, context string) {
	t.Helper()
	edges := g.Edges

	found := false
	for _, edge := range edges {
		// if (edge.From == a && edge.To == b) || (edge.From == b && edge.To == a) {
		if edge.From == a && edge.To == b {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("[%v]:: edge '%v' -> '%v' not found", context, a, b)
	}
}

func expectEdgeOfType(t *testing.T, g *graph.Graph, a, b, expectedType, context string) {
	t.Helper()
	found := false
	for _, edge := range g.Edges {
		if edge.From == a && edge.To == b {
			found = true
			if string(edge.Type) != expectedType {
				t.Errorf("[%v]:: edge '%v' -> '%v' expected type '%v', got '%v'", context, a, b, expectedType, edge.Type)
			}
			break
		}
	}
	if !found {
		t.Errorf("[%v]:: edge '%v' -> '%v' not found", context, a, b)
	}
}

func expectState(t *testing.T, g *graph.Graph, addr string, expected graph.ResourceState, context string) {
	t.Helper()
	node, ok := g.Nodes[addr]
	if !ok {
		t.Errorf("[%v]:: %v not found", context, addr)
		return
	}
	if node.State != expected {
		t.Errorf("[%v]:: expected state %v, got %v", context, expected, node.State)
	}
}

