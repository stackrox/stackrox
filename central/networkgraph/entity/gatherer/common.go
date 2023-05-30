package gatherer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
)

func loadStoredDefaultExtSrcsIDs(entityDS datastore.EntityDataStore) (set.StringSet, error) {
	entities, err := entityDS.GetAllMatchingEntities(networkGraphReadCtx, func(entity *storage.NetworkEntity) bool {
		return entity.GetInfo().GetExternalSource().GetDefault()
	})
	if err != nil {
		return nil, errors.Wrap(err, "loading stored default external networks")
	}

	// Since we know IDs are deterministically computed from CIDRs, we can use IDs to determine CIDR uniqueness.
	ret := set.NewStringSet()
	for _, entity := range entities {
		ret.Add(entity.GetInfo().GetId())
	}
	return ret, nil
}

func updateInStorage(entityDS datastore.EntityDataStore, lastSeenIDs set.StringSet, entities ...*storage.NetworkEntity) ([]string, error) {
	var filtered []*storage.NetworkEntity
	for _, entity := range entities {
		// This is under the assumption that network from one provider does not move to another provider.
		// Otherwise, deep equality is required.
		if !lastSeenIDs.Contains(entity.GetInfo().GetId()) {
			filtered = append(filtered, entity)
		}
	}
	return entityDS.CreateExtNetworkEntitiesForCluster(networkGraphWriteCtx, "", filtered...)
}

func removeOutdatedNetworks(entityDS datastore.EntityDataStore, ids ...string) error {
	var errs errorhelpers.ErrorList
	for _, id := range ids {
		if err := entityDS.DeleteExternalNetworkEntity(networkGraphWriteCtx, id); err != nil {
			errs.AddError(err)
		}
	}
	return errs.ToError()
}
