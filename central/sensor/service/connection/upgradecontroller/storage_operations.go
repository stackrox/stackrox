package upgradecontroller

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	upgradeControllerCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))
)

func (u *upgradeController) getCluster() (*storage.Cluster, error) {
	cluster, _, err := u.storage.GetCluster(upgradeControllerCtx, u.clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve cluster %q", u.clusterID)
	}
	if cluster == nil {
		return nil, errors.Errorf("cluster %q not found in DB", u.clusterID)
	}
	return cluster, nil
}

// getClusterUpgradeStatus gets the upgrade status for the given cluster from storage.
// It returns an error if the cluster doesn't exist, or if there's an error.
// The error it returns will be wrapped and formatted.
// It ALWAYS returns a non-nil cluster upgrade status if err == nil
// (if the cluster in the DB had a nil cluster upgrade status, it allocates a new, empty, object).
func (u *upgradeController) getClusterUpgradeStatus() (*storage.ClusterUpgradeStatus, error) {
	cluster, err := u.getCluster()
	if err != nil {
		return nil, err
	}
	if upgradeStatus := cluster.GetStatus().GetUpgradeStatus(); upgradeStatus != nil {
		return upgradeStatus, nil
	}
	return &storage.ClusterUpgradeStatus{}, nil
}

func (u *upgradeController) setUpgradeStatus(status *storage.ClusterUpgradeStatus) error {
	if err := u.storage.UpdateClusterUpgradeStatus(upgradeControllerCtx, u.clusterID, status); err != nil {
		return errors.Wrapf(err, "failed to update cluster status for %q", u.clusterID)
	}
	return nil
}

func (u *upgradeController) setUpgradeStatusOrTerminate(status *storage.ClusterUpgradeStatus) {
	if err := u.setUpgradeStatus(status); err != nil {
		u.errorSig.SignalWithError(err)
	}
}

func (u *upgradeController) setUpgradeProgress(expectedProcessID string, state storage.UpgradeProgress_UpgradeState, detail string) error {
	u.storageLock.Lock()
	defer u.storageLock.Unlock()
	upgradeStatus, err := u.getClusterUpgradeStatus()
	if err != nil {
		return err
	}
	if upgradeStatus.GetCurrentUpgradeProcessId() != expectedProcessID {
		return errors.Errorf("upgrade process ID %s is now old, not updating upgrade process", expectedProcessID)
	}
	upgradeStatus.CurrentUpgradeProgress = &storage.UpgradeProgress{
		UpgradeState:        state,
		UpgradeStatusDetail: detail,
	}
	return u.storage.UpdateClusterUpgradeStatus(upgradeControllerCtx, u.clusterID, upgradeStatus)
}
