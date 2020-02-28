package resources

import (
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/apimachinery/pkg/labels"
)

// DeploymentStore stores deployments (by namespace and id).
type DeploymentStore struct {
	lock        sync.RWMutex
	deployments map[string]map[string]*deploymentWrap
}

// newDeploymentStore creates and returns a new deployment store.
func newDeploymentStore() *DeploymentStore {
	return &DeploymentStore{
		deployments: make(map[string]map[string]*deploymentWrap),
	}
}

func (ds *DeploymentStore) addOrUpdateDeployment(wrap *deploymentWrap) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	nsMap := ds.deployments[wrap.GetNamespace()]
	if nsMap == nil {
		nsMap = make(map[string]*deploymentWrap)
		ds.deployments[wrap.GetNamespace()] = nsMap
	}
	nsMap[wrap.GetId()] = wrap
}

func (ds *DeploymentStore) removeDeployment(wrap *deploymentWrap) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	nsMap := ds.deployments[wrap.GetNamespace()]
	if nsMap == nil {
		return
	}
	delete(nsMap, wrap.GetId())
}

func (ds *DeploymentStore) getDeploymentsByIDs(namespace string, idSet set.StringSet) []*deploymentWrap {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	deployments := make([]*deploymentWrap, 0, len(idSet))
	for _, wrap := range ds.deployments[namespace] {
		if idSet.Contains(wrap.GetId()) {
			deployments = append(deployments, wrap)
		}
	}
	return deployments
}

func (ds *DeploymentStore) getMatchingDeployments(namespace string, sel selector) (matching []*deploymentWrap) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	for _, wrap := range ds.deployments[namespace] {
		if sel.Matches(labels.Set(wrap.PodLabels)) {
			matching = append(matching, wrap)
		}
	}
	return
}

// CountDeploymentsForNamespace returns the number of deployments in a namespace
func (ds *DeploymentStore) CountDeploymentsForNamespace(namespace string) int {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return len(ds.deployments[namespace])
}

// OnNamespaceDeleted reacts to a namespace deletion, deleting all deployments in this namespace from the store.
func (ds *DeploymentStore) OnNamespaceDeleted(ns string) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	delete(ds.deployments, ns)
}
