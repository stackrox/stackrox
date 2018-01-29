package detection

import (
	"time"

	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
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
	deployments, err := d.database.GetDeployments(&v1.GetDeploymentsRequest{})
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
	deployments, err := d.database.GetDeployments(&v1.GetDeploymentsRequest{})
	if err != nil {
		logger.Error(err)
		return
	}

	for _, deploy := range deployments {
		d.taskC <- task{
			deployment: deploy,
			policy:     policy,
			action:     v1.ResourceAction_REFRESH_RESOURCE,
		}
	}
}

func (d *Detector) reprocessRegistry(registry registries.ImageRegistry) {
	deployments, err := d.database.GetDeployments(&v1.GetDeploymentsRequest{})
	if err != nil {
		logger.Error(err)
		return
	}

	policies := d.getCurrentPolicies()

	for _, deploy := range deployments {
		if d.enricher.EnrichWithRegistry(deploy, registry) {
			d.queueTasks(deploy, policies)
		}
	}
}

func (d *Detector) reprocessScanner(scanner scannerTypes.ImageScanner) {
	deployments, err := d.database.GetDeployments(&v1.GetDeploymentsRequest{})
	if err != nil {
		logger.Error(err)
		return
	}

	policies := d.getCurrentPolicies()

	for _, deploy := range deployments {
		if d.enricher.EnrichWithScanner(deploy, scanner) {
			d.queueTasks(deploy, policies)
		}
	}
}

func (d *Detector) queueTasks(deployment *v1.Deployment, policies []*matcher.Policy) {
	for _, p := range policies {
		d.taskC <- task{
			deployment: deployment,
			policy:     p,
			action:     v1.ResourceAction_REFRESH_RESOURCE,
		}
	}
}
