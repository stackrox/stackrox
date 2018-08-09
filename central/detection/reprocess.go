package detection

import (
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies"
	"github.com/stackrox/rox/pkg/search"
)

// EnrichAndReprocess enriches all deployments. If new data is available, it re-assesses all policies for that deployment.
func (d *detectorImpl) EnrichAndReprocess() {
	deployments, err := d.deploymentStorage.GetDeployments()
	if err != nil {
		logger.Error(err)
		return
	}
	polices := d.getCurrentPolicies()

	for _, deploy := range deployments {
		enriched, err := d.enricher.Enrich(deploy)
		if err != nil {
			logger.Error(err)
			continue
		}
		if enriched {
			d.queueTasks(deploy, polices)
		} else {
			// Even if the deployment is not enriched, we reprocess the risk explicitly because
			// some of the risk factors are external to the enrichment process. (Ex: D&R alerts.)
			go d.reprocessDeploymentRiskAndLogError(deploy)
		}
	}
}

func (d *detectorImpl) reprocessLoop() {
	logger.Info("Detector started reprocess loop")
	for t := range d.taskC {
		d.processTask(t)
	}
	logger.Info("Detector stopped reprocess loop")
	d.stoppedC <- struct{}{}
}

func (d *detectorImpl) periodicallyEnrich() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		d.EnrichAndReprocess()
	}
}

func (d *detectorImpl) reprocessPolicy(policy compiledpolicies.DeploymentMatcher) {
	deployments, err := d.deploymentStorage.GetDeployments()
	if err != nil {
		logger.Error(err)
		return
	}

	deploymentMap := make(map[string]struct{})

	for _, deploy := range deployments {
		d.taskC <- Task{
			deployment: deploy,
			policy:     policy,
			action:     v1.ResourceAction_UPDATE_RESOURCE,
		}

		deploymentMap[deploy.GetId()] = struct{}{}
	}

	alerts, err := d.alertStorage.SearchRawAlerts(
		search.NewQueryBuilder().
			AddBools(search.Stale, false).
			AddStrings(search.PolicyID, policy.GetProto().GetId()).
			ToParsedSearchRequest())
	if err != nil {
		logger.Error(err)
		return
	}

	for _, alert := range alerts {
		if _, ok := deploymentMap[alert.GetDeployment().GetId()]; !ok {
			d.taskC <- Task{
				deployment: alert.GetDeployment(),
				policy:     policy,
				action:     v1.ResourceAction_REMOVE_RESOURCE,
			}
		}
	}
}

func (d *detectorImpl) queueTasks(deployment *v1.Deployment, policies []compiledpolicies.DeploymentMatcher) {
	for _, p := range policies {
		d.taskC <- Task{
			deployment: deployment,
			policy:     p,
			action:     v1.ResourceAction_UPDATE_RESOURCE,
		}
	}
}
