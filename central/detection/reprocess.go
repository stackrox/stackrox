package detection

import (
	"time"

	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

func (d *Detector) reprocessLoop() {
	logger.Info("Detector started reprocess loop")
	for t := range d.taskC {
		d.processTask(t)
	}
	logger.Info("Detector stopped reprocess loop")
	d.stoppedC <- struct{}{}
}

func (d *Detector) periodicallyEnrich() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		d.EnrichAndReprocess()
	}
}

// EnrichAndReprocess enriches all deployments. If new data is available, it re-assesses all policies for that deployment.
func (d *Detector) EnrichAndReprocess() {
	deployments, err := d.deploymentStorage.GetDeployments()
	if err != nil {
		logger.Error(err)
		return
	}
	polices := d.getCurrentPolicies()

	for _, deploy := range deployments {
		if enriched, err := d.enricher.Enrich(deploy); err != nil {
			logger.Error(err)
		} else if enriched {
			d.queueTasks(deploy, polices)
		}
	}
}

func (d *Detector) reprocessPolicy(policy *matcher.Policy) {
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

	qb := search.NewQueryBuilder().
		AddBool(search.Stale, false).
		AddString(search.PolicyID, policy.GetId())
	alerts, err := d.alertStorage.GetAlerts(&v1.ListAlertsRequest{
		Query: qb.Query(),
	})
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

func (d *Detector) reprocessImageIntegration(integration *sources.ImageIntegration) {
	deployments, err := d.deploymentStorage.GetDeployments()
	if err != nil {
		logger.Error(err)
		return
	}

	policies := d.getCurrentPolicies()

	for _, deploy := range deployments {
		if d.enricher.EnrichWithImageIntegration(deploy, integration) {
			d.queueTasks(deploy, policies)
		}
	}
}

func (d *Detector) queueTasks(deployment *v1.Deployment, policies []*matcher.Policy) {
	for _, p := range policies {
		d.taskC <- Task{
			deployment: deployment,
			policy:     p,
			action:     v1.ResourceAction_UPDATE_RESOURCE,
		}
	}
}
