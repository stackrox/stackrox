package upgradecontroller

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

var (
	// nonRecordableUpgradeStates are upgrade states that cannot be set through the API.
	// These states can only be set by the logic in Central.
	nonRecordableUpgradeStates = set.NewFrozenStorageUpgradeProgress_UpgradeStateSet(
		storage.UpgradeProgress_UNSET,
		storage.UpgradeProgress_UPGRADE_COMPLETE,
		storage.UpgradeProgress_UPGRADE_TRIGGER_SENT,
		storage.UpgradeProgress_UPGRADE_TIMED_OUT,
	)

	upgradeInProgressStates = set.NewFrozenStorageUpgradeProgress_UpgradeStateSet(
		storage.UpgradeProgress_UPGRADE_TRIGGER_SENT,
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
		storage.UpgradeProgress_UPGRADE_OPERATIONS_DONE,
	)

	upgradeErrorStates = set.NewFrozenStorageUpgradeProgress_UpgradeStateSet(
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_FAILED,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLBACK_FAILED,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLED_BACK,
		storage.UpgradeProgress_UPGRADE_TIMED_OUT,
	)
)

func upgradeInProgress(upgradeStatus *storage.ClusterUpgradeStatus) bool {
	if upgradeStatus.GetCurrentUpgradeProcessId() == "" {
		return false
	}
	return upgradeInProgressStates.Contains(upgradeStatus.GetCurrentUpgradeProgress().GetUpgradeState())
}
