//go:build sql_integration

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterInitStore "github.com/stackrox/rox/central/clusterinit/store"
	installationMock "github.com/stackrox/rox/central/installation/store/mocks"
	"github.com/stackrox/rox/central/sensor/service/connection"
	connectionMock "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	pipelineMock "github.com/stackrox/rox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/grpc/authn"
	authnMock "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/mtls"
	mtlsTestutils "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
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

func TestIsCARotationSupported(t *testing.T) {
	testCases := map[string]struct {
		capabilities []string
		expected     bool
	}{
		"should return true when CA rotation capability is present": {
			capabilities: []string{
				string(centralsensor.ComplianceInNodesCap),
				string(centralsensor.SensorCARotationSupported),
				string(centralsensor.ScannerV4Supported),
			},
			expected: true,
		},
		"should return false when CA rotation capability is missing": {
			capabilities: []string{
				string(centralsensor.ComplianceInNodesCap),
				string(centralsensor.ScannerV4Supported),
			},
			expected: false,
		},
		"should return false when no capabilities are provided": {
			capabilities: []string{},
			expected:     false,
		},
		"should return false when nil capabilities": {
			capabilities: nil,
			expected:     false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			sensorHello := &central.SensorHello{
				Capabilities: tc.capabilities,
			}

			result := isCARotationSupported(sensorHello)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Sensor Service Test Suite (primarily, but not exclusively, for CRS handshake).
type sensorServiceTestSuite struct {
	suite.Suite
	mockCtrl         *gomock.Controller
	internalContext  context.Context
	db               *pgtest.TestPostgres
	clusterInitStore clusterInitStore.Store
	clusterDataStore clusterDataStore.DataStore
}

func (s *sensorServiceTestSuite) SetupSuite() {
	imageFlavor := "rhacs"
	utils.Must(os.Setenv("ROX_IMAGE_FLAVOR", imageFlavor))
	testutils.SetExampleVersion(s.T())
	s.internalContext = sac.WithAllAccess(context.Background())
	s.db = pgtest.ForT(s.T())
	var err error
	s.clusterDataStore, err = clusterDataStore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.Require().NoError(err, "failed to create cluster data store")
	s.clusterInitStore = clusterDataStore.IntrospectClusterInitStore(s.T(), s.clusterDataStore)

	s.mockCtrl = gomock.NewController(s.T())
	utils.Should(mtlsTestutils.LoadTestMTLSCerts(s.T()))
}

func (s *sensorServiceTestSuite) SetupTest() {
	log.Infof("Running test: %s", s.T().Name())
}

func TestClusterRegistration(t *testing.T) {
	suite.Run(t, new(sensorServiceTestSuite))
}

func newClusterNames() (string, string, string) {
	baseUuid := uuid.NewV4().String()
	base := baseUuid[:9]
	return base + "a", base + "b", base + "c"

}

func (s *sensorServiceTestSuite) TestCrsCentralReturnsAllServiceCertificates() {
	_, crsMeta := newCrsMeta(0)
	sensorService := newSensorService(s, crsMeta)
	clusterName, _, _ := newClusterNames()
	mockServer := newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationA, clusterName)

	err := sensorService.Communicate(mockServer)
	s.NoError(err)
	centralHello := retrieveCentralHello(s, mockServer)
	_ = centralHello

	assertCertificateBundleComplete(s, centralHello.GetCertBundle())
}

func (s *sensorServiceTestSuite) TestCrsFlowCanBeRepeated() {
	_, crsMeta := newCrsMeta(0)
	sensorService := newSensorService(s, crsMeta)
	clusterName, _, _ := newClusterNames()

	// First-time CRS cluster registration.
	mockServer := newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationA, clusterName)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Initiating the CRS flow a second time should still work.
	mockServer = newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationA, clusterName)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)
}

