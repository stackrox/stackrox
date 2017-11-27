package inmem

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

func (i *InMemoryStore) loadDeployments() error {
	i.deploymentsMutex.Lock()
	defer i.deploymentsMutex.Unlock()
	deployments, err := i.persistent.GetDeployments()
	if err != nil {
		return err
	}
	for _, d := range deployments {
		i.deployments[d.Id] = d
	}
	return nil
}

// GetDeployment retrieves a deployment by id.
func (i *InMemoryStore) GetDeployment(id string) (d *v1.Deployment, exist bool, err error) {
	i.deploymentsMutex.Lock()
	defer i.deploymentsMutex.Unlock()
	d, exist = i.deployments[id]
	return
}

// GetDeployments retrieves all deployments.
func (i *InMemoryStore) GetDeployments() (deployments []*v1.Deployment, err error) {
	i.deploymentsMutex.Lock()
	defer i.deploymentsMutex.Unlock()
	deployments = make([]*v1.Deployment, 0, len(i.deployments))
	for _, d := range i.deployments {
		deployments = append(deployments, d)
	}
	sort.SliceStable(deployments, func(i, j int) bool { return deployments[i].Id < deployments[j].Id })
	return
}

// AddDeployment adds a new deployment.
func (i *InMemoryStore) AddDeployment(deployment *v1.Deployment) (err error) {
	if err = i.persistent.AddDeployment(deployment); err != nil {
		return
	}
	i.upsertDeployment(deployment)
	return
}

// UpdateDeployment updates a deployment.
func (i *InMemoryStore) UpdateDeployment(deployment *v1.Deployment) (err error) {
	if err = i.persistent.UpdateDeployment(deployment); err != nil {
		return
	}
	i.upsertDeployment(deployment)
	return
}

func (i *InMemoryStore) upsertDeployment(deployment *v1.Deployment) {
	i.alertMutex.Lock()
	defer i.alertMutex.Unlock()
	i.deployments[deployment.Id] = deployment
}

// RemoveDeployment removes a deployment.
func (i *InMemoryStore) RemoveDeployment(id string) (err error) {
	if err = i.persistent.RemoveDeployment(id); err != nil {
		return
	}

	i.deploymentsMutex.Lock()
	defer i.deploymentsMutex.Unlock()
	delete(i.deployments, id)
	return
}
