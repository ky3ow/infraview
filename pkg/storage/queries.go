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
		var parent *string
		if node.Parent != "" {
			parent = &node.Parent
		}
		_, err = tx.Exec(
			`INSERT INTO nodes (revision_id, address, name, mode, kind, state, parent_address)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			revisionID, addr, node.Name, string(node.Mode), string(node.Kind),
			string(node.State), parent,
		)
		if err != nil {
			return err
		}
	}

	// Insert edges
	for _, edge := range g.Edges {
		_, err = tx.Exec(
			`INSERT INTO edges (revision_id, from_address, to_address, type)
			 VALUES (?, ?, ?, ?)`,
			revisionID, edge.From, edge.To, edge.Type,
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
// identified by namespace name and hash, returning a reconstructed graph.Graph.
// Returns ErrRevisionNotFound if no revision matches.
func (s *Store) GetGraph(namespace, hash string) (*graph.Graph, error) {
	// Verify the revision exists
	var exists int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM revisions r
		 JOIN namespaces ns ON ns.id = r.namespace_id
		 WHERE ns.name = ? AND r.hash = ?`,
		namespace, hash,
	).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, ErrRevisionNotFound
	}

	g := &graph.Graph{
		Nodes: make(map[string]graph.Node),
		Edges: make([]graph.Edge, 0),
	}

	// Load nodes + details in one pass
	rows, err := s.db.Query(`
		SELECT n.address, n.name, n.mode, n.kind, n.state, n.parent_address,
		       nd.type, nd.changes
		FROM nodes n
		JOIN revisions r ON r.id = n.revision_id
		JOIN namespaces ns ON ns.id = r.namespace_id
		LEFT JOIN node_details nd ON nd.revision_id = n.revision_id AND nd.address = n.address
		WHERE ns.name = ? AND r.hash = ?
	`, namespace, hash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var n DBNode
		var mode, kind, state, typ sql.NullString
		var parent sql.NullString
		var changesJSON sql.NullString

		err := rows.Scan(&n.Address, &n.Name, &mode, &kind, &state, &parent, &typ, &changesJSON)
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

		node := graph.Node{
			Kind:    n.Kind,
			Name:    n.Name,
			Type:    "",
			Mode:    n.Mode,
			State:   n.State,
			Address: n.Address,
			Parent:  n.ParentAddress,
			Changes: nil,
		}

		if typ.Valid {
			node.Type = typ.String
		}
		if changesJSON.Valid {
			var changes graph.NodeChange
			if err := json.Unmarshal([]byte(changesJSON.String), &changes); err != nil {
				return nil, err
			}
			node.Changes = &changes
		}

		g.Nodes[n.Address] = node
	}

	// Load edges
	edgeRows, err := s.db.Query(`
		SELECT e.from_address, e.to_address, e.type
		FROM edges e
		JOIN revisions r ON r.id = e.revision_id
		JOIN namespaces ns ON ns.id = r.namespace_id
		WHERE ns.name = ? AND r.hash = ?
	`, namespace, hash)
	if err != nil {
		return nil, err
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var from, to, edgeType string
		if err := edgeRows.Scan(&from, &to, &edgeType); err != nil {
			return nil, err
		}
		g.Edges = append(g.Edges, graph.Edge{From: from, To: to, Type: graph.EdgeType(edgeType)})
	}

	return g, nil
}

// GetNamespaceByName returns a namespace by name.
// Returns sql.ErrNoRows if not found.
func (s *Store) GetNamespaceByName(name string) (*DBNamespace, error) {
	var ns DBNamespace
	var createdAt string

	err := s.db.QueryRow("SELECT id, name, created_at FROM namespaces WHERE name = ?", name).
		Scan(&ns.ID, &ns.Name, &createdAt)
	if err != nil {
		return nil, err
	}

	ns.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
	return &ns, err
}

// GetNamespaces returns all namespaces ordered by creation time.
func (s *Store) GetNamespaces() ([]DBNamespace, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM namespaces ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var namespaces []DBNamespace
	for rows.Next() {
		var ns DBNamespace
		var createdAt string
		if err := rows.Scan(&ns.ID, &ns.Name, &createdAt); err != nil {
			return nil, err
		}
		ns.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
		if err != nil {
			return nil, err
		}
		namespaces = append(namespaces, ns)
	}
	return namespaces, rows.Err()
}

// GetRevisionsByNamespaceID returns all revisions for a namespace,
// newest first.
func (s *Store) GetRevisionsByNamespaceID(namespaceID int) ([]DBRevision, error) {
	rows, err := s.db.Query(
		`SELECT id, namespace_id, name, hash, creates, updates, deletes, created_at
		 FROM revisions WHERE namespace_id = ? ORDER BY id DESC`,
		namespaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []DBRevision
	for rows.Next() {
		var rev DBRevision
		var createdAt string
		if err := rows.Scan(&rev.ID, &rev.NamespaceID, &rev.Name, &rev.Hash, &rev.Creates, &rev.Updates, &rev.Deletes, &createdAt); err != nil {
			return nil, err
		}
		rev.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, rev)
	}
	return revisions, rows.Err()
}

