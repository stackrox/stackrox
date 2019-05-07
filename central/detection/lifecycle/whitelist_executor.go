package lifecycle

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/generated/storage"
)

type whitelistExecutor struct {
	deploymentIDs           []string
	deploymentsToIndicators map[string][]*storage.ProcessIndicator

	deployments datastore.DataStore
	alerts      []*storage.Alert
}

func newWhitelistExecutor(deployments datastore.DataStore, deploymentsToIndicators map[string][]*storage.ProcessIndicator) *whitelistExecutor {
	deploymentIDs := make([]string, 0, len(deploymentsToIndicators))
	for k := range deploymentsToIndicators {
		deploymentIDs = append(deploymentIDs, k)
	}

	return &whitelistExecutor{
		deploymentIDs:           deploymentIDs,
		deploymentsToIndicators: deploymentsToIndicators,
		deployments:             deployments,
	}
}

func (w *whitelistExecutor) Execute(compiled detection.CompiledPolicy) error {
	if !runtime.IsProcessWhitelistPolicy(compiled) {
		return nil
	}

	ctx := context.TODO()

	violationsByDeployment, err := compiled.Matcher().MatchMany(ctx, w.deployments, w.deploymentIDs...)
	if err != nil {
		return errors.Wrapf(err, "matching policy %s", compiled.Policy().GetName())
	}

	for deploymentID, violations := range violationsByDeployment {
		violations.ProcessViolation = &storage.Alert_ProcessViolation{
			Processes: w.deploymentsToIndicators[deploymentID],
		}
		dep, exists, err := w.deployments.GetDeployment(ctx, deploymentID)
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
		w.alerts = append(w.alerts, runtime.PolicyDeploymentAndViolationsToAlert(compiled.Policy(), dep, violations))
	}
	return nil
}
