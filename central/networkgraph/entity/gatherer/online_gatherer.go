package gatherer

import (
	"bytes"
	"context"
	"io/ioutil"
	"time"

	"github.com/stackrox/rox/central/license/manager"
	entityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph/defaultexternalsrcs"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log                  = logging.LoggerForModule()
	networkGraphWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
)

type onlineDefaultExtSrcsGathererImpl struct {
	licenseMgr      manager.LicenseManager
	networkEntityDS entityDataStore.EntityDataStore
	stopSig         concurrency.Signal
	lock            sync.Mutex
}

func (g *onlineDefaultExtSrcsGathererImpl) Start() {
	go g.run()
}

func (g *onlineDefaultExtSrcsGathererImpl) run() {
	// TODO: Remove the following once there is bundled data.
	g.reconcileDefaultExternalSrcs()

	ticker := time.NewTicker(env.NetworkGraphDefaultExtSrcsGatherFreq.DurationSetting())
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

func (g *onlineDefaultExtSrcsGathererImpl) reconcileDefaultExternalSrcs() {
	src, err := NewDefaultNetworksRemoteSource(g.licenseMgr)
	if err != nil {
		log.Errorf("Failed to determine DefaultNetworksRemoteSource for provider networks: %v", err)
		return
	}

	remoteChecksum, err := httputil.HTTPGet(src.ChecksumURL())
	if err != nil {
		log.Errorf("Failed to pull remote provider networks checksum: %v", err)
		return
	}

	localChecksum, err := ioutil.ReadFile(defaultexternalsrcs.LocalChecksumFile)
	if err != nil {
		log.Errorf("Failed to read local provider networks checksum from %q: %v", defaultexternalsrcs.LocalChecksumFile, err)
		return
	}

	if bytes.Equal(localChecksum, remoteChecksum) {
		return
	}

	data, err := httputil.HTTPGet(src.DataURL())
	if err != nil {
		log.Errorf("Failed to pull remote provider networks data: %v", err)
	}

	var entities []*storage.NetworkEntity
	if entities, err = defaultexternalsrcs.ParseProviderNetworkData(data); err != nil {
		log.Error(err)
		return
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	if err := writeChecksumLocally(remoteChecksum); err != nil {
		log.Error(err)
		return
	}

	for _, entity := range entities {
		if err := g.networkEntityDS.UpsertExternalNetworkEntity(networkGraphWriteCtx, entity); err != nil {
			log.Errorf("Failed to update default external networks: %v", err)
		}
	}
}
