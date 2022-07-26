package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// NewDeploymentStore returns new instance of DeploymentStore.
func NewDeploymentStore(pods *PodStore) *DeploymentStore {
	return &DeploymentStore{
		deployments: make(map[string]map[string]*storage.Deployment),
		pods:        pods,
	}
}

// DeploymentStore stores the deployments.
type DeploymentStore struct {
	deployments map[string]map[string]*storage.Deployment
	pods        *PodStore

	mutex sync.RWMutex
}

// ProcessEvent processes deployment event.
func (m *DeploymentStore) ProcessEvent(action central.ResourceAction, obj interface{}) {
	deployment, isDeployment := obj.(*storage.Deployment)
	if !isDeployment {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		depMap := m.deployments[deployment.GetNamespace()]
		if depMap == nil {
			depMap = make(map[string]*storage.Deployment)
			m.deployments[deployment.GetNamespace()] = depMap
		}
		depMap[deployment.GetId()] = deployment
	case central.ResourceAction_REMOVE_RESOURCE:
		// Deployment remove event contains full deployment object.
		delete(m.deployments[deployment.GetNamespace()], deployment.GetId())
		m.pods.OnDeploymentDelete(deployment.GetNamespace(), deployment.GetId())
	}
}

// Get returns a deployment given namespace and deployment id.
func (m *DeploymentStore) Get(namespace, deploymentID string) *storage.Deployment {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	depMap := m.deployments[namespace]
	if depMap == nil {
		return nil
	}
	return depMap[deploymentID]
}

// OnNamespaceDelete removes deployments in supplied namespace.
func (m *DeploymentStore) OnNamespaceDelete(namespace string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.deployments, namespace)
}
