package execution

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	log = logging.LoggerForModule()
)

type executor struct {
	ctx *upgradectx.UpgradeContext
}

func (e *executor) ExecutePlan(execPlan *plan.ExecutionPlan) error {
	actions := execPlan.Actions()
	for _, act := range actions {
		log.Infof("Performing action %s on object %v", act.ActionName, act.ObjectRef)
		if err := e.executeAction(act); err != nil {
			return errors.Wrapf(err, "executing action %s object %v", act.ActionName, act.ObjectRef)
		}
	}
	return nil
}

func (e *executor) executeAction(act plan.ActionDesc) error {
	res := e.ctx.GetResourceMetadata(act.ObjectRef.GVK)
	if res == nil {
		return errors.Errorf("no resource information available for object kind %v", act.ObjectRef.GVK)
	}

	client, err := e.ctx.DynamicClientForResource(res, act.ObjectRef.Namespace)
	if err != nil {
		return errors.Wrapf(err, "obtaining dynamic client for resource %v", res)
	}

	var obj unstructured.Unstructured
	if act.Object != nil {
		if err := e.ctx.Scheme().Convert(act.Object, &obj, nil); err != nil {
			return errors.Wrapf(err, "converting object %s to unstructured", act.ObjectRef)
		}

		ann := obj.GetAnnotations()
		if ann == nil {
			ann = make(map[string]string)
		}
		ann[common.LastUpgradeIDAnnotationKey] = e.ctx.ProcessID()
		obj.SetAnnotations(ann)
		obj.SetResourceVersion("")
	}

	switch act.ActionName {
	case plan.CreateAction:
		if _, err := client.Create(&obj); err != nil {
			if k8sErrors.IsAlreadyExists(err) && common.IsSharedObject(act.ObjectRef) {
				log.Warnf("Skipping creation of shared object %v", act.ObjectRef)
			} else {
				return err
			}
		}
	case plan.UpdateAction:
		if _, err := client.Update(&obj); err != nil {
			return err
		}
	case plan.DeleteAction:
		if err := client.Delete(act.ObjectRef.Name, &metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return errors.Errorf("invalid action %q on object %v", act.ActionName, act.ObjectRef)
	}

	return nil
}
