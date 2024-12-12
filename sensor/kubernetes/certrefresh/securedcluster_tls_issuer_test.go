package certrefresh

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/mtls"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certificates"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	sensorSecretName           = "tls-cert-sensor"             // #nosec G101 not a hardcoded credential
	collectorSecretName        = "tls-cert-collector"          // #nosec G101 not a hardcoded credential
	admissionControlSecretName = "tls-cert-admission-control"  // #nosec G101 not a hardcoded credential
	scannerSecretName          = "tls-cert-scanner"            // #nosec G101 not a hardcoded credential
	scannerDbSecretName        = "tls-cert-scanner-db"         // #nosec G101 not a hardcoded credential
	scannerV4IndexerSecretName = "tls-cert-scanner-v4-indexer" // #nosec G101 not a hardcoded credential
	scannerV4DbSecretName      = "tls-cert-scanner-v4-db"      // #nosec G101 not a hardcoded credential
)

type securedClusterTLSIssuerFixture struct {
	k8sClient       *fake.Clientset
	certRequester   *certificateRequesterMock
	certRefresher   *certificateRefresherMock
	repo            *certsRepoMock
	componentGetter *componentGetterMock
	tlsIssuer       *securedClusterTLSIssuerImpl
}

func newSecuredClusterTLSIssuerFixture(k8sClientConfig fakeK8sClientConfig) *securedClusterTLSIssuerFixture {
	fixture := &securedClusterTLSIssuerFixture{
		certRequester:   &certificateRequesterMock{},
		certRefresher:   &certificateRefresherMock{},
		repo:            &certsRepoMock{},
		componentGetter: &componentGetterMock{},
		k8sClient:       getFakeK8sClient(k8sClientConfig),
	}
	fixture.tlsIssuer = &securedClusterTLSIssuerImpl{
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    fixture.k8sClient,
		certRefreshBackoff:           certRefreshBackoff,
		getCertificateRefresherFn:    fixture.componentGetter.getCertificateRefresher,
		getServiceCertificatesRepoFn: fixture.componentGetter.getServiceCertificatesRepo,
		certRequester:                fixture.certRequester,
	}

	return fixture
}

func (f *securedClusterTLSIssuerFixture) assertMockExpectations(t *testing.T) {
	f.certRequester.AssertExpectations(t)
	f.certRequester.AssertExpectations(t)
	f.componentGetter.AssertExpectations(t)
}

// mockForStart setups the mocks for the happy path of Start
func (f *securedClusterTLSIssuerFixture) mockForStart(conf mockForStartConfig) {
	f.certRefresher.On("Start").Once().Return(conf.refresherStartErr)

	f.repo.On("GetServiceCertificates", mock.Anything).Once().
		Return((*storage.TypedServiceCertificateSet)(nil), conf.getCertsErr)

	f.componentGetter.On("getServiceCertificatesRepo", mock.Anything,
		mock.Anything, mock.Anything).Once().Return(f.repo, nil)

	f.componentGetter.On("getCertificateRefresher", "secured cluster certificates", mock.Anything, f.repo,
		certRefreshTimeout, certRefreshBackoff).Once().Return(f.certRefresher)
}

func TestSecuredClusterTLSIssuerStartStopSuccess(t *testing.T) {
	testCases := map[string]struct {
		getCertsErr error
	}{
		"no error":            {getCertsErr: nil},
		"missing secret data": {getCertsErr: errors.Wrap(certrepo.ErrMissingSecretData, "wrap error")},
		"inconsistent CAs":    {getCertsErr: errors.Wrap(certrepo.ErrDifferentCAForDifferentServiceTypes, "wrap error")},
		"missing secret":      {getCertsErr: k8sErrors.NewNotFound(schema.GroupResource{Group: "Core", Resource: "Secret"}, "scanner-db-slim-tls")},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
			fixture.mockForStart(mockForStartConfig{getCertsErr: tc.getCertsErr})
			fixture.certRefresher.On("Stop").Once()

			startErr := fixture.tlsIssuer.Start()
			fixture.tlsIssuer.Stop(nil)

			assert.NoError(t, startErr)
			assert.Nil(t, fixture.tlsIssuer.certRefresher)
			fixture.assertMockExpectations(t)
		})
	}
}

