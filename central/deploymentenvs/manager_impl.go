package deploymentenvs

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/deploymentenvs"
	"github.com/stackrox/rox/pkg/providers"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	fetchMetadataTimeout = 60 * time.Second
)

type manager struct {
	mutex sync.RWMutex

	hasCentralDeploymentEnv   concurrency.Signal
	deploymentEnvsByClusterID map[string][]string

	listeners map[Listener]struct{}
}

func newManager() *manager {
	m := &manager{
		deploymentEnvsByClusterID: make(map[string][]string),
		listeners:                 make(map[Listener]struct{}),
		hasCentralDeploymentEnv:   concurrency.NewSignal(),
	}
	go m.fetchCentralDeploymentEnvs()
	return m
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

func (m *manager) fetchCentralDeploymentEnvs() {
	ctx, cancel := context.WithTimeout(context.Background(), fetchMetadataTimeout)
	defer cancel()

	providerMetadata := providers.GetMetadata(ctx)
	centralDeploymentEnv := deploymentenvs.GetDeploymentEnvFromProviderMetadata(providerMetadata)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.deploymentEnvsByClusterID[CentralClusterID] = []string{centralDeploymentEnv}
	m.hasCentralDeploymentEnv.Signal()
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

func (m *manager) GetDeploymentEnvironmentsByClusterID(block bool) map[string][]string {
	if block {
		m.hasCentralDeploymentEnv.Wait()
	} else if !m.hasCentralDeploymentEnv.IsDone() {
		return nil
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string][]string, len(m.deploymentEnvsByClusterID))

	for clusterID, envs := range m.deploymentEnvsByClusterID {
		result[clusterID] = sliceutils.ShallowClone(envs)
	}

	return result
}
