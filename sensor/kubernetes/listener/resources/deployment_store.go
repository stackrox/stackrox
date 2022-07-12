package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/selector"
)

// DeploymentStore stores deployments.
type DeploymentStore struct {
	lock sync.RWMutex

	// Stores deployment IDs by namespaces.
	deploymentIDs map[string]map[string]struct{}
	// Stores deployments by IDs.
	deployments map[string]*deploymentWrap
}

// NewDeploymentStore creates and returns a new deployment store.
func NewDeploymentStore() *DeploymentStore {
	return &DeploymentStore{
		deploymentIDs: make(map[string]map[string]struct{}),
		deployments:   make(map[string]*deploymentWrap),
	}
}

func (ds *DeploymentStore) addOrUpdateDeployment(wrap *deploymentWrap) {
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

func (ds *DeploymentStore) removeDeployment(wrap *deploymentWrap) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	ids := ds.deploymentIDs[wrap.GetNamespace()]
	if ids == nil {
		return
	}
	delete(ids, wrap.GetId())
	delete(ds.deployments, wrap.GetId())
}

func (ds *DeploymentStore) getDeploymentsByIDs(namespace string, idSet set.StringSet) []*deploymentWrap {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	deployments := make([]*deploymentWrap, 0, len(idSet))
	for id := range idSet {
		wrap := ds.deployments[id]
		if wrap != nil {
			deployments = append(deployments, wrap)
		}
	}
	return deployments
}

func (ds *DeploymentStore) GetMatchingDeployments(namespace string, sel selector.Selector) []*storage.Deployment {
	var result []*storage.Deployment
	for _, wrap := range ds.getMatchingDeployments(namespace, sel) {
		result = append(result, wrap.GetDeployment())
	}
	return result
}

func (ds *DeploymentStore) getMatchingDeployments(namespace string, sel selector.Selector) (matching []*deploymentWrap) {
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

		if sel.Matches(selector.CreateLabelsWithLen(wrap.PodLabels)) {
			matching = append(matching, wrap)
		}
	}
	return
}

// CountDeploymentsForNamespace returns the number of deployments in a namespace
func (ds *DeploymentStore) CountDeploymentsForNamespace(namespace string) int {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return len(ds.deploymentIDs[namespace])
}

// OnNamespaceDeleted reacts to a namespace deletion, deleting all deployments in this namespace from the store.
func (ds *DeploymentStore) OnNamespaceDeleted(namespace string) {
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
func (ds *DeploymentStore) GetAll() []*storage.Deployment {
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
func (ds *DeploymentStore) Get(id string) *storage.Deployment {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	wrap := ds.deployments[id]
	if wrap == nil {
		return nil
	}
	return wrap.GetDeployment()
}
