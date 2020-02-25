package gatherers

import (
	"github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/central/license/manager"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/pkg/telemetry"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/pkg/telemetry/gatherers"
)

// CentralGatherer objects will gather and return telemetry information about this Central
type CentralGatherer struct {
	licenseMgr            manager.LicenseManager
	installationInfoStore store.Store

	databaseGatherer      *databaseGatherer
	apiGatherer           *apiGatherer
	componentInfoGatherer *gatherers.ComponentInfoGatherer
}

func newCentralGatherer(licenseMgr manager.LicenseManager, installationInfoStore store.Store, databaseGatherer *databaseGatherer, apiGatherer *apiGatherer, componentInfoGatherer *gatherers.ComponentInfoGatherer) *CentralGatherer {
	return &CentralGatherer{
		licenseMgr:            licenseMgr,
		installationInfoStore: installationInfoStore,
		databaseGatherer:      databaseGatherer,
		apiGatherer:           apiGatherer,
		componentInfoGatherer: componentInfoGatherer,
	}
}

// Gather returns telemetry information about this Central
func (c *CentralGatherer) Gather() *data.CentralInfo {
	var activeLicense *licenseproto.License
	if c.licenseMgr != nil {
		activeLicense = c.licenseMgr.GetActiveLicense()
	}

	var errList []string
	installationInfo, err := c.installationInfoStore.GetInstallationInfo()
	if err != nil {
		errList = append(errList, err.Error())
	}

	centralComponent := &data.CentralInfo{
		RoxComponentInfo: c.componentInfoGatherer.Gather(),
		// Despite GoLand's warning it's okay for installationInfo to be nil, GetID() will return ""
		ID:               installationInfo.GetId(),
		InstallationTime: telemetry.GetTimeOrNil(installationInfo.GetCreated()),
		License:          (*data.LicenseJSON)(activeLicense),
		Storage:          c.databaseGatherer.Gather(),
		APIStats:         c.apiGatherer.Gather(),
		Errors:           errList,
	}
	return centralComponent
}
