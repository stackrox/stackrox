package upgradecontroller

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/version"
)

func (u *upgradeController) RegisterConnection(sensorCtx context.Context, injector common.MessageInjector) concurrency.ErrorWaitable {
	var errCond concurrency.ErrorWaitable

	errorhelpers.PanicOnDevelopment(u.do(func() error {
		u.errorSig.Reset()

		if u.doHandleNewConnection(sensorCtx, injector) {
			errCond = u.errorSig.Snapshot()
		}
		return nil
	}))

	if errCond == nil { // indicates connection was not accepted (useless, sensor version too old)
		return nil
	}

	go u.watchConnection(sensorCtx, injector)

	return errCond
}

func (u *upgradeController) watchConnection(sensorCtx context.Context, injector common.MessageInjector) {
	select {
	case <-u.errorSig.Done():
		return
	case <-sensorCtx.Done():
	}

	errorhelpers.PanicOnDevelopment(u.do(func() error {
		if u.injector == injector {
			u.injector = nil
		}
		return nil
	}))
}

func determineUpgradabilityFromVersionInfo(versionInfo *centralsensor.SensorVersionInfo) (storage.ClusterUpgradeStatus_Upgradability, string) {
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

func (u *upgradeController) markUpgradeDone(state storage.UpgradeProgress_UpgradeState) {
	if u.active == nil {
		return
	}

	errorhelpers.PanicOnDevelopment(u.setUpgradeProgress(u.active.status.GetId(), state, ""))
	u.active.status.Active = false
	u.upgradeStatusChanged = true
	u.active = nil
}

func (u *upgradeController) reconcileInitialUpgradeStatus(versionInfo *centralsensor.SensorVersionInfo) {
	upgradability, reason := determineUpgradabilityFromVersionInfo(versionInfo)
	log.Infof("Determined upgradability status for sensor from cluster %s: %s. Reason: %s", u.clusterID, upgradability, reason)
	u.upgradeStatus.Upgradability, u.upgradeStatus.UpgradabilityStatusReason = upgradability, reason
	u.upgradeStatusChanged = true // we don't check for this but sensor checking in should be comparatively rare

	if u.active != nil {
		// Check relative to the target version, not central's current version (we might have upgraded central since
		// the upgrade was initiated). If the versions are incomparable, we assume the upgrade is not complete, otherwise
		// we erroneously mark upgrades as complete when testing with dev builds.
		versionCmp := version.CompareReleaseVersionsOr(versionInfo.MainVersion, u.active.status.GetTargetVersion(), -1)

		state := u.active.status.GetProgress().GetUpgradeState()
		if versionCmp >= 0 /* TODO: && state == storage.UpgradeProgress_UPGRADE_OPERATIONS_DONE */ {
			u.markUpgradeDone(storage.UpgradeProgress_UPGRADE_COMPLETE)
		} else if versionCmp < 0 && state == storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK {
			u.markUpgradeDone(storage.UpgradeProgress_UPGRADE_ERROR_ROLLED_BACK)
		}
	} else if u.shouldAutoTriggerUpgrade() { // && active == nil
		cluster := u.getCluster()
		process, err := u.newUpgradeProcess()
		if err != nil {
			// This is not a critical error, it just means we can't auto-trigger. NBD.
			log.Errorf("Cannot automatically trigger auto-upgrade for sensor in cluster %s: %v", u.clusterID, err)
		} else {
			u.makeProcessActive(cluster, process)
		}
	}
}

func (u *upgradeController) doHandleNewConnection(sensorCtx context.Context, injector common.MessageInjector) bool {
	versionInfo, err := centralsensor.DeriveSensorVersionInfo(sensorCtx)
	if err != nil {
		u.injector = nil
		log.Errorf("Could not derive sensor version info for cluster %s from context: %v. Auto-upgrade functionality will not work.", u.clusterID, err)
		return false
	}

	u.reconcileInitialUpgradeStatus(versionInfo)

	// In either case, send the sensor a message telling it about the upgrade status.
	var trigger *central.SensorUpgradeTrigger
	if u.active != nil {
		trigger = u.active.trigger
	} else {
		trigger = &central.SensorUpgradeTrigger{} // empty trigger indicates "no upgrade should be in progress"
	}

	// Send the trigger asynchronously - we are holding the lock so don't do anything blocking.
	go func() {
		if err := sendTrigger(sensorCtx, injector, trigger); err != nil {
			log.Errorf("Could not send initial upgrade trigger: %v. Connection went away before being fully registered?", err)
		}
	}()
	u.injector = injector
	return true
}
