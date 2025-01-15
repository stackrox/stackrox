package certrefresh

import (
	"context"
	"sync/atomic"
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
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	k8sClient            *fake.Clientset
	certRefresher        *certificateRefresherMock
	repo                 *certsRepoMock
	componentGetter      *componentGetterMock
	tlsIssuer            *tlsIssuerImpl
	interceptedRequestID atomic.Value
}

func newSecuredClusterTLSIssuerFixture(k8sClientConfig fakeK8sClientConfig) *securedClusterTLSIssuerFixture {
	fixture := &securedClusterTLSIssuerFixture{
		certRefresher:   &certificateRefresherMock{},
		repo:            &certsRepoMock{},
		componentGetter: &componentGetterMock{},
		k8sClient:       getFakeK8sClient(k8sClientConfig),
	}
	fixture.tlsIssuer = &tlsIssuerImpl{
		componentName:                securedClusterComponentName,
		sensorCapability:             securedClusterSensorCapability,
		getResponseFn:                securedClusterResponseFn,
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    fixture.k8sClient,
		certRefreshBackoff:           certRefreshBackoff,
		getCertificateRefresherFn:    fixture.componentGetter.getCertificateRefresher,
		getServiceCertificatesRepoFn: fixture.componentGetter.getServiceCertificatesRepo,
		stopSig:                      concurrency.NewErrorSignal(),
		msgToCentralC:                make(chan *message.ExpiringMessage),
		newMsgFromSensorFn:           newSecuredClusterMsgFromSensor,
		responseReceived:             concurrency.NewSignal(),
		requiredCentralCapability:    nil,
	}

	return fixture
}

func (f *securedClusterTLSIssuerFixture) assertMockExpectations(t *testing.T) {
	f.componentGetter.AssertExpectations(t)
}

// mockForStart setups the mocks for the happy path of Start
func (f *securedClusterTLSIssuerFixture) mockForStart(conf mockForStartConfig) {
	f.certRefresher.On("Start").Once().Return(conf.refresherStartErr)

	f.repo.On("GetServiceCertificates", mock.Anything).Once().
		Return((*storage.TypedServiceCertificateSet)(nil), conf.getCertsErr)

	f.componentGetter.On("getServiceCertificatesRepo", mock.Anything,
		mock.Anything, mock.Anything).Once().Return(f.repo, nil)

	f.componentGetter.On("getCertificateRefresher", securedClusterComponentName, mock.Anything, f.repo,
		certRefreshTimeout, certRefreshBackoff).Once().Return(f.certRefresher)
}

// respondRequest reads a request from `f.tlsIssuer.MsgToCentralC` and responds with `responseOverwrite` if not nil,
// or with a response with the same ID as the request otherwise.
// Before sending the response, it stores in `f.interceptedRequestID` the ID of the request.
func (f *securedClusterTLSIssuerFixture) respondRequest(
	ctx context.Context, t *testing.T,
	responseOverwrite *central.IssueSecuredClusterCertsResponse) {
	select {
	case <-ctx.Done():
	case request := <-f.tlsIssuer.msgToCentralC:
		interceptedRequestID := request.GetIssueSecuredClusterCertsRequest().GetRequestId()
		assert.NotEmpty(t, interceptedRequestID)
		var response *central.IssueSecuredClusterCertsResponse
		if responseOverwrite != nil {
			response = responseOverwrite
		} else {
			response = &central.IssueSecuredClusterCertsResponse{RequestId: interceptedRequestID}
		}
		f.interceptedRequestID.Store(response.GetRequestId())
		f.tlsIssuer.dispatch(NewResponseFromSecuredClusterCerts(response))
	}
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

	fixture.tlsIssuer.ongoingRequestID = expectedResponse.RequestId
	fixture.tlsIssuer.requestOngoing.Store(true)

	assert.NoError(t, fixture.tlsIssuer.ProcessMessage(msg))
	assert.Eventually(t, func() bool {
		return fixture.tlsIssuer.responseReceived.IsDone()
	}, 2*time.Second, 100*time.Millisecond)
}

func TestSecuredClusterTLSIssuerProcessMessageUnknownMessage(t *testing.T) {
	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ReprocessDeployments{},
	}

	assert.NoError(t, fixture.tlsIssuer.ProcessMessage(msg))
	assert.Never(t, func() bool {
		return fixture.tlsIssuer.responseReceived.IsDone()
	}, 200*time.Millisecond, 50*time.Millisecond)
}

func TestSecuredClusterTLSIssuerRequestCancellation(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.SecuredClusterCertificatesReissue})
	f := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	cancel()

	certs, requestErr := f.tlsIssuer.requestCertificates(ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.Canceled, requestErr)
}

func TestSecuredClusterTLSIssuerRequestSuccess(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.SecuredClusterCertificatesReissue})
	f := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go f.respondRequest(ctx, t, nil)

	response, err := f.tlsIssuer.requestCertificates(ctx)
	require.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.RequestId)
	oldRequestId := response.RequestId

	// Check that a second call also works
	go f.respondRequest(ctx, t, nil)

	response, err = f.tlsIssuer.requestCertificates(ctx)
	assert.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.RequestId)
	assert.NotEqual(t, oldRequestId, response.RequestId)
}

func TestSecuredClusterTLSIssuerResponsesWithUnknownIDAreIgnored(t *testing.T) {
	f := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	response := &central.IssueSecuredClusterCertsResponse{RequestId: "UNKNOWN"}
	// Request with different request ID should be ignored.
	go f.respondRequest(ctx, t, response)

	certs, requestErr := f.tlsIssuer.requestCertificates(ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
}

func TestSecuredClusterCertificateRequesterNoReplyFromCentral(t *testing.T) {
	f := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	certs, requestErr := f.tlsIssuer.requestCertificates(ctx)

	// No response was set using `f.respondRequest`, which simulates not receiving a reply from Central
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
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

			secretsCerts := getAllSecuredClusterCertificates(s.T())

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

			verifySecrets(ctx, s.T(), k8sClient, sensorNamespace, ca, secretsCerts)
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

func getAllSecuredClusterCertificates(t require.TestingT) map[string]*mtls.IssuedCert {
	return map[string]*mtls.IssuedCert{
		sensorSecretName:           getCertificate(t, storage.ServiceType_SENSOR_SERVICE),
		collectorSecretName:        getCertificate(t, storage.ServiceType_COLLECTOR_SERVICE),
		admissionControlSecretName: getCertificate(t, storage.ServiceType_ADMISSION_CONTROL_SERVICE),
		scannerSecretName:          getCertificate(t, storage.ServiceType_SCANNER_SERVICE),
		scannerDbSecretName:        getCertificate(t, storage.ServiceType_SCANNER_DB_SERVICE),
		scannerV4IndexerSecretName: getCertificate(t, storage.ServiceType_SCANNER_V4_INDEXER_SERVICE),
		scannerV4DbSecretName:      getCertificate(t, storage.ServiceType_SCANNER_V4_DB_SERVICE),
	}
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
) *tlsIssuerImpl {
	tlsIssuer := NewSecuredClusterTLSIssuer(k8sClient, sensorNamespace, sensorPodName)
	require.IsType(t, &tlsIssuerImpl{}, tlsIssuer)
	return tlsIssuer.(*tlsIssuerImpl)
}
