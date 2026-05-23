package catalog

import (
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/ky3ow/infraview/pkg/graph"
	"github.com/ky3ow/infraview/pkg/storage"
	"time"
)

type PlanMetadata struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
}

type NamespaceView struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type RevisionView struct {
	ID          int          `json:"id"`
	NamespaceID int          `json:"namespace_id"`
	Name        string       `json:"name,omitempty"`
	Hash        string       `json:"hash"`
	Summary     PlanMetadata `json:"summary"`
	CreatedAt   time.Time    `json:"created_at"`
}

func (n *NamespaceView) fromDB(db *storage.DBNamespace) {
	n.ID = db.ID
	n.Name = db.Name
}

func (r *RevisionView) fromDB(db *storage.DBRevision) {
	if db.Name != nil {
		r.Name = *db.Name
	}
	r.ID = db.ID
	r.NamespaceID = db.NamespaceID
	r.Hash = db.Hash
	r.CreatedAt = db.CreatedAt
	r.Summary = PlanMetadata{
		Created: db.Creates,
		Updated: db.Updates,
		Deleted: db.Deletes,
	}
}

// Ensure the namespace exists, then create or update a revision.
// If name is provided, it is assigned to the new revision and stripped from
// any other revision in the same namespace, like Git Branches
func AddRevision(store *storage.Store, namespaceName, hash string, name *string, metadata PlanMetadata) (*RevisionView, error) {
	ns, err := store.EnsureNamespace(namespaceName)
	if err != nil {
		return nil, err
	}

	// If a name is given, unname any other revision holding it
	if name != nil {
		err = store.ClearRevisionName(ns.ID, *name)
		if err != nil {
			return nil, err
		}
	}

	dbRev, err := store.UpsertRevision(ns.ID, hash, name, metadata.Created, metadata.Updated, metadata.Deleted)
	if err != nil {
		return nil, err
	}

	res := RevisionView{}
	res.fromDB(dbRev)

	return &res, nil
}

func ExtractPlanMetadata(plan tfjson.Plan) *PlanMetadata {
	var metadata PlanMetadata

	for _, c := range plan.ResourceChanges {
		if c.Change == nil || c.Change.Actions == nil {
			continue
		}

		if c.Change.Actions.Create() {
			metadata.Created += 1
		} else if c.Change.Actions.Update() {
			metadata.Updated += 1
		} else if c.Change.Actions.Delete() {
			metadata.Deleted += 1
		} else if c.Change.Actions.Replace() {
			metadata.Deleted += 1
			metadata.Created += 1
		}
	}

	return &metadata
}

// SavePlan persists a parsed graph into the database under the given
// namespace/revision. If the revision already exists (same hash), the
// stored graph is replaced.
func SavePlan(store *storage.Store, namespaceName, hash string, name *string, metadata PlanMetadata, g *graph.Graph) (*RevisionView, error) {
	rev, err := AddRevision(store, namespaceName, hash, name, metadata)
	if err != nil {
		return nil, err
	}

	if err := store.SaveGraph(rev.ID, g); err != nil {
		return nil, err
	}

	return rev, nil
}

func ListNamespaces(store *storage.Store) ([]NamespaceView, error) {
	ns, err := store.GetNamespaces()
	if err != nil {
		return nil, err
	}

	res := make([]NamespaceView, 0, len(ns))

	for _, n := range ns {
		view := NamespaceView{}
		view.fromDB(&n)
		res = append(res, view)
	}

	return res, nil
}

func ListRevisions(store *storage.Store, namespaceID int) ([]RevisionView, error) {
	revs, err := store.GetRevisionsByNamespaceID(namespaceID)
	if err != nil {
		return nil, err
	}

	res := make([]RevisionView, 0, len(revs))

	for _, r := range revs {
		view := RevisionView{}
		view.fromDB(&r)
		res = append(res, view)
	}

	return res, nil
}

func SearchNamespaces(store *storage.Store, query string) (map[string][]RevisionView, error) {
	pattern := "%" + query + "%"

	rows, err := store.SearchCatalog(pattern)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]RevisionView)

	for _, row := range rows {
		rev := RevisionView{}
		rev.fromDB(&row.Revision)
		result[row.NamespaceName] = append(result[row.NamespaceName], rev)
	}

	return result, nil
}

func GetRevisionGraph(store *storage.Store, revisionID int) (*graph.Graph, error) {
	// TODO it might need to be in business logic and not in storage
	return store.GetGraph(revisionID)
}
