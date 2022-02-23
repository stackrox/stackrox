package localscanner

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/mtls"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsApiv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	sensorNamespace      = "stackrox-ns"
	sensorReplicasetName = "sensor-replicaset"
	sensorPodName        = "sensor-pod"
)

type localScannerTLSIssuerFixture struct {
	k8sClient       *fake.Clientset
	certRequester   *certificateRequesterMock
	certRefresher   *certificateRefresherMock
	repo            *certsRepoMock
	componentGetter *componentGetterMock
	tlsIssuer       *localScannerTLSIssuerImpl
}

func newLocalScannerTLSIssuerFixture(k8sClientConfig fakeK8sClientConfig) *localScannerTLSIssuerFixture {
	fixture := &localScannerTLSIssuerFixture{
		certRequester:   &certificateRequesterMock{},
		certRefresher:   &certificateRefresherMock{},
		repo:            &certsRepoMock{},
		componentGetter: &componentGetterMock{},
		k8sClient:       getFakeK8sClient(k8sClientConfig),
	}
	msgToCentralC := make(chan *central.MsgFromSensor)
	msgFromCentralC := make(chan *central.IssueLocalScannerCertsResponse)
	fixture.tlsIssuer = &localScannerTLSIssuerImpl{
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    fixture.k8sClient,
		sensorManagedBy:              storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		msgToCentralC:                msgToCentralC,
		msgFromCentralC:              msgFromCentralC,
		getCertificateRefresherFn:    fixture.componentGetter.getCertificateRefresher,
		getServiceCertificatesRepoFn: fixture.componentGetter.getServiceCertificatesRepo,
		certRequester:                fixture.certRequester,
	}

	return fixture
}

func (f *localScannerTLSIssuerFixture) assertMockExpectations(t *testing.T) {
	f.certRequester.AssertExpectations(t)
	f.certRequester.AssertExpectations(t)
	f.componentGetter.AssertExpectations(t)
}

// mockForStart setups the mocks for the happy path of Start
func (f *localScannerTLSIssuerFixture) mockForStart(conf mockForStartConfig) {
	f.certRequester.On("Start").Once()
	f.certRefresher.On("Start").Once().Return(conf.refresherStartErr)

	f.repo.On("getServiceCertificates", mock.Anything).Once().
		Return((*storage.TypedServiceCertificateSet)(nil), conf.getCertsErr)

	f.componentGetter.On("getServiceCertificatesRepo", mock.Anything,
		mock.Anything, mock.Anything).Once().Return(f.repo, nil)

	f.componentGetter.On("getCertificateRefresher", mock.Anything, f.repo,
		certRefreshTimeout, certRefreshBackoff).Once().Return(f.certRefresher)
}

type mockForStartConfig struct {
	getCertsErr       error
	refresherStartErr error
}

func TestLocalScannerTLSIssuerStartStopSuccess(t *testing.T) {
	testCases := map[string]struct {
		getCertsErr error
	}{
		"no error":            {getCertsErr: nil},
		"missing secret data": {getCertsErr: ErrMissingSecretData},
		"inconsistent CAs":    {getCertsErr: ErrDifferentCAForDifferentServiceTypes},
		"missing secret":      {getCertsErr: k8sErrors.NewNotFound(schema.GroupResource{Group: "Core", Resource: "Secret"}, "scanner-db-slim-tls")},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
			fixture.mockForStart(mockForStartConfig{getCertsErr: tc.getCertsErr})
			fixture.certRefresher.On("Stop").Once()
			fixture.certRequester.On("Stop").Once()

			startErr := fixture.tlsIssuer.Start()
			fixture.tlsIssuer.Stop(nil)

			assert.NoError(t, startErr)
			assert.Nil(t, fixture.tlsIssuer.certRefresher)
			fixture.assertMockExpectations(t)
		})
	}
}

func TestLocalScannerTLSIssuerRefresherFailureStartFailure(t *testing.T) {
	fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.mockForStart(mockForStartConfig{refresherStartErr: errForced})
	fixture.certRefresher.On("Stop").Once()
	fixture.certRequester.On("Stop").Once()

	startErr := fixture.tlsIssuer.Start()

	assert.NoError(t, startErr)
	fixture.assertMockExpectations(t)
}

func TestLocalScannerTLSIssuerStartAlreadyStartedFailure(t *testing.T) {
	fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.mockForStart(mockForStartConfig{})
	fixture.certRefresher.On("Stop").Once()
	fixture.certRequester.On("Stop").Once()

	startErr := fixture.tlsIssuer.Start()
	secondStartErr := fixture.tlsIssuer.Start()

	assert.NoError(t, startErr)
	assert.NoError(t, secondStartErr)
	fixture.assertMockExpectations(t)
}

