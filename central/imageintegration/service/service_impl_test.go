package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/enrichment/mocks"
	enrichMocks "github.com/stackrox/rox/central/enrichment/mocks"
	integrationMocks "github.com/stackrox/rox/central/imageintegration/datastore/mocks"
	loopMocks "github.com/stackrox/rox/central/reprocessor/mocks"
	"github.com/stackrox/rox/central/sensor/service/connection"
	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	nodeMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/sac"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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

func (*fakeNodeScanner) GetNodeInventoryScan(_ *storage.Node, _ *storage.NodeInventory) (*storage.NodeScan, error) {
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

func TestBroadcast(t *testing.T) {
	var connMgr *connMocks.MockManager
	var conn *connMocks.MockSensorConnection
	var s *serviceImpl
	var msg *central.MsgToSensor

	setup := func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connMgr = connMocks.NewMockManager(ctrl)
		conn = connMocks.NewMockSensorConnection(ctrl)
		s = &serviceImpl{connManager: connMgr}
		msg = &central.MsgToSensor{}
	}

	ii := &storage.ImageIntegration{}

	t.Run("success", func(t *testing.T) {
		setup(t)
		conn.EXPECT().ClusterID()
		conn.EXPECT().HasCapability(gomock.Any()).Return(true)
		conn.EXPECT().InjectMessage(gomock.Any(), msg)
		connMgr.EXPECT().GetActiveConnections().Return([]connection.SensorConnection{conn})

		s.broadcast(context.Background(), "action", ii, msg)
	})

	t.Run("noop on no conns", func(t *testing.T) {
		setup(t)
		connMgr.EXPECT().GetActiveConnections().Return(nil)

		s.broadcast(context.Background(), "action", ii, msg)
	})

	t.Run("noop on conns not valid", func(t *testing.T) {
		setup(t)
		conn.EXPECT().HasCapability(gomock.Any()).Return(false)
		connMgr.EXPECT().GetActiveConnections().Return([]connection.SensorConnection{conn})

		s.broadcast(context.Background(), "action", ii, msg)
	})

	t.Run("noop on inject err", func(t *testing.T) {
		setup(t)
		conn.EXPECT().ClusterID()
		conn.EXPECT().HasCapability(gomock.Any()).Return(true)
		conn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).Return(errors.New("broken"))
		connMgr.EXPECT().GetActiveConnections().Return([]connection.SensorConnection{conn})

		s.broadcast(context.Background(), "action", ii, msg)
	})
}

func TestBroadcastOnDelete(t *testing.T) {
	var s *serviceImpl
	var iiDS *integrationMocks.MockDataStore
	var intMgr *enrichMocks.MockManager
	var connMgr *connMocks.MockManager
	var conn *connMocks.MockSensorConnection

	ii := &storage.ImageIntegration{Id: "id", Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY}}

	setup := func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connMgr = connMocks.NewMockManager(ctrl)
		conn = connMocks.NewMockSensorConnection(ctrl)
		iiDS = integrationMocks.NewMockDataStore(ctrl)
		intMgr = enrichMocks.NewMockManager(ctrl)
		s = &serviceImpl{connManager: connMgr, datastore: iiDS, integrationManager: intMgr}
	}

	t.Run("success", func(t *testing.T) {
		setup(t)
		conn.EXPECT().ClusterID().Return(ii.GetId())
		conn.EXPECT().HasCapability(gomock.Any()).Return(true)
		conn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).Return(nil)

		connMgr.EXPECT().GetActiveConnections().Return([]connection.SensorConnection{conn})

		intMgr.EXPECT().Remove(gomock.Any()).Return(nil)

		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(ii, true, nil)
		iiDS.EXPECT().RemoveImageIntegration(gomock.Any(), gomock.Any()).Return(nil)

		_, err := s.DeleteImageIntegration(context.Background(), &v1.ResourceByID{Id: "id"})
		assert.NoError(t, err)
	})

	t.Run("no broadcast on no exist", func(t *testing.T) {
		setup(t)

		intMgr.EXPECT().Remove(gomock.Any()).Return(nil)

		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(nil, false, nil)
		iiDS.EXPECT().RemoveImageIntegration(gomock.Any(), gomock.Any()).Return(nil)

		_, err := s.DeleteImageIntegration(context.Background(), &v1.ResourceByID{Id: "id"})
		assert.NoError(t, err)
	})

	t.Run("err on failure to get existing", func(t *testing.T) {
		setup(t)

		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(nil, false, errors.New("broken"))

		_, err := s.DeleteImageIntegration(context.Background(), &v1.ResourceByID{Id: "id"})
		assert.Error(t, err)
	})
}

func TestScannerV4Restrictions(t *testing.T) {
	ii := &storage.ImageIntegration{
		Name: "fake",
		Id:   "fake",
		Type: types.ScannerV4,
		Categories: []storage.ImageIntegrationCategory{
			storage.ImageIntegrationCategory_SCANNER,
		},
		IntegrationConfig: &storage.ImageIntegration_ScannerV4{},
	}

	t.Run("prevent scannerv4 create", func(t *testing.T) {
		s := &serviceImpl{}

		iiNew := ii.Clone()
		iiNew.Id = ""

		_, err := s.PostImageIntegration(context.Background(), iiNew)
		assert.ErrorContains(t, err, "scanner V4")
	})

	t.Run("prevent scannerv4 delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		iiDS := integrationMocks.NewMockDataStore(ctrl)
		s := &serviceImpl{datastore: iiDS}

		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(ii, true, nil)

		_, err := s.DeleteImageIntegration(context.Background(), &v1.ResourceByID{Id: "fake"})
		assert.ErrorContains(t, err, "scanner V4")
	})

	t.Run("prevent type change FROM scannerv4", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		iiDS := integrationMocks.NewMockDataStore(ctrl)
		s := &serviceImpl{datastore: iiDS}

		iiOld := ii.Clone()
		iiOld.Type = types.Clairify

		iiDS.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).Return([]*storage.ImageIntegration{iiOld}, nil)
		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(iiOld, true, nil)

		_, err := s.UpdateImageIntegration(context.Background(), &v1.UpdateImageIntegrationRequest{Config: ii})
		assert.ErrorContains(t, err, "scanner V4")
	})

	t.Run("prevent type change TO scannerv4", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		iiDS := integrationMocks.NewMockDataStore(ctrl)
		s := &serviceImpl{datastore: iiDS}

		iiNew := ii.Clone()
		iiNew.Type = types.Clairify
		iiNew.IntegrationConfig = &storage.ImageIntegration_Clairify{}

		iiDS.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).Return([]*storage.ImageIntegration{ii}, nil)
		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(ii, true, nil)

		_, err := s.UpdateImageIntegration(context.Background(), &v1.UpdateImageIntegrationRequest{Config: iiNew})
		assert.ErrorContains(t, err, "scanner V4")
	})
}