// GetRevisionsByNamespaceName returns all revisions for a namespace given by name,
// newest first.
func (s *Store) GetRevisionsByNamespaceName(name string) ([]DBRevision, error) {
	rows, err := s.db.Query(
		`SELECT r.id, r.namespace_id, r.name, r.hash, r.creates, r.updates, r.deletes, r.created_at
		 FROM revisions r
		 JOIN namespaces ns ON ns.id = r.namespace_id
		 WHERE ns.name = ?
		 ORDER BY r.id DESC`,
		name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []DBRevision
	for rows.Next() {
		var rev DBRevision
		var createdAt string
		if err := rows.Scan(&rev.ID, &rev.NamespaceID, &rev.Name, &rev.Hash, &rev.Creates, &rev.Updates, &rev.Deletes, &createdAt); err != nil {
			return nil, err
		}
		rev.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, rev)
	}
	return revisions, rows.Err()
}

// LatestRevision holds a namespace name and its latest revision hash.
type LatestRevision struct {
	NamespaceName string
	Hash          string
}

// GetLatestRevision returns the globally newest revision (highest id).
func (s *Store) GetLatestRevision() (*LatestRevision, error) {
	var lr LatestRevision
	err := s.db.QueryRow(
		`SELECT ns.name, r.hash
		 FROM revisions r
		 JOIN namespaces ns ON ns.id = r.namespace_id
		 ORDER BY r.id DESC
		 LIMIT 1`,
	).Scan(&lr.NamespaceName, &lr.Hash)
	if err != nil {
		return nil, err
	}
	return &lr, nil
}

// GetLatestRevisionByNamespaceName returns the newest revision for a given namespace.
func (s *Store) GetLatestRevisionByNamespaceName(name string) (*LatestRevision, error) {
	var lr LatestRevision
	err := s.db.QueryRow(
		`SELECT ns.name, r.hash
		 FROM revisions r
		 JOIN namespaces ns ON ns.id = r.namespace_id
		 WHERE ns.name = ?
		 ORDER BY r.id DESC
		 LIMIT 1`,
		name,
	).Scan(&lr.NamespaceName, &lr.Hash)
	if err != nil {
		return nil, err
	}
	return &lr, nil
}

type SearchResultRow struct {
	NamespaceID   int
	NamespaceName string
	Revision      DBRevision
}

// SearchCatalog returns all (namespace, revision) pairs where the namespace
// name, revision hash, or revision name contains the given query.
func (s *Store) SearchCatalog(query string) ([]SearchResultRow, error) {
	rows, err := s.db.Query(
		`SELECT n.id, n.name,
		        r.id, r.namespace_id, r.name, r.hash, r.creates, r.updates, r.deletes, r.created_at
		 FROM namespaces n
		 JOIN revisions r ON r.namespace_id = n.id
		 WHERE n.name LIKE '%' || ? || '%'
		    OR r.hash LIKE '%' || ? || '%'
		    OR r.name LIKE '%' || ? || '%'
		 ORDER BY n.name, r.created_at DESC`,
		query, query, query,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResultRow
	for rows.Next() {
		var rev DBRevision
		var nsID int
		var nsName string
		var createdAt string

		if err := rows.Scan(&nsID, &nsName,
			&rev.ID, &rev.NamespaceID, &rev.Name, &rev.Hash,
			&rev.Creates, &rev.Updates, &rev.Deletes, &createdAt); err != nil {
			return nil, err
		}

		rev.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
		if err != nil {
			return nil, err
		}

		results = append(results, SearchResultRow{
			NamespaceID:   nsID,
			NamespaceName: nsName,
			Revision:      rev,
		})
	}
	return results, rows.Err()
}

func (s *Store) DeleteTokensByType(tokenType string) error {
	_, err := s.db.Exec(`DELETE FROM tokens WHERE type = ?`, tokenType)
	return err
}

func (s *Store) CreateToken(tokenType, hash string) error {
	_, err := s.db.Exec(
		`INSERT INTO tokens (type, hash) VALUES (?, ?)`,
		tokenType, hash,
	)
	return err
}

func (s *Store) FindTokenByHash(hash string) (*DBToken, error) {
	var t DBToken
	var createdAt string
	err := s.db.QueryRow(
		`SELECT id, type, hash, created_at FROM tokens WHERE hash = ?`,
		hash,
	).Scan(&t.ID, &t.Type, &t.Hash, &createdAt)
	if err != nil {
		return nil, err
	}
	t.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
	return &t, err
}

// parentOrNil returns nil for empty parent so it stores as NULL.
func parentOrNil(p string) *string {
	if p == "" {
		return nil
	}
	return &p
}
