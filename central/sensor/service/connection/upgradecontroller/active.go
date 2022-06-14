package upgradecontroller

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/protoconv"
	"github.com/stackrox/stackrox/pkg/utils"
	"github.com/stackrox/stackrox/pkg/version"
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
		utils.Should(errors.Errorf("Making process %s active when there already is an active one. This should not happen...", processStatus.GetId()))
	}

	u.active = &activeUpgradeInfo{
		trigger: constructTriggerUpgradeRequest(cluster, processStatus),
		status:  processStatus,
	}
	u.upgradeStatus.MostRecentProcess = processStatus
	u.upgradeStatusChanged = true
	go u.reconcileUpgradeStateRegularly(processStatus.GetId())
}

func (u *upgradeController) maybeTimeoutUpgrade(processID string) error {
	currState := u.active.status.GetProgress().GetUpgradeState()
	var relevantTimestamp *types.Timestamp
	if currState == storage.UpgradeProgress_UPGRADE_INITIALIZING {
		relevantTimestamp = u.active.status.GetInitiatedAt()
	}
	if relevantTimestamp == nil {
		relevantTimestamp = u.active.status.GetProgress().GetSince()
	}
	if relevantTimestamp == nil {
		// This should never happen -- it violates one of our invariants.
		return errors.Errorf("got no relevant timestamp for upgrade controller with status: %+v", u.upgradeStatus)
	}
	relevantGoTime := protoconv.ConvertTimestampToTimeOrNow(relevantTimestamp)
	if time.Since(relevantGoTime) > u.timeouts.AbsoluteNoProgressTimeout() {
		return u.setUpgradeProgress(processID, storage.UpgradeProgress_UPGRADE_TIMED_OUT, fmt.Sprintf("The upgrade has been aborted due to timeout -- it was stuck in the %s state for too long.", currState))
	}
	return nil
}

func (u *upgradeController) maybeReconcileStateWithActiveConnInfo(processID string) (bool, error) {
	// No active connection, OR the sensor is too old. In either case, not interesting
	// to this function.
	if u.activeSensorConn == nil || u.activeSensorConn.sensorVersion == "" {
		return false, nil
	}

	// If it's a cert rotation, then we don't need to do any state reconciliation.
	if u.active.status.Type == storage.ClusterUpgradeStatus_UpgradeProcessStatus_CERT_ROTATION {
		return false, nil
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
		if err := u.setUpgradeProgress(processID, storage.UpgradeProgress_UPGRADE_COMPLETE, ""); err != nil {
			return false, err
		}
		return true, nil
	}

	// Next, if the state is ROLLING_BACK, check if we have an active connection with a rolled back sensor.
	// If so, mark the upgrade as rolled back. (This is not perfect, since it doesn't guarantee that all
	// the other objects in the cluster have been rolled back, but it's the best we can do until we have
	// better in-product health checks...)
	if currState == storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK && versionCmp < 0 {
		if time.Since(protoconv.ConvertTimestampToTimeOrNow(u.active.status.GetProgress().GetSince())) < u.timeouts.RollBackSuccessPeriod() {
			// Don't mark a rollback as successful too early.
			return false, nil
		}
		newDetail := fmt.Sprintf("The upgrade was rolled back due to an error in the previous stage. Details: %s", u.active.status.GetProgress().GetUpgradeStatusDetail())
		if err := u.setUpgradeProgress(processID, storage.UpgradeProgress_UPGRADE_ERROR_ROLLED_BACK, newDetail); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (u *upgradeController) reconcileUpgradeStateRegularly(processID string) {
	t := time.NewTicker(u.timeouts.StateReconcilePollInterval())
	defer t.Stop()
	for range t.C {
		var done bool

		// This function should never return an error unless there's a programming mistake.
		// Note that setUpgradeProgress does no DB operations.
		utils.Should(u.do(func() error {
			// The upgrade progress we were monitoring is complete. Exit this goroutine.
			if u.active == nil || u.active.status.GetId() != processID {
				done = true
				return nil
			}

			if upgradeIsDone, err := u.maybeReconcileStateWithActiveConnInfo(processID); err != nil {
				return err
			} else if upgradeIsDone {
				done = true
				return nil
			}

			if err := u.maybeTimeoutUpgrade(processID); err != nil {
				return err
			}
			return nil
		}))

		if done {
			return
		}
	}
}
