package upgradecontroller

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/sensor/service/connection/upgradecontroller/stateutils"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/protoconv"
	"github.com/stackrox/stackrox/pkg/sensorupgrader"
)

var (
	// ErrNoUpgradeInProgress represents the error that no upgrade is in progress.
	ErrNoUpgradeInProgress = errors.New("no upgrade is currently in progress")
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

	nextState, workflowToExecute, updateDetail := stateutils.DetermineNextStateAndWorkflowForUpgrader(
		processStatus.GetType(), processStatus.GetProgress().GetUpgradeState(), req.GetCurrentWorkflow(), stage, req.GetLastExecutedStageError())

	var detail string
	if updateDetail {
		detail = constructUpgradeDetail(req)
	} else {
		// Carry over the previous detail.
		detail = processStatus.GetProgress().GetUpgradeStatusDetail()
	}

	if err := u.setUpgradeProgress(req.GetUpgradeProcessId(), nextState, detail); err != nil {
		return nil, err
	}

	return &central.UpgradeCheckInFromUpgraderResponse{WorkflowToExecute: workflowToExecute}, nil
}

func (u *upgradeController) ProcessCheckInFromSensor(req *central.UpgradeCheckInFromSensorRequest) error {
	return u.do(func() error {
		return u.doProcessCheckInFromSensor(req)
	})
}

// Returns the most relevant error condition among all errors of all pods.
// Image-related errors are more relevant than non-image related errors.
func findMostRelevantErrorCondition(states []*central.UpgradeCheckInFromSensorRequest_UpgraderPodState) *central.UpgradeCheckInFromSensorRequest_PodErrorCondition {
	var anyErrCond *central.UpgradeCheckInFromSensorRequest_PodErrorCondition

	for _, state := range states {
		if errCond := state.GetError(); errCond != nil {
			if errCond.GetImageRelated() {
				return errCond
			}
			if anyErrCond == nil {
				anyErrCond = errCond
			}
		}
	}
	return anyErrCond
}

func analyzeUpgraderPodStates(states []*central.UpgradeCheckInFromSensorRequest_UpgraderPodState) (string, bool) {
	relevantErrCond := findMostRelevantErrorCondition(states)

	if relevantErrCond == nil {
		return "upgrader pods are waiting to start", true
	}

	errMsg := relevantErrCond.GetMessage()
	if relevantErrCond.GetImageRelated() {
		errMsg = fmt.Sprintf("The upgrader pods have trouble pulling the new image: %s", errMsg)
	}
	return errMsg, false
}

func (u *upgradeController) doProcessCheckInFromSensor(req *central.UpgradeCheckInFromSensorRequest) error {
	if u.active == nil {
		return ErrNoUpgradeInProgress
	}

	processStatus := u.active.status
	if processStatus.GetId() != req.GetUpgradeProcessId() {
		return errors.Errorf("current upgrade process id (%s) is different; perhaps this upgrade process (id %s) has timed out?", processStatus.GetId(), req.GetUpgradeProcessId())
	}

	currState := processStatus.GetProgress().GetUpgradeState()
	inStateSince := protoconv.ConvertTimestampToTimeOrNow(processStatus.GetProgress().GetSince())

	var nextState storage.UpgradeProgress_UpgradeState
	var detail string

	switch s := req.GetState().(type) {
	case *central.UpgradeCheckInFromSensorRequest_LaunchError:
		if currState >= storage.UpgradeProgress_UPGRADER_LAUNCHED {
			return nil // not interesting if upgrader has already launched
		}

		if s.LaunchError != "" {
			nextState = storage.UpgradeProgress_UPGRADE_INITIALIZATION_ERROR
			detail = fmt.Sprintf("Sensor failed to launch upgrader deployment: %s", s.LaunchError)
		} else {
			nextState = storage.UpgradeProgress_UPGRADER_LAUNCHING
		}
	case *central.UpgradeCheckInFromSensorRequest_DeploymentGone:
		if currState == storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK {
			return nil
		}
		// This is always an error unless the upgrader was rolling back - we only tell the deployment to delete itself (or sensor to delete it) if the
		// process is complete (deletion of upgrader deployment is not a precondition for deletion!), and in this
		// case we would have exited this function right at the top.
		nextState = storage.UpgradeProgress_UPGRADE_ERROR_UNKNOWN
		detail = "Sensor reported the upgrader deployment no longer exists."
	case *central.UpgradeCheckInFromSensorRequest_PodStates:
		if currState >= storage.UpgradeProgress_UPGRADER_LAUNCHED && time.Since(inStateSince) < u.timeouts.StuckInSameStateTimeout() {
			// Generally, not interesting if the upgrader has already launched, unless it's been stuck in the same
			// state for a really long time.
			return nil
		}

		var ok bool
		detail, ok = analyzeUpgraderPodStates(s.PodStates.GetStates())
		if ok {
			nextState = storage.UpgradeProgress_UPGRADER_LAUNCHING
			break
		}

		if time.Since(inStateSince) < u.timeouts.UpgraderStartGracePeriod() {
			// Do not jump to any conclusions before the grace period is over.
			return nil
		}
		// Errors before the upgrader has launched are initialization errors, all others are unknown errors.
		if currState >= storage.UpgradeProgress_UPGRADER_LAUNCHED {
			nextState = storage.UpgradeProgress_UPGRADE_ERROR_UNKNOWN
		} else {
			nextState = storage.UpgradeProgress_UPGRADE_INITIALIZATION_ERROR
		}
	default:
		return errors.Errorf("Unknown or malformed upgrade check-in from sensor: %+v", req)
	}

	return u.setUpgradeProgress(req.GetUpgradeProcessId(), nextState, detail)
}