func TestSecuredClusterTLSIssuerRefresherFailureStartFailure(t *testing.T) {
	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.mockForStart(mockForStartConfig{refresherStartErr: errForced})
	fixture.certRefresher.On("Stop").Once()

	startErr := fixture.tlsIssuer.Start()

	require.Error(t, startErr)
	fixture.assertMockExpectations(t)
}

func TestSecuredClusterTLSIssuerStartAlreadyStartedFailure(t *testing.T) {
	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.mockForStart(mockForStartConfig{})
	fixture.certRefresher.On("Stop").Once()

	startErr := fixture.tlsIssuer.Start()
	secondStartErr := fixture.tlsIssuer.Start()

	assert.NoError(t, startErr)
	require.Error(t, secondStartErr)
	fixture.assertMockExpectations(t)
}

func TestSecuredClusterTLSIssuerFetchSensorDeploymentOwnerRefErrorStartFailure(t *testing.T) {
	testCases := map[string]struct {
		k8sClientConfig fakeK8sClientConfig
	}{
		"sensor replica set missing": {k8sClientConfig: fakeK8sClientConfig{skipSensorReplicaSet: true}},
		"sensor pod missing":         {k8sClientConfig: fakeK8sClientConfig{skipSensorPod: true}},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			fixture := newSecuredClusterTLSIssuerFixture(tc.k8sClientConfig)
			fixture.certRefresher.On("Stop").Once()

			startErr := fixture.tlsIssuer.Start()

			require.Error(t, startErr)
			fixture.assertMockExpectations(t)
		})
	}
}

func TestSecuredClusterTLSIssuerProcessMessageKnownMessage(t *testing.T) {
	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	expectedResponse := &central.IssueSecuredClusterCertsResponse{
		RequestId: uuid.NewDummy().String(),
	}
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueSecuredClusterCertsResponse{
			IssueSecuredClusterCertsResponse: expectedResponse,
		},
	}

	done := make(chan struct{})
	fixture.certRequester.On("DispatchResponse",
		certificates.NewResponseFromSecuredClusterCerts(expectedResponse)).Run(func(args mock.Arguments) {
		close(done)
	}).Once().Return()

	assert.NoError(t, fixture.tlsIssuer.ProcessMessage(msg))

	select {
	case <-done:
		fixture.certRequester.AssertExpectations(t)
	case <-time.After(5 * time.Second):
		t.Fatalf("Test timed out waiting for DispatchResponse to be called")
	}
}

func TestSecuredClusterTLSIssuerProcessMessageUnknownMessage(t *testing.T) {
	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ReprocessDeployments{},
	}

	assert.NoError(t, fixture.tlsIssuer.ProcessMessage(msg))

	time.Sleep(100 * time.Millisecond)
	fixture.certRequester.AssertNotCalled(t, "DispatchResponse", mock.Anything)
}

func TestSecuredClusterTLSIssuerIntegrationTests(t *testing.T) {
	suite.Run(t, new(securedClusterTLSIssueIntegrationTests))
}

type securedClusterTLSIssueIntegrationTests struct {
	suite.Suite
}

func (s *securedClusterTLSIssueIntegrationTests) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
}