func (s *sensorServiceTestSuite) TestCrsFlowFailsAfterRegistrationComplete() {
	_, crsMeta := newCrsMeta(0)
	sensorService := newSensorService(s, crsMeta)
	clusterName, _, _ := newClusterNames()

	// CRS cluster registration.
	mockServer := newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationA, clusterName)
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Regular connect.
	mockServer = newMockServerForRegularConnect(s, sensorDeploymentIdentificationA, clusterName)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)

	// Initiating the CRS should fail now.
	mockServer = newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationA, clusterName)
	err = sensorService.Communicate(mockServer)
	s.Error(err, "CRS flow succeeded even after regular connect.")
	s.ErrorContains(err, "forbidden to use a Cluster Registration Certificate for already-existing cluster")
}

func lookupCrs(s *sensorServiceTestSuite, crsId string) *storage.InitBundleMeta {
	crsMeta, err := s.clusterInitStore.Get(s.internalContext, crsId)
	s.Require().NoError(err)
	return crsMeta
}

func (s *sensorServiceTestSuite) TestClusterRegistrationWithOneShotCrs() {
	crsMetaId, crsMeta := newCrsMeta(1)
	sensorService := newSensorService(s, crsMeta)
	clusterNameA, clusterNameB, _ := newClusterNames()

	mockServer := newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationA, clusterNameA)

	// CRS cluster registration.
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Verify that cluster registrations is initiated.
	crsMeta = lookupCrs(s, crsMetaId)
	registrationsInitiated := crsMeta.GetRegistrationsInitiated()
	s.Len(registrationsInitiated, 1, "Unexpected number of initiated registrations for CRS")
	assert.Containsf(s.T(), registrationsInitiated, clusterNameA, "registrationsInitiated (%v) does not contain %q", registrationsInitiated, clusterNameA)

	// Attempt another cluster registration.
	mockServer = newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationA, clusterNameB)
	err = sensorService.Communicate(mockServer)
	s.Error(err)

	// Verify that no other cluster registration is initiated.
	crsMeta = lookupCrs(s, crsMetaId)
	registrationsInitiated = crsMeta.GetRegistrationsInitiated()
	s.Len(registrationsInitiated, 1, "Unexpected number of initiated registrations for CRS after second registration attempt")

	// Execute a regular connect.
	mockServer = newMockServerForRegularConnect(s, sensorDeploymentIdentificationA, clusterNameA)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)

	// Verify that cluster registrations is completed.
	crsMeta = lookupCrs(s, crsMetaId)
	registrationsInitiated = crsMeta.GetRegistrationsInitiated()
	registrationsCompleted := crsMeta.GetRegistrationsCompleted()
	s.Len(registrationsInitiated, 0, "Unexpected number of initiated registrations for CRS")
	s.Len(registrationsCompleted, 1, "Unexpected number of initiated registrations for CRS")
	assert.Containsf(s.T(), registrationsCompleted, clusterNameA, "registrationsCompleted (%v) does not contain %q", registrationsCompleted, clusterNameA)

	// Verify CRS is revoked.
	crsMeta = lookupCrs(s, crsMetaId)
	s.True(crsMeta.GetIsRevoked(), "CRS is not revoked after one-shot use")
}

func toPrettyJson(s *sensorServiceTestSuite, v any) string {
	bytes, err := json.MarshalIndent(v, "|", "  ")
	s.Require().NoErrorf(err, "JSON marshalling of value %+v failed", v)
	return string(bytes)
}

