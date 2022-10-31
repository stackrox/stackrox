package service

import (
	routeV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/common/store/service/servicewrapper"
	v1 "k8s.io/api/core/v1"
	k8sLabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

type routeRef struct {
	namespace string
	name      string
}

// serviceStoreImpl stores service objects (by namespace and UID)
type serviceStoreImpl struct {
	// namespace->name->svcWrap
	services map[string]map[string]*servicewrapper.SelectorWrap
	// namespace->serviceName->Routes
	routesByServiceMetadata map[string]map[string][]*routeV1.Route
	routesByRouteRef        map[routeRef]*routeV1.Route
	nodePortServices        map[types.UID]*servicewrapper.SelectorWrap

	// Protects all fields
	lock sync.RWMutex
}

// NewServiceStore creates and returns a new service store.
func NewServiceStore() store.ServiceStore {
	return &serviceStoreImpl{
		services:                make(map[string]map[string]*servicewrapper.SelectorWrap),
		routesByServiceMetadata: make(map[string]map[string][]*routeV1.Route),
		routesByRouteRef:        make(map[routeRef]*routeV1.Route),
		nodePortServices:        make(map[types.UID]*servicewrapper.SelectorWrap),
	}
}

func (ss *serviceStoreImpl) UpsertRoute(route *routeV1.Route) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	ref := routeRef{name: route.Name, namespace: route.Namespace}
	if existing, ok := ss.routesByRouteRef[ref]; ok {
		ss.RemoveRouteNoLock(existing)
	}
	ss.routesByRouteRef[ref] = route
	nsMap := ss.routesByServiceMetadata[route.Namespace]
	if nsMap == nil {
		nsMap = make(map[string][]*routeV1.Route)
		ss.routesByServiceMetadata[route.Namespace] = nsMap
	}
	nsMap[route.Spec.To.Name] = append(nsMap[route.Spec.To.Name], route)
}

func (ss *serviceStoreImpl) RemoveRoute(route *routeV1.Route) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	ss.RemoveRouteNoLock(route)
}
func (ss *serviceStoreImpl) RemoveRouteNoLock(route *routeV1.Route) {
	nsMap := ss.routesByServiceMetadata[route.Namespace]
	thisRouteIdx := -1
	svcName := route.Spec.To.Name
	// Not very efficient, but we expect:
	// 1. route updates/deletions are rare
	// 2. there are usually very few Routes per service (often just one)
	for i, routeFromMap := range nsMap[svcName] {
		if routeFromMap.Name == route.Name {
			thisRouteIdx = i
			break
		}
	}
	if thisRouteIdx != -1 {
		nsMap[svcName] = append(nsMap[svcName][:thisRouteIdx], nsMap[svcName][thisRouteIdx+1:]...)
	}
	if len(nsMap[svcName]) == 0 {
		delete(nsMap, svcName)
	}
	if len(nsMap) == 0 {
		delete(ss.routesByServiceMetadata, route.Namespace)
	}
	delete(ss.routesByRouteRef, routeRef{name: route.Name, namespace: route.Namespace})
}

func (ss *serviceStoreImpl) UpsertService(svc *servicewrapper.SelectorWrap) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	nsMap := ss.services[svc.Namespace]
	if nsMap == nil {
		nsMap = make(map[string]*servicewrapper.SelectorWrap)
		ss.services[svc.Namespace] = nsMap
	}
	nsMap[svc.Name] = svc
	if svc.Spec.Type == v1.ServiceTypeNodePort || svc.Spec.Type == v1.ServiceTypeLoadBalancer {
		ss.nodePortServices[svc.UID] = svc
	} else {
		delete(ss.nodePortServices, svc.UID)
	}
}

// NodePortServicesSnapshot returns a snapshot of the service wraps
func (ss *serviceStoreImpl) NodePortServicesSnapshot() []*servicewrapper.SelectorWrap {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	wraps := make([]*servicewrapper.SelectorWrap, 0, len(ss.nodePortServices))
	for _, wrap := range ss.nodePortServices {
		wraps = append(wraps, wrap)
	}
	return wraps
}

func (ss *serviceStoreImpl) RemoveService(svc *v1.Service) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	nsMap := ss.services[svc.Namespace]
	delete(nsMap, svc.Name)
	if len(nsMap) == 0 {
		delete(ss.services, svc.Namespace)
	}
	delete(ss.nodePortServices, svc.UID)
}

// OnNamespaceDeleted reacts to a namespace deletion, deleting all services in that namespace from the store.
func (ss *serviceStoreImpl) OnNamespaceDeleted(ns string) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	delete(ss.services, ns)
	delete(ss.routesByServiceMetadata, ns)
}

func (ss *serviceStoreImpl) GetMatchingServicesWithRoutes(namespace string, labels map[string]string) (matching []servicewrapper.SelectorRouteWrap) {
	labelSet := k8sLabels.Set(labels)
	ss.lock.RLock()
	defer ss.lock.RUnlock()
	for _, entry := range ss.services[namespace] {
		if entry.Selector.Matches(selector.CreateLabelsWithLen(labelSet)) {
			svcWithRoutes := servicewrapper.SelectorRouteWrap{
				SelectorWrap: entry,
				Routes:       ss.routesByServiceMetadata[namespace][entry.Name],
			}
			matching = append(matching, svcWithRoutes)
		}
	}
	return matching
}

func (ss *serviceStoreImpl) GetService(namespace string, name string) *servicewrapper.SelectorWrap {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	return ss.services[namespace][name]
}

func (ss *serviceStoreImpl) GetRoutesForService(svcWrap *servicewrapper.SelectorWrap) []*routeV1.Route {
	return ss.routesByServiceMetadata[svcWrap.Namespace][svcWrap.Name]
}