func (s *securedClusterTLSIssueIntegrationTests) TestSuccessfulRefresh() {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.SecuredClusterCertificatesReissue})
	defer func() {
		centralcaps.Set([]centralsensor.CentralCapability{})
	}()

	testCases := map[string]struct {
		k8sClientConfig    fakeK8sClientConfig
		numFailedResponses int
	}{
		"no secrets": {k8sClientConfig: fakeK8sClientConfig{}},
		"corrupted data in sensor secret": {
			k8sClientConfig: fakeK8sClientConfig{
				secretsData: map[string]map[string][]byte{sensorSecretName: nil},
			},
		},
		"corrupted data in scanner DB secret": {
			k8sClientConfig: fakeK8sClientConfig{
				secretsData: map[string]map[string][]byte{scannerDbSecretName: nil},
			},
		},
		"corrupted data in all secured cluster secrets": {
			k8sClientConfig: fakeK8sClientConfig{
				secretsData: map[string]map[string][]byte{
					sensorSecretName:           nil,
					collectorSecretName:        nil,
					admissionControlSecretName: nil,
					scannerSecretName:          nil,
					scannerDbSecretName:        nil,
					scannerV4IndexerSecretName: nil,
					scannerV4DbSecretName:      nil,
				},
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

			secretsCerts := map[string]*mtls.IssuedCert{
				sensorSecretName:           s.getCertificate(storage.ServiceType_SENSOR_SERVICE),
				collectorSecretName:        s.getCertificate(storage.ServiceType_COLLECTOR_SERVICE),
				admissionControlSecretName: s.getCertificate(storage.ServiceType_ADMISSION_CONTROL_SERVICE),
				scannerSecretName:          s.getCertificate(storage.ServiceType_SCANNER_SERVICE),
				scannerDbSecretName:        s.getCertificate(storage.ServiceType_SCANNER_DB_SERVICE),
				scannerV4IndexerSecretName: s.getCertificate(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE),
				scannerV4DbSecretName:      s.getCertificate(storage.ServiceType_SCANNER_V4_DB_SERVICE),
			}

			k8sClient := getFakeK8sClient(tc.k8sClientConfig)
			tlsIssuer := newSecuredClusterTLSIssuer(s.T(), k8sClient, sensorNamespace, sensorPodName)
			tlsIssuer.certRefreshBackoff = wait.Backoff{
				Duration: time.Millisecond,
			}

			s.Require().NoError(tlsIssuer.Start())
			defer tlsIssuer.Stop(nil)
			s.Require().NotNil(tlsIssuer.certRefresher)
			s.Require().False(tlsIssuer.certRefresher.Stopped())

			for i := 0; i < tc.numFailedResponses; i++ {
				request := s.waitForRequest(ctx, tlsIssuer)
				response := getSecuredClusterIssueCertsFailureResponse(request.GetRequestId())
				err = tlsIssuer.ProcessMessage(response)
				s.Require().NoError(err)
			}

			request := s.waitForRequest(ctx, tlsIssuer)
			response := getSecuredClusterIssueCertsSuccessResponse(request.GetRequestId(), ca.CertPEM(), secretsCerts)
			err = tlsIssuer.ProcessMessage(response)
			s.Require().NoError(err)

			var secrets *v1.SecretList
			ok := concurrency.PollWithTimeout(func() bool {
				secrets, err = k8sClient.CoreV1().Secrets(sensorNamespace).List(context.Background(), metav1.ListOptions{})
				s.Require().NoError(err)

				allSecretsHaveData := true
				for _, secret := range secrets.Items {
					if len(secret.Data) == 0 {
						allSecretsHaveData = false
						break
					}
				}
				return allSecretsHaveData && len(secrets.Items) == len(secretsCerts)
			}, 10*time.Millisecond, testTimeout)
			s.Require().True(ok, "expected exactly %d secrets with non-empty data available in the k8s API", len(secretsCerts))

			for _, secret := range secrets.Items {
				expectedCert, exists := secretsCerts[secret.GetName()]
				if !exists {
					s.Require().Failf("unexpected secret name %q", secret.GetName())
					continue
				}
				s.Equal(ca.CertPEM(), secret.Data[mtls.CACertFileName])
				s.Equal(expectedCert.CertPEM, secret.Data[mtls.ServiceCertFileName])
				s.Equal(expectedCert.KeyPEM, secret.Data[mtls.ServiceKeyFileName])
			}
		})
	}
}

func (s *securedClusterTLSIssueIntegrationTests) TestUnexpectedOwnerStop() {
	testCases := map[string]struct {
		secretNames []string
	}{
		"wrong owner for sensor secret":                  {secretNames: []string{sensorSecretName}},
		"wrong owner for collector secret":               {secretNames: []string{collectorSecretName}},
		"wrong owner for admission controller secret":    {secretNames: []string{admissionControlSecretName}},
		"wrong owner for scanner secret":                 {secretNames: []string{scannerSecretName}},
		"wrong owner for scanner db secret":              {secretNames: []string{scannerDbSecretName}},
		"wrong owner for scanner v4 indexer secret":      {secretNames: []string{scannerV4IndexerSecretName}},
		"wrong owner for scanner v4 db secret":           {secretNames: []string{scannerV4DbSecretName}},
		"wrong owner for scanner and scanner db secrets": {secretNames: []string{scannerSecretName, scannerDbSecretName}},
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
			tlsIssuer := newSecuredClusterTLSIssuer(s.T(), k8sClient, sensorNamespace, sensorPodName)

			s.Require().NoError(tlsIssuer.Start())
			defer tlsIssuer.Stop(nil)

			ok := concurrency.PollWithTimeout(func() bool {
				return tlsIssuer.certRefresher != nil && tlsIssuer.certRefresher.Stopped()
			}, 10*time.Millisecond, 100*time.Millisecond)
			s.True(ok, "cert refresher should be stopped")
		})
	}
}

