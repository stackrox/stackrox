package upgradecontroller

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timeutil"
)

const (
	upgradeAttemptTimeout = 10 * time.Minute
)

func (u *upgradeController) markUpgradeTimedOutAt(deadline time.Time, upgradeProcessID string) {
	u.upgradeDoneSig.Reset()

	waitTime := time.Until(deadline)

	// NewTimer handles a negative waitTime correctly.
	timer := time.NewTimer(waitTime)

	select {
	case <-timer.C:
		u.storageLock.Lock()
		defer u.storageLock.Unlock()
		upgradeStatus, err := u.getClusterUpgradeStatus()
		if err != nil {
			u.errorSig.SignalWithError(err)
			return
		}
		if upgradeStatus.GetCurrentUpgradeProcessId() != upgradeProcessID {
			log.Infof("Not marking upgrade %s as timed out, since a new upgrade (%s) was started.", upgradeStatus.GetCurrentUpgradeProcessId(), upgradeProcessID)
			return
		}
		// If the upgrade is still in an in-progress state, it has now officially timed out.
		if upgradeInProgress(upgradeStatus) {
			upgradeStatus.CurrentUpgradeProgress.UpgradeState = storage.UpgradeProgress_UPGRADE_TIMED_OUT
			upgradeStatus.CurrentUpgradeProgress.UpgradeStatusDetail = ""
			if err := u.storage.UpdateClusterUpgradeStatus(upgradeControllerCtx, u.clusterID, upgradeStatus); err != nil {
				u.errorSig.SignalWithError(errors.Wrap(err, "failed to mark upgrade timed out"))
			}
		}
	case <-u.upgradeDoneSig.Done():
		timeutil.StopTimer(timer)
	}
}
