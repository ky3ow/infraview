package graph_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ky3ow/infraview/pkg/graph"
)

func TestPlan(t *testing.T) {
	file, err := os.Open("data/module-changes.json")
	if err != nil {
		t.Error("No file")
	}

	graph, err := graph.New(file)

	if err != nil {
		t.Error("Invalid plan")
	}

	j, err := json.Marshal(graph)

	t.Logf(string(j))
}
