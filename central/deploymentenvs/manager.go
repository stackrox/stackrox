package deploymentenvs

const (
	// CentralClusterID is the fake cluster ID that is used to register the deployment
	// environment of central.
	CentralClusterID = `CENTRAL`
)

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
	GetDeploymentEnvironmentsByClusterID(block bool) map[string][]string
}

//go:generate mockgen-wrapper Listener,Manager
