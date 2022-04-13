package gatherer

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/networkgraph/entity/datastore"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/networkgraph/defaultexternalsrcs"
	"github.com/stackrox/stackrox/pkg/set"
)

func writeChecksumLocally(checksum []byte) error {
	if err := os.WriteFile(defaultexternalsrcs.LocalChecksumFile, checksum, 0644); err != nil {
		return errors.Wrapf(err, "writing provider networks checksum %s", defaultexternalsrcs.LocalChecksumFile)
	}
	return nil
}

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
