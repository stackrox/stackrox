package resources

import (
	routeV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type routeDispatcher struct {
	serviceStore           *serviceStore
	portExposureReconciler portExposureReconciler
}

func newRouteDispatcher(serviceStore *serviceStore, portExposureReconciler portExposureReconciler) *routeDispatcher {
	return &routeDispatcher{
		serviceStore:           serviceStore,
		portExposureReconciler: portExposureReconciler,
	}
}

func (r *routeDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	route, _ := obj.(*routeV1.Route)
	if route == nil {
		return nil
	}
	to := route.Spec.To
	// Currently, this is the only allowed kind, but this guards against future OpenShift updates.
	if to.Kind != "Service" || to.Name == "" {
		return nil
	}

	if action == central.ResourceAction_CREATE_RESOURCE || action == central.ResourceAction_UPDATE_RESOURCE {
		r.serviceStore.upsertRoute(route)
	}
	if action == central.ResourceAction_REMOVE_RESOURCE {
		r.serviceStore.removeRoute(route)
	}
	existingService := r.serviceStore.getService(route.Namespace, to.Name)
	// The route has a dangling reference to a service that doesn't exist.
	// We can just return now. If the service is created later, the route will be considered
	// at that time since we've put up-to-date route information into the store.
	if existingService == nil {
		return nil
	}
	// We do not append any Route event here because Routes, just like Services, are not tracked by central.
	event := component.NewEvent()
	event.AddDeploymentReference(resolver.ResolveDeploymentLabels(existingService.GetNamespace(), existingService.selector))
	return event
}
