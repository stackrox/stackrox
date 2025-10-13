package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// NewDeploymentStore returns new instance of DeploymentStore.
func NewDeploymentStore(pods *PodStore) *DeploymentStore {
	return &DeploymentStore{
		deployments:          make(map[string]map[string]*storage.Deployment),
		deploymentNamesToIds: make(map[string]map[string]string),
		pods:                 pods,
	}
}

// DeploymentStore stores the deployments.
type DeploymentStore struct {
	// A map of maps of deployment ID -> deployment object, by namespace
	deployments map[string]map[string]*storage.Deployment
	// A map of maps of deployment names to IDs, by namespace.
	deploymentNamesToIds map[string]map[string]string

	pods *PodStore

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
		depNameToIdMap := m.deploymentNamesToIds[deployment.GetNamespace()]
		if depMap == nil {
			depMap = make(map[string]*storage.Deployment)
			m.deployments[deployment.GetNamespace()] = depMap
		}
		depMap[deployment.GetId()] = deployment

		if depNameToIdMap == nil {
			depNameToIdMap = make(map[string]string)
			m.deploymentNamesToIds[deployment.GetNamespace()] = depNameToIdMap
		}
		depNameToIdMap[deployment.GetName()] = deployment.GetId()

	case central.ResourceAction_REMOVE_RESOURCE:
		// Deployment remove event contains full deployment object.
		delete(m.deployments[deployment.GetNamespace()], deployment.GetId())
		delete(m.deploymentNamesToIds[deployment.GetNamespace()], deployment.GetName())
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

// GetByName returns a deployment given namespace and deployment name.
func (m *DeploymentStore) GetByName(namespace, name string) *storage.Deployment {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	deploymentNamesToIdsMap := m.deploymentNamesToIds[namespace]
	if deploymentNamesToIdsMap == nil {
		return nil
	}

	depMap := m.deployments[namespace]
	if depMap == nil {
		return nil
	}

	deploymentID := deploymentNamesToIdsMap[name]
	return depMap[deploymentID]
}

// OnNamespaceDelete removes deployments in supplied namespace.
func (m *DeploymentStore) OnNamespaceDelete(namespace string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.deployments, namespace)
	delete(m.deploymentNamesToIds, namespace)
}
