package upgradecontroller

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sensorupgrader"
)

func constructTriggerUpgradeRequest(cluster *storage.Cluster, process *storage.ClusterUpgradeStatus_UpgradeProcessStatus) *central.SensorUpgradeTrigger {
	t := &central.SensorUpgradeTrigger{
		UpgradeProcessId: process.GetId(),
		Image:            process.GetUpgraderImage(),
		Command:          []string{"sensor-upgrader"},
		EnvVars: []*central.SensorUpgradeTrigger_EnvVarDef{
			{
				Name:         env.CentralEndpoint.EnvVar(),
				SourceEnvVar: env.CentralEndpoint.EnvVar(),
				DefaultValue: cluster.GetCentralApiEndpoint(),
			},
			{
				Name:         "ROX_UPGRADE_PROCESS_ID",
				DefaultValue: process.GetId(),
			},
			{
				Name:         sensorupgrader.ClusterIDEnvVarName,
				DefaultValue: cluster.GetId(),
			},
		},
	}
	if process.GetType() == storage.ClusterUpgradeStatus_UpgradeProcessStatus_CERT_ROTATION {
		t.EnvVars = append(t.EnvVars, &central.SensorUpgradeTrigger_EnvVarDef{
			Name:         env.UpgraderCertsOnly.EnvVar(),
			DefaultValue: "true",
		})
	}
	adjustTrigger(t, process.GetProgress().GetUpgradeState())
	return t
}

func adjustTrigger(trigger *central.SensorUpgradeTrigger, state storage.UpgradeProgress_UpgradeState) {
	if state >= storage.UpgradeProgress_UPGRADER_LAUNCHED {
		trigger.Image = "" // indicate to sensor that it should not launch another upgrader
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
	return u.trigger(ctx, u.doTriggerUpgrade)
}

func (u *upgradeController) TriggerCertRotation(ctx concurrency.Waitable) error {
	return u.trigger(ctx, u.doTriggerCertRotation)
}

func (u *upgradeController) trigger(ctx concurrency.Waitable, triggerGenerateFunc func() (common.MessageInjector, *central.SensorUpgradeTrigger, error)) error {
	var injector common.MessageInjector
	var trigger *central.SensorUpgradeTrigger

	err := u.do(func() error {
		var err error
		injector, trigger, err = triggerGenerateFunc()
		return err
	})
	if err != nil {
		return err
	}

	return sendTrigger(ctx, injector, trigger)
}

func (u *upgradeController) checkActiveConnAndNoActiveProcess() error {
	if u.activeSensorConn == nil {
		return errors.Errorf("no active sensor connection for cluster %s exists, cannot trigger cert rotation", u.clusterID)
	}
	if err := u.activeSensorConn.conn.CheckAutoUpgradeSupport(); err != nil {
		return err
	}

	if u.active != nil {
		return errors.Errorf("an upgrade is already in progress in cluster %s", u.clusterID)
	}
	return nil
}

func (u *upgradeController) doTriggerCertRotation() (common.MessageInjector, *central.SensorUpgradeTrigger, error) {
	if err := u.checkActiveConnAndNoActiveProcess(); err != nil {
		return nil, nil, err
	}

	if u.upgradeStatus.GetUpgradability() != storage.ClusterUpgradeStatus_UP_TO_DATE {
		return nil, nil, errors.New("sensor is not up-to-date; cannot trigger cert rotation")
	}

	cluster := u.getCluster()
	process := newCertRotationProcess()
	u.makeProcessActive(cluster, process)
	return u.activeSensorConn.conn, u.active.trigger, nil
}

func (u *upgradeController) doTriggerUpgrade() (common.MessageInjector, *central.SensorUpgradeTrigger, error) {
	if err := u.checkActiveConnAndNoActiveProcess(); err != nil {
		return nil, nil, err
	}

	// Check upgradability.
	switch u.upgradeStatus.GetUpgradability() {
	case storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE:
		// yay!
	case storage.ClusterUpgradeStatus_SENSOR_VERSION_HIGHER:
		// We still allow upgrade triggers in this case.
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

	return u.activeSensorConn.conn, u.active.trigger, nil
}
