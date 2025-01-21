//go:build sql_integration

package service

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterInitStore "github.com/stackrox/rox/central/clusterinit/store"
	installationMock "github.com/stackrox/rox/central/installation/store/mocks"
	"github.com/stackrox/rox/central/sensor/service/connection"
	connectionMock "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	pipelineMock "github.com/stackrox/rox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authn"
	authnMock "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/mtls"
	mtlsTestutils "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetCertExpiryStatus(t *testing.T) {
	type testCase struct {
		notBefore, notAfter time.Time
		expectedStatus      *storage.ClusterCertExpiryStatus
	}
	testCases := map[string]testCase{
		"should return nil when no dates": {
			expectedStatus: nil,
		},
		"should fill not before only if expiry is not set": {
			notBefore: time.Unix(1646870400, 0), // Thu Mar 10 2022 00:00:00 GMT+0000
			expectedStatus: &storage.ClusterCertExpiryStatus{
				SensorCertNotBefore: protocompat.GetProtoTimestampFromSeconds(1646870400),
			},
		},
		"should fill expiry only if notbefore is not set": {
			notAfter: time.Unix(1646956799, 0), // Thu Mar 10 2022 23:59:59 GMT+0000
			expectedStatus: &storage.ClusterCertExpiryStatus{
				SensorCertExpiry: protocompat.GetProtoTimestampFromSeconds(1646956799),
			},
		},
		"should fill status if both bounds are set": {
			notBefore: time.Unix(1646870400, 0), // Thu Mar 10 2022 00:00:00 GMT+0000
			notAfter:  time.Unix(1646956799, 0), // Thu Mar 10 2022 23:59:59 GMT+0000
			expectedStatus: &storage.ClusterCertExpiryStatus{
				SensorCertNotBefore: protocompat.GetProtoTimestampFromSeconds(1646870400),
				SensorCertExpiry:    protocompat.GetProtoTimestampFromSeconds(1646956799),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			identity := service.WrapMTLSIdentity(mtls.IdentityFromCert(mtls.CertInfo{
				NotBefore: tc.notBefore,
				NotAfter:  tc.notAfter,
			}))
			result, err := getCertExpiryStatus(identity)
			assert.NoError(t, err)
			protoassert.Equal(t, tc.expectedStatus, result)
		})
	}
}

// CRS Test Suite.

type crsTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
	context  context.Context
	db       *pgtest.TestPostgres
	ctx      context.Context
}

func (s *crsTestSuite) SetupSuite() {
	imageFlavor := "rhacs"
	utils.Must(os.Setenv("ROX_IMAGE_FLAVOR", imageFlavor))

	testutils.SetExampleVersion(s.T())
	s.db = pgtest.ForT(s.T())
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Administration, resources.Integration)))
}

func (s *crsTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.context = sac.WithAllAccess(context.Background())
	utils.Should(mtlsTestutils.LoadTestMTLSCerts(s.T()))
}

func TestCrs(t *testing.T) {
	suite.Run(t, new(crsTestSuite))
}

func (s *crsTestSuite) TestCrsCentralReturnsAllServiceCertificates() {
	crsMeta := storage.InitBundleMeta{
		Id:               crsRegistrantId,
		Name:             "xxx",
		CreatedAt:        timestamppb.New(time.Now()),
		ExpiresAt:        timestamppb.New(time.Now().Add(10 * time.Minute)),
		Version:          storage.InitBundleMeta_CRS,
		MaxRegistrations: 0,
	}
	sensorService, mockServer := s.newSensorService(s.context, s.mockCtrl, &crsMeta)

	mockServer.prepareNewHandshake(defaultSensorHello)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)
	centralHello := retrieveCentralHello(s, mockServer)
	clusters, err := mockServer.getClusters()
	s.NoError(err)
	s.Len(clusters, 1, "expected exactly one registered cluster")
	s.NoError(err)
	assertCertificateBundleComplete(s, centralHello.GetCertBundle())
}

func assertCertificateBundleComplete(s *crsTestSuite, certBundle map[string]string) {
	s.Len(certBundle, 15, "expected 15 entries (1 CA cert, 7 service certs, 7 service keys) in bundle")
}

