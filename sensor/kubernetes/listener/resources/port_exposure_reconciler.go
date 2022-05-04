package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/store"
)

// portExposureReconciler reconciles the port exposures in the deployment store on receiving
// service or route updates.
type portExposureReconciler interface {
	UpdateExposuresForMatchingDeployments(namespace string, sel store.Selector) []*central.SensorEvent
	UpdateExposureOnServiceCreate(svc store.ServiceWithRoutes) []*central.SensorEvent
}

type portExposureReconcilerImpl struct {
	deploymentStore store.DeploymentStore
	serviceStore    store.ServiceStore
}

func newPortExposureReconciler(deploymentStore store.DeploymentStore, serviceStore store.ServiceStore) portExposureReconciler {
	return &portExposureReconcilerImpl{
		deploymentStore: deploymentStore,
		serviceStore:    serviceStore,
	}
}

func (p *portExposureReconcilerImpl) UpdateExposuresForMatchingDeployments(namespace string, sel store.Selector) []*central.SensorEvent {
	var events []*central.SensorEvent
	for _, deploymentWrap := range p.deploymentStore.GetMatchingDeployments(namespace, sel) {
		if svcs := p.serviceStore.GetMatchingServicesWithRoutes(deploymentWrap.GetNamespace(), deploymentWrap.GetPodLabels()); len(svcs) > 0 || deploymentWrap.AnyNonHostPort() {
			cloned := deploymentWrap.Clone()
			cloned.UpdatePortExposureFromServices(svcs...)
			p.deploymentStore.AddOrUpdateDeployment(cloned)
		}

		events = append(events, deploymentWrap.ToEvent(central.ResourceAction_UPDATE_RESOURCE))
	}
	return events
}

func (p *portExposureReconcilerImpl) UpdateExposureOnServiceCreate(svc store.ServiceWithRoutes) []*central.SensorEvent {
	var events []*central.SensorEvent
	for _, deploymentWrap := range p.deploymentStore.GetMatchingDeployments(svc.GetNamespace(), svc.GetSelector()) {
		cloned := deploymentWrap.Clone()
		cloned.UpdatePortExposure(svc)
		p.deploymentStore.AddOrUpdateDeployment(cloned)
		events = append(events, cloned.ToEvent(central.ResourceAction_UPDATE_RESOURCE))
	}
	return events
}
