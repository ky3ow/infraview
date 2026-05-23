package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/ky3ow/infraview/pkg/graph"
)

// Get or create and get a namespace
func (s *Store) EnsureNamespace(name string) (*DBNamespace, error) {
	var ns DBNamespace
	var createdAt string

	err := s.db.QueryRow("SELECT id, name, created_at FROM namespaces WHERE name = ?", name).
		Scan(&ns.ID, &ns.Name, &createdAt)

	if err == nil {
		ns.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
		return &ns, err
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Namespace doesn't exist - create it
	_, err = s.db.Exec("INSERT INTO namespaces (name) VALUES (?)", name)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow("SELECT id, name, created_at FROM namespaces WHERE name = ?", name).
		Scan(&ns.ID, &ns.Name, &createdAt)
	if err != nil {
		return nil, err
	}

	ns.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
	return &ns, err
}

// Insert or update a revision.
// Update in case of pipeline re-runs on same commit.
func (s *Store) UpsertRevision(namespaceID int, hash string, name *string, creates, updates, deletes int) (*DBRevision, error) {
	var rev DBRevision
	var createdAt string

	_, err := s.db.Exec(
		`INSERT INTO revisions (namespace_id, hash, name, creates, updates, deletes)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (namespace_id, hash) DO UPDATE SET
			name       = excluded.name,
			creates    = excluded.creates,
			updates    = excluded.updates,
			deletes    = excluded.deletes,
			created_at = CURRENT_TIMESTAMP`,
		namespaceID, hash, name, creates, updates, deletes,
	)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow(
		`SELECT id, namespace_id, name, hash, creates, updates, deletes, created_at
		 FROM revisions WHERE namespace_id = ? AND hash = ?`,
		namespaceID, hash,
	).Scan(&rev.ID, &rev.NamespaceID, &rev.Name, &rev.Hash, &rev.Creates, &rev.Updates, &rev.Deletes, &createdAt)
	if err != nil {
		return nil, err
	}

	rev.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
	return &rev, err
}

func (s *Store) ClearRevisionName(namespaceID int, name string) error {
	_, err := s.db.Exec(
		`UPDATE revisions SET name = NULL WHERE namespace_id = ? AND name = ?`,
		namespaceID, name,
	)
	return err
}

func (s *Store) GetRevisionByID(id int) (*DBRevision, error) {
	var rev DBRevision
	var createdAt string

	err := s.db.QueryRow(
		`SELECT id, namespace_id, name, hash, creates, updates, deletes, created_at
		 FROM revisions WHERE id = ?`,
		id,
	).Scan(&rev.ID, &rev.NamespaceID, &rev.Name, &rev.Hash, &rev.Creates, &rev.Updates, &rev.Deletes, &createdAt)
	if err != nil {
		return nil, err
	}

	rev.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
	return &rev, err
}

// SaveGraph persists nodes, edges, and node details for a revision.
// Existing graph data for the revision is replaced.
func (s *Store) SaveGraph(revisionID int, g *graph.Graph) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing graph data for this revision
	_, err = tx.Exec(`DELETE FROM node_details WHERE revision_id = ?`, revisionID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM nodes WHERE revision_id = ?`, revisionID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM edges WHERE revision_id = ?`, revisionID)
	if err != nil {
		return err
	}

	// Insert nodes
	for addr, node := range g.Nodes {
		_, err = tx.Exec(
			`INSERT INTO nodes (revision_id, address, name, mode, kind, state, parent_address)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			revisionID, addr, node.Name, string(node.Mode), string(node.Kind),
			string(node.State), parentOrNil(node.Parent),
		)
		if err != nil {
			return err
		}
	}

	// Insert edges
	for _, edge := range g.Edges {
		_, err = tx.Exec(
			`INSERT INTO edges (revision_id, from_address, to_address)
			 VALUES (?, ?, ?)`,
			revisionID, edge.From, edge.To,
		)
		if err != nil {
			return err
		}
	}

	// Insert node details
	for addr, node := range g.Nodes {
		var changesJSON *string
		if node.Changes != nil {
			b, err := json.Marshal(node.Changes)
			if err != nil {
				return err
			}
			s := string(b)
			changesJSON = &s
		}

		_, err = tx.Exec(
			`INSERT INTO node_details (revision_id, address, type, changes)
			 VALUES (?, ?, ?, ?)`,
			revisionID, addr, node.Type, changesJSON,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetGraph loads nodes, edges, and node details for a revision
// and returns a reconstructed graph.Graph.
func (s *Store) GetGraph(revisionID int) (*graph.Graph, error) {
	g := &graph.Graph{
		Nodes: make(map[string]graph.Node),
		Edges: make([]graph.Edge, 0),
	}

	// Load nodes
	rows, err := s.db.Query(
		`SELECT address, name, mode, kind, state, parent_address FROM nodes WHERE revision_id = ?`,
		revisionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var n DBNode
		var mode, kind, state sql.NullString
		var parent sql.NullString
		err := rows.Scan(&n.Address, &n.Name, &mode, &kind, &state, &parent)
		if err != nil {
			return nil, err
		}
		if mode.Valid {
			n.Mode = tfjson.ResourceMode(mode.String)
		}
		n.Kind = graph.NodeKind(kind.String)
		n.State = graph.ResourceState(state.String)
		if parent.Valid {
			n.ParentAddress = parent.String
		}

		g.Nodes[n.Address] = graph.Node{
			Kind:     n.Kind,
			Name:     n.Name,
			Type:     "", // filled from details
			Mode:     n.Mode,
			State:    n.State,
			Address:  n.Address,
			Parent:   n.ParentAddress,
			Changes:  nil, // filled from details
		}
	}

	// Load node details
	detailRows, err := s.db.Query(
		`SELECT address, type, changes FROM node_details WHERE revision_id = ?`,
		revisionID,
	)
	if err != nil {
		return nil, err
	}
	defer detailRows.Close()

	for detailRows.Next() {
		var addr, typ string
		var changesJSON sql.NullString
		err := detailRows.Scan(&addr, &typ, &changesJSON)
		if err != nil {
			return nil, err
		}
		if node, ok := g.Nodes[addr]; ok {
			node.Type = typ
			if changesJSON.Valid {
				var changes graph.NodeChange
				if err := json.Unmarshal([]byte(changesJSON.String), &changes); err != nil {
					return nil, err
				}
				node.Changes = &changes
			}
			g.Nodes[addr] = node
		}
	}

	// Load edges
	edgeRows, err := s.db.Query(
		`SELECT from_address, to_address FROM edges WHERE revision_id = ?`,
		revisionID,
	)
	if err != nil {
		return nil, err
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var from, to string
		if err := edgeRows.Scan(&from, &to); err != nil {
			return nil, err
		}
		g.Edges = append(g.Edges, graph.Edge{From: from, To: to})
	}

	return g, nil
}

// parentOrNil returns nil for empty parent so it stores as NULL.
func parentOrNil(p string) *string {
	if p == "" {
		return nil
	}
	return &p
}