func (s *crsTestSuite) TestCrsFlowCanBeRepeated() {
	crsMeta := storage.InitBundleMeta{
		Id:               crsRegistrantId,
		Name:             "xxx",
		CreatedAt:        timestamppb.New(time.Now()),
		ExpiresAt:        timestamppb.New(time.Now().Add(10 * time.Minute)),
		Version:          storage.InitBundleMeta_CRS,
		MaxRegistrations: 0,
	}
	sensorService, mockServer := s.newSensorService(s.context, s.mockCtrl, &crsMeta)

	// First-time CRS cluster registration.
	mockServer.prepareNewHandshake(defaultSensorHello)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)
	centralHello := retrieveCentralHello(s, mockServer)
	clusters, err := mockServer.getClusters()
	s.NoError(err)
	s.Len(clusters, 1, "expected exactly one registered cluster")

	// Initiating the CRS flow a second time should still work.
	mockServer.prepareNewHandshake(defaultSensorHello)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)
	clusters, err = mockServer.getClusters()
	s.NoError(err)
	s.Len(clusters, 1, "expected exactly one registered cluster")
	// Verify that we again got all certificates we need.
	assertCertificateBundleComplete(s, centralHello.GetCertBundle())
}

func (s *crsTestSuite) TestCrsFlowFailsAfterLastContact() {
	crsMeta := storage.InitBundleMeta{
		Id:               crsRegistrantId,
		Name:             "xxx",
		CreatedAt:        timestamppb.New(time.Now()),
		ExpiresAt:        timestamppb.New(time.Now().Add(10 * time.Minute)),
		Version:          storage.InitBundleMeta_CRS,
		MaxRegistrations: 0,
	}
	sensorService, mockServer := s.newSensorService(s.context, s.mockCtrl, &crsMeta)

	// CRS cluster registration.
	mockServer.prepareNewHandshake(defaultSensorHello)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Simulate a connection with service certificates has occurred by updating
	// the LastContact field of a cluster.
	clusters, err := mockServer.getClusters()
	s.NoError(err)
	s.Len(clusters, 1)
	cluster := clusters[0]
	cluster.HealthStatus = &storage.ClusterHealthStatus{
		LastContact: timestamppb.Now(),
	}
	err = mockServer.clusterDataStore.UpdateCluster(s.ctx, cluster)
	s.NoError(err)

	// Initiating the CRS should fail now.
	mockServer.prepareNewHandshake(defaultSensorHello)
	err = sensorService.Communicate(mockServer)
	s.ErrorContains(err, "forbidden to use a Cluster Registration Certificate for already-existing cluster")
	s.Error(err, "CRS flow succeeded even after LastContact field was updated.")
}

func (s *crsTestSuite) TestClusterReissuingWithDifferentDeploymentIdFails() {
	crsMeta := &storage.InitBundleMeta{
		Id:               crsRegistrantId,
		Name:             "xxx",
		CreatedAt:        timestamppb.New(time.Now()),
		ExpiresAt:        timestamppb.New(time.Now().Add(10 * time.Minute)),
		Version:          storage.InitBundleMeta_CRS,
		MaxRegistrations: 1,
	}

	sensorService, mockServer := s.newSensorService(s.context, s.mockCtrl, crsMeta)

	mockServer.prepareNewHandshake(defaultSensorHello)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Attempt another cluster registration.
	helloB := defaultSensorHello.CloneVT()
	helloB.DeploymentIdentification = sensorDeploymentIdentificationB
	mockServer.prepareNewHandshake(helloB)
	err = sensorService.Communicate(mockServer)
	s.Error(err)
}

func (s *crsTestSuite) TestClusterRegistrationWithOneShotCrs() {
	crsMeta := &storage.InitBundleMeta{
		Id:               crsRegistrantId,
		Name:             "xxx",
		CreatedAt:        timestamppb.New(time.Now()),
		ExpiresAt:        timestamppb.New(time.Now().Add(10 * time.Minute)),
		Version:          storage.InitBundleMeta_CRS,
		MaxRegistrations: 1,
	}

	sensorService, mockServer := s.newSensorService(s.context, s.mockCtrl, crsMeta)

	// CRS cluster registration.
	mockServer.prepareNewHandshake(defaultSensorHello)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Attempt another cluster registration.
	helloB := defaultSensorHello.CloneVT()
	helloB.HelmManagedConfigInit.ClusterName = "foo-bar"
	mockServer.prepareNewHandshake(helloB)
	err = sensorService.Communicate(mockServer)
	s.Error(err)
}

