package execution

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/resources"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			return errors.Wrapf(err, "executing action %s on object %v", act.ActionName, act.ObjectRef)
		}
	}
	return nil
}

func (e *executor) executeAction(act plan.ActionDesc) error {
	res := e.ctx.GetResourceMetadata(act.ObjectRef.GVK, resources.BundleResource)
	if res == nil {
		return errors.Errorf("no resource information available for object kind %v", act.ObjectRef.GVK)
	}

	client := e.ctx.DynamicClientForResource(res, act.ObjectRef.Namespace)

	var obj *unstructured.Unstructured
	if act.Object != nil {
		obj = act.Object.DeepCopy()
		k8sutil.SetAnnotation(obj, common.LastUpgradeIDAnnotationKey, e.ctx.ProcessID())
	}

	switch act.ActionName {
	case plan.CreateAction:
		if _, err := client.Create(e.ctx.Context(), obj, metaV1.CreateOptions{}); err != nil {
			if k8sErrors.IsAlreadyExists(err) && common.IsSharedObject(act.ObjectRef) {
				log.Warnf("Skipping creation of shared object %v", act.ObjectRef)
			} else {
				return err
			}
		}
	case plan.UpdateAction:
		if _, err := client.Update(e.ctx.Context(), obj, metaV1.UpdateOptions{}); err != nil {
			// The upgrader is very focused on getting the Kubernetes objects to the end state
			// obtained from the bundle. Hence, it doesn't make sense for us to bother with
			// the resourceVersion, since our update is NOT a function of the original object.
			// However, for some objects (specifically, we've observed this with the admission webhook),
			// the update is rejected by the Kube API server unless a resourceVersion is specified.
			// To mitigate this, we try all updates with the resourceVersion specified. If we hit a conflict,
			// then we unset the resourceVersion and try again.
			// Of course, if there is a conflict with a resource like an admission controller, this will still fail --
			// however, that should be relatively rare, and in those cases, the upgrade can be retried.
			if !k8sErrors.IsConflict(err) {
				return err
			}
			log.Warnf("The update for object %v hit a conflict. Trying again without setting resourceVersion: %v", act.ObjectRef, err)
			obj.SetResourceVersion("")
			_, err := client.Update(e.ctx.Context(), obj, metaV1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "updating after clearing resourceVersion of object %v", act.ObjectRef)
			}
		}
	case plan.DeleteAction:
		if err := client.Delete(e.ctx.Context(), act.ObjectRef.Name, kubernetes.DeleteBackgroundOption); err != nil {
			return err
		}
	default:
		return errors.Errorf("invalid action %q on object %v", act.ActionName, act.ObjectRef)
	}

	return nil
}
