package gatherers

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore"
	"github.com/stackrox/rox/pkg/telemetry"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/pkg/telemetry/gatherers"
)

// CentralGatherer objects will gather and return telemetry information about this Central
type CentralGatherer struct {
	installationInfoStore store.Store

	databaseGatherer             *databaseGatherer
	apiGatherer                  *apiGatherer
	componentInfoGatherer        *gatherers.ComponentInfoGatherer
	sensorUpgradeConfigDatastore datastore.DataStore
}

func newCentralGatherer(installationInfoStore store.Store, databaseGatherer *databaseGatherer, apiGatherer *apiGatherer, componentInfoGatherer *gatherers.ComponentInfoGatherer, sensorUpgradeConfigDatastore datastore.DataStore) *CentralGatherer {
	return &CentralGatherer{
		installationInfoStore:        installationInfoStore,
		databaseGatherer:             databaseGatherer,
		apiGatherer:                  apiGatherer,
		componentInfoGatherer:        componentInfoGatherer,
		sensorUpgradeConfigDatastore: sensorUpgradeConfigDatastore,
	}
}

// Gather returns telemetry information about this Central
func (c *CentralGatherer) Gather(ctx context.Context) *data.CentralInfo {
	var errList []string
	installationInfo, _, err := c.installationInfoStore.Get(ctx)
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
		Storage:            c.databaseGatherer.Gather(ctx),
		APIStats:           c.apiGatherer.Gather(),
		Errors:             errList,
		AutoUpgradeEnabled: autoUpgradeEnabled.GetEnableAutoUpgrade(),
	}
	return centralComponent
}
