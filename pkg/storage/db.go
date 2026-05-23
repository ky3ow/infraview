package storage

import (
	"database/sql"
	"fmt"

	"github.com/ky3ow/infraview/pkg/graph"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(location string) (*Store, error) {
	db, err := sql.Open("sqlite", location)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func SaveGraph(g *graph.Graph) error {
	return fmt.Errorf("")
}

func LoadGraph(revision, namespace string) (*graph.Graph, error) {
	return nil, fmt.Errorf("")
}
