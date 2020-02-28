package runtime

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

func newSingleDeploymentExecutor(deployment *storage.Deployment, images []*storage.Image, process *storage.ProcessIndicator) AlertCollectingExecutor {
	return &policyExecutor{
		deployment: deployment,
		images:     images,
		process:    process,
	}
}

type policyExecutor struct {
	deployment *storage.Deployment
	images     []*storage.Image
	alerts     []*storage.Alert
	process    *storage.ProcessIndicator
}

func (d *policyExecutor) GetAlerts() []*storage.Alert {
	return d.alerts
}

func (d *policyExecutor) ClearAlerts() {
	d.alerts = nil
}

func (d *policyExecutor) Execute(compiled detection.CompiledPolicy) error {
	if compiled.Policy().GetDisabled() {
		return nil
	}
	// Check predicate on deployment.
	if !compiled.AppliesTo(d.deployment) {
		return nil
	}

	violation, err := d.getViolations(context.Background(), compiled.Matcher())
	if err != nil {
		return errors.Wrapf(err, "evaluating violations for policy %s; deployment %s/%s", compiled.Policy().GetName(), d.deployment.GetNamespace(), d.deployment.GetName())
	}

	if alert := policyDeploymentAndViolationsToAlert(compiled.Policy(), d.deployment, violation); alert != nil {
		d.alerts = append(d.alerts, alert)
	}
	return nil
}

func (d *policyExecutor) getViolations(ctx context.Context, matcher searchbasedpolicies.Matcher) (searchbasedpolicies.Violations, error) {
	return matcher.MatchOne(ctx, d.deployment, d.images, d.process)
}
