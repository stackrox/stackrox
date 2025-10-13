package stateutils

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/pkg/utils"
)

// DetermineNextStateAndWorkflowForUpgrader takes the current state, and the input from the upgrader, and determines the final state.
func DetermineNextStateAndWorkflowForUpgrader(upgradeType storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType,
	currentUpgradeState storage.UpgradeProgress_UpgradeState, workflow string, stage sensorupgrader.Stage, upgraderErr string) (nextState storage.UpgradeProgress_UpgradeState, workflowToExecute string, updateDetail bool) {
	resp := computeNextStateAndResp(upgradeType, currentUpgradeState, workflow, stage, upgraderErr)
	if resp != nil {
		return resp.nextUpgradeState, resp.upgraderWorkflowToExecute, resp.updateDetail
	}

	// This should never happen in practice; it means that we're in an unexpected situation.
	// Respond by telling the upgrader to clean up.
	utils.Should(errors.Errorf("UNEXPECTED: No transition found for state: %s; workflow: %s; state; %s; upgraderErr: %s", currentUpgradeState, workflow, stage, upgraderErr))
	return currentUpgradeState, sensorupgrader.CleanupWorkflow, false
}

func computeNextStateAndResp(upgradeType storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType, currentUpgradeState storage.UpgradeProgress_UpgradeState, workflow string,
	stage sensorupgrader.Stage, upgraderErr string) *nextStateAndResponse {

	req := stateAndUpgraderReq{
		upgradeType:         upgradeType,
		currentState:        currentUpgradeState,
		workflow:            workflow,
		stage:               stage,
		upgraderErrOccurred: upgraderErr != "",
	}

	for _, transition := range allTransitions {
		resp := transition.GetNextState(req)
		if resp != nil {
			return resp
		}
	}
	return nil
}
