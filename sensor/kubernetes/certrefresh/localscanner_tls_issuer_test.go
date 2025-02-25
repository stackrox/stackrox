package certrefresh

import (
	"context"
	"sync/atomic"
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

// localScannerTLSIssuerFixture tests the legacy Local Scanner certificate refresh feature, which will be deprecated.
// The tests are similar to the ones in securedcluster_tls_issuer_test.go, and improvements to this file should also
// be made there. Expect these tests to be deleted soon.
// TODO(ROX-27734): Remove local scanner certificate refresh from Sensor
type localScannerTLSIssuerFixture struct {
	k8sClient            *fake.Clientset
	certRefresher        *certificateRefresherMock
	repo                 *certsRepoMock
	componentGetter      *componentGetterMock
	tlsIssuer            *tlsIssuerImpl
	interceptedRequestID atomic.Value
}

func newLocalScannerTLSIssuerFixture(k8sClientConfig fakeK8sClientConfig) *localScannerTLSIssuerFixture {
	fixture := &localScannerTLSIssuerFixture{
		certRefresher:   &certificateRefresherMock{},
		repo:            &certsRepoMock{},
		componentGetter: &componentGetterMock{},
		k8sClient:       getFakeK8sClient(k8sClientConfig),
	}
	fixture.tlsIssuer = &tlsIssuerImpl{
		componentName:                localScannerComponentName,
		sensorCapability:             localScannerSensorCapability,
		getResponseFn:                localScannerResponseFn,
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    fixture.k8sClient,
		certRefreshBackoff:           certRefreshBackoff,
		getCertificateRefresherFn:    fixture.componentGetter.getCertificateRefresher,
		getServiceCertificatesRepoFn: fixture.componentGetter.getServiceCertificatesRepo,
		msgToCentralC:                make(chan *message.ExpiringMessage),
		newMsgFromSensorFn:           newLocalScannerMsgFromSensor,
		responseReceived:             concurrency.NewSignal(),
		requiredCentralCapability:    nil,
	}

	return fixture
}

func (f *localScannerTLSIssuerFixture) assertMockExpectations(t *testing.T) {
	f.componentGetter.AssertExpectations(t)
}

// mockForStart setups the mocks for the happy path of Start
func (f *localScannerTLSIssuerFixture) mockForStart(conf mockForStartConfig) {
	f.certRefresher.On("Start", mock.Anything).Once().Return(conf.refresherStartErr)

	f.repo.On("GetServiceCertificates", mock.Anything).Once().
		Return((*storage.TypedServiceCertificateSet)(nil), conf.getCertsErr)

	f.componentGetter.On("getServiceCertificatesRepo", mock.Anything,
		mock.Anything, mock.Anything).Once().Return(f.repo, nil)

	f.componentGetter.On("getCertificateRefresher", localScannerComponentName, mock.Anything, f.repo,
		certRefreshTimeout, certRefreshBackoff).Once().Return(f.certRefresher)
}

// respondRequest reads a request from `f.tlsIssuer.MsgToCentralC` and responds with `responseOverwrite` if not nil,
// or with a response with the same ID as the request otherwise.
// Before sending the response, it stores in `f.interceptedRequestID` the ID of the request.
func (f *localScannerTLSIssuerFixture) respondRequest(
	ctx context.Context, t *testing.T,
	responseOverwrite *central.IssueLocalScannerCertsResponse) {
	select {
	case <-ctx.Done():
	case request := <-f.tlsIssuer.msgToCentralC:
		interceptedRequestID := request.GetIssueLocalScannerCertsRequest().GetRequestId()
		assert.NotEmpty(t, interceptedRequestID)
		var response *central.IssueLocalScannerCertsResponse
		if responseOverwrite != nil {
			response = responseOverwrite
		} else {
			response = &central.IssueLocalScannerCertsResponse{RequestId: interceptedRequestID}
		}
		f.interceptedRequestID.Store(response.GetRequestId())
		f.tlsIssuer.dispatch(NewResponseFromLocalScannerCerts(response))
	}
}

func TestLocalScannerTLSIssuerStartStopSuccess(t *testing.T) {
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
			fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
			fixture.mockForStart(mockForStartConfig{getCertsErr: tc.getCertsErr})
			fixture.certRefresher.On("Stop").Once()

			startErr := fixture.tlsIssuer.Start()
			fixture.tlsIssuer.Notify(common.SensorComponentEventCentralReachable)
			assert.NotNil(t, fixture.tlsIssuer.certRefresher)
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

	fixture.tlsIssuer.Notify(common.SensorComponentEventCentralReachable)
	startErr := fixture.tlsIssuer.Start()

	require.Error(t, startErr)
	fixture.assertMockExpectations(t)
}

func TestLocalScannerTLSIssuerStartAlreadyStarted(t *testing.T) {
	fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.mockForStart(mockForStartConfig{})

	startErr := fixture.tlsIssuer.Start()
	fixture.tlsIssuer.Notify(common.SensorComponentEventCentralReachable)
	secondStartErr := fixture.tlsIssuer.Start()

	require.NoError(t, startErr)
	require.NoError(t, secondStartErr)
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

			fixture.tlsIssuer.Notify(common.SensorComponentEventCentralReachable)
			startErr := fixture.tlsIssuer.Start()

			require.Error(t, startErr)
			fixture.assertMockExpectations(t)
		})
	}
}