func (s *sensorServiceTestSuite) TestClusterRegistrationWithTwoLimitCrs() {
	crsMetaId, crsMeta := newCrsMeta(2)
	sensorService := newSensorService(s, crsMeta)
	clusterNameA, clusterNameB, clusterNameC := newClusterNames()
	mockServer := newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationA, clusterNameA)

	// CRS cluster registration.
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Verify that cluster registrations is initiated.
	crsMeta = lookupCrs(s, crsMetaId)
	registrationsInitiated := crsMeta.GetRegistrationsInitiated()
	s.Len(registrationsInitiated, 1, "Unexpected number of initiated registrations for CRS:\n%s\n", toPrettyJson(s, crsMeta))
	assert.Containsf(s.T(), registrationsInitiated, clusterNameA, "registrationsInitiated (%v) does not contain %q", registrationsInitiated, clusterNameA)

	// Attempt another cluster registration.
	mockServer = newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationB, clusterNameB)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)

	// Verify that cluster registrations is initiated.
	crsMeta = lookupCrs(s, crsMetaId)
	registrationsInitiated = crsMeta.GetRegistrationsInitiated()
	s.Len(registrationsInitiated, 2, "Unexpected number of initiated registrations for CRS")
	assert.Containsf(s.T(), registrationsInitiated, clusterNameB, "registrationsInitiated (%v) does not contain %q", registrationsInitiated, clusterNameB)

	// Execute regular connects.
	mockServer = newMockServerForRegularConnect(s, sensorDeploymentIdentificationA, clusterNameA)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)
	mockServer = newMockServerForRegularConnect(s, sensorDeploymentIdentificationB, clusterNameB)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)

	// Verify that cluster registrations is completed.
	crsMeta = lookupCrs(s, crsMetaId)
	registrationsInitiated = crsMeta.GetRegistrationsInitiated()
	registrationsCompleted := crsMeta.GetRegistrationsCompleted()
	s.Len(registrationsInitiated, 0, "Unexpected number of initiated registrations for CRS")
	s.Len(registrationsCompleted, 2, "Unexpected number of initiated registrations for CRS")
	assert.Containsf(s.T(), registrationsCompleted, clusterNameA, "registrationsCompleted (%v) does not contain %q", registrationsCompleted, clusterNameA)
	assert.Containsf(s.T(), registrationsCompleted, clusterNameB, "registrationsCompleted (%v) does not contain %q", registrationsCompleted, clusterNameB)

	// Attempt another cluster registration.
	mockServer = newMockServerForCrsHandshake(s, crsMeta, sensorDeploymentIdentificationA, clusterNameC)
	err = sensorService.Communicate(mockServer)
	s.Error(err)

	// Verify that no other cluster registration has been recorded.
	crsMeta = lookupCrs(s, crsMetaId)
	registrationsInitiated = crsMeta.GetRegistrationsInitiated()
	registrationsCompleted = crsMeta.GetRegistrationsCompleted()
	s.Len(registrationsInitiated, 0, "Unexpected number of initiated registrations for CRS after third registration attempt")
	s.Len(registrationsCompleted, 2, "Unexpected number of completed registrations for CRS after third registration attempt")

	// Verify CRS is revoked.
	crsMeta = lookupCrs(s, crsMetaId)
	s.True(crsMeta.GetIsRevoked(), "CRS is not revoked after registering two clusters")
}

func (s *sensorServiceTestSuite) lookupClusterByName(name string) *storage.Cluster {
	query := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Cluster, name).ProtoQuery()
	results, err := s.clusterDataStore.Search(s.internalContext, query)
	s.NoError(err)
	s.Lenf(results, 1, "unexpected number of search results when looking for cluster by name %s", name)

	resultIds := pkgSearch.ResultsToIDs(results)
	s.Len(resultIds, 1)

	clusterId := resultIds[0]
	cluster, ok, err := s.clusterDataStore.GetCluster(s.internalContext, clusterId)
	s.NoErrorf(err, "failed to retrieve cluster %s (%s)", name)
	s.True(ok)

	return cluster
}

