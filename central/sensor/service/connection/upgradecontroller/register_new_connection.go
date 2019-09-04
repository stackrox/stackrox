package upgradecontroller

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/version"
)

func (u *upgradeController) RegisterConnection(sensorCtx context.Context, injector common.MessageInjector) {
	u.errorSig.Reset()
	u.setInjector(injector)
	u.handleNewConnection(sensorCtx)
}

func (u *upgradeController) determineUpgradabilityFromVersionInfo(versionInfo *centralsensor.SensorVersionInfo) (storage.ClusterUpgradeStatus_Upgradability, string) {
	if versionInfo == nil {
		return storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED, "sensor is from an old version that doesn't support auto-upgrade"
	}

	if versionInfo.MainVersion == version.GetMainVersion() {
		return storage.ClusterUpgradeStatus_UP_TO_DATE, "sensor is running the same version as Central"
	}
	cmp := version.CompareReleaseVersions(versionInfo.MainVersion, version.GetMainVersion())
	// The sensor is newer! See comments on the below enum value in the proto file
	// for more details on how we handle this case.
	if cmp > 0 {
		return storage.ClusterUpgradeStatus_SENSOR_VERSION_HIGHER, fmt.Sprintf("sensor is running a newer version (%s)", versionInfo.MainVersion)
	}
	// We don't differentiate between cmp == -1 and cmp == 0.
	// The former means we definitely know sensor is an older version.
	// The latter means we don't know (ex: we're on a development version)
	// In such a case, it seems reasonable to assume that the sensor is older.
	// Ideally, we would panic if cmp == 0 on release builds, since that should
	// only happen if the versions are exactly equal (which is checked above),
	// but panic-ing on release builds doesn't help anyone with on-prem software, so...
	return storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE, fmt.Sprintf("sensor is running an old version (%s)", versionInfo.MainVersion)
}

func (u *upgradeController) reconcileInitialUpgradeStatus(versionInfo *centralsensor.SensorVersionInfo) {
	upgradability, reason := u.determineUpgradabilityFromVersionInfo(versionInfo)
	log.Infof("Determined upgradability status for sensor from cluster %s: %s. Reason: %s", u.clusterID, upgradability, reason)

	u.storageLock.Lock()
	defer u.storageLock.Unlock()

	// Merge these fields into the existing upgrade status.
	upgradeStatus, err := u.getClusterUpgradeStatus()
	if err != nil {
		u.errorSig.SignalWithError(err)
		return
	}

	upgradeStatus.Upgradability = upgradability
	upgradeStatus.UpgradabilityStatusReason = reason

	// If we have an upgrade in progress, and the sensor that has checked in is
	// is an up-to-date one, mark the upgrade as a success.
	if upgradeInProgress(upgradeStatus) && upgradability == storage.ClusterUpgradeStatus_UP_TO_DATE {
		upgradeStatus.CurrentUpgradeProgress.UpgradeState = storage.UpgradeProgress_UPGRADE_COMPLETE
		u.upgradeDoneSig.Signal()
	}

	u.setUpgradeStatusOrTerminate(upgradeStatus)
}

func (u *upgradeController) handleNewConnection(sensorCtx context.Context) {
	versionInfo, err := centralsensor.DeriveSensorVersionInfo(sensorCtx)
	if err != nil {
		// This ONLY happens when the sensor gives an inconsistent version, so panic on development.
		u.errorSig.SignalWithErrorf("couldn't derive version info from context: %v", errorhelpers.PanicOnDevelopment(err))
		return
	}

	u.reconcileInitialUpgradeStatus(versionInfo)
}
