package upgradecontroller

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
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
		errorhelpers.PanicOnDevelopmentf("Making process %s active when there already is an active one. This should not happen...", processStatus.GetId())
	}

	u.active = &activeUpgradeInfo{
		trigger: constructTriggerUpgradeRequest(cluster, processStatus),
		status:  processStatus,
	}
	u.upgradeStatus.MostRecentProcess = processStatus
	u.upgradeStatusChanged = true
}
