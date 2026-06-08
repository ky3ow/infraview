package storage

import (
	"errors"
	"os"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/ky3ow/infraview/pkg/graph"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	f, err := os.CreateTemp("", "infraview_*.db")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(path) })

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestEnsureTables(t *testing.T) {
	s := newTestStore(t)

	rows, err := s.db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, name)
	}

	expected := []string{"edges", "namespaces", "node_details", "nodes", "revisions", "tokens"}
	filtered := make([]string, 0, len(tables))
	for _, t := range tables {
		if t == "sqlite_sequence" {
			continue
		}
		filtered = append(filtered, t)
	}

	if len(filtered) != len(expected) {
		t.Fatalf("expected %d tables, got %d: %v", len(expected), len(filtered), filtered)
	}
	for i, tbl := range filtered {
		if tbl != expected[i] {
			t.Fatalf("expected table %q at pos %d, got %q", expected[i], i, tbl)
		}
	}
}

func TestEnsureNamespace(t *testing.T) {
	s := newTestStore(t)

	// First call - should create
	ns1, err := s.EnsureNamespace("production")
	if err != nil {
		t.Fatalf("EnsureNamespace(production): %v", err)
	}
	if ns1.Name != "production" {
		t.Fatalf("expected name %q, got %q", "production", ns1.Name)
	}
	if ns1.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	id1 := ns1.ID

	// Second call - should return existing
	ns2, err := s.EnsureNamespace("production")
	if err != nil {
		t.Fatalf("EnsureNamespace(production) again: %v", err)
	}
	if ns2.ID != id1 {
		t.Fatalf("expected same ID %d, got %d", id1, ns2.ID)
	}

	// Different namespace
	ns3, err := s.EnsureNamespace("staging")
	if err != nil {
		t.Fatalf("EnsureNamespace(staging): %v", err)
	}
	if ns3.ID == id1 {
		t.Fatal("expected different ID for staging")
	}
}

func TestUpsertRevision(t *testing.T) {
	s := newTestStore(t)

	ns, err := s.EnsureNamespace("testns")
	if err != nil {
		t.Fatal(err)
	}

	rev, err := s.UpsertRevision(ns.ID, "abc123", nil, 3, 2, 1)
	if err != nil {
		t.Fatalf("UpsertRevision: %v", err)
	}

	if rev.NamespaceID != ns.ID {
		t.Fatalf("expected namespace_id %d, got %d", ns.ID, rev.NamespaceID)
	}
	if rev.Hash != "abc123" {
		t.Fatalf("expected hash %q, got %q", "abc123", rev.Hash)
	}
	if rev.Name != nil {
		t.Fatalf("expected nil name, got %q", *rev.Name)
	}
	if rev.Creates != 3 || rev.Updates != 2 || rev.Deletes != 1 {
		t.Fatalf("expected 3/2/1, got %d/%d/%d", rev.Creates, rev.Updates, rev.Deletes)
	}
	if rev.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestUpsertRevisionSameRow(t *testing.T) {
	s := newTestStore(t)

	ns, err := s.EnsureNamespace("testns")
	if err != nil {
		t.Fatal(err)
	}

	rev1, err := s.UpsertRevision(ns.ID, "aaa", nil, 1, 0, 0)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// Upsert same hash - should update in place, not create a new row
	rev2, err := s.UpsertRevision(ns.ID, "aaa", nil, 5, 3, 2)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if rev2.ID != rev1.ID {
		t.Fatalf("expected same ID %d, got %d", rev1.ID, rev2.ID)
	}

	// Counters should reflect the upsert values
	if rev2.Creates != 5 || rev2.Updates != 3 || rev2.Deletes != 2 {
		t.Fatalf("expected 5/3/2 after upsert, got %d/%d/%d", rev2.Creates, rev2.Updates, rev2.Deletes)
	}

	// Verify only one row exists
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM revisions WHERE namespace_id = ? AND hash = ?", ns.ID, "aaa").
		Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}
}

