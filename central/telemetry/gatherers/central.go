package gatherers

import (
	"context"
	"fmt"

	"github.com/stackrox/stackrox/central/installation/store"
	"github.com/stackrox/stackrox/central/license/manager"
	"github.com/stackrox/stackrox/central/sensorupgradeconfig/datastore"
	licenseproto "github.com/stackrox/stackrox/generated/shared/license"
	"github.com/stackrox/stackrox/pkg/telemetry"
	"github.com/stackrox/stackrox/pkg/telemetry/data"
	"github.com/stackrox/stackrox/pkg/telemetry/gatherers"
)

// CentralGatherer objects will gather and return telemetry information about this Central
type CentralGatherer struct {
	licenseMgr            manager.LicenseManager
	installationInfoStore store.Store

	databaseGatherer             *databaseGatherer
	apiGatherer                  *apiGatherer
	componentInfoGatherer        *gatherers.ComponentInfoGatherer
	sensorUpgradeConfigDatastore datastore.DataStore
}

func newCentralGatherer(licenseMgr manager.LicenseManager, installationInfoStore store.Store, databaseGatherer *databaseGatherer, apiGatherer *apiGatherer, componentInfoGatherer *gatherers.ComponentInfoGatherer, sensorUpgradeConfigDatastore datastore.DataStore) *CentralGatherer {
	return &CentralGatherer{
		licenseMgr:                   licenseMgr,
		installationInfoStore:        installationInfoStore,
		databaseGatherer:             databaseGatherer,
		apiGatherer:                  apiGatherer,
		componentInfoGatherer:        componentInfoGatherer,
		sensorUpgradeConfigDatastore: sensorUpgradeConfigDatastore,
	}
}

// Gather returns telemetry information about this Central
func (c *CentralGatherer) Gather(ctx context.Context) *data.CentralInfo {
	var activeLicense *licenseproto.License
	if c.licenseMgr != nil {
		activeLicense = c.licenseMgr.GetActiveLicense()
	}

	var errList []string
	installationInfo, err := c.installationInfoStore.GetInstallationInfo()
	if err != nil {
		errList = append(errList, fmt.Sprintf("Installation info error: %v", err.Error()))
	}

	autoUpgradeEnabled, err := c.sensorUpgradeConfigDatastore.GetSensorUpgradeConfig(ctx)
	if err != nil {
		errList = append(errList, fmt.Sprintf("Sensor upgrade config error: %v", err.Error()))
	}

	centralComponent := &data.CentralInfo{
		RoxComponentInfo: c.componentInfoGatherer.Gather(),
		// Despite GoLand's warning it's okay for installationInfo to be nil, GetID() will return ""
		ID:                 installationInfo.GetId(),
		InstallationTime:   telemetry.GetTimeOrNil(installationInfo.GetCreated()),
		License:            (*data.LicenseJSON)(activeLicense),
		Storage:            c.databaseGatherer.Gather(),
		APIStats:           c.apiGatherer.Gather(),
		Errors:             errList,
		AutoUpgradeEnabled: autoUpgradeEnabled.GetEnableAutoUpgrade(),
	}
	return centralComponent
}
