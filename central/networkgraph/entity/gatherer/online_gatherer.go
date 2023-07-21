package gatherer

import (
	"bytes"
	"context"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	blobstore "github.com/stackrox/rox/central/blob/datastore"
	entityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph/defaultexternalsrcs"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
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

	blobAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	blobReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
)

type defaultExtSrcsGathererImpl struct {
	networkEntityDS entityDataStore.EntityDataStore
	stopSig         concurrency.Signal
	blobStore       blobstore.Datastore
	currentChecksum []byte
	mutex           sync.RWMutex
}

// newDefaultExtNetworksGatherer returns an instance of NetworkGraphDefaultExtSrcsGatherer that reaches out internet to fetch the data.
func newDefaultExtNetworksGatherer(networkEntityDS entityDataStore.EntityDataStore, blobStore blobstore.Datastore) NetworkGraphDefaultExtSrcsGatherer {
	return &defaultExtSrcsGathererImpl{
		networkEntityDS: networkEntityDS,
		blobStore:       blobStore,
	}
}

func (g *defaultExtSrcsGathererImpl) Start() {
	go func() {
		if err := g.loadBundledExternalSrcs(g.blobStore, g.networkEntityDS); err != nil {
			log.Errorf("UNEXPECTED: Failed to load pre-bundled external networks data: %v", err)
		}
		go g.run()
	}()
}

func (g *defaultExtSrcsGathererImpl) run() {
	// In offline mode, don't try to reconcile.
	if env.OfflineModeEnv.BooleanSetting() {
		return
	}
	ticker := time.NewTicker(env.ExtNetworkSrcsGatherInterval.DurationSetting())
	defer ticker.Stop()

	for {
		select {
		case <-g.stopSig.Done():
			return
		case <-ticker.C:
			if err := g.reconcileDefaultExternalSrcs(); err != nil {
				log.Errorf("Failed to update default external networks: %v", err)
			}
		}
	}
}

func (g *defaultExtSrcsGathererImpl) Stop() {
	g.stopSig.Signal()
}

func (g *defaultExtSrcsGathererImpl) Update() error {
	return g.reconcileDefaultExternalSrcs()
}

func (g *defaultExtSrcsGathererImpl) reconcileDefaultExternalSrcs() error {
	remoteDataURL, remoteCksumURL, err := defaultexternalsrcs.GetRemoteDataAndCksumURLs()
	if err != nil {
		return errors.Wrap(err, "getting remote data and checksum URLs")
	}

	remoteChecksum, err := httputil.HTTPGet(remoteCksumURL)
	if err != nil {
		return errors.Wrap(err, "pulling remote external networks checksum")
	}

	localChecksum, err := g.loadLocalChecksum(g.blobStore)
	if err != nil {
		return errors.Wrapf(err, "reading local external networks checksum from %q", defaultexternalsrcs.LocalChecksumBlobPath)
	}

	if bytes.Equal(localChecksum, remoteChecksum) {
		return nil
	}

	data, err := httputil.HTTPGet(remoteDataURL)
	if err != nil {
		return errors.Wrap(err, "pulling remote external networks data")
	}

	var entities []*storage.NetworkEntity
	if entities, err = defaultexternalsrcs.ParseProviderNetworkData(data); err != nil {
		return err
	}

	log.Infof("Successfully fetched %d external networks", len(entities))

	lastSeenIDs, err := loadStoredDefaultExtSrcsIDs(g.networkEntityDS)
	if err != nil {
		return err
	}

	inserted, err := updateInStorage(g.networkEntityDS, lastSeenIDs, entities...)
	if err != nil {
		return errors.Wrapf(err, "updated %d/%d networks", len(inserted), len(entities))
	}

	log.Infof("Found %d external networks in DB. Successfully stored %d/%d new external networks", len(lastSeenIDs), len(inserted), len(entities))

	// Update checksum only if all the pulled data is successfully written.
	if err := g.writeLocalChecksum(g.blobStore, remoteChecksum); err != nil {
		return err
	}

	newIDs := set.NewStringSet()
	for _, entity := range entities {
		newIDs.Add(entity.GetInfo().GetId())
	}

	if err := removeOutdatedNetworks(g.networkEntityDS, lastSeenIDs.Difference(newIDs).AsSlice()...); err != nil {
		return errors.Wrap(err, "removing outdated default external networks")
	}
	return nil
}

// loadLocalChecksum loads local checksum if it exists.
func (g *defaultExtSrcsGathererImpl) loadLocalChecksum(store blobstore.Datastore) ([]byte, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if len(g.currentChecksum) > 0 {
		return g.currentChecksum, nil
	}
	buf, _, exists, err := store.GetBlobWithDataInBuffer(blobReadCtx, defaultexternalsrcs.LocalChecksumBlobPath)
	if err != nil || !exists {
		return nil, err
	}
	g.currentChecksum = buf.Bytes()
	return g.currentChecksum, nil
}

func (g *defaultExtSrcsGathererImpl) writeLocalChecksum(store blobstore.Datastore, checksum []byte) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	b := &storage.Blob{
		Name:         defaultexternalsrcs.LocalChecksumBlobPath,
		Length:       int64(len(checksum)),
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: timestamp.TimestampNow(),
	}
	buf := bytes.NewBuffer(checksum)
	if err := store.Upsert(blobAccessCtx, b, buf); err != nil {
		return errors.Wrapf(err, "writing provider networks checksum %s", defaultexternalsrcs.LocalChecksumBlobPath)
	}
	g.currentChecksum = checksum
	return nil
}
