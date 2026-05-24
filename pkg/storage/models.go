package storage

import (
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/ky3ow/infraview/pkg/graph"
	"time"
)

type DBNamespace struct {
	ID        int
	Name      string
	CreatedAt time.Time
}

type DBRevision struct {
	ID          int
	NamespaceID int
	Name        *string
	Hash        string
	Creates     int
	Updates     int
	Deletes     int
	CreatedAt   time.Time
}

type DBNode struct {
	RevisionID    int
	Address       string
	Name          string
	Mode          tfjson.ResourceMode // TODO make our own
	Kind          graph.NodeKind
	State         graph.ResourceState
	ParentAddress string
}

type DBNodeDetails struct {
	RevisionID int
	Address    string
	Type       string
	Changes    graph.NodeChange
}

type DBEdge struct {
	RevisionID  int
	FromAddress string
	ToAddress   string
	Type        graph.EdgeType
}

type DBToken struct {
	ID        int
	Type      string
	Hash      string
	CreatedAt time.Time
}
