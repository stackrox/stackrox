package upgradecontroller

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/connection/upgradecontroller/stateutils"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sensorupgrader"
)

func constructUpgradeDetail(req *central.UpgradeCheckInFromUpgraderRequest) string {
	if req.GetLastExecutedStageError() != "" {
		return fmt.Sprintf("Upgrader failed to execute %s of the %s workflow: %s", req.GetLastExecutedStage(), req.GetCurrentWorkflow(), req.GetLastExecutedStageError())
	}

	return fmt.Sprintf("Upgrader successfully executed %s of the %s workflow", req.GetLastExecutedStage(), req.GetCurrentWorkflow())
}

func (u *upgradeController) ProcessCheckInFromUpgrader(req *central.UpgradeCheckInFromUpgraderRequest) (*central.UpgradeCheckInFromUpgraderResponse, error) {
	if err := u.checkErrSig(); err != nil {
		return nil, err
	}

	u.storageLock.Lock()
	defer u.storageLock.Unlock()

	upgradeStatus, err := u.getClusterUpgradeStatus()
	if err != nil {
		return nil, err
	}
	if upgradeStatus.GetCurrentUpgradeProcessId() != req.GetUpgradeProcessId() {
		return nil, errors.Errorf("current upgrade process id (%s) is different; perhaps this upgrade process (id %s) has timed out?", upgradeStatus.GetCurrentUpgradeProcessId(), req.GetUpgradeProcessId())
	}

	stage := sensorupgrader.GetStage(req.GetLastExecutedStage())

	currentState := upgradeStatus.GetCurrentUpgradeProgress().GetUpgradeState()
	nextState, workflowToExecute, updateDetail := stateutils.DetermineNextStateAndWorkflowForUpgrader(currentState, req.GetCurrentWorkflow(), stage, req.GetLastExecutedStageError())

	upgradeStatus.CurrentUpgradeProgress.UpgradeState = nextState
	if updateDetail {
		upgradeStatus.CurrentUpgradeProgress.UpgradeStatusDetail = constructUpgradeDetail(req)
	}
	if err := u.setUpgradeStatus(upgradeStatus); err != nil {
		return nil, err
	}

	return &central.UpgradeCheckInFromUpgraderResponse{WorkflowToExecute: workflowToExecute}, nil
}
