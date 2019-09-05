package upgradecontroller

import (
	"github.com/stackrox/rox/central/sensor/service/connection/upgradecontroller/stateutils"
	"github.com/stackrox/rox/generated/storage"
)

func upgradeInProgress(upgradeStatus *storage.ClusterUpgradeStatus) bool {
	if upgradeStatus.GetCurrentUpgradeProcessId() == "" {
		return false
	}
	return !stateutils.TerminalStates.Contains(upgradeStatus.GetCurrentUpgradeProgress().GetUpgradeState())
}
