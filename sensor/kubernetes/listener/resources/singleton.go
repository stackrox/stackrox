package resources

import "github.com/stackrox/rox/pkg/sync"

var (
	dsInit   sync.Once
	depStore *DeploymentStore
)

// DeploymentStoreSingleton returns a singleton of the DeploymentStore
func DeploymentStoreSingleton() *DeploymentStore {
	dsInit.Do(func() {
		depStore = newDeploymentStore()
	})
	return depStore
}
