package storage

import (
	"database/sql"
	"errors"
	"time"
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

// New revision in namespace
// Naming like git branches - new revision may steal name from another
// Update revision in place in case of pipeline re-runs on same commit
func (s *Store) AddRevision(namespaceID int, hash string, name *string) (*DBRevision, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// If a name is given, unname any OTHER revision with that name
	if name != nil {
		_, err = tx.Exec(
			`UPDATE revisions SET name = NULL WHERE namespace_id = ? AND name = ?`,
			namespaceID, *name,
		)
		if err != nil {
			return nil, err
		}
	}

	_, err = tx.Exec(
		`INSERT INTO revisions (namespace_id, hash, name, creates, updates, deletes)
		 VALUES (?, ?, ?, 0, 0, 0)
		 ON CONFLICT (namespace_id, hash) DO UPDATE SET
			name      = excluded.name,
			creates   = 0,
			updates   = 0,
			deletes   = 0,
			created_at = CURRENT_TIMESTAMP`,
		namespaceID, hash, name,
	)
	if err != nil {
		return nil, err
	}

	var rev DBRevision
	var createdAt string

	err = tx.QueryRow(
		`SELECT id, namespace_id, name, hash, creates, updates, deletes, created_at
		 FROM revisions WHERE namespace_id = ? AND hash = ?`,
		namespaceID, hash,
	).Scan(&rev.ID, &rev.NamespaceID, &rev.Name, &rev.Hash, &rev.Creates, &rev.Updates, &rev.Deletes, &createdAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	rev.CreatedAt, err = time.ParseInLocation(time.DateTime, createdAt, time.UTC)
	return &rev, err
}
