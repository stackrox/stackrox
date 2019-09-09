package upgradecontroller

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/version"
)

type activeUpgradeInfo struct {
	trigger *central.SensorUpgradeTrigger
	status  *storage.ClusterUpgradeStatus_UpgradeProcessStatus
}

func (u *upgradeController) makeProcessActive(cluster *storage.Cluster, processStatus *storage.ClusterUpgradeStatus_UpgradeProcessStatus) {
	if !processStatus.GetActive() {
		u.active = nil
		return
	}

	if u.active != nil {
		errorhelpers.PanicOnDevelopmentf("Making process %s active when there already is an active one. This should not happen...", processStatus.GetId())
	}

	u.active = &activeUpgradeInfo{
		trigger: constructTriggerUpgradeRequest(cluster, processStatus),
		status:  processStatus,
	}
	u.upgradeStatus.MostRecentProcess = processStatus
	u.upgradeStatusChanged = true
	go u.reconcileUpgradeStateRegularly(processStatus.GetId())
}

func (u *upgradeController) reconcileUpgradeStateRegularly(processID string) {
	t := time.NewTicker(u.timeouts.StateReconcilePollInterval())
	defer t.Stop()
	for range t.C {
		var done bool

		// This function should never return an error.
		// Note that setUpgradeProgress does no DB operations.
		errorhelpers.PanicOnDevelopment(u.do(func() error {
			// The upgrade progress we were monitoring is complete. Exit this goroutine.
			if u.active == nil || u.active.status.GetId() != processID {
				done = true
				return nil
			}

			// No active connection, OR the sensor is too old. In either case, not interesting
			// to this function.
			if u.activeSensorConn == nil || u.activeSensorConn.sensorVersion == "" {
				return nil
			}
			// We check relative to the target version, not central's current version, because we might have upgraded central since
			// the upgrade was initiated. If the versions are incomparable, we assume the upgrade is not complete, otherwise
			// we erroneously mark upgrades as complete when testing with dev builds.
			versionCmp := version.CompareReleaseVersionsOr(u.activeSensorConn.sensorVersion, u.active.status.GetTargetVersion(), -1)

			currState := u.active.status.GetProgress().GetUpgradeState()
			// Couple of simple checks.

			// First, if the state is UPGRADE_OPERATIONS_DONE, check if an up-to-date sensor has connected.
			// If it has, mark the upgrade complete!
			if currState == storage.UpgradeProgress_UPGRADE_OPERATIONS_DONE && versionCmp >= 0 {
				return u.setUpgradeProgress(processID, storage.UpgradeProgress_UPGRADE_COMPLETE, "")
			}

			// Next, if the state is ROLLING_BACK, check if we have an active connection with a rolled back sensor.
			// If so, mark the upgrade as rolled back. (This is not perfect, since it doesn't guarantee that all
			// the other objects in the cluster have been rolled back, but it's the best we can do until we have
			// better in-product health checks...)
			if currState == storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK && versionCmp < 0 {
				return u.setUpgradeProgress(processID, storage.UpgradeProgress_UPGRADE_ERROR_ROLLED_BACK, "")
			}

			return nil
		}))

		if done {
			return
		}
	}
}
