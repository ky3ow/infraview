package graph_test

import (
	"os"
	"testing"

	_ "github.com/hashicorp/terraform-json"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/ky3ow/infraview/pkg/graph"
)

const (
	SIMPLE = "tfcoremock_simple_resource"
)

func TestSimpleResource(t *testing.T) {
	planName := "data/00-simple-resource.json"
	file, err := os.Open(planName)

	if err != nil {
		t.Errorf("No file: %s", planName)
	}

	got, err := graph.New(file)

	if err != nil {
		t.Errorf("%v", err)
	}

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
		"bool": true,
		"float": 2.0,
		"id": "id",
		"integer": 3.0, // integers are not integers in terraform
		"number": nil,
		"string": "string",
	}

	for _, param := range resource.Parameters {
		if _, ok := expectedParams[param]; !ok {
			t.Errorf("Param %v not found", param)
		}
	}

	for k, v := range resource.Changes.After.(map[string]any) {
		if expectedParams[k] != v {
			t.Errorf("Expected param(%v) %v, got: %v", k, expectedParams[k], v)
		}
	}
}

func TestComplexResource(t *testing.T) {
	planName := "data/01-complex-resource.json"
	file, err := os.Open(planName)

	if err != nil {
		t.Errorf("No file: %s", planName)
	}

	got, err := graph.New(file)

	if err != nil {
		t.Errorf("%v", err)
	}

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
	planName := "data/02-simple-create-update-delete.json"
	file, err := os.Open(planName)

	if err != nil {
		t.Errorf("No file: %s", planName)
	}

	got, err := graph.New(file)

	if err != nil {
		t.Errorf("%v", err)
	}

	// check green yellow and red exist and represent proper action to apply
	green, ok := got.Nodes["tfcoremock_simple_resource.green"]

	if !ok {
		t.Errorf("green is missing")
	}

	if green.Changes == nil || green.Changes.Actions[0] != "create" {
		t.Errorf("green should create")
	}

	yellow, ok := got.Nodes["tfcoremock_simple_resource.yellow"]

	if !ok {
		t.Errorf("yellow is missing")
	}

	if yellow.Changes == nil || yellow.Changes.Actions[0] != "update" {
		t.Errorf("yellow should update")
	}

	red, ok := got.Nodes["tfcoremock_simple_resource.red"]

	if !ok {
		t.Errorf("red is missing")
	}

	if red.Changes == nil || red.Changes.Actions[0] != "delete" {
		t.Errorf("red should update")
	}
}
