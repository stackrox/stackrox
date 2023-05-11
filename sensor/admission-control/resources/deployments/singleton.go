package deployments

import (
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/admission-control/resources/pods"
)

var (
	once sync.Once

	depStore *DeploymentStore
)

func initialize() {
	depStore = NewDeploymentStore(pods.Singleton())
}

// Singleton provides the interface for getting annotation values with a datastore backed implementation.
func Singleton() *DeploymentStore {
	once.Do(initialize)
	return depStore
}
