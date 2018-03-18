package inmem

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
)

type deploymentStore struct {
	db.DeploymentStorage
}

func newDeploymentStore(persistent db.DeploymentStorage) *deploymentStore {
	return &deploymentStore{
		DeploymentStorage: persistent,
	}
}

func (s *deploymentStore) GetDeployments(request *v1.GetDeploymentsRequest) ([]*v1.Deployment, error) {
	deployments, err := s.DeploymentStorage.GetDeployments(request)
	if err != nil {
		return nil, err
	}

	nameSet := stringWrap(request.GetName()).asSet()
	typeSet := stringWrap(request.GetType()).asSet()
	imageShaSet := stringWrap(request.GetImageSha()).asSet()

	filteredDeployments := deployments[:0]
	for _, d := range deployments {
		if _, ok := nameSet[d.GetName()]; len(nameSet) > 0 && !ok {
			continue
		}

		if _, ok := typeSet[d.GetType()]; len(typeSet) > 0 && !ok {
			continue
		}

		if len(imageShaSet) > 0 && !s.matchImageSha(imageShaSet, d.GetContainers()) {

			continue
		}

		filteredDeployments = append(filteredDeployments, d)
	}

	sort.SliceStable(filteredDeployments, func(i, j int) bool { return filteredDeployments[i].Id < filteredDeployments[j].Id })
	return filteredDeployments, nil
}

func (s *deploymentStore) matchImageSha(imageShaSet map[string]struct{}, containers []*v1.Container) bool {
	for _, c := range containers {
		digest := images.NewDigest(c.GetImage().GetName().GetSha()).Digest()
		if _, ok := imageShaSet[digest]; !ok {
			return false
		}
	}

	return true
}
