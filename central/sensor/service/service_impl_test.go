package service

import (
	"context"
	"io"
	"testing"
	"time"

	clusterMock "github.com/stackrox/rox/central/cluster/datastore/mocks"
	installationMock "github.com/stackrox/rox/central/installation/store/mocks"
	"github.com/stackrox/rox/central/sensor/service/connection"
	connectionMock "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	pipelineMock "github.com/stackrox/rox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/grpc/authn"
	authnMock "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
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
}

func (s *crsTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.context = sac.WithAllAccess(context.Background())
	utils.Should(testutils.LoadTestMTLSCerts(s.T()))
}

func TestCrs(t *testing.T) {
	suite.Run(t, new(crsTestSuite))
}

func (s *crsTestSuite) TestCrsCentralReturnsAllServiceCertificates() {
	sensorService, mockServer := newSensorService(s.context, s.mockCtrl)

	// First-time CRS cluster registration.
	mockServer.prepareNewHandshake(sensorHello)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)
	s.Len(mockServer.clustersRegistered, 1, "expected exactly one registered cluster")
	centralHello := retrieveCentralHello(s, mockServer)
	assertCertificateBundleComplete(s, centralHello.GetCertBundle())
}

func assertCertificateBundleComplete(s *crsTestSuite, certBundle map[string]string) {
	s.Len(certBundle, 15, "expected 15 entries (1 CA cert, 7 service certs, 7 service keys) in bundle")
}

func (s *crsTestSuite) TestCrsFlowCanBeRepeated() {
	sensorService, mockServer := newSensorService(s.context, s.mockCtrl)

	// First-time CRS cluster registration.
	mockServer.prepareNewHandshake(sensorHello)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)
	s.Len(mockServer.clustersRegistered, 1, "expected exactly one registered cluster")

	// Initiating the CRS flow a second time should still work.
	mockServer.prepareNewHandshake(sensorHello)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)
	s.Len(mockServer.clustersRegistered, 1, "expected exactly one registered cluster")
	// Verify that we again got all certificates we need.
	centralHello := retrieveCentralHello(s, mockServer)
	assertCertificateBundleComplete(s, centralHello.GetCertBundle())
}

func (s *crsTestSuite) TestCrsFlowFailsAfterLastContact() {
	sensorService, mockServer := newSensorService(s.context, s.mockCtrl)

	// CRS cluster registration.
	mockServer.prepareNewHandshake(sensorHello)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Simulate a connection with service certificates has occurred by updating
	// the LastContact field of a cluster.
	mockServer.clustersRegistered[0].HealthStatus = &storage.ClusterHealthStatus{
		LastContact: timestamppb.Now(),
	}

	// Initiating the CRS should fail now.
	mockServer.prepareNewHandshake(sensorHello)
	err = sensorService.Communicate(mockServer)
	s.ErrorContains(err, "forbidden to use a Cluster Registration Certificate for already-existing cluster")
	s.Error(err, "CRS flow succeeded even after LastContact field was updated.")
}

// Implementation of a simple mock server to be used in the CRS test suite.
type mockServer struct {
	grpc.ServerStream
	context            context.Context
	msgsFromSensor     []*central.MsgFromSensor
	msgsToSensor       []*central.MsgToSensor
	clustersRegistered []*storage.Cluster
}

