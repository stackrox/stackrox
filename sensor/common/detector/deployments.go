package detector

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

type deploymentStore struct {
	lock sync.RWMutex
	// deploymentMap is deployment ID -> Deployment object
	// this is a fairly cheap map because we have a map further down which
	// holds references to the *storage.Deployment object so this is really
	// just a minor amount of overhead
	deploymentMap map[string]*storage.Deployment
}

func newDeploymentStore() *deploymentStore {
	return &deploymentStore{
		deploymentMap: make(map[string]*storage.Deployment),
	}
}

func (d *deploymentStore) upsertDeployment(deployment *storage.Deployment) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.deploymentMap[deployment.GetId()] = deployment
}

func (d *deploymentStore) removeDeployment(id string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	delete(d.deploymentMap, id)
}

func (d *deploymentStore) getDeployment(id string) *storage.Deployment {
	d.lock.RLock()
	defer d.lock.RUnlock()

	return d.deploymentMap[id]
}

func (d *deploymentStore) getAll() []*storage.Deployment {
	d.lock.RLock()
	defer d.lock.RUnlock()

	deployments := make([]*storage.Deployment, 0, len(d.deploymentMap))
	for _, deployment := range d.deploymentMap {
		deployments = append(deployments, deployment)
	}
	return deployments
}
