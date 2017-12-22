package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type deploymentStore struct {
	deployments      map[string]*v1.Deployment
	deploymentsMutex sync.Mutex

	persistent db.DeploymentStorage
}

func newDeploymentStore(persistent db.DeploymentStorage) *deploymentStore {
	return &deploymentStore{
		deployments: make(map[string]*v1.Deployment),
		persistent:  persistent,
	}
}

func (s *deploymentStore) loadFromPersistent() error {
	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()
	deployments, err := s.persistent.GetDeployments(&v1.GetDeploymentsRequest{})
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

func (s *deploymentStore) GetDeployments(request *v1.GetDeploymentsRequest) (deployments []*v1.Deployment, err error) {
	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()

	nameSet := stringWrap(request.GetName()).asSet()
	typeSet := stringWrap(request.GetType()).asSet()
	imageShaSet := stringWrap(request.GetImageSha()).asSet()

	for _, d := range s.deployments {
		if _, ok := nameSet[d.GetName()]; len(nameSet) > 0 && !ok {
			continue
		}

		if _, ok := typeSet[d.GetType()]; len(typeSet) > 0 && !ok {
			continue
		}

		if len(imageShaSet) > 0 && !s.matchImageSha(imageShaSet, d.GetContainers()) {

			continue
		}

		deployments = append(deployments, d)
	}

	sort.SliceStable(deployments, func(i, j int) bool { return deployments[i].Id < deployments[j].Id })
	return
}

func (s *deploymentStore) matchImageSha(imageShaSet map[string]struct{}, containers []*v1.Container) bool {
	for _, c := range containers {
		if _, ok := imageShaSet[c.GetImage().GetSha()]; !ok {
			return false
		}
	}

	return true
}

func (s *deploymentStore) AddDeployment(deployment *v1.Deployment) (err error) {
	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()
	if _, ok := s.deployments[deployment.Id]; ok {
		return fmt.Errorf("Cannot add deployment %v because it already exists", deployment.Id)
	}
	if err = s.persistent.AddDeployment(deployment); err != nil {
		return
	}
	s.upsertDeployment(deployment)
	return
}

func (s *deploymentStore) UpdateDeployment(deployment *v1.Deployment) (err error) {
	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()
	if err = s.persistent.UpdateDeployment(deployment); err != nil {
		return
	}
	s.upsertDeployment(deployment)
	return
}

func (s *deploymentStore) upsertDeployment(deployment *v1.Deployment) {
	s.deployments[deployment.Id] = deployment
}

func (s *deploymentStore) RemoveDeployment(id string) (err error) {
	s.deploymentsMutex.Lock()
	defer s.deploymentsMutex.Unlock()
	if err = s.persistent.RemoveDeployment(id); err != nil {
		return
	}
	delete(s.deployments, id)
	return
}
