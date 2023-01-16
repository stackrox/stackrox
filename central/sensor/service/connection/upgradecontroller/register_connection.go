package upgradecontroller

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

func (u *upgradeController) RegisterConnection(sensorCtx context.Context, conn SensorConn) concurrency.ErrorWaitable {
	var errCond concurrency.ErrorWaitable

	utils.Should(u.do(func() error {
		u.errorSig.Reset()

		if u.doHandleNewConnection(sensorCtx, conn) {
			errCond = u.errorSig.Snapshot()
		}
		return nil
	}))

	if errCond == nil { // indicates connection was not accepted (useless, sensor version too old)
		return nil
	}

	go u.watchConnection(sensorCtx, conn)

	return errCond
}

func (u *upgradeController) watchConnection(sensorCtx context.Context, conn SensorConn) {
	select {
	case <-u.errorSig.Done():
		return
	case <-sensorCtx.Done():
	}

	utils.Should(u.do(func() error {
		if u.activeSensorConn != nil && u.activeSensorConn.conn == conn {
			u.activeSensorConn = nil
		}
		return nil
	}))
}

func determineUpgradabilityFromVersionInfoAndConn(sensorVersion string, conn SensorConn) (storage.ClusterUpgradeStatus_Upgradability, string) {
	if sensorVersion == "" {
		return storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED, "sensor is from an old version that doesn't support auto-upgrade"
	}

	if sensorVersion == version.GetMainVersion() {
		return storage.ClusterUpgradeStatus_UP_TO_DATE, "sensor is running the same version as Central"
	}

	// Check if the connection supports auto-upgrade.
	if err := conn.CheckAutoUpgradeSupport(); err != nil {
		return storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED, err.Error()
	}
	cmp := version.CompareReleaseVersions(sensorVersion, version.GetMainVersion())
	// The sensor is newer! See comments on the below enum value in the proto file
	// for more details on how we handle this case.
	if cmp > 0 {
		return storage.ClusterUpgradeStatus_SENSOR_VERSION_HIGHER, fmt.Sprintf("sensor is running a newer version (%s)", sensorVersion)
	}

	// We don't differentiate between cmp == -1 and cmp == 0.
	// The former means we definitely know sensor is an older version.
	// The latter means we don't know (ex: we're on a development version)
	// In such a case, it seems reasonable to assume that the sensor is older.
	// Ideally, we would panic if cmp == 0 on release builds, since that should
	// only happen if the versions are exactly equal (which is checked above),
	// but panic-ing on release builds doesn't help anyone with on-prem software, so...
	return storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE, fmt.Sprintf("sensor is running an old version (%s)", sensorVersion)
}

func (u *upgradeController) maybeTriggerAutoUpgrade() {
	if !u.shouldAutoTriggerUpgrade() {
		return
	}
	cluster := u.getCluster()
	process, err := u.newUpgradeProcess()
	if err != nil {
		// This is not a critical error, it just means we can't auto-trigger. NBD.
		log.Errorf("Cannot automatically trigger auto-upgrade for sensor in cluster %s: %v", u.clusterID, err)
	} else {
		u.makeProcessActive(cluster, process)
	}
}

func (u *upgradeController) reconcileInitialUpgradeStatus(sensorVersion string, conn SensorConn) {
	upgradability, reason := determineUpgradabilityFromVersionInfoAndConn(sensorVersion, conn)
	log.Infof("Determined upgradability status for sensor from cluster %s: %s. Reason: %s", u.clusterID, upgradability, reason)
	u.upgradeStatus.Upgradability, u.upgradeStatus.UpgradabilityStatusReason = upgradability, reason
	u.upgradeStatusChanged = true // we don't check for this but sensor checking in should be comparatively rare

	// No active upgrade process. Maybe trigger an auto-upgrade.
	if u.active == nil {
		u.maybeTriggerAutoUpgrade()
	}
}

func (u *upgradeController) doHandleNewConnection(sensorCtx context.Context, conn SensorConn) (sensorSupportsAutoUpgrade bool) {
	sensorVersion := conn.SensorVersion()
	u.reconcileInitialUpgradeStatus(sensorVersion, conn)

	// Special case: if the sensor is too old to support auto upgrades, then don't send it a trigger that
	// it will not know how to parse.
	if u.upgradeStatus.Upgradability == storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED {
		return false
	}

	// In either case, send the sensor a message telling it about the upgrade status.
	var trigger *central.SensorUpgradeTrigger
	if u.active != nil {
		// Since we send the trigger asynchronously, clone the object -- we do modify the trigger
		// sometimes, and don't want to cause a race.
		trigger = u.active.trigger.Clone()
	} else {
		trigger = &central.SensorUpgradeTrigger{} // empty trigger indicates "no upgrade should be in progress"
	}

	// Send the trigger asynchronously - we are holding the lock so don't do anything blocking.
	go func() {
		if err := sendTrigger(sensorCtx, conn, trigger); err != nil {
			log.Errorf("Could not send initial upgrade trigger: %v. Connection went away before being fully registered?", err)
		}
	}()
	u.activeSensorConn = &activeSensorConnectionInfo{
		conn:          conn,
		sensorVersion: sensorVersion,
	}
	return true
}
