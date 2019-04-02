package deploymentenvs

import "github.com/stackrox/rox/pkg/sync"

type manager struct {
	mutex sync.RWMutex

	deploymentEnvsByClusterID map[string][]string

	listeners map[Listener]struct{}
}

func newManager() *manager {
	return &manager{
		deploymentEnvsByClusterID: make(map[string][]string),
		listeners:                 make(map[Listener]struct{}),
	}
}

func (m *manager) RegisterListener(listener Listener) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.listeners[listener] = struct{}{}
}

func (m *manager) UnregisterListener(listener Listener) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.listeners, listener)
}

func (m *manager) UpdateDeploymentEnvironments(clusterID string, deploymentEnvs []string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.deploymentEnvsByClusterID[clusterID] = deploymentEnvs

	for listener := range m.listeners {
		listener.OnUpdate(clusterID, deploymentEnvs)
	}
}

func (m *manager) MarkClusterInactive(clusterID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.deploymentEnvsByClusterID, clusterID)

	for listener := range m.listeners {
		listener.OnClusterMarkedInactive(clusterID)
	}
}

func (m *manager) GetDeploymentEnvironmentsByClusterID() map[string][]string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string][]string, len(m.deploymentEnvsByClusterID))

	for clusterID, envs := range m.deploymentEnvsByClusterID {
		envsCopy := make([]string, len(envs))
		copy(envsCopy, envs)
		result[clusterID] = envsCopy
	}

	return result
}
