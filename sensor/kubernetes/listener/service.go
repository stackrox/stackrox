package listener

import (
	"strings"

	pkgV1 "bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/containers"
	"bitbucket.org/stack-rox/apollo/pkg/kubernetes"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
)

type serviceWatchLister struct {
	watchLister
	eventC chan<- *listeners.DeploymentEventWrap
}

func newServiceWatchLister(client rest.Interface, eventC chan<- *listeners.DeploymentEventWrap, deploymentsFuncs ...func() (objs []interface{}, deploymentEvents []*pkgV1.DeploymentEvent)) *serviceWatchLister {
	swl := &serviceWatchLister{
		watchLister: newWatchLister(client),
		eventC:      eventC,
	}
	swl.setupWatch(strings.ToLower(kubernetes.Service)+"s", &v1.Service{}, func(serviceObj interface{}, action pkgV1.ResourceAction) {
		for _, deploymentGetter := range deploymentsFuncs {
			deploymentObjs, deploymentEvents := deploymentGetter()

			swl.updateDeployments(serviceObj, action, deploymentObjs, deploymentEvents)
		}
	})

	return swl
}

func (swl *serviceWatchLister) updateDeployments(serviceObj interface{}, action pkgV1.ResourceAction, deploymentObjs []interface{}, deploymentEvents []*pkgV1.DeploymentEvent) {
	for i, obj := range deploymentObjs {
		swl.updateDeployment(serviceObj, action, obj, deploymentEvents[i])
	}
}

func (swl *serviceWatchLister) updateDeployment(serviceObj interface{}, action pkgV1.ResourceAction, deployObj interface{}, deployEvent *pkgV1.DeploymentEvent) {
	switch action {
	case pkgV1.ResourceAction_CREATE_RESOURCE, pkgV1.ResourceAction_PREEXISTING_RESOURCE:
		swl.updateDeploymentUponServiceCreation(serviceObj, deployObj, deployEvent)
	case pkgV1.ResourceAction_UPDATE_RESOURCE, pkgV1.ResourceAction_REMOVE_RESOURCE:
		swl.updateDeploymentUponServiceModification(serviceObj, deployObj, deployEvent)
	}
}

func (swl *serviceWatchLister) updateDeploymentUponServiceCreation(serviceObj interface{}, deployObj interface{}, deployEvent *pkgV1.DeploymentEvent) {
	if swl.updatePortExposure(serviceObj, deployEvent.GetDeployment(), false) {
		deployEvent.Action = pkgV1.ResourceAction_UPDATE_RESOURCE

		swl.eventC <- &listeners.DeploymentEventWrap{
			DeploymentEvent: deployEvent,
			OriginalSpec:    deployObj,
		}
	}
}

func (swl *serviceWatchLister) updateDeploymentUponServiceModification(serviceObj interface{}, deployObj interface{}, deployEvent *pkgV1.DeploymentEvent) {
	if swl.updatePortExposure(serviceObj, deployEvent.GetDeployment(), true) {
		swl.updatePortExposureFromStore(deployEvent)

		deployEvent.Action = pkgV1.ResourceAction_UPDATE_RESOURCE

		swl.eventC <- &listeners.DeploymentEventWrap{
			DeploymentEvent: deployEvent,
			OriginalSpec:    deployObj,
		}
	}
}

func (swl *serviceWatchLister) updatePortExposureFromStore(event *pkgV1.DeploymentEvent) {
	if event.GetAction() == pkgV1.ResourceAction_REMOVE_RESOURCE {
		return
	}

	d := event.GetDeployment()

	for _, obj := range swl.store.List() {
		swl.updatePortExposure(obj, d, false)
	}
}

func (swl *serviceWatchLister) updatePortExposure(serviceObj interface{}, d *pkgV1.Deployment, reset bool) (updated bool) {
	service, ok := serviceObj.(*v1.Service)
	if !ok {
		logger.Warnf("Obj %+v is not of type v1.Service", serviceObj)
		return
	}
	s := serviceWrap(*service)

	if !s.matchSelector(d.GetLabels()) {
		return
	}

	for _, c := range d.GetContainers() {
		if reset && s.resetPorts(c) {
			updated = true
		} else if s.updatePorts(c) {
			updated = true
		}
	}

	return
}

type serviceWrap v1.Service

func (s serviceWrap) matchSelector(labels []*pkgV1.Deployment_KeyValue) bool {
	// Ignoring services without selectors.
	if len(s.Spec.Selector) == 0 {
		return false
	}

	labelMap := protoconv.ConvertDeploymentKeyValues(labels)
	for k, v := range s.Spec.Selector {
		if labelMap[k] != v {
			return false
		}
	}

	return true
}

func (s serviceWrap) updatePorts(container *pkgV1.Container) (updated bool) {
	for _, portConfig := range container.GetPorts() {
		if s.matchPort(portConfig) && s.updateExposure(portConfig) {
			updated = true
		}
	}

	return
}

func (s serviceWrap) resetPorts(container *pkgV1.Container) (updated bool) {
	for _, portConfig := range container.GetPorts() {
		if s.matchPort(portConfig) {
			portConfig.Exposure = pkgV1.PortConfig_INTERNAL
			updated = true
		}
	}

	return
}

func (s serviceWrap) matchPort(portConfig *pkgV1.PortConfig) bool {
	for _, p := range s.Spec.Ports {
		if p.TargetPort.Type == intstr.Int && p.TargetPort.IntVal == portConfig.ContainerPort {
			return true
		}

		if p.TargetPort.Type == intstr.String && p.TargetPort.StrVal == portConfig.Name {
			return true
		}
	}

	return false
}

func (s serviceWrap) updateExposure(portConfig *pkgV1.PortConfig) bool {
	if exposure := s.asExposureLevel(); containers.IncreasedExposureLevel(portConfig.Exposure, exposure) {
		portConfig.Exposure = exposure
		return true
	}

	return false
}

func (s serviceWrap) asExposureLevel() pkgV1.PortConfig_Exposure {
	switch s.Spec.Type {
	case v1.ServiceTypeLoadBalancer:
		return pkgV1.PortConfig_EXTERNAL
	case v1.ServiceTypeNodePort:
		return pkgV1.PortConfig_NODE
	default:
		return pkgV1.PortConfig_INTERNAL
	}
}
