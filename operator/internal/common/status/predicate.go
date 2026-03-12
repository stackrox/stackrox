package status

import (
	"fmt"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/reflectutils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// SkipStatusControllerUpdates filters events triggered by status controller updates to prevent unnecessary reconciliations.
// It skips reconciliation when there is no delta between the old and the new object except for possible
// changes in the status controller owned conditions (Available, Progressing).
type SkipStatusControllerUpdates struct {
	predicate.Funcs
	log logr.Logger
}

// NewSkipStatusControllerUpdates creates a new instance of the SkipStatusControllerUpdates predicate
// with the provided logger and kind for logging context.
func NewSkipStatusControllerUpdates(logger logr.Logger, kind string) SkipStatusControllerUpdates {
	return SkipStatusControllerUpdates{
		log: logger.WithName("predicate-skip-status-ctrl-update").WithName(kind),
	}
}

// Update implements predicate.TypedPredicate.
// Returns true to allow the update event to trigger reconciliation, false to skip it.
func (p SkipStatusControllerUpdates) Update(e event.UpdateEvent) bool {
	// Check for nil using reflection to handle the interface nil gotcha.
	if reflectutils.IsNil(e.ObjectOld) || reflectutils.IsNil(e.ObjectNew) {
		// One of the objects is nil, allow reconciliation.
		p.log.Info("One of the objects is nil, allowing reconciliation")
		return true
	}

	objOldForStatusController, err := toObjectForStatusController(e.ObjectOld)
	if err != nil {
		p.log.Error(err, "Failed to convert old object to ObjectForStatusController, allowing reconciliation",
			"objectOldType", fmt.Sprintf("%T", e.ObjectOld))
		return true
	}

	objNewForStatusController, err := toObjectForStatusController(e.ObjectNew)
	if err != nil {
		p.log.Error(err, "Failed to convert new object to ObjectForStatusController, allowing reconciliation",
			"objectNewType", fmt.Sprintf("%T", e.ObjectNew))
		return true
	}

	if reducedObjectsEqual(objOldForStatusController, objNewForStatusController) {
		p.log.V(1).Info("No noteworthy changes detected in object, skipping reconciliation")
		return false
	}

	return true
}

// reducedObjectsEqual compares two ObjectForStatusController objects for equality, while making sure that
// certain differences are ignored, which would only trigger unnecessary reconciliations.
//
// This includes:
//   - Ignoring differences in conditions owned by the status controller.
//   - Ignoring differences in managed fields (volatile Kubernetes bookkeeping).
//   - Ignoring differences in resource version (the resource version is also bumped on status-only updates).
//
// Important: This function modifies the input objects!
func reducedObjectsEqual(oldObj, newObj platform.ObjectForStatusController) bool {
	for _, conditionType := range statusControllerConditionTypes {
		oldObj.SetCondition(platform.StackRoxCondition{Type: conditionType})
		newObj.SetCondition(platform.StackRoxCondition{Type: conditionType})
	}

	oldObj.SetManagedFields(nil)
	newObj.SetManagedFields(nil)

	oldObj.SetResourceVersion("")
	newObj.SetResourceVersion("")

	return equality.Semantic.DeepEqual(oldObj, newObj)
}

// toObjectForStatusController converts an interface to ObjectForStatusController,
// handling typed and unstructured objects.
func toObjectForStatusController(obj ctrlClient.Object) (platform.ObjectForStatusController, error) {
	// We copy the passed objects because our checks involve modifying the object.
	objCopy := obj.DeepCopyObject().(ctrlClient.Object)
	u, ok := objCopy.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("failed to convert object to unstructured: %T", obj)
	}

	gvk := u.GroupVersionKind()
	var target platform.ObjectForStatusController
	switch gvk.Kind {
	case "Central":
		target = &platform.Central{}
	case "SecuredCluster":
		target = &platform.SecuredCluster{}
	default:
		return nil, fmt.Errorf("unsupported kind for conversion: %s", gvk.Kind)
	}

	// Convert unstructured to typed object.
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, target); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to %s: %w", gvk.Kind, err)
	}

	return target, nil
}

// PassThroughUpdatedStatusPredicate filters deployment events to only react to status changes.
// This prevents reconciliation when HPA or other controllers modify deployment.spec.replicas.
// For example, the status controller only cares about deployment readiness (status), not scaling decisions (spec).
type PassThroughUpdatedStatusPredicate struct {
	predicate.TypedFuncs[*appsv1.Deployment]
	logger logr.Logger
}

func NewPassThroughUpdatedStatusPredicate(logger logr.Logger) PassThroughUpdatedStatusPredicate {
	return PassThroughUpdatedStatusPredicate{
		logger: logger.WithName("passthrough-updated-status-predicate"),
	}
}

// Update returns true only if Deployment status changed (ignores spec-only changes).
// This allows HPA to modify replicas without triggering reconciliation.
func (p PassThroughUpdatedStatusPredicate) Update(e event.TypedUpdateEvent[*appsv1.Deployment]) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return true
	}
	objectStatusOld := reduceDeploymentStatus(&e.ObjectOld.Status)
	objectStatusNew := reduceDeploymentStatus(&e.ObjectNew.Status)

	if equality.Semantic.DeepEqual(objectStatusOld, objectStatusNew) {
		p.logger.V(1).Info("Deployment status unchanged, skipping reconciliation")
		return false
	}

	return true
}

// reduceDeploymentStatus
func reduceDeploymentStatus(status *appsv1.DeploymentStatus) *appsv1.DeploymentStatus {
	statusCopy := status.DeepCopy()
	statusCopy.ObservedGeneration = 0
	return statusCopy
}