func (s *sensorServiceTestSuite) TestClusterRegistrationWithInitBundle() {
	initBundleId, initBundleMeta := newInitBundleMeta()
	sensorService := newSensorService(s, initBundleMeta)
	clusterNameA, _, _ := newClusterNames()

	mockServer := newMockServerForInitBundleHandshake(s, initBundleMeta, sensorDeploymentIdentificationA, clusterNameA)

	// Init Bundle cluster registration.
	err := sensorService.Communicate(mockServer)
	s.NoError(err)

	// Verify that init bundle is still associated with cluster.
	cluster := s.lookupClusterByName(clusterNameA)
	s.NotEmptyf(cluster.InitBundleId, "cluster %s lost association to init bundle %s", clusterNameA, initBundleId)

	// Simulate regular connection with non-init certificate.
	mockServer = newMockServerForRegularConnect(s, sensorDeploymentIdentificationA, clusterNameA)
	err = sensorService.Communicate(mockServer)
	s.NoError(err)

	// Verify that init bundle is not associated with cluster anymore.
	cluster = s.lookupClusterByName(clusterNameA)
	s.Emptyf(cluster.InitBundleId, "cluster %s still association with init bundle %s", clusterNameA, initBundleId)
}

// Implementation of a simple mock server to be used in the CRS test suite.
type mockServer struct {
	grpc.ServerStream
	context            context.Context
	msgsFromSensor     []*central.MsgFromSensor
	msgsToSensor       []*central.MsgToSensor
	clustersRegistered []*storage.Cluster
}

func newMockServer(s *sensorServiceTestSuite, identity *storage.ServiceIdentity) *mockServer {
	notBefore := time.Now().Add(-time.Minute)
	notAfter := time.Now().Add(time.Hour)

	mockIdentity := authnMock.NewMockIdentity(s.mockCtrl)
	mockIdentity.EXPECT().Service().AnyTimes().Return(identity)
	mockIdentity.EXPECT().ValidityPeriod().AnyTimes().Return(notBefore, notAfter)
	md := metadata.Pairs(centralsensor.SensorHelloMetadataKey, "true")
	ctx := authn.ContextWithIdentity(s.internalContext, mockIdentity, nil)
	ctx = metadata.NewIncomingContext(ctx, md)
	return &mockServer{
		context: ctx,
	}
}

func prepareHelloHandshake(m *mockServer, sensorDeploymentId *storage.SensorDeploymentIdentification, clusterName string) {
	hello := &central.SensorHello{
		SensorVersion:            "1.0",
		DeploymentIdentification: sensorDeploymentId,
		HelmManagedConfigInit: &central.HelmManagedConfigInit{
			ClusterName: clusterName,
			ManagedBy:   storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		},
	}
	m.msgsFromSensor = []*central.MsgFromSensor{
		{
			Msg: &central.MsgFromSensor_Hello{Hello: hello},
		},
	}
	m.msgsToSensor = nil
}

func newMockServerForCrsHandshake(s *sensorServiceTestSuite, crsMeta *storage.InitBundleMeta, sensorDeploymentId *storage.SensorDeploymentIdentification, clusterName string) *mockServer {
	identity := &storage.ServiceIdentity{
		Type:         storage.ServiceType_REGISTRANT_SERVICE,
		InitBundleId: crsMeta.Id,
	}

	m := newMockServer(s, identity)
	prepareHelloHandshake(m, sensorDeploymentId, clusterName)
	return m
}

func newMockServerForInitBundleHandshake(s *sensorServiceTestSuite, crsMeta *storage.InitBundleMeta, sensorDeploymentId *storage.SensorDeploymentIdentification, clusterName string) *mockServer {
	identity := &storage.ServiceIdentity{
		Type:         storage.ServiceType_SENSOR_SERVICE,
		InitBundleId: crsMeta.Id,
	}

	m := newMockServer(s, identity)
	prepareHelloHandshake(m, sensorDeploymentId, clusterName)
	return m
}

func newMockServerForRegularConnect(s *sensorServiceTestSuite, sensorDeploymentId *storage.SensorDeploymentIdentification, clusterName string) *mockServer {
	m := newMockServer(s, &storage.ServiceIdentity{
		Type: storage.ServiceType_SENSOR_SERVICE,
	})
	prepareHelloHandshake(m, sensorDeploymentId, clusterName)
	return m
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
		MostRecentSensorId: hello.GetDeploymentIdentification(),
	}
}

