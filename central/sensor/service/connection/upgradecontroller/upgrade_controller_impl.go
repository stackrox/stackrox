package upgradecontroller

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

var (
	errUnknown = errors.New("unknown error")
)

type upgradeController struct {
	clusterID string
	errorSig  concurrency.ErrorSignal

	// The injector needs to be protected with a lock because it can change when new connections
	// to the cluster are created.
	injectorLock sync.Mutex
	injector     common.MessageInjector

	// The storage is safe for concurrent access, but we protect it with a lock to make sure
	// that the stored ClusterUpgradeStatus is in a consistent state.
	// The upgradeController for a cluster "owns" the upgrade status for that cluster.
	// When making an update, it makes sure that, when required, it reads the existing value
	// and preserves fields that have to be preserved.
	storageLock sync.Mutex
	storage     clusterStorage

	upgradeDoneSig concurrency.Signal
}

func (u *upgradeController) initialize() error {
	upgradeStatus, err := u.getClusterUpgradeStatus()
	if err != nil {
		return err
	}

	if !upgradeInProgress(upgradeStatus) {
		return nil
	}
	upgradeInitiatedAt := protoconv.ConvertTimestampToTimeOrNow(upgradeStatus.GetCurrentUpgradeInitiatedAt())
	upgradeDeadline := upgradeInitiatedAt.Add(upgradeAttemptTimeout)
	if upgradeDeadline.Before(time.Now()) {
		return u.setUpgradeProgress(upgradeStatus.GetCurrentUpgradeProcessId(), storage.UpgradeProgress_UPGRADE_TIMED_OUT, "")
	}
	go u.markUpgradeTimedOutAt(upgradeDeadline, upgradeStatus.GetCurrentUpgradeProcessId())
	return nil
}

func (u *upgradeController) ErrorSignal() concurrency.ReadOnlyErrorSignal {
	return &u.errorSig
}

func (u *upgradeController) setInjector(injector common.MessageInjector) {
	u.injectorLock.Lock()
	defer u.injectorLock.Unlock()
	u.injector = injector
}

func (u *upgradeController) getInjector() common.MessageInjector {
	u.injectorLock.Lock()
	defer u.injectorLock.Unlock()
	return u.injector
}

func (u *upgradeController) checkErrSig() error {
	if err := u.errorSig.ErrorWithDefault(errUnknown); err != nil {
		return errors.Wrapf(err, "upgrade controller for cluster %s is in error state", u.clusterID)
	}
	return nil
}
