package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	integrationMocks "github.com/stackrox/rox/central/imageintegration/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestValidateIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	textCtx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	clusterDatastore := clusterMocks.NewMockDataStore(ctrl)
	clusterDatastore.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{}, nil).AnyTimes()

	integrationDatastore := integrationMocks.NewMockDataStore(ctrl)

	s := &serviceImpl{clusterDatastore: clusterDatastore, datastore: integrationDatastore}

	// Test name and categories validation
	assert.Error(t, s.validateIntegration(textCtx, &storage.ImageIntegration{}))

	assert.Error(t, s.validateIntegration(textCtx, &storage.ImageIntegration{
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	// Test should be successful
	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), &v1.GetImageIntegrationsRequest{Name: "name"}).Return([]*storage.ImageIntegration{}, nil)
	assert.NoError(t, s.validateIntegration(textCtx, &storage.ImageIntegration{
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	// Test name scenarios

	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), &v1.GetImageIntegrationsRequest{Name: "name"}).Return([]*storage.ImageIntegration{{Id: "id", Name: "name"}}, nil).AnyTimes()
	// Duplicate name with different ID should fail
	assert.Error(t, s.validateIntegration(textCtx, &storage.ImageIntegration{
		Id:         "diff",
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	// Duplicate name with same ID should succeed
	assert.NoError(t, s.validateIntegration(textCtx, &storage.ImageIntegration{
		Id:         "id",
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	request := &v1.UpdateImageIntegrationRequest{
		Config: &storage.ImageIntegration{
			Id:                  "id",
			Name:                "name",
			Categories:          []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
			IntegrationConfig:   nil,
			SkipTestIntegration: true,
		},
		UpdatePassword: false,
	}

	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), &v1.GetImageIntegrationsRequest{Name: "name"}).Return([]*storage.ImageIntegration{
		{
			Id:         "id",
			Name:       "name",
			Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		}}, nil).AnyTimes()
	integrationDatastore.EXPECT().GetImageIntegration(gomock.Any(), "id").Return(&storage.ImageIntegration{
		Id:         "id",
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}, true, nil).AnyTimes()

	_, err := s.TestUpdatedImageIntegration(textCtx, request)
	assert.Error(t, err)
	assert.EqualErrorf(t, err, "the request doesn't have a valid integration config type", "formatted")

	requestWithADockerConfig := &v1.UpdateImageIntegrationRequest{
		Config: &storage.ImageIntegration{
			Id:         "id2",
			Name:       "name2",
			Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
			IntegrationConfig: &storage.ImageIntegration_Docker{Docker: &storage.DockerConfig{
				Endpoint: "endpoint",
				Username: "username",
				Password: "password",
			}},
			SkipTestIntegration: true,
		},
		UpdatePassword: false,
	}
	integrationDatastore.EXPECT().GetImageIntegration(gomock.Any(), "id2").Return(&storage.ImageIntegration{
		Id:         "id2",
		Name:       "name2",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{Docker: &storage.DockerConfig{
			Endpoint: "endpoint",
			Username: "username",
			Password: "******",
		}},
		SkipTestIntegration: true,
	}, true, nil).AnyTimes()

	maskedIntegrationConfig := &storage.ImageIntegration{
		Id:         "id2",
		Name:       "name2",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{Docker: &storage.DockerConfig{
			Endpoint: "endpoint",
			Username: "username",
			Password: "******",
		}},
		SkipTestIntegration: true,
	}
	tempConfig := &storage.ImageIntegration{
		Id:         "id2",
		Name:       "name2",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{Docker: &storage.DockerConfig{
			Endpoint: "endpoint",
			Username: "username",
			Password: "password",
		}},
		SkipTestIntegration: true,
	}
	storedConfig, err := s.GetImageIntegration(textCtx, &v1.ResourceByID{
		Id: requestWithADockerConfig.GetConfig().GetId(),
	})
	assert.Equal(t, storedConfig, maskedIntegrationConfig)
	assert.NoError(t, err)
	err = s.pullDataFromStoredConfig(requestWithADockerConfig.GetConfig(), tempConfig)
	// Ensure successfully pulled credentials from storedConfig
	assert.Equal(t, requestWithADockerConfig.GetConfig().GetDocker(), tempConfig.GetDocker())
	assert.NoError(t, err)

	//Test case: config request with a different endpoint
	requestWithDifferentEndpoint := &v1.UpdateImageIntegrationRequest{
		Config: &storage.ImageIntegration{
			Id:         "id2",
			Name:       "name2",
			Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
			IntegrationConfig: &storage.ImageIntegration_Docker{Docker: &storage.DockerConfig{
				Endpoint: "endpointDiff",
				Username: "username",
			}},
			SkipTestIntegration: true,
		},
		UpdatePassword: false,
	}
	integrationDatastore.EXPECT().GetImageIntegration(gomock.Any(), "id2").Return(&storage.ImageIntegration{
		Id:         "id2",
		Name:       "name2",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{Docker: &storage.DockerConfig{
			Endpoint: "endpointDiff",
			Username: "username",
			Password: "******",
		}},
		SkipTestIntegration: true,
	}, true, nil).AnyTimes()

	storedConfig, err = s.GetImageIntegration(textCtx, &v1.ResourceByID{
		Id: requestWithDifferentEndpoint.GetConfig().GetId(),
	})
	assert.NoError(t, err)
	err = s.pullDataFromStoredConfig(requestWithDifferentEndpoint.GetConfig(), storedConfig)
	assert.Error(t, err)
	assert.EqualErrorf(t, err, "must explicitly set password when changing username/endpoint", "formatted")

	//Test case: config request with a different username
	requestWithDifferentUsername := &v1.UpdateImageIntegrationRequest{
		Config: &storage.ImageIntegration{
			Id:         "id2",
			Name:       "name2",
			Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
			IntegrationConfig: &storage.ImageIntegration_Docker{Docker: &storage.DockerConfig{
				Endpoint: "endpoint",
				Username: "usernameDiff",
			}},
			SkipTestIntegration: true,
		},
		UpdatePassword: false,
	}
	integrationDatastore.EXPECT().GetImageIntegration(gomock.Any(), "id2").Return(&storage.ImageIntegration{
		Id:         "id2",
		Name:       "name2",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{Docker: &storage.DockerConfig{
			Endpoint: "endpoint",
			Username: "usernameDiff",
			Password: "******",
		}},
		SkipTestIntegration: true,
	}, true, nil).AnyTimes()

	storedConfig, err = s.GetImageIntegration(textCtx, &v1.ResourceByID{
		Id: requestWithDifferentUsername.GetConfig().GetId(),
	})
	assert.NoError(t, err)
	err = s.pullDataFromStoredConfig(requestWithDifferentUsername.GetConfig(), storedConfig)
	assert.Error(t, err)
	assert.EqualErrorf(t, err, "must explicitly set password when changing username/endpoint", "formatted")
}
