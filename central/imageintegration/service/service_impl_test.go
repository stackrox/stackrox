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
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	nodeMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	"github.com/stackrox/rox/pkg/scanners/types"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
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

func (*fakeNodeScanner) GetNodeInventoryScan(_ *storage.Node, _ *storage.NodeInventory, _ *v4.IndexReport) (*storage.NodeScan, error) {
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

	ii := &storage.ImageIntegration{}
	ii.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})
	assert.Error(t, s.validateIntegration(testCtx, ii))

	// Test should be successful
	giir := &v1.GetImageIntegrationsRequest{}
	giir.SetName("name")
	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), giir).Return([]*storage.ImageIntegration{}, nil)
	ii2 := &storage.ImageIntegration{}
	ii2.SetName("name")
	ii2.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})
	assert.NoError(t, s.validateIntegration(testCtx, ii2))

	// Test name scenarios

	giir2 := &v1.GetImageIntegrationsRequest{}
	giir2.SetName("name")
	ii3 := &storage.ImageIntegration{}
	ii3.SetId("id")
	ii3.SetName("name")
	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), giir2).Return([]*storage.ImageIntegration{ii3}, nil).AnyTimes()
	// Duplicate name with different ID should fail
	ii4 := &storage.ImageIntegration{}
	ii4.SetId("diff")
	ii4.SetName("name")
	ii4.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})
	assert.Error(t, s.validateIntegration(testCtx, ii4))

	// Duplicate name with same ID should succeed
	ii5 := &storage.ImageIntegration{}
	ii5.SetId("id")
	ii5.SetName("name")
	ii5.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})
	assert.NoError(t, s.validateIntegration(testCtx, ii5))

	ii6 := &storage.ImageIntegration{}
	ii6.SetId("id")
	ii6.SetName("name")
	ii6.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})
	ii6.ClearIntegrationConfig()
	ii6.SetSkipTestIntegration(true)
	request := &v1.UpdateImageIntegrationRequest{}
	request.SetConfig(ii6)
	request.SetUpdatePassword(false)

	giir3 := &v1.GetImageIntegrationsRequest{}
	giir3.SetName("name")
	ii7 := &storage.ImageIntegration{}
	ii7.SetId("id")
	ii7.SetName("name")
	ii7.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})
	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), giir3).Return([]*storage.ImageIntegration{
		ii7}, nil).AnyTimes()
	ii8 := &storage.ImageIntegration{}
	ii8.SetId("id")
	ii8.SetName("name")
	ii8.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})
	integrationDatastore.EXPECT().GetImageIntegration(gomock.Any(), "id").Return(ii8, true, nil).AnyTimes()

	_, err := s.TestUpdatedImageIntegration(testCtx, request)
	assert.Error(t, err)
	assert.EqualError(t, err, errors.Wrap(errox.InvalidArgs, "the request doesn't have a valid integration config type").Error())

	dockerConfig := &storage.DockerConfig{}
	dockerConfig.SetEndpoint("endpoint")
	dockerConfig.SetUsername("username")
	dockerConfig.SetPassword("password")
	dockerConfigScrubbed := dockerConfig.CloneVT()
	secrets.ScrubSecretsFromStructWithReplacement(dockerConfigScrubbed, secrets.ScrubReplacementStr)
	dockerImageIntegrationConfig := &storage.ImageIntegration{}
	dockerImageIntegrationConfig.SetId("id2")
	dockerImageIntegrationConfig.SetName("name2")
	dockerImageIntegrationConfig.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})
	dockerImageIntegrationConfig.SetSkipTestIntegration(true)

	dockerImageIntegrationConfigStored := dockerImageIntegrationConfig.CloneVT()
	dockerImageIntegrationConfigStored.SetDocker(proto.ValueOrDefault(dockerConfig.CloneVT()))

	integrationDatastore.EXPECT().GetImageIntegration(gomock.Any(),
		dockerImageIntegrationConfig.GetId()).Return(dockerImageIntegrationConfigStored, true, nil).AnyTimes()

	dockerImageIntegrationConfigScrubbed := dockerImageIntegrationConfig.CloneVT()
	dockerImageIntegrationConfigScrubbed.SetDocker(proto.ValueOrDefault(dockerConfigScrubbed))
	requestWithADockerConfig := &v1.UpdateImageIntegrationRequest{}
	requestWithADockerConfig.SetConfig(dockerImageIntegrationConfigScrubbed)
	requestWithADockerConfig.SetUpdatePassword(false)

	storedConfig, exists, err := s.datastore.GetImageIntegration(testCtx,
		requestWithADockerConfig.GetConfig().GetId())
	assert.NoError(t, err)
	assert.True(t, exists)

	// Ensure successfully pulled credentials from storedConfig
	protoassert.NotEqual(t, dockerConfig, requestWithADockerConfig.GetConfig().GetDocker())
	err = s.reconcileImageIntegrationWithExisting(requestWithADockerConfig.GetConfig(), storedConfig)
	assert.NoError(t, err)
	protoassert.Equal(t, dockerConfig, requestWithADockerConfig.GetConfig().GetDocker())

	// Test case: config request with a different endpoint
	dockerConfigDiffEndpoint := dockerConfig.CloneVT()
	dockerConfigDiffEndpoint.SetEndpoint("endpointDiff")
	secrets.ScrubSecretsFromStructWithReplacement(dockerConfigDiffEndpoint, secrets.ScrubReplacementStr)
	dockerImageIntegrationConfigDiffEndpoint := dockerImageIntegrationConfig.CloneVT()
	dockerImageIntegrationConfigDiffEndpoint.SetDocker(proto.ValueOrDefault(dockerConfigDiffEndpoint))
	requestWithDifferentEndpoint := &v1.UpdateImageIntegrationRequest{}
	requestWithDifferentEndpoint.SetConfig(dockerImageIntegrationConfigDiffEndpoint)
	requestWithDifferentEndpoint.SetUpdatePassword(false)

	storedConfig, exists, err = s.datastore.GetImageIntegration(testCtx, requestWithDifferentEndpoint.GetConfig().GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	err = s.reconcileImageIntegrationWithExisting(requestWithDifferentEndpoint.GetConfig(), storedConfig)
	assert.Error(t, err)
	assert.EqualError(t, err, "credentials required to update field 'ImageIntegration.ImageIntegration_Docker.DockerConfig.Endpoint'")

	// Test case: config request with a different username
	dockerConfigDiffUsername := dockerConfig.CloneVT()
	dockerConfigDiffUsername.SetUsername("usernameDiff")
	secrets.ScrubSecretsFromStructWithReplacement(dockerConfigDiffUsername, secrets.ScrubReplacementStr)
	dockerImageIntegrationConfigDiffUsername := dockerImageIntegrationConfig.CloneVT()
	dockerImageIntegrationConfigDiffUsername.SetDocker(proto.ValueOrDefault(dockerConfigDiffUsername))
	requestWithDifferentUsername := &v1.UpdateImageIntegrationRequest{}
	requestWithDifferentUsername.SetConfig(dockerImageIntegrationConfigDiffUsername)
	requestWithDifferentUsername.SetUpdatePassword(false)
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
	giir := &v1.GetImageIntegrationsRequest{}
	giir.SetName("name")
	integrationDatastore.EXPECT().GetImageIntegrations(gomock.Any(), giir).Return([]*storage.ImageIntegration{}, nil)
	ii := &storage.ImageIntegration{}
	ii.SetName("name")
	ii.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_NODE_SCANNER})
	assert.NoError(t, s.validateIntegration(testCtx, ii))

	clairifyConfig := &storage.ClairifyConfig{}
	clairifyConfig.SetEndpoint("https://scanner.stackrox:8080")
	clairifyConfig.SetNumConcurrentScans(30)
	clairifyIntegrationConfig := &storage.ImageIntegration{}
	clairifyIntegrationConfig.SetId("id")
	clairifyIntegrationConfig.SetName("name")
	clairifyIntegrationConfig.SetType(scannerTypes.Clairify)
	clairifyIntegrationConfig.SetClairify(proto.ValueOrDefault(clairifyConfig))
	clairifyIntegrationConfig.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER, storage.ImageIntegrationCategory_NODE_SCANNER})
	clairifyIntegrationConfig.SetSkipTestIntegration(true)
	clairifyNodeIntegrationConfig := &storage.NodeIntegration{}
	clairifyNodeIntegrationConfig.SetId("id")
	clairifyNodeIntegrationConfig.SetName("name")
	clairifyNodeIntegrationConfig.SetType(scannerTypes.Clairify)
	clairifyNodeIntegrationConfig.SetClairify(proto.ValueOrDefault(clairifyConfig))

	clairifyIntegrationConfigStored := clairifyIntegrationConfig.CloneVT()
	clairifyIntegrationConfigStored.SetClairify(proto.ValueOrDefault(clairifyConfig.CloneVT()))

	// Test integration.
	giir2 := &v1.GetImageIntegrationsRequest{}
	giir2.SetName("name")
	integrationDatastore.EXPECT().GetImageIntegrations(
		gomock.Any(), giir2).Return([]*storage.ImageIntegration{clairifyIntegrationConfigStored}, nil).AnyTimes()
	integrationDatastore.EXPECT().GetImageIntegration(gomock.Any(), clairifyIntegrationConfig.GetId()).Return(clairifyIntegrationConfigStored, true, nil).Times(1)
	scannerFactory.EXPECT().CreateScanner(clairifyIntegrationConfig).Return(newFakeImageAndNodeScanner(), nil).Times(1)
	nodeEnricher.EXPECT().CreateNodeScanner(clairifyNodeIntegrationConfig).Return(newFakeImageAndNodeScanner(), nil).Times(1)
	_, err := s.TestImageIntegration(testCtx, clairifyIntegrationConfig)
	assert.NoError(t, err)

	// Put.
	giir3 := &v1.GetImageIntegrationsRequest{}
	giir3.SetName("name")
	integrationDatastore.EXPECT().GetImageIntegrations(
		gomock.Any(), giir3).Return([]*storage.ImageIntegration{clairifyIntegrationConfigStored}, nil).AnyTimes()
	integrationDatastore.EXPECT().UpdateImageIntegration(gomock.Any(), clairifyIntegrationConfig).Return(nil).Times(1)
	integrationDatastore.EXPECT().GetImageIntegration(gomock.Any(), clairifyIntegrationConfig.GetId()).Return(clairifyIntegrationConfigStored, true, nil).Times(1)
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

	ii := &storage.ImageIntegration{}
	ii.SetId("id")
	ii.SetCategories([]storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY})

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

		rbid := &v1.ResourceByID{}
		rbid.SetId("id")
		_, err := s.DeleteImageIntegration(context.Background(), rbid)
		assert.NoError(t, err)
	})

	t.Run("no broadcast on no exist", func(t *testing.T) {
		setup(t)

		intMgr.EXPECT().Remove(gomock.Any()).Return(nil)

		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(nil, false, nil)
		iiDS.EXPECT().RemoveImageIntegration(gomock.Any(), gomock.Any()).Return(nil)

		rbid := &v1.ResourceByID{}
		rbid.SetId("id")
		_, err := s.DeleteImageIntegration(context.Background(), rbid)
		assert.NoError(t, err)
	})

	t.Run("err on failure to get existing", func(t *testing.T) {
		setup(t)

		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(nil, false, errors.New("broken"))

		rbid := &v1.ResourceByID{}
		rbid.SetId("id")
		_, err := s.DeleteImageIntegration(context.Background(), rbid)
		assert.Error(t, err)
	})
}

