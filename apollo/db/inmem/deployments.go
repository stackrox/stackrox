package inmem

import (
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type deploymentStore struct {
	deployments      map[string]*v1.Deployment
	deploymentsMutex sync.Mutex

	persistent db.Storage
}

func newDeploymentStore(persistent db.Storage) *deploymentStore {
	return &deploymentStore{
		deployments: make(map[string]*v1.Deployment),
		persistent:  persistent,
	}
}

func (s *deploymentStore) loadFromPersistent() error {
	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()
	deployments, err := s.persistent.GetDeployments()
	if err != nil {
		return err
	}
	for _, d := range deployments {
		s.deployments[d.Id] = d
	}
	return nil
}

func (s *deploymentStore) GetDeployment(id string) (d *v1.Deployment, exist bool, err error) {
	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()
	d, exist = s.deployments[id]
	return
}

func (s *deploymentStore) GetDeployments() (deployments []*v1.Deployment, err error) {
	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()
	deployments = make([]*v1.Deployment, 0, len(s.deployments))
	for _, d := range s.deployments {
		deployments = append(deployments, d)
	}
	sort.SliceStable(deployments, func(i, j int) bool { return deployments[i].Id < deployments[j].Id })
	return
}

func (s *deploymentStore) AddDeployment(deployment *v1.Deployment) (err error) {
	if err = s.persistent.AddDeployment(deployment); err != nil {
		return
	}
	s.upsertDeployment(deployment)
	return
}

func (s *deploymentStore) UpdateDeployment(deployment *v1.Deployment) (err error) {
	if err = s.persistent.UpdateDeployment(deployment); err != nil {
		return
	}
	s.upsertDeployment(deployment)
	return
}

func (s *deploymentStore) upsertDeployment(deployment *v1.Deployment) {
	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()
	s.deployments[deployment.Id] = deployment
}

func (s *deploymentStore) RemoveDeployment(id string) (err error) {
	if err = s.persistent.RemoveDeployment(id); err != nil {
		return
	}

	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()
	delete(s.deployments, id)
	return
}