func (s *securedClusterTLSIssueIntegrationTests) getCertificate(serviceType storage.ServiceType) *mtls.IssuedCert {
	cert, err := issueCertificate(serviceType, mtls.WithValidityExpiringInHours())
	s.Require().NoError(err)
	return cert
}

func (s *securedClusterTLSIssueIntegrationTests) waitForRequest(ctx context.Context, tlsIssuer common.SensorComponent) *central.IssueSecuredClusterCertsRequest {
	var request *message.ExpiringMessage
	select {
	case request = <-tlsIssuer.ResponsesC():
	case <-ctx.Done():
		s.Require().Fail(ctx.Err().Error())
	}
	s.Require().NotNil(request.GetIssueSecuredClusterCertsRequest())

	return request.GetIssueSecuredClusterCertsRequest()
}

func getSecuredClusterIssueCertsSuccessResponse(
	requestID string,
	caPem []byte,
	secretsCerts map[string]*mtls.IssuedCert,
) *central.MsgToSensor {
	serviceTypeMap := map[string]storage.ServiceType{
		sensorSecretName:           storage.ServiceType_SENSOR_SERVICE,
		collectorSecretName:        storage.ServiceType_COLLECTOR_SERVICE,
		admissionControlSecretName: storage.ServiceType_ADMISSION_CONTROL_SERVICE,
		scannerSecretName:          storage.ServiceType_SCANNER_SERVICE,
		scannerDbSecretName:        storage.ServiceType_SCANNER_DB_SERVICE,
		scannerV4IndexerSecretName: storage.ServiceType_SCANNER_V4_INDEXER_SERVICE,
		scannerV4DbSecretName:      storage.ServiceType_SCANNER_V4_DB_SERVICE,
	}

	var serviceCerts []*storage.TypedServiceCertificate
	for secretName, cert := range secretsCerts {
		serviceType, exists := serviceTypeMap[secretName]
		if !exists {
			continue
		}
		serviceCerts = append(serviceCerts, &storage.TypedServiceCertificate{
			ServiceType: serviceType,
			Cert: &storage.ServiceCertificate{
				KeyPem:  cert.KeyPEM,
				CertPem: cert.CertPEM,
			},
		})
	}

	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueSecuredClusterCertsResponse{
			IssueSecuredClusterCertsResponse: &central.IssueSecuredClusterCertsResponse{
				RequestId: requestID,
				Response: &central.IssueSecuredClusterCertsResponse_Certificates{
					Certificates: &storage.TypedServiceCertificateSet{
						CaPem:        caPem,
						ServiceCerts: serviceCerts,
					},
				},
			},
		},
	}
}

func getSecuredClusterIssueCertsFailureResponse(requestID string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueSecuredClusterCertsResponse{
			IssueSecuredClusterCertsResponse: &central.IssueSecuredClusterCertsResponse{
				RequestId: requestID,
				Response: &central.IssueSecuredClusterCertsResponse_Error{
					Error: &central.SecuredClusterCertsIssueError{
						Message: "forced error",
					},
				},
			},
		},
	}
}

func newSecuredClusterTLSIssuer(
	t *testing.T,
	k8sClient kubernetes.Interface,
	sensorNamespace string,
	sensorPodName string,
) *securedClusterTLSIssuerImpl {
	tlsIssuer := NewSecuredClusterTLSIssuer(k8sClient, sensorNamespace, sensorPodName)
	require.IsType(t, &securedClusterTLSIssuerImpl{}, tlsIssuer)
	return tlsIssuer.(*securedClusterTLSIssuerImpl)
}
