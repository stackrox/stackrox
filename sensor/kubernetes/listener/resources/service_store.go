package resources

import (
	routeV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/pkg/sync"
	v1 "k8s.io/api/core/v1"
	k8sLabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

type routeRef struct {
	namespace string
	name      string
}

// serviceStore stores service objects (by namespace and UID)
type serviceStore struct {
	// namespace->name->svcWrap
	services map[string]map[string]*serviceWrap
	// namespace->serviceName->routes
	routesByServiceMetadata map[string]map[string][]*routeV1.Route
	routesByRouteRef        map[routeRef]*routeV1.Route
	nodePortServices        map[types.UID]*serviceWrap

	// Protects all fields
	lock sync.RWMutex
}

// newServiceStore creates and returns a new service store.
func newServiceStore() *serviceStore {
	return &serviceStore{
		services:                make(map[string]map[string]*serviceWrap),
		routesByServiceMetadata: make(map[string]map[string][]*routeV1.Route),
		routesByRouteRef:        make(map[routeRef]*routeV1.Route),
		nodePortServices:        make(map[types.UID]*serviceWrap),
	}
}

func (ss *serviceStore) upsertRoute(route *routeV1.Route) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	ref := routeRef{name: route.Name, namespace: route.Namespace}
	if existing, ok := ss.routesByRouteRef[ref]; ok {
		ss.removeRouteNoLock(existing)
	}
	ss.routesByRouteRef[ref] = route
	nsMap := ss.routesByServiceMetadata[route.Namespace]
	if nsMap == nil {
		nsMap = make(map[string][]*routeV1.Route)
		ss.routesByServiceMetadata[route.Namespace] = nsMap
	}
	nsMap[route.Spec.To.Name] = append(nsMap[route.Spec.To.Name], route)
}

func (ss *serviceStore) removeRoute(route *routeV1.Route) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	ss.removeRouteNoLock(route)
}
func (ss *serviceStore) removeRouteNoLock(route *routeV1.Route) {
	nsMap := ss.routesByServiceMetadata[route.Namespace]
	thisRouteIdx := -1
	svcName := route.Spec.To.Name
	// Not very efficient, but we expect:
	// 1. route updates/deletions are rare
	// 2. there are usually very few routes per service (often just one)
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

func (ss *serviceStore) addOrUpdateService(svc *serviceWrap) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	nsMap := ss.services[svc.Namespace]
	if nsMap == nil {
		nsMap = make(map[string]*serviceWrap)
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
func (ss *serviceStore) NodePortServicesSnapshot() []*serviceWrap {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	wraps := make([]*serviceWrap, 0, len(ss.nodePortServices))
	for _, wrap := range ss.nodePortServices {
		wraps = append(wraps, wrap)
	}
	return wraps
}

func (ss *serviceStore) removeService(svc *v1.Service) {
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
func (ss *serviceStore) OnNamespaceDeleted(ns string) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	delete(ss.services, ns)
	delete(ss.routesByServiceMetadata, ns)
}

func (ss *serviceStore) getMatchingServicesWithRoutes(namespace string, labels map[string]string) (matching []serviceWithRoutes) {
	labelSet := k8sLabels.Set(labels)
	ss.lock.RLock()
	defer ss.lock.RUnlock()
	for _, entry := range ss.services[namespace] {
		if entry.selector.Matches(createLabelsWithLen(labelSet)) {
			svcWithRoutes := serviceWithRoutes{
				serviceWrap: entry,
				routes:      ss.routesByServiceMetadata[namespace][entry.Name],
			}
			matching = append(matching, svcWithRoutes)
		}
	}
	return matching
}

func (ss *serviceStore) getService(namespace string, name string) *serviceWrap {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	return ss.services[namespace][name]
}

func (ss *serviceStore) getRoutesForService(svcWrap *serviceWrap) []*routeV1.Route {
	return ss.routesByServiceMetadata[svcWrap.Namespace][svcWrap.Name]
}
