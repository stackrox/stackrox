package upgradecontroller

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

func (u *upgradeController) RecordUpgradeProgress(upgradeProcessID string, upgradeProgress *storage.UpgradeProgress) error {
	if err := u.checkErrSig(); err != nil {
		return err
	}

	u.storageLock.Lock()
	defer u.storageLock.Unlock()
	upgradeStatus, err := u.getClusterUpgradeStatus()
	if err != nil {
		return err
	}
	if upgradeStatus.GetCurrentUpgradeProcessId() != upgradeProcessID {
		return errors.Errorf("current upgrade process id (%s) is different; perhaps this upgrade process id (%s) has timed out?", upgradeStatus.GetCurrentUpgradeProcessId(), upgradeProcessID)
	}
	if nonRecordableUpgradeStates.Contains(upgradeProgress.GetUpgradeState()) {
		return errors.Errorf("upgrade state %s cannot be recorded by API", upgradeProgress.GetUpgradeState())
	}
	upgradeStatus.CurrentUpgradeProgress = upgradeProgress
	return u.setUpgradeStatus(upgradeStatus)
}
