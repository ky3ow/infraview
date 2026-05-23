package catalog

import (
	tfjson "github.com/hashicorp/terraform-json"
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

func ExtractPlanMetadata(plan tfjson.Plan) (*PlanMetadata, error) {
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

	return &metadata, nil
}