func (s *crsTestSuite) TestClusterRegistrationWithTwoLimitCrs() {
	crsMeta := &storage.InitBundleMeta{
		Id:               crsRegistrantId,
		Name:             "xxx",
		CreatedAt:        timestamppb.New(time.Now()),
		ExpiresAt:        timestamppb.New(time.Now().Add(10 * time.Minute)),
		Version:          storage.InitBundleMeta_CRS,
		MaxRegistrations: 2,
	}

	sensorService, mockServer := s.newSensorService(s.context, s.mockCtrl, crsMeta)

	// First registration.
	mockServer.prepareNewHandshake(defaultSensorHello)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Second registration.
	hello := defaultSensorHello.CloneVT()
	hello.HelmManagedConfigInit.ClusterName = "foo-1"
	mockServer.prepareNewHandshake(hello)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)

	// Third registration attempt.
	helloB := defaultSensorHello.CloneVT()
	helloB.HelmManagedConfigInit.ClusterName = "foo-2"
	mockServer.prepareNewHandshake(helloB)
	err = sensorService.Communicate(mockServer)
	s.Error(err)
}

// Implementation of a simple mock server to be used in the CRS test suite.
type mockServer struct {
	grpc.ServerStream
	context          context.Context
	msgsFromSensor   []*central.MsgFromSensor
	msgsToSensor     []*central.MsgToSensor
	clusterDataStore clusterDataStore.DataStore
}

func newMockServer(ctx context.Context, ctrl *gomock.Controller, registrantIdentity *storage.ServiceIdentity) *mockServer {
	mockIdentity := authnMock.NewMockIdentity(ctrl)
	mockIdentity.EXPECT().Service().AnyTimes().Return(registrantIdentity)
	md := metadata.Pairs(centralsensor.SensorHelloMetadataKey, "true")
	ctx = authn.ContextWithIdentity(ctx, mockIdentity, nil)
	ctx = metadata.NewIncomingContext(ctx, md)
	return &mockServer{
		context: ctx,
	}
}

func (m *mockServer) getClusters() ([]*storage.Cluster, error) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Cluster, resources.Administration, resources.Integration)))
	return m.clusterDataStore.GetClusters(ctx)
}

func (s *mockServer) prepareNewHandshake(hello *central.SensorHello) {
	s.msgsFromSensor = []*central.MsgFromSensor{
		{
			Msg: &central.MsgFromSensor_Hello{Hello: hello},
		},
	}
	s.msgsToSensor = nil
}

func (s *mockServer) Context() context.Context {
	return s.context
}
func (s *mockServer) Send(msg *central.MsgToSensor) error {
	s.msgsToSensor = append(s.msgsToSensor, msg)
	return nil
}

func (s *mockServer) Recv() (*central.MsgFromSensor, error) {
	if len(s.msgsFromSensor) == 0 {
		return nil, io.EOF
	}
	nextMsg := s.msgsFromSensor[0]
	s.msgsFromSensor = s.msgsFromSensor[1:]
	return nextMsg, nil
}

func (s *mockServer) SendHeader(header metadata.MD) error {
	return nil
}

// func newCluster(clusterName string, hello *central.SensorHello) *storage.Cluster {
// 	return &storage.Cluster{
// 		Id:                 uuid.NewV4().String(),
// 		Name:               clusterName,
// 		HealthStatus:       &storage.ClusterHealthStatus{},
// 		MostRecentSensorId: hello.DeploymentIdentification,
// 	}
// }

func retrieveCentralHello(s *crsTestSuite, server *mockServer) *central.CentralHello {
	s.NotEmpty(server.msgsToSensor, "no central response message")
	centralMsg := server.msgsToSensor[0]
	centralHello := centralMsg.GetHello()
	s.NotNil(centralHello)
	return centralHello
}

var crsRegistrantId = uuid.NewV4().String()

