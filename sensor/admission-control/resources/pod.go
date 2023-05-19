package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// NewPodStore returns new instance of PodStore.
func NewPodStore() *PodStore {
	return &PodStore{
		pods:       make(map[string]map[string]map[string]string),
		podNameMap: make(map[string]map[string]string),
	}
}

// PodStore stores pod mappings.
type PodStore struct {
	// namespace -> deployment ID -> pod ID -> pod name
	pods map[string]map[string]map[string]string

	// namespace -> pod name -> deployment ID
	podNameMap map[string]map[string]string
	mutex      sync.RWMutex
}

// ProcessEvent processes pod event.
func (m *PodStore) ProcessEvent(action central.ResourceAction, pod *storage.Pod) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		// Build the pod map.
		depMap := m.pods[pod.GetNamespace()]
		if depMap == nil {
			depMap = make(map[string]map[string]string)
			m.pods[pod.GetNamespace()] = depMap
		}
		podMap := depMap[pod.GetDeploymentId()]
		if podMap == nil {
			podMap = make(map[string]string)
			depMap[pod.GetDeploymentId()] = podMap
		}
		podMap[pod.GetId()] = pod.GetName()

		// Build the name map.
		podNameMap := m.podNameMap[pod.GetNamespace()]
		if podNameMap == nil {
			podNameMap = make(map[string]string)
			m.podNameMap[pod.GetNamespace()] = podNameMap
		}
		podNameMap[pod.GetName()] = pod.GetDeploymentId()
	case central.ResourceAction_REMOVE_RESOURCE:
		delete(m.pods[pod.GetNamespace()][pod.GetDeploymentId()], pod.GetId())
		delete(m.pods[pod.GetNamespace()], pod.GetName())
	}
}

// GetDeploymentID returns deploymentID containing the pod in namespace.
func (m *PodStore) GetDeploymentID(namespace, pod string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	podNameMap := m.podNameMap[namespace]
	if podNameMap == nil {
		return ""
	}
	return podNameMap[pod]
}

// OnNamespaceDelete removes all pod mapping for pods in supplied namespace.
func (m *PodStore) OnNamespaceDelete(namespace string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.pods, namespace)
	delete(m.podNameMap, namespace)
}

// OnDeploymentDelete removes all pod mapping for pods in supplied deployment.
func (m *PodStore) OnDeploymentDelete(namespace, deploymentID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	depMap := m.pods[namespace]
	if depMap == nil {
		return
	}
	podMap := depMap[deploymentID]
	if podMap == nil {
		return
	}

	for _, podName := range podMap {
		delete(m.podNameMap[namespace], podName)
	}
	delete(m.pods[namespace], deploymentID)
}
