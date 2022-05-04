package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/store"
)

var _ store.DeploymentStore = (*DeploymentStoreImpl)(nil)

// DeploymentStoreImpl stores deployments.
type DeploymentStoreImpl struct {
	lock sync.RWMutex

	// Stores deployment IDs by namespaces.
	deploymentIDs map[string]map[string]struct{}
	// Stores deployments by IDs.
	deployments map[string]store.DeploymentWrap
}

// newDeploymentStore creates and returns a new deployment store.
func newDeploymentStore() *DeploymentStoreImpl {
	return &DeploymentStoreImpl{
		deploymentIDs: make(map[string]map[string]struct{}),
		deployments:   make(map[string]store.DeploymentWrap),
	}
}

// AddOrUpdateDeployment upsert
func (ds *DeploymentStoreImpl) AddOrUpdateDeployment(wrap store.DeploymentWrap) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	ids, ok := ds.deploymentIDs[wrap.GetNamespace()]
	if !ok {
		ids = make(map[string]struct{})
		ds.deploymentIDs[wrap.GetNamespace()] = ids
	}
	ids[wrap.GetId()] = struct{}{}

	ds.deployments[wrap.GetId()] = wrap
}

// RemoveDeployment delete
func (ds *DeploymentStoreImpl) RemoveDeployment(wrap store.DeploymentWrap) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	ids := ds.deploymentIDs[wrap.GetNamespace()]
	if ids == nil {
		return
	}
	delete(ids, wrap.GetId())
	delete(ds.deployments, wrap.GetId())
}

// GetDeploymentsByIDs get deployments by id
func (ds *DeploymentStoreImpl) GetDeploymentsByIDs(namespace string, idSet set.StringSet) []store.DeploymentWrap {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	deployments := make([]store.DeploymentWrap, 0, len(idSet))
	for id := range idSet {
		wrap := ds.deployments[id]
		if wrap != nil {
			deployments = append(deployments, wrap)
		}
	}
	return deployments
}

// GetMatchingDeployments get matching deployments with a given selector
func (ds *DeploymentStoreImpl) GetMatchingDeployments(namespace string, sel store.Selector) (matching []store.DeploymentWrap) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	ids := ds.deploymentIDs[namespace]
	if ids == nil {
		return
	}

	for id := range ids {
		wrap := ds.deployments[id]
		if wrap == nil {
			continue
		}

		if sel.Matches(createLabelsWithLen(wrap.GetPodLabels())) {
			matching = append(matching, wrap)
		}
	}
	return
}

// CountDeploymentsForNamespace returns the number of deployments in a namespace
func (ds *DeploymentStoreImpl) CountDeploymentsForNamespace(namespace string) int {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return len(ds.deploymentIDs[namespace])
}

// OnNamespaceDeleted reacts to a namespace deletion, deleting all deployments in this namespace from the store.
func (ds *DeploymentStoreImpl) OnNamespaceDeleted(namespace string) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	ids := ds.deploymentIDs[namespace]
	if ids == nil {
		return
	}

	for id := range ids {
		delete(ds.deployments, id)
	}
	delete(ds.deploymentIDs, namespace)
}

// GetAll returns all deployments.
func (ds *DeploymentStoreImpl) GetAll() []*storage.Deployment {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	var ret []*storage.Deployment
	for _, wrap := range ds.deployments {
		if wrap != nil {
			ret = append(ret, wrap.GetDeployment())
		}
	}
	return ret
}

// Get returns deployment for supplied id.
func (ds *DeploymentStoreImpl) Get(id string) *storage.Deployment {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	wrap := ds.deployments[id]
	if wrap == nil {
		return nil
	}
	return wrap.GetDeployment()
}
