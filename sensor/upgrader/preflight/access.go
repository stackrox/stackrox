package preflight

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/sensor/upgrader/plan"
	"github.com/stackrox/stackrox/sensor/upgrader/resources"
	"github.com/stackrox/stackrox/sensor/upgrader/upgradectx"
	v1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type accessCheck struct{}

func (accessCheck) Name() string {
	return "Kubernetes authorization"
}

func (accessCheck) getAllResourceAttribs(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan) ([]v1.ResourceAttributes, error) {
	resourceAttribsSet := make(map[v1.ResourceAttributes]struct{})

	for _, act := range execPlan.Actions() {
		var verb, rollbackVerb string
		switch act.ActionName {
		case plan.CreateAction:
			verb, rollbackVerb = "create", "delete"
		case plan.UpdateAction:
			verb = "update"
		case plan.DeleteAction:
			verb, rollbackVerb = "delete", "create"
		default:
			return nil, errors.Errorf("invalid action name %q for object %v", act.ActionName, act.ObjectRef)
		}

		resMD := ctx.GetResourceMetadata(act.ObjectRef.GVK, resources.BundleResource)
		if resMD == nil {
			return nil, errors.Errorf("no metadata available for resource %v", act.ObjectRef.GVK)
		}

		resourceAttribs := v1.ResourceAttributes{
			Namespace: act.ObjectRef.Namespace,
			Verb:      verb,
			Group:     resMD.Group,
			Version:   resMD.Version,
			Resource:  resMD.Name,
		}

		// Name only makes sense for update and delete
		if act.ActionName != plan.CreateAction {
			resourceAttribs.Name = act.ObjectRef.Name
		}

		resourceAttribsSet[resourceAttribs] = struct{}{}

		if rollbackVerb != "" {
			rollbackResourceAttribs := resourceAttribs
			// Name never makes sense for rollback as we'd be talking about objects that don't exist yet or will not
			// exist when we perform the rollback.
			rollbackResourceAttribs.Name = ""
			rollbackResourceAttribs.Verb = rollbackVerb
			resourceAttribsSet[rollbackResourceAttribs] = struct{}{}
		}
	}

	result := make([]v1.ResourceAttributes, 0, len(resourceAttribsSet))

	for resourceAttribs := range resourceAttribsSet {
		// TODO(mi): Sorting would be nice, but seems overkill
		result = append(result, resourceAttribs)
	}

	return result, nil
}

func (c accessCheck) Check(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	sarClient := ctx.ClientSet().AuthorizationV1().SelfSubjectAccessReviews()

	resourceAttribs, err := c.getAllResourceAttribs(ctx, execPlan)
	if err != nil {
		return err
	}

	for i := range resourceAttribs {
		ra := resourceAttribs[i]
		sar := &v1.SelfSubjectAccessReview{
			Spec: v1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &ra,
			},
		}
		sarResult, err := sarClient.Create(ctx.Context(), sar, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to perform SelfSubjectAccessReview check")
		}
		if sarResult.Status.EvaluationError != "" {
			reporter.Warnf("Evaluation error performing access review check for action %s on resource %s: %s", ra.Verb, ra.Resource, sarResult.Status.EvaluationError)
		}
		if !sarResult.Status.Allowed && !sarResult.Status.Denied {
			reporter.Errorf("K8s authorizer did not explicitly allow or deny access to perform action %s on resource %s. This usually means access is denied.", ra.Verb, ra.Resource)
		} else if !sarResult.Status.Allowed {
			reporter.Errorf("Action %s on resource %s is not allowed", ra.Verb, ra.Resource)
		}
	}
	return nil
}