func newMockServer(ctx context.Context, ctrl *gomock.Controller) *mockServer {
	mockIdentity := authnMock.NewMockIdentity(ctrl)
	mockIdentity.EXPECT().Service().AnyTimes().Return(&registrantIdentity)
	md := metadata.Pairs(centralsensor.SensorHelloMetadataKey, "true")
	ctx = authn.ContextWithIdentity(ctx, mockIdentity, nil)
	ctx = metadata.NewIncomingContext(ctx, md)
	return &mockServer{
		context: ctx,
	}
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

func (s *mockServer) LookupOrCreateClusterFromConfig(clusterId string, hello *central.SensorHello) (*storage.Cluster, error) {
	helmConfig := hello.GetHelmManagedConfigInit()
	clusterName := helmConfig.GetClusterName()

	for _, c := range s.clustersRegistered {
		if clusterName != "" && c.GetName() == clusterName ||
			clusterId != "" && c.GetId() == clusterId {
			return c, nil
		}
	}

	// Cluster not found, need to create new cluster.
	cluster := newCluster(clusterName, hello)
	s.clustersRegistered = append(s.clustersRegistered, cluster)
	return cluster, nil
}

func newCluster(clusterName string, hello *central.SensorHello) *storage.Cluster {
	return &storage.Cluster{
		Id:                 uuid.NewV4().String(),
		Name:               clusterName,
		HealthStatus:       &storage.ClusterHealthStatus{},
		MostRecentSensorId: hello.DeploymentIdentification,
	}
}

func retrieveCentralHello(s *crsTestSuite, server *mockServer) *central.CentralHello {
	s.NotEmpty(server.msgsToSensor, "no central response message")
	centralMsg := server.msgsToSensor[0]
	centralHello := centralMsg.GetHello()
	s.NotNil(centralHello)
	return centralHello
}

var registrantIdentity = storage.ServiceIdentity{
	Type: storage.ServiceType_REGISTRANT_SERVICE,
}

var installInfo *storage.InstallationInfo = &storage.InstallationInfo{
	Id: "some-central-id",
}

var sensorDeploymentIdentification = &storage.SensorDeploymentIdentification{
	SystemNamespaceId:   uuid.NewV4().String(),
	DefaultNamespaceId:  uuid.NewV4().String(),
	AppNamespace:        "my-stackrox-namespace",
	AppNamespaceId:      uuid.NewV4().String(),
	AppServiceaccountId: uuid.NewV4().String(),
	K8SNodeName:         "my-node",
}

var sensorHello = &central.SensorHello{
	SensorVersion:            "1.0",
	DeploymentIdentification: sensorDeploymentIdentification,
	HelmManagedConfigInit: &central.HelmManagedConfigInit{
		ClusterName: "my-new-cluster",
		ManagedBy:   storage.ManagerType_MANAGER_TYPE_HELM_CHART,
	},
}

func newSensorService(ctx context.Context, ctrl *gomock.Controller) (Service, *mockServer) {
	mockServer := newMockServer(ctx, ctrl)

	mockInstallation := installationMock.NewMockStore(ctrl)
	mockInstallation.EXPECT().Get(gomock.Any()).AnyTimes().Return(installInfo, true, nil)

	mockConnetionManager := connectionMock.NewMockManager(ctrl)
	mockConnetionManager.EXPECT().GetConnectionPreference(gomock.Any()).AnyTimes().Return(connection.Preferences{})

	pipelineFactory := pipelineMock.NewMockFactory(ctrl)
	pipeline := pipelineMock.NewMockClusterPipeline(ctrl)
	pipeline.EXPECT().Capabilities().AnyTimes().Return([]centralsensor.CentralCapability{})
	pipelineFactory.EXPECT().PipelineForCluster(gomock.Any(), gomock.Any()).AnyTimes().Return(pipeline, nil)

	clustersDataStore := clusterMock.NewMockDataStore(ctrl)
	clustersDataStore.EXPECT().LookupOrCreateClusterFromConfig(
		gomock.Any(), // ctx
		gomock.Any(), // clusterID
		gomock.Any(), // bundleID
		gomock.Any(), // sensorHello
	).
		AnyTimes().
		DoAndReturn(
			func(_ctx context.Context, clusterId, _bundleId string, hello *central.SensorHello) (*storage.Cluster, error) {
				return mockServer.LookupOrCreateClusterFromConfig(clusterId, hello)
			},
		)

	sensorService := New(mockConnetionManager, pipelineFactory, clustersDataStore, mockInstallation)
	return sensorService, mockServer
}