func TestClearRevisionName(t *testing.T) {
	s := newTestStore(t)

	ns, err := s.EnsureNamespace("testns")
	if err != nil {
		t.Fatal(err)
	}

	name := "main"
	rev, err := s.UpsertRevision(ns.ID, "aaa", &name, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	err = s.ClearRevisionName(ns.ID, "main")
	if err != nil {
		t.Fatal(err)
	}

	rev, err = s.GetRevisionByID(rev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rev.Name != nil {
		t.Fatalf("expected nil name after clear, got %q", *rev.Name)
	}

	// Clearing a non-existent name should not error
	err = s.ClearRevisionName(ns.ID, "nonexistent")
	if err != nil {
		t.Fatalf("clearing non-existent name should not error: %v", err)
	}
}

func TestSaveGraphRoundTrip(t *testing.T) {
	s := newTestStore(t)

	ns, err := s.EnsureNamespace("testns")
	if err != nil {
		t.Fatal(err)
	}

	rev, err := s.UpsertRevision(ns.ID, "abc", nil, 2, 1, 0)
	if err != nil {
		t.Fatal(err)
	}

	g := &graph.Graph{
		Nodes: map[string]graph.Node{
			"aws_s3_bucket.logs": {
				Kind:    graph.ResourceNode,
				Name:    "logs",
				Type:    "aws_s3_bucket",
				Mode:    tfjson.ManagedResourceMode,
				State:   graph.StateCreate,
				Address: "aws_s3_bucket.logs",
			},
			"aws_s3_bucket.app": {
				Kind:    graph.ResourceNode,
				Name:    "app",
				Type:    "aws_s3_bucket",
				Mode:    tfjson.ManagedResourceMode,
				State:   graph.StateModify,
				Address: "aws_s3_bucket.app",
				Parent:  "module.vpc",
				Changes: &graph.NodeChange{
					Before: map[string]any{"versioning": false},
					After:  map[string]any{"versioning": true},
				},
			},
		},
		Edges: []graph.Edge{
			{From: "aws_s3_bucket.logs", To: "aws_s3_bucket.app"},
		},
	}

	err = s.SaveGraph(rev.ID, g)
	if err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}

	// Load it back
	loaded, err := s.GetGraph("testns", "abc")
	if err != nil {
		t.Fatalf("GetGraph: %v", err)
	}

	if len(loaded.Nodes) != len(g.Nodes) {
		t.Fatalf("expected %d nodes, got %d", len(g.Nodes), len(loaded.Nodes))
	}

	if len(loaded.Edges) != len(g.Edges) {
		t.Fatalf("expected %d edges, got %d", len(g.Edges), len(loaded.Edges))
	}

	// Check a node with changes
	appNode := loaded.Nodes["aws_s3_bucket.app"]
	if appNode.State != graph.StateModify {
		t.Fatalf("expected state modify, got %q", appNode.State)
	}
	if appNode.Type != "aws_s3_bucket" {
		t.Fatalf("expected type aws_s3_bucket, got %q", appNode.Type)
	}
	if appNode.Changes == nil {
		t.Fatal("expected changes, got nil")
	}
	if appNode.Changes.Before["versioning"] != false {
		t.Fatalf("expected before.versioning=false, got %v", appNode.Changes.Before["versioning"])
	}
	if appNode.Changes.After["versioning"] != true {
		t.Fatalf("expected after.versioning=true, got %v", appNode.Changes.After["versioning"])
	}
	if appNode.Parent != "module.vpc" {
		t.Fatalf("expected parent module.vpc, got %q", appNode.Parent)
	}

	// Check edge
	if loaded.Edges[0].From != "aws_s3_bucket.logs" {
		t.Fatalf("edge from %q, expected %q", loaded.Edges[0].From, "aws_s3_bucket.logs")
	}
}

func TestSaveGraphReplacesOnSecondCall(t *testing.T) {
	s := newTestStore(t)

	ns, err := s.EnsureNamespace("testns")
	if err != nil {
		t.Fatal(err)
	}

	rev, err := s.UpsertRevision(ns.ID, "abc", nil, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	// First save - 2 nodes, 1 edge
	g1 := &graph.Graph{
		Nodes: map[string]graph.Node{
			"a": {Kind: graph.ResourceNode, Name: "a", Address: "a"},
			"b": {Kind: graph.ResourceNode, Name: "b", Address: "b"},
		},
		Edges: []graph.Edge{{From: "a", To: "b"}},
	}
	if err := s.SaveGraph(rev.ID, g1); err != nil {
		t.Fatal(err)
	}

	// Second save - 1 node, 0 edges (replace entirely)
	g2 := &graph.Graph{
		Nodes: map[string]graph.Node{
			"c": {Kind: graph.ResourceNode, Name: "c", Address: "c"},
		},
		Edges: []graph.Edge{},
	}
	if err := s.SaveGraph(rev.ID, g2); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.GetGraph("testns", "abc")
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Nodes) != 1 {
		t.Fatalf("expected 1 node after replace, got %d", len(loaded.Nodes))
	}
	if _, ok := loaded.Nodes["c"]; !ok {
		t.Fatal("expected node 'c', not found")
	}
	if len(loaded.Edges) != 0 {
		t.Fatalf("expected 0 edges after replace, got %d", len(loaded.Edges))
	}
}

func TestGetNamespaces(t *testing.T) {
	s := newTestStore(t)

	// Initially empty
	namespaces, err := s.GetNamespaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != 0 {
		t.Fatalf("expected 0 namespaces, got %d", len(namespaces))
	}

	// Add some namespaces
	s.EnsureNamespace("production")
	s.EnsureNamespace("staging")
	s.EnsureNamespace("dev")

	namespaces, err = s.GetNamespaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != 3 {
		t.Fatalf("expected 3 namespaces, got %d", len(namespaces))
	}

	// Should be ordered by creation time
	if namespaces[0].Name != "production" {
		t.Fatalf("first should be production, got %q", namespaces[0].Name)
	}
	if namespaces[2].Name != "dev" {
		t.Fatalf("third should be dev, got %q", namespaces[2].Name)
	}
}

func TestGetRevisionsByNamespaceID(t *testing.T) {
	s := newTestStore(t)

	ns, _ := s.EnsureNamespace("testns")

	// Initially empty
	revisions, err := s.GetRevisionsByNamespaceID(ns.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 0 {
		t.Fatalf("expected 0 revisions, got %d", len(revisions))
	}

	// Add some revisions
	s.UpsertRevision(ns.ID, "aaa", nil, 1, 0, 0)
	s.UpsertRevision(ns.ID, "bbb", nil, 2, 1, 0)
	s.UpsertRevision(ns.ID, "ccc", nil, 0, 0, 1)

	revisions, err = s.GetRevisionsByNamespaceID(ns.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 3 {
		t.Fatalf("expected 3 revisions, got %d", len(revisions))
	}

	// Should be newest first (highest ID first)
	if revisions[0].Hash != "ccc" {
		t.Fatalf("first should be ccc, got %q", revisions[0].Hash)
	}
	if revisions[2].Hash != "aaa" {
		t.Fatalf("third should be aaa, got %q", revisions[2].Hash)
	}
}

func TestGetGraphEmptyPlan(t *testing.T) {
	s := newTestStore(t)

	ns, _ := s.EnsureNamespace("empty")
	rev, _ := s.UpsertRevision(ns.ID, "emptyhash", nil, 0, 0, 0)

	g := &graph.Graph{
		Nodes: map[string]graph.Node{},
		Edges: []graph.Edge{},
	}
	if err := s.SaveGraph(rev.ID, g); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.GetGraph("empty", "emptyhash")
	if err != nil {
		t.Fatalf("GetGraph empty plan: %v", err)
	}
	if len(loaded.Nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(loaded.Nodes))
	}
}

func TestGetGraphNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetGraph("nonexistent", "nope")
	if !errors.Is(err, ErrRevisionNotFound) {
		t.Fatalf("expected ErrRevisionNotFound, got %v", err)
	}
}
