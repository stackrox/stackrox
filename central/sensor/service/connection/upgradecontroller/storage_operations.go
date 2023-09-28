package upgradecontroller

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/connection/upgradecontroller/stateutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	upgradeControllerCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))
)

func (u *upgradeController) getClusterOrError() (*storage.Cluster, error) {
	cluster, _, err := u.storage.GetCluster(upgradeControllerCtx, u.clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve cluster %q", u.clusterID)
	}
	if cluster == nil {
		return nil, errors.Errorf("cluster %q not found in DB", u.clusterID)
	}
	return cluster, nil
}

func (u *upgradeController) getCluster() *storage.Cluster {
	cluster, err := u.getClusterOrError()
	u.expectNoError(err)
	return cluster
}

func (u *upgradeController) flushUpgradeStatus() error {
	if !u.upgradeStatusChanged {
		return nil
	}

	if err := u.storage.UpdateClusterUpgradeStatus(upgradeControllerCtx, u.clusterID, u.upgradeStatus); err != nil {
		return err
	}
	u.upgradeStatusChanged = false
	return nil
}

func (u *upgradeController) setUpgradeProgress(expectedProcessID string, state storage.UpgradeProgress_UpgradeState, detail string) error {
	if expectedProcessID == "" {
		return errors.New("expected upgrade process ID must not be empty")
	}

	if u.active == nil || u.active.status.GetId() != expectedProcessID {
		return errors.Errorf("upgrade process ID %s is no longer valid, not updating upgrade progress", expectedProcessID)
	}

	prevState := u.active.status.GetProgress().GetUpgradeState()
	since := u.active.status.GetProgress().GetSince()
	if prevState != state || since == nil {
		since = types.TimestampNow()
	}

	// Carryover the detail if the state did not change and no new detail was specified.
	if detail == "" && prevState == state {
		detail = u.active.status.GetProgress().GetUpgradeStatusDetail()
	}
	u.upgradeStatus.MostRecentProcess.Progress = &storage.UpgradeProgress{
		UpgradeState:        state,
		UpgradeStatusDetail: detail,
		Since:               since,
	}
	adjustTrigger(u.active.trigger, state)

	if stateutils.TerminalStates.Contains(state) {
		u.upgradeStatus.MostRecentProcess.Active = false
		u.active = nil
	}

	u.upgradeStatusChanged = true

	if prevState != state {
		log.Infof("Changing upgrade state for cluster %s from %s to %s (detail: %s)", u.clusterID, prevState, state, detail)
	}

	return nil
}