func TestLocalScannerTLSIssuerFetchSensorDeploymentOwnerRefErrorStartFailure(t *testing.T) {
	testCases := map[string]struct {
		testK8sClientConfig fakeK8sClientConfig
	}{
		"sensor replica set missing": {testK8sClientConfig: fakeK8sClientConfig{skipSensorReplicaSet: true}},
		"sensor pod missing":         {testK8sClientConfig: fakeK8sClientConfig{skipSensorPod: true}},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			fixture := newLocalScannerTLSIssuerFixture(tc.testK8sClientConfig)
			fixture.certRefresher.On("Stop").Once()
			fixture.certRequester.On("Stop").Once()

			startErr := fixture.tlsIssuer.Start()

			assert.NoError(t, startErr)
			fixture.assertMockExpectations(t)
		})
	}
}

func TestLocalScannerTLSIssuerWrongManagerTypeStartNoop(t *testing.T) {
	testCases := map[string]struct {
		sensorManagedBy storage.ManagerType
	}{
		"bundle installations":      {sensorManagedBy: storage.ManagerType_MANAGER_TYPE_MANUAL},
		"unknown installation type": {sensorManagedBy: storage.ManagerType_MANAGER_TYPE_UNKNOWN},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
			fixture.tlsIssuer.sensorManagedBy = tc.sensorManagedBy

			startErr := fixture.tlsIssuer.Start()

			assert.NoError(t, startErr)
			assert.Nil(t, fixture.tlsIssuer.certRefresher)
			fixture.assertMockExpectations(t)
		})
	}
}

func TestLocalScannerTLSIssuerProcessMessageKnownMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	processMessageDoneSignal := concurrency.NewErrorSignal()
	fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
	expectedResponse := &central.IssueLocalScannerCertsResponse{
		RequestId: uuid.NewDummy().String(),
	}
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueLocalScannerCertsResponse{
			IssueLocalScannerCertsResponse: expectedResponse,
		},
	}

	go func() {
		assert.NoError(t, fixture.tlsIssuer.ProcessMessage(msg))
		processMessageDoneSignal.Signal()
	}()

	select {
	case <-ctx.Done():
		assert.Fail(t, ctx.Err().Error())
	case response := <-fixture.tlsIssuer.msgFromCentralC:
		assert.Equal(t, expectedResponse, response)
	}

	_, ok := processMessageDoneSignal.WaitWithTimeout(100 * time.Millisecond)
	assert.True(t, ok)
}

func TestLocalScannerTLSIssuerProcessMessageUnknownMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	processMessageDoneSignal := concurrency.NewErrorSignal()
	fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ReprocessDeployments{},
	}

	go func() {
		assert.NoError(t, fixture.tlsIssuer.ProcessMessage(msg))
		processMessageDoneSignal.Signal()
	}()

	select {
	case <-ctx.Done():
	case <-fixture.tlsIssuer.msgFromCentralC:
		assert.Fail(t, "unknown message is not ignored")
	}
	_, ok := processMessageDoneSignal.WaitWithTimeout(100 * time.Millisecond)
	assert.True(t, ok)
}

func TestLocalScannerTLSIssuerIntegrationTests(t *testing.T) {
	suite.Run(t, new(localScannerTLSIssueIntegrationTests))
}

type localScannerTLSIssueIntegrationTests struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (s *localScannerTLSIssueIntegrationTests) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *localScannerTLSIssueIntegrationTests) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.envIsolator)
	s.Require().NoError(err)
}

func (s *localScannerTLSIssueIntegrationTests) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *localScannerTLSIssueIntegrationTests) TestHappyPath() {
	testTimeout := 100 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	client := getFakeK8sClient(fakeK8sClientConfig{})
	tlsIssuer := newLocalScannerTLSIssuer(
		s.T(),
		client,
		storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		sensorNamespace,
		sensorPodName,
	)

	err := tlsIssuer.Start()
	defer tlsIssuer.Stop(nil)
	s.Require().NoError(err)
	s.Require().NotNil(tlsIssuer.certRefresher)

	var request *central.MsgFromSensor
	select {
	case request = <-tlsIssuer.ResponsesC():
	case <-ctx.Done():
		s.Require().Fail(ctx.Err().Error())
	}

	s.Require().NotNil(request.GetIssueLocalScannerCertsRequest())
	ca, err := mtls.CAForSigning()
	s.Require().NoError(err)
	// TODO(ROX-9463): use short expiration for testing renewal when ROX-9010 implementing `WithCustomCertLifetime` is merged
	scannerCert, err := issueCertificate(mtls.WithValidityExpiringInHours())
	s.Require().NoError(err)
	scannerDBCert, err := issueCertificate(mtls.WithValidityExpiringInHours())
	s.Require().NoError(err)
	response := &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueLocalScannerCertsResponse{
			IssueLocalScannerCertsResponse: &central.IssueLocalScannerCertsResponse{
				RequestId: request.GetIssueLocalScannerCertsRequest().GetRequestId(),
				Response: &central.IssueLocalScannerCertsResponse_Certificates{
					Certificates: &storage.TypedServiceCertificateSet{
						CaPem: ca.CertPEM(),
						ServiceCerts: []*storage.TypedServiceCertificate{
							{
								ServiceType: storage.ServiceType_SCANNER_SERVICE,
								Cert: &storage.ServiceCertificate{
									KeyPem:  scannerCert.KeyPEM,
									CertPem: scannerCert.CertPEM,
								},
							},
							{
								ServiceType: storage.ServiceType_SCANNER_DB_SERVICE,
								Cert: &storage.ServiceCertificate{
									KeyPem:  scannerDBCert.KeyPEM,
									CertPem: scannerDBCert.CertPEM,
								},
							},
						},
					},
				},
			},
		},
	}
	err = tlsIssuer.ProcessMessage(response)
	s.Require().NoError(err)

	var secrets *v1.SecretList
	ok := concurrency.PollWithTimeout(func() bool {
		secrets, err = client.CoreV1().Secrets(sensorNamespace).List(context.Background(), metav1.ListOptions{})
		s.Require().NoError(err)
		return len(secrets.Items) == 2
	}, 10*time.Millisecond, testTimeout)
	s.Require().True(ok)
	for _, secret := range secrets.Items {
		var expectedCert *mtls.IssuedCert
		switch secretName := secret.GetName(); secretName {
		case "scanner-slim-tls":
			expectedCert = scannerCert
		case "scanner-db-slim-tls":
			expectedCert = scannerDBCert
		default:
			s.Require().Failf("expected secret name should be either %q or %q, found %q instead",
				"scanner-slim-tls", "scanner-db-slim-tls", secretName)
		}
		s.Equal(ca.CertPEM(), secret.Data[mtls.CACertFileName])
		s.Equal(expectedCert.CertPEM, secret.Data[mtls.ServiceCertFileName])
		s.Equal(expectedCert.KeyPEM, secret.Data[mtls.ServiceKeyFileName])
	}
}

