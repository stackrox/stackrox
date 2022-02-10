package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/enrichment/mocks"
	integrationMocks "github.com/stackrox/rox/central/imageintegration/datastore/mocks"
	loopMocks "github.com/stackrox/rox/central/reprocessor/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	nodeMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/sac"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

type fakeScanner struct{}

func (*fakeScanner) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	panic("implement me")
}

func (*fakeScanner) GetNodeScan(node *storage.Node) (*storage.NodeScan, error) {
	panic("implement me")
}

func (*fakeScanner) Match(image *storage.ImageName) bool {
	panic("implement me")
}

func (*fakeScanner) Test() error {
	return nil
}

func (*fakeScanner) TestNodeScanner() error {
	return nil
}

func (*fakeScanner) Name() string {
	panic("implement me")
}

func (f *fakeScanner) Type() string {
	return "type"
}

func (f *fakeScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(10)
}

func (f *fakeScanner) MaxConcurrentNodeScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(10)
}

func (f *fakeScanner) DataSource() *storage.DataSource {
	return nil
}

func (f *fakeScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return &v1.VulnDefinitionsInfo{}, nil
}

func TestValidateIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCtx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	clusterDatastore := clusterMocks.NewMockDataStore(ctrl)
	clusterDatastore.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{}, nil).AnyTimes()

	integrationDatastore := integrationMocks.NewMockDataStore(ctrl)

	s := &serviceImpl{clusterDatastore: clusterDatastore, datastore: integrationDatastore}

	// Test name and categories validation
	assert.Error(t, s.validateIntegration(testCtx, &storage.ImageIntegration{}))

	assert.Error(t, s.validateIntegration(testCtx, &storage.ImageIntegration{
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	// Test should be successful
	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), &v1.GetImageIntegrationsRequest{Name: "name"}).Return([]*storage.ImageIntegration{}, nil)
	assert.NoError(t, s.validateIntegration(testCtx, &storage.ImageIntegration{
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	// Test name scenarios

	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), &v1.GetImageIntegrationsRequest{Name: "name"}).Return([]*storage.ImageIntegration{{Id: "id", Name: "name"}}, nil).AnyTimes()
	// Duplicate name with different ID should fail
	assert.Error(t, s.validateIntegration(testCtx, &storage.ImageIntegration{
		Id:         "diff",
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
	}))

	// Duplicate name with same ID should succeed
	assert.NoError(t, s.validateIntegration(testCtx, &storage.ImageIntegration{
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

	_, err := s.TestUpdatedImageIntegration(testCtx, request)
	assert.Error(t, err)
	assert.EqualError(t, err, errors.Wrap(errorhelpers.ErrInvalidArgs, "the request doesn't have a valid integration config type").Error())

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

	storedConfig, exists, err := s.datastore.GetImageIntegration(testCtx,
		requestWithADockerConfig.GetConfig().GetId())
	assert.NoError(t, err)
	assert.True(t, exists)

	// Ensure successfully pulled credentials from storedConfig
	assert.NotEqual(t, dockerConfig, requestWithADockerConfig.GetConfig().GetDocker())
	err = s.reconcileImageIntegrationWithExisting(requestWithADockerConfig.GetConfig(), storedConfig)
	assert.NoError(t, err)
	assert.Equal(t, dockerConfig, requestWithADockerConfig.GetConfig().GetDocker())

	// Test case: config request with a different endpoint
	dockerConfigDiffEndpoint := dockerConfig.Clone()
	dockerConfigDiffEndpoint.Endpoint = "endpointDiff"
	secrets.ScrubSecretsFromStructWithReplacement(dockerConfigDiffEndpoint, secrets.ScrubReplacementStr)
	dockerImageIntegrationConfigDiffEndpoint := dockerImageIntegrationConfig.Clone()
	dockerImageIntegrationConfigDiffEndpoint.IntegrationConfig = &storage.ImageIntegration_Docker{Docker: dockerConfigDiffEndpoint}
	requestWithDifferentEndpoint := &v1.UpdateImageIntegrationRequest{
		Config:         dockerImageIntegrationConfigDiffEndpoint,
		UpdatePassword: false,
	}

	storedConfig, exists, err = s.datastore.GetImageIntegration(testCtx, requestWithDifferentEndpoint.GetConfig().GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	err = s.reconcileImageIntegrationWithExisting(requestWithDifferentEndpoint.GetConfig(), storedConfig)
	assert.Error(t, err)
	assert.EqualError(t, err, "credentials required to update field 'ImageIntegration.ImageIntegration_Docker.DockerConfig.Endpoint'")

	// Test case: config request with a different username
	dockerConfigDiffUsername := dockerConfig.Clone()
	dockerConfigDiffUsername.Username = "usernameDiff"
	secrets.ScrubSecretsFromStructWithReplacement(dockerConfigDiffUsername, secrets.ScrubReplacementStr)
	dockerImageIntegrationConfigDiffUsername := dockerImageIntegrationConfig.Clone()
	dockerImageIntegrationConfigDiffUsername.IntegrationConfig = &storage.ImageIntegration_Docker{Docker: dockerConfigDiffUsername}
	requestWithDifferentUsername := &v1.UpdateImageIntegrationRequest{
		Config:         dockerImageIntegrationConfigDiffUsername,
		UpdatePassword: false,
	}
	storedConfig, exists, err = s.datastore.GetImageIntegration(testCtx, requestWithDifferentEndpoint.GetConfig().GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	err = s.reconcileImageIntegrationWithExisting(requestWithDifferentUsername.GetConfig(), storedConfig)
	assert.Error(t, err)
	assert.EqualError(t, err, "credentials required to update field 'ImageIntegration.ImageIntegration_Docker.DockerConfig.Username'")
}

func TestValidateNodeIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCtx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	clusterDatastore := clusterMocks.NewMockDataStore(ctrl)
	clusterDatastore.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{}, nil).AnyTimes()

	integrationDatastore := integrationMocks.NewMockDataStore(ctrl)

	scannerFactory := scannerMocks.NewMockFactory(ctrl)
	nodeEnricher := nodeMocks.NewMockNodeEnricher(ctrl)
	reprocessorLoop := loopMocks.NewMockLoop(ctrl)
	integrationManager := mocks.NewMockManager(ctrl)

	s := &serviceImpl{
		clusterDatastore:   clusterDatastore,
		datastore:          integrationDatastore,
		nodeEnricher:       nodeEnricher,
		integrationManager: integrationManager,
		scannerFactory:     scannerFactory,
		reprocessorLoop:    reprocessorLoop,
	}

	// Test should be successful
	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), &v1.GetImageIntegrationsRequest{Name: "name"}).Return([]*storage.ImageIntegration{}, nil)
	assert.NoError(t, s.validateIntegration(testCtx, &storage.ImageIntegration{
		Name:       "name",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_NODE_SCANNER},
	}))

	clairifyConfig := &storage.ClairifyConfig{
		Endpoint:           "https://scanner.stackrox:8080",
		NumConcurrentScans: 30,
	}
	clairifyIntegrationConfig := &storage.ImageIntegration{
		Id:                  "id",
		Name:                "name",
		IntegrationConfig:   &storage.ImageIntegration_Clairify{Clairify: clairifyConfig},
		Categories:          []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER, storage.ImageIntegrationCategory_NODE_SCANNER},
		SkipTestIntegration: true,
	}
	clairifyNodeIntegrationConfig := &storage.NodeIntegration{
		Id:                "id",
		Name:              "name",
		IntegrationConfig: &storage.NodeIntegration_Clairify{Clairify: clairifyConfig},
	}

	clairifyIntegrationConfigStored := clairifyIntegrationConfig.Clone()
	clairifyIntegrationConfigStored.IntegrationConfig = &storage.ImageIntegration_Clairify{Clairify: clairifyConfig.Clone()}

	// Test integration.
	integrationDatastore.EXPECT().GetImageIntegrations(
		gomock.Any(),
		&v1.GetImageIntegrationsRequest{Name: "name"},
	).Return([]*storage.ImageIntegration{clairifyIntegrationConfigStored}, nil).AnyTimes()
	scannerFactory.EXPECT().CreateScanner(clairifyIntegrationConfig).Return(&fakeScanner{}, nil).Times(1)
	nodeEnricher.EXPECT().CreateNodeScanner(clairifyNodeIntegrationConfig).Return(&fakeScanner{}, nil).Times(1)
	_, err := s.TestImageIntegration(testCtx, clairifyIntegrationConfig)
	assert.NoError(t, err)

	// Put.
	integrationDatastore.EXPECT().GetImageIntegrations(
		gomock.Any(),
		&v1.GetImageIntegrationsRequest{Name: "name"},
	).Return([]*storage.ImageIntegration{clairifyIntegrationConfigStored}, nil).AnyTimes()
	integrationDatastore.EXPECT().UpdateImageIntegration(gomock.Any(), clairifyIntegrationConfig).Return(nil).Times(1)
	integrationManager.EXPECT().Upsert(clairifyIntegrationConfig).Return(nil)
	reprocessorLoop.EXPECT().ShortCircuit().Times(1)
	_, err = s.PutImageIntegration(testCtx, clairifyIntegrationConfig)
	assert.NoError(t, err)
}
