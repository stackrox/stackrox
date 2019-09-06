package upgradecontroller

import (
	"fmt"

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
	var resp *central.UpgradeCheckInFromUpgraderResponse
	err := u.do(func() error {
		var err error
		resp, err = u.doProcessCheckInFromUpgrader(req)
		return err
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (u *upgradeController) doProcessCheckInFromUpgrader(req *central.UpgradeCheckInFromUpgraderRequest) (*central.UpgradeCheckInFromUpgraderResponse, error) {
	if u.active == nil {
		// No upgrade is currently in progress. Tell the upgrader to clean up.
		return &central.UpgradeCheckInFromUpgraderResponse{WorkflowToExecute: sensorupgrader.CleanupWorkflow}, nil
	}

	processStatus := u.active.status
	if processStatus.GetId() != req.GetUpgradeProcessId() {
		// Current upgrade process id is different. Tell the upgrader to clean up.
		return &central.UpgradeCheckInFromUpgraderResponse{WorkflowToExecute: sensorupgrader.CleanupWorkflow}, nil
	}

	stage := sensorupgrader.GetStage(req.GetLastExecutedStage())

	currentState := processStatus.GetProgress().GetUpgradeState()
	nextState, workflowToExecute, updateDetail := stateutils.DetermineNextStateAndWorkflowForUpgrader(currentState, req.GetCurrentWorkflow(), stage, req.GetLastExecutedStageError())

	processStatus.Progress.UpgradeState = nextState
	if updateDetail {
		processStatus.Progress.UpgradeStatusDetail = constructUpgradeDetail(req)
	}
	u.upgradeStatusChanged = true

	return &central.UpgradeCheckInFromUpgraderResponse{WorkflowToExecute: workflowToExecute}, nil
}
