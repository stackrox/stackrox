package gatherer

import (
	"context"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/defaultexternalsrcs"
	"github.com/stackrox/rox/pkg/sac"
)

func writeChecksumLocally(checksum []byte) error {
	if err := ioutil.WriteFile(defaultexternalsrcs.LocalChecksumFile, checksum, 0644); err != nil {
		return errors.Wrapf(err, "writing provider networks checksum %s", defaultexternalsrcs.LocalChecksumFile)
	}
	return nil
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