func TestLocalScannerTLSIssuerProcessMessageKnownMessage(t *testing.T) {
	fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
	expectedResponse := &central.IssueLocalScannerCertsResponse{
		RequestId: uuid.NewDummy().String(),
	}
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueLocalScannerCertsResponse{
			IssueLocalScannerCertsResponse: expectedResponse,
		},
	}

	fixture.tlsIssuer.ongoingRequestID = expectedResponse.RequestId
	fixture.tlsIssuer.requestOngoing.Store(true)

	assert.NoError(t, fixture.tlsIssuer.ProcessMessage(msg))
	assert.Eventually(t, func() bool {
		return fixture.tlsIssuer.responseReceived.IsDone()
	}, 2*time.Second, 100*time.Millisecond)
}

func TestLocalScannerTLSIssuerProcessMessageUnknownMessage(t *testing.T) {
	fixture := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ReprocessDeployments{},
	}

	assert.NoError(t, fixture.tlsIssuer.ProcessMessage(msg))
	assert.Never(t, func() bool {
		return fixture.tlsIssuer.responseReceived.IsDone()
	}, 200*time.Millisecond, 50*time.Millisecond)
}

func TestLocalScannerTLSIssuerRequestCancellation(t *testing.T) {
	f := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})

	testTimeout := 2 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	cancel()

	certs, requestErr := f.tlsIssuer.requestCertificates(ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.Canceled, requestErr)
}

func TestLocalScannerTLSIssuerRequestSuccess(t *testing.T) {
	f := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
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

func TestLocalScannerTLSIssuerResponsesWithUnknownIDAreIgnored(t *testing.T) {
	f := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	response := &central.IssueLocalScannerCertsResponse{RequestId: "UNKNOWN"}
	// Request with different request ID should be ignored.
	go f.respondRequest(ctx, t, response)

	certs, requestErr := f.tlsIssuer.requestCertificates(ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
}

func TestLocalScannerCertificateRequesterNoReplyFromCentral(t *testing.T) {
	f := newLocalScannerTLSIssuerFixture(fakeK8sClientConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	certs, requestErr := f.tlsIssuer.requestCertificates(ctx)

	// No response was set using `f.respondRequest`, which simulates not receiving a reply from Central
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
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
			scannerCert := getCertificate(s.T(), storage.ServiceType_SCANNER_SERVICE)
			scannerDBCert := getCertificate(s.T(), storage.ServiceType_SCANNER_DB_SERVICE)
			k8sClient := getFakeK8sClient(tc.k8sClientConfig)
			tlsIssuer := newLocalScannerTLSIssuer(s.T(), k8sClient, sensorNamespace, sensorPodName)
			tlsIssuer.certRefreshBackoff = wait.Backoff{
				Duration: time.Millisecond,
			}

			s.Require().NoError(tlsIssuer.Start())
			tlsIssuer.Notify(common.SensorComponentEventCentralReachable)
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

			verifySecrets(ctx, s.T(), k8sClient, sensorNamespace, ca,
				map[string]*mtls.IssuedCert{"scanner-tls": scannerCert, "scanner-db-tls": scannerDBCert})
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

			tlsIssuer.Notify(common.SensorComponentEventCentralReachable)
			s.Require().NoError(tlsIssuer.Start())
			defer tlsIssuer.Stop(nil)

			require.Eventually(s.T(), func() bool {
				return tlsIssuer.certRefresher != nil && tlsIssuer.certRefresher.Stopped()
			}, 100*time.Millisecond, 10*time.Millisecond, "cert refresher should be stopped")
		})
	}
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

func newLocalScannerTLSIssuer(
	t *testing.T,
	k8sClient kubernetes.Interface,
	sensorNamespace string,
	sensorPodName string,
) *tlsIssuerImpl {
	tlsIssuer := NewLocalScannerTLSIssuer(k8sClient, sensorNamespace, sensorPodName)
	require.IsType(t, &tlsIssuerImpl{}, tlsIssuer)
	return tlsIssuer.(*tlsIssuerImpl)
}
