package localscanner

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/mtls"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
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
	"k8s.io/apimachinery/pkg/types"
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
	msgToCentralC := make(chan *message.ExpiringMessage)
	msgFromCentralC := make(chan *central.IssueLocalScannerCertsResponse)
	fixture.tlsIssuer = &localScannerTLSIssuerImpl{
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    fixture.k8sClient,
		msgToCentralC:                msgToCentralC,
		msgFromCentralC:              msgFromCentralC,
		certRefreshBackoff:           certRefreshBackoff,
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
		"missing secret data": {getCertsErr: errors.Wrap(ErrMissingSecretData, "wrap error")},
		"inconsistent CAs":    {getCertsErr: errors.Wrap(ErrDifferentCAForDifferentServiceTypes, "wrap error")},
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

	require.Error(t, startErr)
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
	require.Error(t, secondStartErr)
	fixture.assertMockExpectations(t)
}

func TestLocalScannerTLSIssuerFetchSensorDeploymentOwnerRefErrorStartFailure(t *testing.T) {
	testCases := map[string]struct {
		k8sClientConfig fakeK8sClientConfig
	}{
		"sensor replica set missing": {k8sClientConfig: fakeK8sClientConfig{skipSensorReplicaSet: true}},
		"sensor pod missing":         {k8sClientConfig: fakeK8sClientConfig{skipSensorPod: true}},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			fixture := newLocalScannerTLSIssuerFixture(tc.k8sClientConfig)
			fixture.certRefresher.On("Stop").Once()
			fixture.certRequester.On("Stop").Once()

			startErr := fixture.tlsIssuer.Start()

			require.Error(t, startErr)
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
}

func (s *localScannerTLSIssueIntegrationTests) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
}

func (s *localScannerTLSIssueIntegrationTests) TestSuccessfulRefresh() {
	testCases := map[string]struct {
		k8sClientConfig    fakeK8sClientConfig
		numFailedResponses int
	}{
		"no secrets": {k8sClientConfig: fakeK8sClientConfig{}},
		"corrupted data in scanner secret": {
			k8sClientConfig: fakeK8sClientConfig{
				secretsData: map[string]map[string][]byte{"scanner-tls": nil},
			},
		},
		"corrupted data in scanner DB secret": {
			k8sClientConfig: fakeK8sClientConfig{
				secretsData: map[string]map[string][]byte{"scanner-db-tls": nil},
			},
		},
		"corrupted data in all local scanner secrets": {
			k8sClientConfig: fakeK8sClientConfig{
				secretsData: map[string]map[string][]byte{"scanner-tls": nil, "scanner-db-tls": nil},
			},
		},
		"refresh failure and retries": {k8sClientConfig: fakeK8sClientConfig{}, numFailedResponses: 2},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			testTimeout := 2 * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()
			ca, err := mtls.CAForSigning()
			s.Require().NoError(err)
			scannerCert := s.getCertificate(storage.ServiceType_SCANNER_SERVICE)
			scannerDBCert := s.getCertificate(storage.ServiceType_SCANNER_DB_SERVICE)
			k8sClient := getFakeK8sClient(tc.k8sClientConfig)
			tlsIssuer := newLocalScannerTLSIssuer(s.T(), k8sClient, sensorNamespace, sensorPodName)
			tlsIssuer.certRefreshBackoff = wait.Backoff{
				Duration: time.Millisecond,
			}

			s.Require().NoError(tlsIssuer.Start())
			defer tlsIssuer.Stop(nil)
			s.Require().NotNil(tlsIssuer.certRefresher)
			s.Require().False(tlsIssuer.certRefresher.Stopped())

			for i := 0; i < tc.numFailedResponses; i++ {
				request := s.waitForRequest(ctx, tlsIssuer)
				response := getIssueCertsFailureResponse(request.GetRequestId())
				err = tlsIssuer.ProcessMessage(response)
				s.Require().NoError(err)
			}

			request := s.waitForRequest(ctx, tlsIssuer)
			response := getIssueCertsSuccessResponse(request.GetRequestId(), ca.CertPEM(), scannerCert, scannerDBCert)
			err = tlsIssuer.ProcessMessage(response)
			s.Require().NoError(err)

			var secrets *v1.SecretList
			ok := concurrency.PollWithTimeout(func() bool {
				secrets, err = k8sClient.CoreV1().Secrets(sensorNamespace).List(context.Background(), metav1.ListOptions{})
				s.Require().NoError(err)
				return len(secrets.Items) == 2 && len(secrets.Items[0].Data) > 0 && len(secrets.Items[1].Data) > 0
			}, 10*time.Millisecond, testTimeout)
			s.Require().True(ok, "expected exactly 2 secrets with non-empty data available in the k8s API")
			for _, secret := range secrets.Items {
				var expectedCert *mtls.IssuedCert
				switch secretName := secret.GetName(); secretName {
				case "scanner-tls":
					expectedCert = scannerCert
				case "scanner-db-tls":
					expectedCert = scannerDBCert
				default:
					s.Require().Failf("expected secret name should be either %q or %q, found %q instead",
						"scanner-tls", "scanner-db-tls", secretName)
				}
				s.Equal(ca.CertPEM(), secret.Data[mtls.CACertFileName])
				s.Equal(expectedCert.CertPEM, secret.Data[mtls.ServiceCertFileName])
				s.Equal(expectedCert.KeyPEM, secret.Data[mtls.ServiceKeyFileName])
			}
		})
	}
}

