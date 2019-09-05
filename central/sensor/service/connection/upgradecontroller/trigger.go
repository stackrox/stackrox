package upgradecontroller

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
)

func constructTriggerUpgradeRequest(cluster *storage.Cluster, process *storage.ClusterUpgradeStatus_UpgradeProcessStatus) *central.SensorUpgradeTrigger {
	return &central.SensorUpgradeTrigger{
		UpgradeProcessId: process.GetId(),
		Image:            process.GetUpgraderImage(),
		// TODO: remove sleeps and adjust command, this is just for better debuggability
		Command: []string{"sh", "-c", "sleep 10 ; sensor-upgrader -workflow roll-forward && sleep 30 && sensor-upgrader -workflow cleanup"},
		EnvVars: []*central.SensorUpgradeTrigger_EnvVarDef{
			{
				Name:         env.ClusterID.EnvVar(),
				SourceEnvVar: env.ClusterID.EnvVar(),
				DefaultValue: cluster.GetId(),
			},
			{
				Name:         env.CentralEndpoint.EnvVar(),
				SourceEnvVar: env.CentralEndpoint.EnvVar(),
				DefaultValue: cluster.GetCentralApiEndpoint(),
			},
			{
				Name:         "ROX_UPGRADE_PROCESS_ID",
				DefaultValue: process.GetId(),
			},
		},
	}
}

func sendTrigger(ctx concurrency.Waitable, injector common.MessageInjector, trigger *central.SensorUpgradeTrigger) error {
	if injector == nil {
		return errors.New("sensor is not connected")
	}

	if trigger == nil {
		return nil
	}

	return injector.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorUpgradeTrigger{
			SensorUpgradeTrigger: trigger,
		},
	})
}

func (u *upgradeController) Trigger(ctx concurrency.Waitable) error {
	var injector common.MessageInjector
	var trigger *central.SensorUpgradeTrigger

	err := u.do(func() error {
		var err error
		injector, trigger, err = u.doTrigger()
		return err
	})
	if err != nil {
		return err
	}

	return sendTrigger(ctx, injector, trigger)
}

func (u *upgradeController) doTrigger() (common.MessageInjector, *central.SensorUpgradeTrigger, error) {
	if u.injector == nil {
		return nil, nil, errors.Errorf("no active sensor connection for cluster %s exists, cannot trigger upgrade", u.clusterID)
	}

	if u.active != nil {
		return nil, nil, errors.Errorf("an upgrade is already in progress in cluster %s", u.clusterID)
	}

	// Check upgradability.
	switch u.upgradeStatus.GetUpgradability() {
	case storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE:
		// yay!
	case storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED:
		return nil, nil, errors.Errorf("manual upgrade required for cluster %s; cannot trigger upgrade", u.clusterID)
	case storage.ClusterUpgradeStatus_UP_TO_DATE:
		return nil, nil, errors.Errorf("sensor for cluster %s is already up-to-date; cannot trigger upgrade", u.clusterID)
	default:
		return nil, nil, errors.Errorf("unknown upgradability status of sensor for cluster %s; cannot trigger upgrades", u.clusterID)
	}

	cluster := u.getCluster()
	process, err := newUpgradeProcess(cluster)
	if err != nil {
		return nil, nil, err
	}

	u.makeProcessActive(cluster, process)

	return u.injector, u.active.trigger, nil
}
