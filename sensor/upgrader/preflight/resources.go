package preflight

import (
	"github.com/stackrox/stackrox/sensor/upgrader/plan"
	"github.com/stackrox/stackrox/sensor/upgrader/resources"
	"github.com/stackrox/stackrox/sensor/upgrader/upgradectx"
)

type resourcesCheck struct{}

func (resourcesCheck) Name() string {
	return "Resources"
}

func (resourcesCheck) Check(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	actsByResources := plan.GroupActionsByResource(execPlan.Actions())

	for gvk, acts := range actsByResources {
		res := ctx.GetResourceMetadata(gvk, resources.BundleResource)
		if res == nil {
			reporter.Errorf("server does not support resource type %v", gvk)
			continue
		}
		for _, act := range acts {
			if res.Namespaced && act.ObjectRef.Namespace == "" {
				reporter.Errorf("Object %v does not specify a namespace, but the resource is namespaced", act.ObjectRef)
			} else if !res.Namespaced && act.ObjectRef.Namespace != "" {
				reporter.Errorf("Object %v specifies a namespace, but the resource is not namespaced", act.ObjectRef)
			}
		}
	}

	return nil
}
