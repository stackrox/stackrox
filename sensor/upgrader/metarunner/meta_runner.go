package metarunner

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/sensor/upgrader/runner"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

const (
	maxRetries = 5

	timeout                     = 30 * time.Second
	sleepDurationBetweenRetries = 10 * time.Second

	sleepDurationBetweenPolls = 10 * time.Second
)

var (
	log = logging.LoggerForModule()
)

func sendRequest(upgradeCtx *upgradectx.UpgradeContext, svc central.SensorUpgradeControlServiceClient, workflow string, stage sensorupgrader.Stage, lastExecutedStageError string) (*central.UpgradeCheckInFromUpgraderResponse, error) {
	ctx, cancel := context.WithTimeout(upgradeCtx.Context(), timeout)
	defer cancel()
	return svc.UpgradeCheckInFromUpgrader(ctx, &central.UpgradeCheckInFromUpgraderRequest{
		UpgradeProcessId:       upgradeCtx.ProcessID(),
		ClusterId:              upgradeCtx.ClusterID(),
		CurrentWorkflow:        workflow,
		LastExecutedStage:      stage.String(),
		LastExecutedStageError: lastExecutedStageError,
	})
}

func sendRequestWithRetries(upgradeCtx *upgradectx.UpgradeContext, svc central.SensorUpgradeControlServiceClient, workflow string, stage sensorupgrader.Stage, lastExecutedStageError string) (*central.UpgradeCheckInFromUpgraderResponse, error) {
	for tryNumber := 1; ; tryNumber++ {
		resp, err := sendRequest(upgradeCtx, svc, workflow, stage, lastExecutedStageError)
		if err == nil {
			return resp, err
		}
		if tryNumber >= maxRetries {
			return nil, err
		}
		log.Errorf("Error getting instructions from Central remote control service: %v. Retrying (%d) ...", err, tryNumber)
		time.Sleep(sleepDurationBetweenRetries)
	}
}

// Run manages the entire run of the upgrader, talking to the remote control service in Central
// every step of the way to figure out what to do next.
func Run(upgradeCtx *upgradectx.UpgradeContext) error {
	client := upgradeCtx.GetGRPCClient()
	if client == nil {
		return errors.New("no gRPC client to Central found")
	}
	svc := central.NewSensorUpgradeControlServiceClient(client)
	err := runLoop(upgradeCtx, svc)
	log.Errorf("Couldn't successfully run remote control service, despite retries: %v. Giving up now, and cleaning up...", err)
	return runner.Run(upgradeCtx, sensorupgrader.CleanupWorkflow)
}

func runLoop(upgradeCtx *upgradectx.UpgradeContext, svc central.SensorUpgradeControlServiceClient) error {
	var workflow, lastExecutedStageError string
	var stage sensorupgrader.Stage

	var currRunner runner.Runner

	for {
		resp, err := sendRequestWithRetries(upgradeCtx, svc, workflow, stage, lastExecutedStageError)
		if err != nil {
			return err
		}

		newWorkflow := resp.GetWorkflowToExecute()
		if newWorkflow == "" {
			return errors.New("central did not specify a workflow to execute")
		}

		log.Infof("Received instruction from Central to run workflow: %s", newWorkflow)
		if newWorkflow != workflow {
			// New workflow. Abandon old runner.
			currRunner, err = runner.New(upgradeCtx, newWorkflow)
			if err != nil {
				return err
			}
		}

		// An error occurred running the currRunner but Central told us to run the same workflow.
		// In this case, we instantiate a new runner to retry the workflow.
		if currRunner.Err() != nil {
			currRunner, err = runner.New(upgradeCtx, newWorkflow)
			if err != nil {
				return err
			}
		}

		// If the current runner is finished, but Central told us to keep running the same workflow,
		// there's nothing for us to do. Let's just keep polling it until it tells us to do something
		// else.
		if currRunner.Finished() {
			time.Sleep(sleepDurationBetweenPolls)
		} else {
			currRunner.RunNextStage()
		}
		workflow = newWorkflow
		lastExecutedStageError = errToStrOrEmpty(currRunner.Err())
		stage = currRunner.MostRecentStage()
		log.Infof("Ran workflow %s/stage %s. %s", workflow, stage, formatErrForLogs(currRunner.Err()))
	}
}

func formatErrForLogs(runnerErr error) string {
	if runnerErr != nil {
		return fmt.Sprintf("There was an error executing it: %v", runnerErr.Error())
	}
	return "Executed successfully"
}

func errToStrOrEmpty(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
