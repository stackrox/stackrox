package gatherer

import (
	"context"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/networkgraph/defaultexternalsrcs"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
)

func writeChecksumLocally(checksum []byte) error {
	if err := ioutil.WriteFile(defaultexternalsrcs.LocalChecksumFile, checksum, 0644); err != nil {
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

func updateInStorage(entityDS datastore.EntityDataStore, lastSeenIDs set.StringSet, entities ...*storage.NetworkEntity) error {
	var errs errorhelpers.ErrorList
	for _, entity := range entities {
		if lastSeenIDs.Contains(entity.GetInfo().GetId()) {
			continue
		}

		if err := entityDS.CreateExternalNetworkEntity(networkGraphWriteCtx, entity, true); err != nil {
			errs.AddError(err)
		}
	}
	return errs.ToError()
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

func doPushExternalNetworkEntitiesToAllSensor(connMgr connection.Manager) {
	// If push request if for a global network entity, push to all known clusters once and return.
	elevateCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))

	if err := connMgr.PushExternalNetworkEntitiesToAllSensors(elevateCtx); err != nil {
		log.Errorf("failed to sync external networks with some clusters: %v", err)
	}
}
