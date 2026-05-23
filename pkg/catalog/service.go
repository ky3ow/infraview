package catalog

import (
	"github.com/ky3ow/infraview/pkg/storage"
)

type PlanMetadata struct {
	Created int
	Updated int
	Deleted int
}

// Ensure the namespace exists, then create or update a revision.
// If name is provided, it is assigned to the new revision and stripped from
// any other revision in the same namespace, like Git Branches
func AddRevision(store *storage.Store, namespaceName, hash string, name *string, metadata PlanMetadata) (*storage.DBRevision, error) {
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

	return store.UpsertRevision(ns.ID, hash, name, metadata.Created, metadata.Updated, metadata.Deleted)
}
