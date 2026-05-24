package storage

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

var ErrRevisionNotFound = errors.New("revision not found")

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS namespaces (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

CREATE TABLE IF NOT EXISTS revisions (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	namespace_id INTEGER NOT NULL REFERENCES namespaces(id),
	name         TEXT,
	hash         TEXT NOT NULL,
	creates      INTEGER NOT NULL DEFAULT 0,
	updates      INTEGER NOT NULL DEFAULT 0,
	deletes      INTEGER NOT NULL DEFAULT 0,
	created_at   TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
	UNIQUE (namespace_id, hash)
);

CREATE TABLE IF NOT EXISTS nodes (
	revision_id    INTEGER NOT NULL REFERENCES revisions(id),
	address        TEXT    NOT NULL,
	name           TEXT    NOT NULL,
	mode           TEXT,
	kind           TEXT    NOT NULL,
	state          TEXT    NOT NULL,
	parent_address TEXT,
	PRIMARY KEY (revision_id, address)
);

CREATE TABLE IF NOT EXISTS node_details (
	revision_id INTEGER NOT NULL,
	address     TEXT    NOT NULL,
	type        TEXT,
	changes     TEXT,
	PRIMARY KEY (revision_id, address),
	FOREIGN KEY (revision_id, address) REFERENCES nodes(revision_id, address)
);

CREATE TABLE IF NOT EXISTS edges (
	revision_id  INTEGER NOT NULL REFERENCES revisions(id),
	from_address TEXT    NOT NULL,
	to_address   TEXT    NOT NULL,
	type         TEXT    NOT NULL,
	PRIMARY KEY (revision_id, from_address, to_address)
);

CREATE TABLE IF NOT EXISTS tokens (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	type       TEXT NOT NULL CHECK(type IN ('read', 'write')),
	hash       TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);
`

func New(location string) (*Store, error) {
	db, err := sql.Open("sqlite", location)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// WAL mode for concurrent reads
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("set journal mode: %w", err)
	}
	// Enable foreign key enforcement (off by default in SQLite)
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("set foreign keys: %w", err)
	}

	store := &Store{db: db}

	if err := store.ensureTables(); err != nil {
		return nil, fmt.Errorf("ensure tables: %w", err)
	}

	return store, nil
}

func (s *Store) ensureTables() error {
	_, err := s.db.Exec(schema)
	return err
}
