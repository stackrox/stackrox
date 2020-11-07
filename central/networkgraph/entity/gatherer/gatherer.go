package gatherer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/license/manager"
	entityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
)

// NetworkGraphDefaultExtSrcsGatherer provides functionality to update the storage.NetworkEntity storage with default network graph
// external sources.
type NetworkGraphDefaultExtSrcsGatherer interface {
	Start()
	Stop()
}

// NewNetworkGraphDefaultExtSrcsGatherer returns an instance of NetworkGraphDefaultExtSrcsGatherer as per the offline mode setting.
func NewNetworkGraphDefaultExtSrcsGatherer(networkEntityDS entityDataStore.EntityDataStore, licenseMgr manager.LicenseManager) (NetworkGraphDefaultExtSrcsGatherer, error) {
	if !features.NetworkGraphExternalSrcs.Enabled() {
		return nil, nil
	}

	var mgr NetworkGraphDefaultExtSrcsGatherer
	if env.OfflineModeEnv.BooleanSetting() {
		// TODO: support offline mode
	} else {
		mgr = &onlineDefaultExtSrcsGathererImpl{
			licenseMgr:      licenseMgr,
			networkEntityDS: networkEntityDS,
		}
	}

	if err := loadBundledData(); err != nil {
		return nil, errors.Wrap(err, "loading bundled provider networks data")
	}

	return mgr, nil
}

func loadBundledData() error {
	// TODO: add to bundle
	//if err := os.MkdirAll(defaultNetworksBaseDir, 0744); err != nil {
	//	log.Errorf("failed to create directory %q: %v", defaultNetworksBaseDir, err)
	//	return err
	//}
	//
	//if err := fileutils.CopyNoOverwrite(bundledDataFile, localDataFile); err != nil {
	//	return err
	//}
	//
	//if err := fileutils.CopyNoOverwrite(bundledChecksumFile, localChecksumFile); err != nil {
	//	return err
	//}
	return nil
}
