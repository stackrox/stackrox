package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	clusterMocks "github.com/stackrox/stackrox/central/cluster/datastore/mocks"
	"github.com/stackrox/stackrox/central/enrichment/mocks"
	integrationMocks "github.com/stackrox/stackrox/central/imageintegration/datastore/mocks"
	loopMocks "github.com/stackrox/stackrox/central/reprocessor/mocks"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errox"
	nodeMocks "github.com/stackrox/stackrox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/stackrox/pkg/sac"
	scannerMocks "github.com/stackrox/stackrox/pkg/scanners/mocks"
	"github.com/stackrox/stackrox/pkg/scanners/types"
	"github.com/stackrox/stackrox/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

var _ types.Scanner = (*fakeScanner)(nil)

type fakeScanner struct{}

func (*fakeScanner) GetScan(*storage.Image) (*storage.ImageScan, error) {
	panic("implement me")
}

func (*fakeScanner) Match(*storage.ImageName) bool {
	panic("implement me")
}

func (*fakeScanner) Test() error {
	return nil
}

func (*fakeScanner) Name() string {
	panic("implement me")
}

func (*fakeScanner) Type() string {
	return "type"
}

func (*fakeScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(10)
}

func (*fakeScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return &v1.VulnDefinitionsInfo{}, nil
}

var _ types.NodeScanner = (*fakeNodeScanner)(nil)

type fakeNodeScanner struct{}

func (*fakeNodeScanner) Name() string {
	panic("implement me")
}

func (*fakeNodeScanner) Type() string {
	return "type"
}

func (*fakeNodeScanner) GetNodeScan(*storage.Node) (*storage.NodeScan, error) {
	panic("implement me")
}

func (*fakeNodeScanner) TestNodeScanner() error {
	return nil
}

func (*fakeNodeScanner) MaxConcurrentNodeScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(10)
}

var (
	_ types.ImageScannerWithDataSource = (*fakeImageAndNodeScanner)(nil)
	_ types.NodeScannerWithDataSource  = (*fakeImageAndNodeScanner)(nil)
)

type fakeImageAndNodeScanner struct {
	scanner     types.Scanner
	nodeScanner types.NodeScanner
}

func newFakeImageAndNodeScanner() *fakeImageAndNodeScanner {
	return &fakeImageAndNodeScanner{
		scanner:     &fakeScanner{},
		nodeScanner: &fakeNodeScanner{},
	}
}

func (f *fakeImageAndNodeScanner) GetScanner() types.Scanner {
	return f.scanner
}

func (f *fakeImageAndNodeScanner) GetNodeScanner() types.NodeScanner {
	return f.nodeScanner
}

func (*fakeImageAndNodeScanner) DataSource() *storage.DataSource {
	return nil
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
	assert.EqualError(t, err, errors.Wrap(errox.InvalidArgs, "the request doesn't have a valid integration config type").Error())

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
	scannerFactory.EXPECT().CreateScanner(clairifyIntegrationConfig).Return(newFakeImageAndNodeScanner(), nil).Times(1)
	nodeEnricher.EXPECT().CreateNodeScanner(clairifyNodeIntegrationConfig).Return(newFakeImageAndNodeScanner(), nil).Times(1)
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
