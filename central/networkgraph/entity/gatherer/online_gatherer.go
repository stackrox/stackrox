package gatherer

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"time"

	"github.com/stackrox/rox/central/license/manager"
	entityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph/defaultexternalsrcs"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log                 = logging.LoggerForModule()
	networkGraphReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
	networkGraphWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
)

type onlineDefaultExtSrcsGathererImpl struct {
	lastSeenCIDRs   map[string]string
	licenseMgr      manager.LicenseManager
	networkEntityDS entityDataStore.EntityDataStore
	sensorConnMgr   connection.Manager
	stopSig         concurrency.Signal
	lock            sync.Mutex
}

func (g *onlineDefaultExtSrcsGathererImpl) Start() {
	go g.run()
}

func (g *onlineDefaultExtSrcsGathererImpl) run() {
	g.loadStoredCIDRs()
	// TODO: Remove the following once there is bundled data.
	g.reconcileDefaultExternalSrcs()

	ticker := time.NewTicker(env.ExtNetworkSrcsGatherInterval.DurationSetting())
	defer ticker.Stop()

	for {
		select {
		case <-g.stopSig.Done():
			return
		case <-ticker.C:
			g.reconcileDefaultExternalSrcs()
		}
	}
}

func (g *onlineDefaultExtSrcsGathererImpl) Stop() {
	g.stopSig.Signal()
}

func (g *onlineDefaultExtSrcsGathererImpl) loadStoredCIDRs() {
	g.lock.Lock()
	defer g.lock.Unlock()

	entities, err := g.networkEntityDS.GetAllMatchingEntities(networkGraphReadCtx, func(entity *storage.NetworkEntity) bool {
		return entity.GetInfo().GetExternalSource().GetDefault()
	})
	if err != nil {
		log.Errorf("Failed to load stored default external networks: %v", err)
	}

	for _, entity := range entities {
		g.lastSeenCIDRs[entity.GetInfo().GetExternalSource().GetCidr()] = entity.GetInfo().GetId()
	}
}

func (g *onlineDefaultExtSrcsGathererImpl) reconcileDefaultExternalSrcs() {
	remoteChecksum, err := httputil.HTTPGet(defaultexternalsrcs.RemoteChecksumURL)
	if err != nil {
		log.Errorf("Failed to pull remote external networks checksum: %v", err)
		return
	}

	var localChecksum []byte
	_, err = os.Open(defaultexternalsrcs.LocalChecksumFile)
	if os.IsExist(err) {
		localChecksum, err = ioutil.ReadFile(defaultexternalsrcs.LocalChecksumFile)
		if err != nil {
			log.Errorf("Failed to read local external networks checksum from %q: %v", defaultexternalsrcs.LocalChecksumFile, err)
			return
		}
	}

	if bytes.Equal(localChecksum, remoteChecksum) {
		return
	}

	data, err := httputil.HTTPGet(defaultexternalsrcs.RemoteDataURL)
	if err != nil {
		log.Errorf("Failed to pull remote external networks data: %v", err)
	}

	var entities []*storage.NetworkEntity
	if entities, err = defaultexternalsrcs.ParseProviderNetworkData(data); err != nil {
		log.Error(err)
		return
	}

	log.Infof("Successfully fetched %d external networks", len(entities))

	var errs errorhelpers.ErrorList
	newCIDRs := set.NewStringSet()
	for _, entity := range entities {
		if err := g.updateInStorage(entity); err != nil {
			errs.AddError(err)
			continue
		}
		newCIDRs.Add(entity.GetInfo().GetExternalSource().GetCidr())
	}

	if err := errs.ToError(); err != nil {
		log.Errorf("Failed to update default external networks: %v", err)
		return
	}

	go doPushExternalNetworkEntitiesToAllSensor(g.sensorConnMgr)

	// Update checksum only if all the pulled data is successfully written.
	if err := writeChecksumLocally(remoteChecksum); err != nil {
		log.Error(err)
		return
	}

	if err := g.removeOutdatedNetworks(newCIDRs); err != nil {
		log.Errorf("Failed to remove outdated default external networks: %v", err)
		return
	}
}

func (g *onlineDefaultExtSrcsGathererImpl) updateInStorage(entity *storage.NetworkEntity) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if id := g.lastSeenCIDRs[entity.GetInfo().GetExternalSource().GetCidr()]; id != "" {
		return nil
	}

	if err := g.networkEntityDS.UpsertExternalNetworkEntity(networkGraphWriteCtx, entity, true); err != nil {
		return err
	}

	g.lastSeenCIDRs[entity.GetInfo().GetExternalSource().GetCidr()] = entity.GetInfo().GetId()
	return nil
}

func (g *onlineDefaultExtSrcsGathererImpl) removeOutdatedNetworks(newCIDRs set.StringSet) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	for cidr, id := range g.lastSeenCIDRs {
		if newCIDRs.Contains(cidr) {
			continue
		}

		if err := g.networkEntityDS.DeleteExternalNetworkEntity(networkGraphWriteCtx, id); err != nil {
			return err
		}
	}
	return nil
}