// var initBundleRegistrantIdentity = storage.ServiceIdentity{
// 	Type:         storage.ServiceType_REGISTRANT_SERVICE,
// 	InitBundleId: initBundleRegistrantId,
// }

var crsRegistrantIdentity = storage.ServiceIdentity{
	Type:         storage.ServiceType_REGISTRANT_SERVICE,
	InitBundleId: crsRegistrantId,
}

var installInfo *storage.InstallationInfo = &storage.InstallationInfo{
	Id: "some-central-id",
}

var sensorDeploymentIdentificationA = &storage.SensorDeploymentIdentification{
	SystemNamespaceId:   uuid.NewV4().String(),
	DefaultNamespaceId:  uuid.NewV4().String(),
	AppNamespace:        "my-stackrox-namespace-a",
	AppNamespaceId:      uuid.NewV4().String(),
	AppServiceaccountId: uuid.NewV4().String(),
	K8SNodeName:         "my-node",
}

var sensorDeploymentIdentificationB = &storage.SensorDeploymentIdentification{
	SystemNamespaceId:   uuid.NewV4().String(),
	DefaultNamespaceId:  uuid.NewV4().String(),
	AppNamespace:        "my-stackrox-namespace-b",
	AppNamespaceId:      uuid.NewV4().String(),
	AppServiceaccountId: uuid.NewV4().String(),
	K8SNodeName:         "my-node",
}

var defaultSensorHello = &central.SensorHello{
	SensorVersion:            "1.0",
	DeploymentIdentification: sensorDeploymentIdentificationA,
	HelmManagedConfigInit: &central.HelmManagedConfigInit{
		ClusterName: "my-new-cluster-x",
		ManagedBy:   storage.ManagerType_MANAGER_TYPE_HELM_CHART,
	},
}

func (s *crsTestSuite) newSensorService(ctx context.Context, ctrl *gomock.Controller, crsMeta *storage.InitBundleMeta) (Service, *mockServer) {
	mockServer := newMockServer(ctx, ctrl, &crsRegistrantIdentity)

	mockInstallation := installationMock.NewMockStore(ctrl)
	mockInstallation.EXPECT().Get(gomock.Any()).AnyTimes().Return(installInfo, true, nil)

	mockConnetionManager := connectionMock.NewMockManager(ctrl)
	mockConnetionManager.EXPECT().GetConnectionPreference(gomock.Any()).AnyTimes().Return(connection.Preferences{})

	pipelineFactory := pipelineMock.NewMockFactory(ctrl)
	pipeline := pipelineMock.NewMockClusterPipeline(ctrl)
	pipeline.EXPECT().Capabilities().AnyTimes().Return([]centralsensor.CentralCapability{})
	pipelineFactory.EXPECT().PipelineForCluster(gomock.Any(), gomock.Any()).AnyTimes().Return(pipeline, nil)

	clusterInitStore, err := clusterInitStore.GetTestClusterInitDataStore(s.T(), s.db.DB)
	s.NoError(err, "test cluster init data store creation failed")

	if crsMeta != nil {
		err = clusterInitStore.Add(s.ctx, crsMeta)
		s.NoError(err, "failed adding dummy CRS to cluster init store")
		s.T().Logf("Added dummy CRS with ID %q and name %q", crsMeta.GetId(), crsMeta.GetName())
	}

	clusterDataStore, err := clusterDataStore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.NoError(err, "FIXME")

	s.purgeClusters(clusterDataStore)

	mockServer.clusterDataStore = clusterDataStore
	sensorService := New(mockConnetionManager, pipelineFactory, clusterDataStore, mockInstallation, clusterInitStore)
	return sensorService, mockServer
}

func (s *crsTestSuite) purgeClusters(clusterDataStore clusterDataStore.DataStore) {
	clusters, err := clusterDataStore.GetClusters(s.context)
	s.NoError(err)
	s.T().Log("Purging clusters")
	for _, cluster := range clusters {
		done := concurrency.NewSignal()
		s.T().Logf("Removed cluster %q", cluster.GetName())
		err = clusterDataStore.RemoveCluster(s.context, cluster.GetId(), &done)
		s.NoError(err)
		done.Wait()
	}
}
