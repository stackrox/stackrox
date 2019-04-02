package deploymentenvs

// Listener allows to listen to deployment environments events.
type Listener interface {
	OnUpdate(clusterID string, deploymentEnvs []string)
	OnClusterMarkedInactive(clusterID string)
}

// Manager manages the active deployment environments.
type Manager interface {
	UpdateDeploymentEnvironments(clusterID string, deploymentEnvs []string)
	MarkClusterInactive(clusterID string)

	RegisterListener(listener Listener)
	UnregisterListener(listener Listener)
	GetDeploymentEnvironmentsByClusterID() map[string][]string
}

//go:generate mockgen-wrapper Listener,Manager
