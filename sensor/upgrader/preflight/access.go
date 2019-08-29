package preflight

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	v1 "k8s.io/api/authorization/v1"
)

type accessCheck struct{}

func (accessCheck) Name() string {
	return "Kubernetes authorization"
}

func (accessCheck) Check(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	sarClient := ctx.ClientSet().AuthorizationV1().SelfSubjectAccessReviews()

	for _, act := range execPlan.Actions() {
		var verb string
		switch act.ActionName {
		case plan.CreateAction:
			verb = "create"
		case plan.UpdateAction:
			verb = "update"
		case plan.DeleteAction:
			verb = "delete"
		default:
			return errors.Errorf("invalid action name %q for object %v", act.ActionName, act.ObjectRef)
		}

		resMD := ctx.GetResourceMetadata(act.ObjectRef.GVK)
		if resMD == nil {
			return errors.Errorf("no metadata available for resource %v", act.ObjectRef.GVK)
		}

		sar := &v1.SelfSubjectAccessReview{
			Spec: v1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &v1.ResourceAttributes{
					Namespace: act.ObjectRef.Namespace,
					Verb:      verb,
					Group:     resMD.Group,
					Version:   resMD.Version,
					Resource:  resMD.Name,
				},
			},
		}

		if act.ActionName != plan.CreateAction {
			sar.Spec.ResourceAttributes.Name = act.ObjectRef.Name
		}

		sarResult, err := sarClient.Create(sar)
		if err != nil {
			return errors.Wrap(err, "failed to perform SelfSubjectAccessReview check")
		}
		if sarResult.Status.EvaluationError != "" {
			reporter.Warnf("Evaluation error performing access review check for action %s on object %v: %s", act.ActionName, act.ObjectRef, sarResult.Status.EvaluationError)
		}
		if !sarResult.Status.Allowed && !sarResult.Status.Denied {
			reporter.Warnf("K8s authorizer seems to have no opinion on whether action %s on object %v is allowed", act.ActionName, act.ObjectRef)
		} else if !sarResult.Status.Allowed {
			reporter.Errorf("ActionName %s on object %v is not allowed", act.ActionName, act.ObjectRef)
		}
	}
	return nil
}