func TestScannerV4Restrictions(t *testing.T) {
	ii := &storage.ImageIntegration{}
	ii.SetName("fake")
	ii.SetId("fake")
	ii.SetType(types.ScannerV4)
	ii.SetCategories([]storage.ImageIntegrationCategory{
		storage.ImageIntegrationCategory_SCANNER,
	})
	ii.IntegrationConfig = &storage.ImageIntegration_ScannerV4{}

	t.Run("prevent scannerv4 create", func(t *testing.T) {
		s := &serviceImpl{}

		iiNew := ii.CloneVT()
		iiNew.SetId("")

		_, err := s.PostImageIntegration(context.Background(), iiNew)
		assert.ErrorContains(t, err, "scanner V4")
	})

	t.Run("prevent scannerv4 delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		iiDS := integrationMocks.NewMockDataStore(ctrl)
		s := &serviceImpl{datastore: iiDS}

		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(ii, true, nil)

		rbid := &v1.ResourceByID{}
		rbid.SetId("fake")
		_, err := s.DeleteImageIntegration(context.Background(), rbid)
		assert.ErrorContains(t, err, "scanner V4")
	})

	t.Run("prevent type change FROM scannerv4", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		iiDS := integrationMocks.NewMockDataStore(ctrl)
		s := &serviceImpl{datastore: iiDS}

		iiOld := ii.CloneVT()
		iiOld.SetType(types.Clairify)

		iiDS.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).Return([]*storage.ImageIntegration{iiOld}, nil)
		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(iiOld, true, nil)

		uiir := &v1.UpdateImageIntegrationRequest{}
		uiir.SetConfig(ii)
		_, err := s.UpdateImageIntegration(context.Background(), uiir)
		assert.ErrorContains(t, err, "scanner V4")
	})

	t.Run("prevent type change TO scannerv4", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		iiDS := integrationMocks.NewMockDataStore(ctrl)
		s := &serviceImpl{datastore: iiDS}

		iiNew := ii.CloneVT()
		iiNew.SetType(types.Clairify)
		iiNew.IntegrationConfig = &storage.ImageIntegration_Clairify{}

		iiDS.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).Return([]*storage.ImageIntegration{ii}, nil)
		iiDS.EXPECT().GetImageIntegration(gomock.Any(), gomock.Any()).Return(ii, true, nil)

		uiir := &v1.UpdateImageIntegrationRequest{}
		uiir.SetConfig(iiNew)
		_, err := s.UpdateImageIntegration(context.Background(), uiir)
		assert.ErrorContains(t, err, "scanner V4")
	})
}
