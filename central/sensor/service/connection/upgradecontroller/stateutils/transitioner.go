package stateutils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/pkg/set"
)

type stateAndUpgraderReq struct {
	upgradeType         storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType
	currentState        storage.UpgradeProgress_UpgradeState
	workflow            string
	stage               sensorupgrader.Stage
	upgraderErrOccurred bool
}

type nextStateAndResponse struct {
	nextUpgradeState          storage.UpgradeProgress_UpgradeState
	upgraderWorkflowToExecute string
	updateDetail              bool
}

type transitioner struct {
	workflowMatch     *string
	stageMatch        *set.Set[sensorupgrader.Stage]
	currentStateMatch *set.Set[storage.UpgradeProgress_UpgradeState]
	errOccurredMatch  *bool
	upgradeTypeMatch  *storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType

	workflowToExecute string
	// noStateChange indicates that the final state is the same as the current state.
	// If noStateChange is true, nextState is ignored.
	// If it is false, nextState MUST be non-nil.
	// This is redundant since the nil-ness of newState is sufficient to infer the value
	// of noStateChange, but spelling it out this way for better readability.
	// It is also enforced by a unit test.
	noStateChange bool
	nextState     *storage.UpgradeProgress_UpgradeState

	updateDetail bool
}

func (e *transitioner) GetNextState(req stateAndUpgraderReq) *nextStateAndResponse {
	if e.currentStateMatch != nil {
		if !e.currentStateMatch.Contains(req.currentState) {
			return nil
		}
	}
	if e.workflowMatch != nil {
		if req.workflow != *e.workflowMatch {
			return nil
		}
	}
	if e.stageMatch != nil {
		if !e.stageMatch.Contains(req.stage) {
			return nil
		}
	}
	if e.errOccurredMatch != nil {
		if req.upgraderErrOccurred != *e.errOccurredMatch {
			return nil
		}
	}
	if e.upgradeTypeMatch != nil {
		if req.upgradeType != *e.upgradeTypeMatch {
			return nil
		}
	}

	resp := &nextStateAndResponse{
		upgraderWorkflowToExecute: e.workflowToExecute,
		updateDetail:              e.updateDetail,
	}
	if e.noStateChange {
		resp.nextUpgradeState = req.currentState
	} else {
		resp.nextUpgradeState = *e.nextState
	}

	return resp
}
