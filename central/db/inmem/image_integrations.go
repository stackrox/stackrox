package inmem

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/set"
)

type imageIntegrationStore struct {
	db.ImageIntegrationStorage
}

func newImageIntegrationStore(persistent db.ImageIntegrationStorage) *imageIntegrationStore {
	return &imageIntegrationStore{
		ImageIntegrationStorage: persistent,
	}
}

func (s *imageIntegrationStore) GetImageIntegrations(request *v1.GetImageIntegrationsRequest) ([]*v1.ImageIntegration, error) {
	integrations, err := s.ImageIntegrationStorage.GetImageIntegrations(request)
	if err != nil {
		return nil, err
	}
	integrationSlice := integrations[:0]
	for _, integration := range integrations {
		clusterSet := set.NewSetFromStringSlice(integration.GetClusters())
		if len(request.GetCluster()) != 0 && !clusterSet.Contains(request.GetCluster()) {
			continue
		}
		integrationSlice = append(integrationSlice, integration)
	}
	return integrationSlice, nil
}
