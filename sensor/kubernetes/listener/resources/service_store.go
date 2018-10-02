package resources

import (
	"k8s.io/api/core/v1"
	k8sLabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

// serviceStore stores service objects (by namespace and UID)
type serviceStore struct {
	services map[string]map[types.UID]*serviceWrap
}

// newServiceStore creates and returns a new service store.
func newServiceStore() *serviceStore {
	return &serviceStore{
		services: make(map[string]map[types.UID]*serviceWrap),
	}
}

func (ss *serviceStore) addOrUpdateService(svc *serviceWrap) {
	nsMap := ss.services[svc.Namespace]
	if nsMap == nil {
		nsMap = make(map[types.UID]*serviceWrap)
		ss.services[svc.Namespace] = nsMap
	}
	nsMap[svc.UID] = svc
}

func (ss *serviceStore) removeService(svc *v1.Service) {
	nsMap := ss.services[svc.Namespace]
	if nsMap == nil {
		return
	}
	delete(nsMap, svc.UID)
}

// OnNamespaceDeleted reacts to a namespace deletion, deleting all services in that namespace from the store.
func (ss *serviceStore) OnNamespaceDeleted(ns string) {
	delete(ss.services, ns)
}

func (ss *serviceStore) getMatchingServices(namespace string, labels map[string]string) (matching []*serviceWrap) {
	labelSet := k8sLabels.Set(labels)
	for _, entry := range ss.services[namespace] {
		if entry.selector.Matches(labelSet) {
			matching = append(matching, entry)
		}
	}
	return
}

func (ss *serviceStore) getService(namespace string, uid types.UID) *serviceWrap {
	return ss.services[namespace][uid]
}
