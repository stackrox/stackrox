package resources

import (
	"strings"
	"time"

	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/containers"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watchlister"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
)

var logger = logging.LoggerForModule()

// ServiceWatchLister watches k8s services
type ServiceWatchLister struct {
	watchlister.WatchLister
	eventC       chan<- *listeners.EventWrap
	resyncPeriod time.Duration
}

// NewServiceWatchLister creates a watcher that listeners to services
func NewServiceWatchLister(client rest.Interface, eventC chan<- *listeners.EventWrap, resyncPeriod time.Duration, deploymentsFuncs ...func() (objs []interface{}, deployments []*pkgV1.Deployment)) *ServiceWatchLister {
	swl := &ServiceWatchLister{
		WatchLister:  watchlister.NewWatchLister(client, resyncPeriod),
		eventC:       eventC,
		resyncPeriod: resyncPeriod,
	}
	swl.SetupWatch(strings.ToLower(kubernetes.Service)+"s", &v1.Service{}, func(serviceObj interface{}, action pkgV1.ResourceAction) {
		for _, deploymentGetter := range deploymentsFuncs {
			deploymentObjs, deploymentEvents := deploymentGetter()

			swl.updateDeployments(serviceObj, action, deploymentObjs, deploymentEvents)
		}
	})

	return swl
}

func (swl *ServiceWatchLister) updateDeployments(serviceObj interface{}, action pkgV1.ResourceAction, deploymentObjs []interface{}, deploymentEvents []*pkgV1.Deployment) {
	for i, obj := range deploymentObjs {
		swl.updateDeployment(serviceObj, action, obj, deploymentEvents[i])
	}
}

func (swl *ServiceWatchLister) updateDeployment(serviceObj interface{}, action pkgV1.ResourceAction, deployObj interface{}, deployEvent *pkgV1.Deployment) {
	switch action {
	case pkgV1.ResourceAction_CREATE_RESOURCE, pkgV1.ResourceAction_PREEXISTING_RESOURCE:
		swl.updateDeploymentUponServiceCreation(serviceObj, deployObj, deployEvent)
	case pkgV1.ResourceAction_UPDATE_RESOURCE:
		swl.updateDeploymentUponServiceModification(serviceObj, deployObj, deployEvent)
	}
}

func (swl *ServiceWatchLister) updateDeploymentUponServiceCreation(serviceObj interface{}, deployObj interface{}, deployment *pkgV1.Deployment) {
	if swl.updatePortExposure(serviceObj, deployment, false) {
		swl.eventC <- &listeners.EventWrap{
			SensorEvent: &pkgV1.SensorEvent{
				Action: pkgV1.ResourceAction_UPDATE_RESOURCE,
				Resource: &pkgV1.SensorEvent_Deployment{
					Deployment: deployment,
				},
			},
			OriginalSpec: deployObj,
		}
	}
}

func (swl *ServiceWatchLister) updateDeploymentUponServiceModification(serviceObj interface{}, deployObj interface{}, deployment *pkgV1.Deployment) {
	if swl.updatePortExposure(serviceObj, deployment, true) {
		swl.updatePortExposureFromStore(deployment)

		swl.eventC <- &listeners.EventWrap{
			SensorEvent: &pkgV1.SensorEvent{
				Action: pkgV1.ResourceAction_UPDATE_RESOURCE,
				Resource: &pkgV1.SensorEvent_Deployment{
					Deployment: deployment,
				},
			},
			OriginalSpec: deployObj,
		}
	}
}

func (swl *ServiceWatchLister) updatePortExposureFromStore(deployment *pkgV1.Deployment) {
	for _, obj := range swl.Store.List() {
		swl.updatePortExposure(obj, deployment, false)
	}
}

func (swl *ServiceWatchLister) updatePortExposure(serviceObj interface{}, d *pkgV1.Deployment, reset bool) (updated bool) {
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

func (s serviceWrap) matchSelector(labels map[string]string) bool {
	// Ignoring services without selectors.
	if len(s.Spec.Selector) == 0 {
		return false
	}

	for k, v := range s.Spec.Selector {
		if labels[k] != v {
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