func (s *localScannerTLSIssueIntegrationTests) TestUnexpectedOwnerStop() {
	testCases := map[string]struct {
		secretNames []string
	}{
		"wrong owner for scanner secret":                 {secretNames: []string{"scanner-tls"}},
		"wrong owner for scanner db secret":              {secretNames: []string{"scanner-db-tls"}},
		"wrong owner for scanner and scanner db secrets": {secretNames: []string{"scanner-tls", "scanner-db-tls"}},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			secretsData := make(map[string]map[string][]byte, len(tc.secretNames))
			for _, secretName := range tc.secretNames {
				secretsData[secretName] = nil
			}
			k8sClient := getFakeK8sClient(fakeK8sClientConfig{
				secretsData: secretsData,
				secretsOwner: &metav1.OwnerReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "another-deployment",
					UID:        types.UID(uuid.NewDummy().String()),
				},
			})
			tlsIssuer := newLocalScannerTLSIssuer(s.T(), k8sClient, sensorNamespace, sensorPodName)

			s.Require().NoError(tlsIssuer.Start())
			defer tlsIssuer.Stop(nil)

			ok := concurrency.PollWithTimeout(func() bool {
				return tlsIssuer.certRefresher != nil && tlsIssuer.certRefresher.Stopped()
			}, 10*time.Millisecond, 100*time.Millisecond)
			s.True(ok, "cert refresher should be stopped")
		})
	}
}

func (s *localScannerTLSIssueIntegrationTests) getCertificate(serviceType storage.ServiceType) *mtls.IssuedCert {
	// TODO(ROX-9463): use short expiration for testing renewal when ROX-9010 implementing `WithCustomCertLifetime` is merged
	cert, err := issueCertificate(serviceType, mtls.WithValidityExpiringInHours())
	s.Require().NoError(err)
	return cert
}

func (s *localScannerTLSIssueIntegrationTests) waitForRequest(ctx context.Context, tlsIssuer common.SensorComponent) *central.IssueLocalScannerCertsRequest {
	var request *message.ExpiringMessage
	select {
	case request = <-tlsIssuer.ResponsesC():
	case <-ctx.Done():
		s.Require().Fail(ctx.Err().Error())
	}
	s.Require().NotNil(request.GetIssueLocalScannerCertsRequest())

	return request.GetIssueLocalScannerCertsRequest()
}

func getIssueCertsSuccessResponse(requestID string, caPem []byte, scannerCert, scannerDBCert *mtls.IssuedCert) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueLocalScannerCertsResponse{
			IssueLocalScannerCertsResponse: &central.IssueLocalScannerCertsResponse{
				RequestId: requestID,
				Response: &central.IssueLocalScannerCertsResponse_Certificates{
					Certificates: &storage.TypedServiceCertificateSet{
						CaPem: caPem,
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
}

func getIssueCertsFailureResponse(requestID string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueLocalScannerCertsResponse{
			IssueLocalScannerCertsResponse: &central.IssueLocalScannerCertsResponse{
				RequestId: requestID,
				Response: &central.IssueLocalScannerCertsResponse_Error{
					Error: &central.LocalScannerCertsIssueError{
						Message: "forced error",
					},
				},
			},
		},
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

		sensorReplicaSetGVK := sensorReplicaSet.GroupVersionKind()
		sensorReplicaSetOwnerRef := metav1.OwnerReference{
			APIVersion: sensorReplicaSetGVK.GroupVersion().String(),
			Kind:       sensorReplicaSet.Kind,
			Name:       sensorReplicaSet.GetName(),
			UID:        sensorReplicaSet.GetUID(),
		}

		if !conf.skipSensorPod {
			sensorPod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:            sensorPodName,
					Namespace:       sensorNamespace,
					OwnerReferences: []metav1.OwnerReference{sensorReplicaSetOwnerRef},
				},
			}
			objects = append(objects, sensorPod)
		}

		secretsOwnerRef := sensorReplicaSetOwnerRef
		if conf.secretsOwner != nil {
			secretsOwnerRef = *conf.secretsOwner
		}
		for secretName, secretData := range conf.secretsData {
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretName,
					Namespace:       sensorNamespace,
					OwnerReferences: []metav1.OwnerReference{secretsOwnerRef},
				},
				Data: secretData,
			}
			objects = append(objects, secret)
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
	// if skipSensorReplicaSet is false, then a secret will be added to the test client for
	// each entry in this map, using the key as the secret name and the value as the secret data.
	secretsData map[string]map[string][]byte
	// owner reference to used for the secrets specified in `secretsData`. If `nil` then the sensor
	// replica set is used as owner
	secretsOwner *metav1.OwnerReference
}

func newLocalScannerTLSIssuer(
	t *testing.T,
	k8sClient kubernetes.Interface,
	sensorNamespace string,
	sensorPodName string,
) *localScannerTLSIssuerImpl {
	tlsIssuer := NewLocalScannerTLSIssuer(k8sClient, sensorNamespace, sensorPodName)
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
	stopped bool
}

func (m *certificateRefresherMock) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *certificateRefresherMock) Stop() {
	m.Called()
	m.stopped = true
}

func (m *certificateRefresherMock) Stopped() bool {
	return m.stopped
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

func (m *certsRepoMock) ensureServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) ([]*storage.TypedServiceCertificate, error) {
	args := m.Called(ctx, certificates)
	return certificates.ServiceCerts, args.Error(0)
}
