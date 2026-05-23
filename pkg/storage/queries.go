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
