package preflight

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/resources"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type objectPreconditionsCheck struct{}

func (objectPreconditionsCheck) Name() string {
	return "Object preconditions"
}

func (objectPreconditionsCheck) Check(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	groupedActions := plan.GroupActionsByResource(execPlan.Actions())

	for gvk, acts := range groupedActions {
		res := ctx.GetResourceMetadata(gvk, resources.BundleResource)
		if res == nil {
			return errors.Errorf("could not find resource metadata for resource type %v", gvk)
		}

		for _, act := range acts {
			resClient := ctx.DynamicClientForResource(res, act.ObjectRef.Namespace)

			_, err := resClient.Get(ctx.Context(), act.ObjectRef.Name, metav1.GetOptions{})
			if err != nil && !k8sErrors.IsNotFound(err) {
				return errors.Wrapf(err, "could not retrieve resource %v", act.ObjectRef)
			}

			exists := err == nil

			if act.ActionName == plan.CreateAction && exists && !common.IsSharedObject(act.ObjectRef) {
				reporter.Errorf("To-be-created object %v already exists", act.ObjectRef)
			} else if act.ActionName != plan.CreateAction && !exists {
				reporter.Errorf("To-be-%sd object %v does not exist", act.ActionName, act.ObjectRef)
			}
		}
	}

	return nil
}
