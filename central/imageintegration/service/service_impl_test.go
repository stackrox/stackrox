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
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	assert.Equal(t, err, status.Error(codes.InvalidArgument, "the request doesn't have a valid integration config type"))

	dockerConfig := &storage.DockerConfig{
		Endpoint: "endpoint",
		Username: "username",
		Password: "password",
	}
	dockerConfigScrubbed := dockerConfig.Clone()
	secrets.ScrubSecretsFromStructWithReplacement(dockerConfigScrubbed, secrets.ScrubReplacementStr)
	dockerImageIntegrationConfig := &storage.ImageIntegration{
		Id:                  "id2",
		Name:                "name2",
		Categories:          []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		SkipTestIntegration: true,
	}

	dockerImageIntegrationConfigStored := dockerImageIntegrationConfig.Clone()
	dockerImageIntegrationConfigStored.IntegrationConfig = &storage.ImageIntegration_Docker{Docker: dockerConfig.Clone()}

	integrationDatastore.EXPECT().GetImageIntegration(gomock.Any(),
		dockerImageIntegrationConfig.GetId()).Return(dockerImageIntegrationConfigStored, true, nil).AnyTimes()

	dockerImageIntegrationConfigScrubbed := dockerImageIntegrationConfig.Clone()
	dockerImageIntegrationConfigScrubbed.IntegrationConfig = &storage.ImageIntegration_Docker{Docker: dockerConfigScrubbed}
	requestWithADockerConfig := &v1.UpdateImageIntegrationRequest{
		Config:         dockerImageIntegrationConfigScrubbed,
		UpdatePassword: false,
	}

	storedConfig, exists, err := s.datastore.GetImageIntegration(textCtx,
		requestWithADockerConfig.GetConfig().GetId())
	assert.NoError(t, err)
	assert.True(t, exists)

	// Ensure successfully pulled credentials from storedConfig
	assert.NotEqual(t, dockerConfig, requestWithADockerConfig.GetConfig().GetDocker())
	err = s.reconcileImageIntegrationWithExisting(requestWithADockerConfig.GetConfig(), storedConfig)
	assert.NoError(t, err)
	assert.Equal(t, dockerConfig, requestWithADockerConfig.GetConfig().GetDocker())

	//Test case: config request with a different endpoint
	dockerConfigDiffEndpoint := dockerConfig.Clone()
	dockerConfigDiffEndpoint.Endpoint = "endpointDiff"
	secrets.ScrubSecretsFromStructWithReplacement(dockerConfigDiffEndpoint, secrets.ScrubReplacementStr)
	dockerImageIntegrationConfigDiffEndpoint := dockerImageIntegrationConfig.Clone()
	dockerImageIntegrationConfigDiffEndpoint.IntegrationConfig = &storage.ImageIntegration_Docker{Docker: dockerConfigDiffEndpoint}
	requestWithDifferentEndpoint := &v1.UpdateImageIntegrationRequest{
		Config:         dockerImageIntegrationConfigDiffEndpoint,
		UpdatePassword: false,
	}

	storedConfig, exists, err = s.datastore.GetImageIntegration(textCtx, requestWithDifferentEndpoint.GetConfig().GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	err = s.reconcileImageIntegrationWithExisting(requestWithDifferentEndpoint.GetConfig(), storedConfig)
	assert.Error(t, err)
	assert.EqualError(t, err, "credentials required to update field 'ImageIntegration.ImageIntegration_Docker.DockerConfig.Endpoint'")

	//Test case: config request with a different username
	dockerConfigDiffUsername := dockerConfig.Clone()
	dockerConfigDiffUsername.Username = "usernameDiff"
	secrets.ScrubSecretsFromStructWithReplacement(dockerConfigDiffUsername, secrets.ScrubReplacementStr)
	dockerImageIntegrationConfigDiffUsername := dockerImageIntegrationConfig.Clone()
	dockerImageIntegrationConfigDiffUsername.IntegrationConfig = &storage.ImageIntegration_Docker{Docker: dockerConfigDiffUsername}
	requestWithDifferentUsername := &v1.UpdateImageIntegrationRequest{
		Config:         dockerImageIntegrationConfigDiffUsername,
		UpdatePassword: false,
	}
	storedConfig, exists, err = s.datastore.GetImageIntegration(textCtx, requestWithDifferentEndpoint.GetConfig().GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	err = s.reconcileImageIntegrationWithExisting(requestWithDifferentUsername.GetConfig(), storedConfig)
	assert.Error(t, err)
	assert.EqualError(t, err, "credentials required to update field 'ImageIntegration.ImageIntegration_Docker.DockerConfig.Username'")
}
