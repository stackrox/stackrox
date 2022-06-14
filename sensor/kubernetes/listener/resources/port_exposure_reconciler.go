package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// portExposureReconciler reconciles the port exposures in the deployment store on receiving
// service or route updates.
type portExposureReconciler interface {
	UpdateExposuresForMatchingDeployments(namespace string, sel selector) []*central.SensorEvent
	UpdateExposureOnServiceCreate(svc serviceWithRoutes) []*central.SensorEvent
}

type portExposureReconcilerImpl struct {
	deploymentStore *DeploymentStore
	serviceStore    *serviceStore
}

func newPortExposureReconciler(deploymentStore *DeploymentStore, serviceStore *serviceStore) portExposureReconciler {
	return &portExposureReconcilerImpl{
		deploymentStore: deploymentStore,
		serviceStore:    serviceStore,
	}
}

func (p *portExposureReconcilerImpl) UpdateExposuresForMatchingDeployments(namespace string, sel selector) []*central.SensorEvent {
	var events []*central.SensorEvent
	for _, deploymentWrap := range p.deploymentStore.getMatchingDeployments(namespace, sel) {
		if svcs := p.serviceStore.getMatchingServicesWithRoutes(deploymentWrap.Namespace, deploymentWrap.PodLabels); len(svcs) > 0 || deploymentWrap.anyNonHostPort() {
			cloned := deploymentWrap.Clone()
			cloned.updatePortExposureFromServices(svcs...)
			p.deploymentStore.addOrUpdateDeployment(cloned)
		}

		events = append(events, deploymentWrap.toEvent(central.ResourceAction_UPDATE_RESOURCE))
	}
	return events
}

func (p *portExposureReconcilerImpl) UpdateExposureOnServiceCreate(svc serviceWithRoutes) []*central.SensorEvent {
	var events []*central.SensorEvent
	for _, deploymentWrap := range p.deploymentStore.getMatchingDeployments(svc.Namespace, svc.selector) {
		cloned := deploymentWrap.Clone()
		cloned.updatePortExposure(svc)
		p.deploymentStore.addOrUpdateDeployment(cloned)
		events = append(events, cloned.toEvent(central.ResourceAction_UPDATE_RESOURCE))
	}
	return events
}
