package deploytime

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

func newSingleDeploymentExecutor(ctx DetectionContext, deployment *storage.Deployment, images []*storage.Image) AlertCollectingExecutor {
	return &policyExecutor{
		ctx:        ctx,
		deployment: deployment,
		images:     images,
	}
}

type policyExecutor struct {
	executorCtx context.Context
	ctx         DetectionContext
	deployment  *storage.Deployment
	images      []*storage.Image
	alerts      []*storage.Alert
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

	// Check enforcement on deployment if we don't want unenforced alerts.
	enforcement, _ := buildEnforcement(compiled.Policy(), d.deployment)
	if enforcement == storage.EnforcementAction_UNSET_ENFORCEMENT && d.ctx.EnforcementOnly {
		return nil
	}

	// Generate violations.
	violations, err := d.getViolations(d.executorCtx, enforcement, compiled.Matcher())
	if err != nil {
		return errors.Wrapf(err, "evaluating violations for policy %s; deployment %s/%s", compiled.Policy().GetName(), d.deployment.GetNamespace(), d.deployment.GetName())
	}
	if len(violations) > 0 {
		d.alerts = append(d.alerts, PolicyDeploymentAndViolationsToAlert(compiled.Policy(), d.deployment, violations))
	}
	return nil
}

func (d *policyExecutor) getViolations(ctx context.Context, enforcement storage.EnforcementAction, matcher searchbasedpolicies.Matcher) ([]*storage.Alert_Violation, error) {
	violations, err := matcher.MatchOne(ctx, d.deployment, d.images, nil)
	if err != nil {
		return nil, err
	}

	return violations.AlertViolations, nil
}
