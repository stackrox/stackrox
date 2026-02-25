package status

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/reflectutils"
)

// SkipStatusControllerUpdates filters events triggered by status controller updates to prevent unnecessary reconciliations.
// It skips reconciliation when status controller owned conditions (Available, Progressing) have changed.
//
// This can be instantiated with either:
//   - A specific CR type (e.g., SkipStatusControllerUpdates[*Central]{}) for typed usage in controller.go.
//   - ctrlClient.Object (e.g., SkipStatusControllerUpdates[ctrlClient.Object]{}) for untyped usage as predicate.Predicate in reconciler.go.
type SkipStatusControllerUpdates[T ctrlClient.Object] struct {
	predicate.TypedFuncs[T]
	Logger logr.Logger
}

// Update implements predicate.TypedPredicate.
// Returns true to allow the update event to trigger reconciliation, false to skip it.
func (p SkipStatusControllerUpdates[T]) Update(e event.TypedUpdateEvent[T]) bool {
	log := p.Logger.WithName("predicate-skip-status-ctrl-update")

	// Check for nil using reflection to handle the interface nil gotcha.
	if reflectutils.IsNil(e.ObjectOld) || reflectutils.IsNil(e.ObjectNew) {
		// One of the objects is nil, allow reconciliation.
		log.Info("One of the objects is nil, allowing reconciliation")
		return true
	}

	objOldT, err := toObjectForStatusController(e.ObjectOld, log)
	if err != nil {
		log.Info("Failed to convert old object to ObjectForStatusController, allowing reconciliation",
			"error", err,
			"objectOldType", fmt.Sprintf("%T", e.ObjectOld))
		return true
	}

	objNewT, err := toObjectForStatusController(e.ObjectNew, log)
	if err != nil {
		log.Info("Failed to convert new object to ObjectForStatusController, allowing reconciliation",
			"error", err,
			"objectNewType", fmt.Sprintf("%T", e.ObjectNew))
		return true
	}

	statusControllerConditionsChanged :=
		conditionsChanged(objOldT, objNewT, platform.ConditionProgressing) ||
			conditionsChanged(objOldT, objNewT, platform.ConditionAvailable)

	if statusControllerConditionsChanged {
		log.Info("Detected changed status controller condition, skipping reconciliation")
		return false
	}

	return true
}

// toObjectForStatusController converts an interface to ObjectForStatusController,
// handling typed and unstructured objects.
func toObjectForStatusController(obj any, log logr.Logger) (platform.ObjectForStatusController, error) {
	// First try direct type assertion.
	if typed, ok := obj.(platform.ObjectForStatusController); ok {
		return typed, nil
	}

	// Next try unstructured.
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("object is neither ObjectForStatusController nor unstructured.Unstructured: %T", obj)
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

// conditionsChanged returns true if the conditions are different.
func conditionsChanged[T platform.ObjectForStatusController](oldObj, newObj T, condType platform.ConditionType) bool {
	old := oldObj.GetCondition(condType)
	new := newObj.GetCondition(condType)

	// One exists, one doesn't.
	if (old == nil) != (new == nil) {
		return true
	}

	// Both nil.
	if old == nil {
		return false
	}

	// Check if meaningful fields changed.
	return old.Status != new.Status ||
		old.Reason != new.Reason ||
		old.Message != new.Message
}

// SkipDeploymentSpecUpdates filters deployment events to only react to status changes.
// This prevents reconciliation when HPA or other controllers modify deployment.spec.replicas.
// The status controller only cares about deployment readiness (status), not scaling decisions (spec).
type SkipDeploymentSpecUpdates struct {
	predicate.TypedFuncs[*appsv1.Deployment]
}

// Update returns true only if deployment status changed (not spec).
// This allows HPA to modify replicas without triggering reconciliation.
func (p SkipDeploymentSpecUpdates) Update(e event.TypedUpdateEvent[*appsv1.Deployment]) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return true
	}
	return !reflect.DeepEqual(e.ObjectOld.Status, e.ObjectNew.Status)
}