func retrieveCentralHello(s *sensorServiceTestSuite, server *mockServer) *central.CentralHello {
	s.NotEmpty(server.msgsToSensor, "no central response message")
	centralMsg := server.msgsToSensor[0]
	centralHello := centralMsg.GetHello()
	s.NotNil(centralHello)
	return centralHello
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

func newSensorService(s *sensorServiceTestSuite, crsMeta *storage.InitBundleMeta) Service {
	mockInstallation := installationMock.NewMockStore(s.mockCtrl)
	installInfo := &storage.InstallationInfo{
		Id: "some-central-id",
	}
	mockInstallation.EXPECT().Get(gomock.Any()).AnyTimes().Return(installInfo, true, nil)

	mockConnectionManager := connectionMock.NewMockManager(s.mockCtrl)
	mockConnectionManager.EXPECT().
		GetConnectionPreference(gomock.Any()).
		AnyTimes().Return(connection.Preferences{})
	mockConnectionManager.EXPECT().
		HandleConnection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(mockHandleConnection(s.clusterDataStore))

	pipelineFactory := pipelineMock.NewMockFactory(s.mockCtrl)
	pipeline := pipelineMock.NewMockClusterPipeline(s.mockCtrl)
	pipeline.EXPECT().Capabilities().AnyTimes().Return([]centralsensor.CentralCapability{})
	pipelineFactory.EXPECT().PipelineForCluster(gomock.Any(), gomock.Any()).AnyTimes().Return(pipeline, nil)

	clusterInitStore := clusterDataStore.IntrospectClusterInitStore(s.T(), s.clusterDataStore)

	if crsMeta != nil {
		err := clusterInitStore.Add(s.internalContext, crsMeta)
		s.NoError(err, "failed adding dummy CRS to cluster init store")
		s.T().Logf("Added dummy CRS with ID %q and name %q", crsMeta.GetId(), crsMeta.GetName())
	}

	sensorService := New(mockConnectionManager, pipelineFactory, s.clusterDataStore, mockInstallation, clusterInitStore)
	return sensorService
}

func newCrsMeta(maxRegistrations uint64) (string, *storage.InitBundleMeta) {
	id := uuid.NewV4().String()
	meta := &storage.InitBundleMeta{
		Id:               id,
		Name:             fmt.Sprintf("name-%s", id),
		CreatedAt:        timestamppb.New(time.Now()),
		ExpiresAt:        timestamppb.New(time.Now().Add(10 * time.Minute)),
		Version:          storage.InitBundleMeta_CRS,
		MaxRegistrations: maxRegistrations,
	}
	return meta.Id, meta
}

func newInitBundleMeta() (string, *storage.InitBundleMeta) {
	id := uuid.NewV4().String()
	meta := &storage.InitBundleMeta{
		Id:        id,
		Name:      fmt.Sprintf("init-bundle-%s", id),
		CreatedAt: timestamppb.New(time.Now()),
		ExpiresAt: timestamppb.New(time.Now().Add(10 * time.Minute)),
		Version:   storage.InitBundleMeta_INIT_BUNDLE,
	}
	return meta.Id, meta
}

func assertCertificateBundleComplete(s *sensorServiceTestSuite, certBundle map[string]string) {
	s.Len(certBundle, 15, "expected 15 entries (1 CA cert, 7 service certs, 7 service keys) in bundle")
}

type HandleConnectionFunc = func(ctx context.Context, _ *central.SensorHello, cluster *storage.Cluster, _ pipeline.ClusterPipeline, _ central.SensorService_CommunicateServer) error

func mockHandleConnection(clusterDataStore clusterDataStore.DataStore) HandleConnectionFunc {
	return func(ctx context.Context, _ *central.SensorHello, cluster *storage.Cluster, _ pipeline.ClusterPipeline, _ central.SensorService_CommunicateServer) error {
		cluster.HealthStatus = &storage.ClusterHealthStatus{
			LastContact: timestamppb.New(time.Now()),
		}
		return clusterDataStore.UpdateCluster(ctx, cluster)
	}
}
