package graph_test

import (
	"testing"
	"github.com/ky3ow/infraview/pkg/graph"
)

func TestDummyPass(t *testing.T) {
	t.Helper()

	res := graph.SomethingUseful(1, 2)

	if res != 3 {
		t.Error("Why not 3")
	}
}
