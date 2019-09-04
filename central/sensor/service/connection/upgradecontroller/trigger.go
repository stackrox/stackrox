package upgradecontroller

import (
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uuid"
)

func constructTriggerUpgradeMessage(upgradeProcessID string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorUpgradeTrigger{
			SensorUpgradeTrigger: &central.SensorUpgradeTrigger{
				UpgradeProcessId: upgradeProcessID,
			},
		},
	}
}

func (u *upgradeController) shouldInitiateNewUpgrade(status *storage.ClusterUpgradeStatus) bool {
	if status.GetCurrentUpgradeProcessId() == "" {
		return true
	}
	return upgradeErrorStates.Contains(status.GetCurrentUpgradeProgress().GetUpgradeState())
}

func (u *upgradeController) Trigger(ctx concurrency.Waitable) error {
	if err := u.checkErrSig(); err != nil {
		return err
	}

	injector := u.getInjector()
	if injector == nil {
		return errors.New("no active sensor connection, cannot trigger upgrade")
	}

	u.storageLock.Lock()
	defer u.storageLock.Unlock()

	clusterUpgradeStatus, err := u.getClusterUpgradeStatus()
	if err != nil {
		return err
	}

	// Check upgradability.
	switch clusterUpgradeStatus.GetUpgradability() {
	case storage.ClusterUpgradeStatus_UNSET:
		return errors.Errorf("unknown upgradability status of sensor for cluster %s; cannot trigger upgrades", u.clusterID)
	case storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED:
		return errors.Errorf("manual upgrade required for cluster %s; cannot trigger upgrade", u.clusterID)
	case storage.ClusterUpgradeStatus_UP_TO_DATE:
		return errors.Errorf("sensor for cluster %s is already up-to-date; cannot trigger upgrade", u.clusterID)
	}

	isNewUpgrade := u.shouldInitiateNewUpgrade(clusterUpgradeStatus)
	var upgradeProcessID string
	if isNewUpgrade {
		upgradeProcessID = uuid.NewV4().String()
	} else {
		upgradeProcessID = clusterUpgradeStatus.GetCurrentUpgradeProcessId()
	}

	// Always send the sensor a message about the current upgrade process. The sensor handles these
	// in an idempotent way.
	if err := injector.InjectMessage(ctx, constructTriggerUpgradeMessage(upgradeProcessID)); err != nil {
		return errors.Wrap(err, "failed to send trigger upgrade message to sensor")
	}

	// Only if it's a new upgrade, update the status of these fields in the DB.
	if isNewUpgrade {
		clusterUpgradeStatus.CurrentUpgradeProcessId = upgradeProcessID
		clusterUpgradeStatus.CurrentUpgradeInitiatedAt = types.TimestampNow()
		clusterUpgradeStatus.CurrentUpgradeProgress = &storage.UpgradeProgress{
			UpgradeState: storage.UpgradeProgress_UPGRADE_TRIGGER_SENT,
		}

		u.upgradeDoneSig.Signal()
	}

	return nil
}
