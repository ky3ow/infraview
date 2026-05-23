package catalog_test

import (
	"os"
	"testing"

	"github.com/ky3ow/infraview/pkg/catalog"
	"github.com/ky3ow/infraview/pkg/graph"
	"github.com/ky3ow/infraview/pkg/storage"
)

func newStore(t *testing.T) *storage.Store {
	t.Helper()
	f, err := os.CreateTemp("", "infraview_*.db")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(path) })

	s, err := storage.New(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAddRevision(t *testing.T) {
	s := newStore(t)

	rev, err := catalog.AddRevision(s, "production", "abc123", nil, catalog.PlanMetadata{
		Created: 3,
		Updated: 2,
		Deleted: 1,
	})
	if err != nil {
		t.Fatalf("AddRevision: %v", err)
	}

	if rev.Hash != "abc123" {
		t.Fatalf("expected hash %q, got %q", "abc123", rev.Hash)
	}
	if rev.Summary.Created != 3 || rev.Summary.Updated != 2 || rev.Summary.Deleted != 1 {
		t.Fatalf("expected 3/2/1, got %d/%d/%d", rev.Summary.Created, rev.Summary.Updated, rev.Summary.Deleted)
	}
}

func TestAddRevisionNameSteals(t *testing.T) {
	s := newStore(t)

	name1 := "main"
	rev1, err := catalog.AddRevision(s, "testns", "aaa", &name1, catalog.PlanMetadata{})
	if err != nil {
		t.Fatalf("rev1: %v", err)
	}
	if rev1.Name != "main" {
		t.Fatalf("expected name %q, got %q", "main", rev1.Name)
	}

	// Same name, different hash — should steal
	name2 := "main"
	rev2, err := catalog.AddRevision(s, "testns", "bbb", &name2, catalog.PlanMetadata{})
	if err != nil {
		t.Fatalf("rev2: %v", err)
	}
	if rev2.Name != "main" {
		t.Fatalf("expected rev2 name %q, got %q", "main", rev2.Name)
	}

	// Old revision lost its name
	oldRev, err := s.GetRevisionByID(rev1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if oldRev.Name != nil {
		t.Fatalf("expected old revision name to be nil, got %q", *oldRev.Name)
	}
}

func TestAddRevisionUpsert(t *testing.T) {
	s := newStore(t)

	rev1, err := catalog.AddRevision(s, "testns", "aaa", nil, catalog.PlanMetadata{Created: 1})
	if err != nil {
		t.Fatal(err)
	}

	// Same hash — should update in place
	rev2, err := catalog.AddRevision(s, "testns", "aaa", nil, catalog.PlanMetadata{Created: 5})
	if err != nil {
		t.Fatal(err)
	}

	if rev2.ID != rev1.ID {
		t.Fatalf("expected same ID %d, got %d", rev1.ID, rev2.ID)
	}
	if rev2.Summary.Created != 5 {
		t.Fatalf("expected updated creates=5, got %d", rev2.Summary.Created)
	}
}

func TestAddRevisionNoCrossNamespaceSteal(t *testing.T) {
	s := newStore(t)

	name := "main"
	rev1, err := catalog.AddRevision(s, "ns1", "aaa", &name, catalog.PlanMetadata{})
	if err != nil {
		t.Fatal(err)
	}

	// Same name in different namespace — should NOT steal
	name2 := "main"
	rev2, err := catalog.AddRevision(s, "ns2", "bbb", &name2, catalog.PlanMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if rev2.Name != "main" {
		t.Fatalf("expected ns2 rev name %q, got %q", "main", rev2.Name)
	}

	// Original in ns1 should still have its name
	ns1Rev, err := s.GetRevisionByID(rev1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ns1Rev.Name == nil || *ns1Rev.Name != "main" {
		t.Fatalf("expected ns1 rev to still have name %q, got %v", "main", ns1Rev.Name)
	}
}

func TestSavePlanRoundTrip(t *testing.T) {
	s := newStore(t)

	name := "main"
	g := &graph.Graph{
		Nodes: map[string]graph.Node{
			"aws_instance.web": {
				Kind:    graph.ResourceNode,
				Name:    "web",
				Type:    "aws_instance",
				State:   graph.StateCreate,
				Address: "aws_instance.web",
			},
		},
		Edges: []graph.Edge{},
	}

	rev, err := catalog.SavePlan(s, "production", "abc123", &name, catalog.PlanMetadata{
		Created: 1,
	}, g)
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	if rev.Hash != "abc123" {
		t.Fatalf("expected hash abc123, got %q", rev.Hash)
	}
	if rev.Summary.Created != 1 {
		t.Fatalf("expected creates=1, got %d", rev.Summary.Created)
	}

	loaded, err := s.GetGraph(rev.ID)
	if err != nil {
		t.Fatalf("GetGraph: %v", err)
	}

	if len(loaded.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(loaded.Nodes))
	}
	node := loaded.Nodes["aws_instance.web"]
	if node.State != graph.StateCreate {
		t.Fatalf("expected state create, got %q", node.State)
	}
}

func TestListNamespaces(t *testing.T) {
	s := newStore(t)

	// Empty initially
	namespaces, err := catalog.ListNamespaces(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != 0 {
		t.Fatalf("expected 0 namespaces, got %d", len(namespaces))
	}

	catalog.SavePlan(s, "production", "aaa", nil, catalog.PlanMetadata{}, &graph.Graph{Nodes: map[string]graph.Node{}})
	catalog.SavePlan(s, "staging", "bbb", nil, catalog.PlanMetadata{}, &graph.Graph{Nodes: map[string]graph.Node{}})

	namespaces, err = catalog.ListNamespaces(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(namespaces))
	}
}

func TestListRevisions(t *testing.T) {
	s := newStore(t)

	name := "main"
	rev1, _ := catalog.SavePlan(s, "production", "aaa", &name, catalog.PlanMetadata{Created: 1}, &graph.Graph{Nodes: map[string]graph.Node{}})
	_, _ = catalog.SavePlan(s, "production", "bbb", nil, catalog.PlanMetadata{Updated: 2}, &graph.Graph{Nodes: map[string]graph.Node{}})

	revisions, err := catalog.ListRevisions(s, rev1.NamespaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 2 {
		t.Fatalf("expected 2 revisions, got %d", len(revisions))
	}

	// Newest first
	if revisions[0].Hash != "bbb" {
		t.Fatalf("first should be bbb, got %q", revisions[0].Hash)
	}
	if revisions[1].Name != "main" {
		t.Fatalf("second revision should have name main, got %q", revisions[1].Name)
	}
}

func TestGetRevisionGraph(t *testing.T) {
	s := newStore(t)

	name := "prod"
	g := &graph.Graph{
		Nodes: map[string]graph.Node{
			"aws_instance.web": {Kind: graph.ResourceNode, Name: "web", Address: "aws_instance.web"},
		},
		Edges: []graph.Edge{},
	}
	rev, _ := catalog.SavePlan(s, "production", "abc123", &name, catalog.PlanMetadata{Created: 1}, g)

	loaded, err := catalog.GetRevisionGraph(s, rev.ID)
	if err != nil {
		t.Fatalf("GetRevisionGraph: %v", err)
	}
	if len(loaded.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(loaded.Nodes))
	}
}