func getFakeK8sClient(conf fakeK8sClientConfig) *fake.Clientset {
	objects := make([]runtime.Object, 0)
	if !conf.skipSensorReplicaSet {
		sensorDeploymentGVK := sensorDeployment.GroupVersionKind()
		sensorReplicaSet := &appsApiv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensorReplicasetName,
				Namespace: sensorNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: sensorDeploymentGVK.GroupVersion().String(),
						Kind:       sensorDeploymentGVK.Kind,
						Name:       sensorDeployment.GetName(),
						UID:        sensorDeployment.GetUID(),
					},
				},
			},
		}
		objects = append(objects, sensorReplicaSet)

		if !conf.skipSensorPod {
			sensorReplicaSetGVK := sensorReplicaSet.GroupVersionKind()
			sensorPod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sensorPodName,
					Namespace: sensorNamespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: sensorReplicaSetGVK.GroupVersion().String(),
							Kind:       sensorReplicaSet.Kind,
							Name:       sensorReplicaSet.GetName(),
							UID:        sensorReplicaSet.GetUID(),
						},
					},
				},
			}
			objects = append(objects, sensorPod)
		}
	}

	k8sClient := fake.NewSimpleClientset(objects...)

	return k8sClient
}

type fakeK8sClientConfig struct {
	// if true then no sensor replica set and no sensor pod will be added to the test client.
	skipSensorReplicaSet bool
	// if true then no sensor pod set will be added to the test client.
	skipSensorPod bool
}

func newLocalScannerTLSIssuer(
	t *testing.T,
	k8sClient kubernetes.Interface,
	sensorManagedBy storage.ManagerType,
	sensorNamespace string,
	sensorPodName string,
) *localScannerTLSIssuerImpl {
	tlsIssuer := NewLocalScannerTLSIssuer(k8sClient, sensorManagedBy, sensorNamespace, sensorPodName)
	require.IsType(t, &localScannerTLSIssuerImpl{}, tlsIssuer)
	return tlsIssuer.(*localScannerTLSIssuerImpl)
}

type certificateRequesterMock struct {
	mock.Mock
}

func (m *certificateRequesterMock) Start() {
	m.Called()
}
func (m *certificateRequesterMock) Stop() {
	m.Called()
}
func (m *certificateRequesterMock) RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*central.IssueLocalScannerCertsResponse), args.Error(1)
}

type certificateRefresherMock struct {
	mock.Mock
}

func (m *certificateRefresherMock) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *certificateRefresherMock) Stop() {
	m.Called()
}

type componentGetterMock struct {
	mock.Mock
}

func (m *componentGetterMock) getCertificateRefresher(requestCertificates requestCertificatesFunc,
	repository serviceCertificatesRepo, timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker {
	args := m.Called(requestCertificates, repository, timeout, backoff)
	return args.Get(0).(concurrency.RetryTicker)
}

func (m *componentGetterMock) getServiceCertificatesRepo(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) serviceCertificatesRepo {
	args := m.Called(ownerReference, namespace, secretsClient)
	return args.Get(0).(serviceCertificatesRepo)
}

type certsRepoMock struct {
	mock.Mock
}

func (m *certsRepoMock) getServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
	args := m.Called(ctx)
	return args.Get(0).(*storage.TypedServiceCertificateSet), args.Error(1)
}

func (m *certsRepoMock) ensureServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) error {
	args := m.Called(ctx, certificates)
	return args.Error(0)
}
