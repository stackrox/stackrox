package deploytime

import (
	"context"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/generated/storage"
)

func newAllDeploymentsExecutor(executorCtx context.Context, deployments datastore.DataStore) alertCollectingExecutor {
	return &allDeploymentsExecutor{
		deployments: deployments,
		executorCtx: executorCtx,
	}
}

type allDeploymentsExecutor struct {
	executorCtx context.Context
	deployments datastore.DataStore
	alerts      []*storage.Alert
}

func (d *allDeploymentsExecutor) GetAlerts() []*storage.Alert {
	return d.alerts
}

func (d *allDeploymentsExecutor) ClearAlerts() {
	d.alerts = nil
}

func (d *allDeploymentsExecutor) Execute(compiled detection.CompiledPolicy) error {
	if compiled.Policy().GetDisabled() {
		return nil
	}
	violationsByDeployment, err := compiled.Matcher().Match(d.executorCtx, d.deployments)
	if err != nil {
		return err
	}
	for deploymentID, violations := range violationsByDeployment {
		dep, exists, err := d.deployments.GetDeployment(d.executorCtx, deploymentID)
		if err != nil {
			return err
		}
		if !exists {
			log.Errorf("deployment with id %q had violations, but doesn't exist", deploymentID)
			continue
		}
		if !compiled.AppliesTo(dep) {
			continue
		}
		d.alerts = append(d.alerts, policyDeploymentAndViolationsToAlert(compiled.Policy(), dep, violations.AlertViolations))
	}
	return nil
}
