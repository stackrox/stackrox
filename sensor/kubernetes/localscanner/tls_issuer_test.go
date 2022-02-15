package localscanner

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsApiv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	sensorNamespace      = "stackrox-ns"
	sensorReplicasetName = "sensor-replicaset"
	sensorDeploymentName = "sensor-deployment"
)

type localScannerTLSIssuerFixture struct {
	k8sClient *fake.Clientset
	requester *certificateRequesterMock
	refresher *certificateRefresherMock
	repo      *certsRepoMock
	supplier  *suppliersMock
	issuer    *localScannerTLSIssuerImpl
}

func newLocalScannerTLSIssuerFixture(createSensorDeployment, createSensorReplicaSet bool) *localScannerTLSIssuerFixture {
	fixture := &localScannerTLSIssuerFixture{}
	fixture.requester = &certificateRequesterMock{}
	fixture.refresher = &certificateRefresherMock{}
	fixture.repo = &certsRepoMock{}
	fixture.supplier = &suppliersMock{}
	fixture.k8sClient = testK8sClient(createSensorDeployment, createSensorReplicaSet)
	msgToCentralC := make(chan *central.MsgFromSensor)
	msgFromCentralC := make(chan *central.IssueLocalScannerCertsResponse)
	fixture.issuer = &localScannerTLSIssuerImpl{
		sensorNamespace:                 sensorNamespace,
		podOwnerName:                    sensorReplicasetName,
		k8sClient:                       fixture.k8sClient,
		msgToCentralC:                   msgToCentralC,
		msgFromCentralC:                 msgFromCentralC,
		certificateRefresherSupplier:    fixture.supplier.supplyCertificateRefresher,
		serviceCertificatesRepoSupplier: fixture.supplier.supplyServiceCertificatesRepoSupplier,
		requester:                       fixture.requester,
	}

	return fixture
}

func (f *localScannerTLSIssuerFixture) assertExpectations(t *testing.T) {
	f.requester.AssertExpectations(t)
	f.requester.AssertExpectations(t)
	f.supplier.AssertExpectations(t)
}

// mockForStart setups the mocks for the happy path of Start
func (f *localScannerTLSIssuerFixture) mockForStart(refresherStartReturn error) {
	f.requester.On("Start").Once()
	f.refresher.On("Start").Once().Return(refresherStartReturn)
	f.repo.On("getServiceCertificates", mock.Anything).Once().Return((*storage.TypedServiceCertificateSet)(nil), nil)
	f.supplier.On("supplyServiceCertificatesRepoSupplier", mock.Anything,
		mock.Anything, mock.Anything).Once().Return(f.repo, nil)
	f.supplier.On("supplyCertificateRefresher", mock.Anything, f.repo,
		certRefreshTimeout, certRefreshBackoff).Once().Return(f.refresher)
}

func TestLocalScannerTLSIssuerStartSuccess(t *testing.T) {
	fixture := newLocalScannerTLSIssuerFixture(true, true)
	fixture.mockForStart(nil)

	startErr := fixture.issuer.Start()

	assert.NoError(t, startErr)
	fixture.assertExpectations(t)
}

func TestLocalScannerTLSIssuerRefresherFailureStartFailure(t *testing.T) {
	fixture := newLocalScannerTLSIssuerFixture(true, true)
	fixture.mockForStart(errForced)

	startErr := fixture.issuer.Start()

	assert.Error(t, startErr)
	fixture.assertExpectations(t)
}

func TestLocalScannerTLSIssuerStartAlreadyStartedFailure(t *testing.T) {
	fixture := newLocalScannerTLSIssuerFixture(true, true)
	fixture.mockForStart(nil)

	startErr := fixture.issuer.Start()
	secondStartErr := fixture.issuer.Start()

	assert.NoError(t, startErr)
	assert.Error(t, secondStartErr)
	fixture.assertExpectations(t)
}

func TestLocalScannerTLSIssuerFetchSensorDeploymentErrorStartFailure(t *testing.T) {
	testCases := map[string]struct {
		createSensorDeployment bool
		createSensorReplicaset bool
	}{
		"sensor replica set missing":    {createSensorDeployment: true, createSensorReplicaset: false},
		"sensor deployment set missing": {createSensorDeployment: false, createSensorReplicaset: false},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			fixture := newLocalScannerTLSIssuerFixture(tc.createSensorDeployment, tc.createSensorReplicaset)

			startErr := fixture.issuer.Start()

			assert.Error(t, startErr)
			fixture.assertExpectations(t)
		})
	}
}

// TODO sensor component interface methods

// if createSensorDeployment is false then createSensorReplicaSet is not treated as false.
func testK8sClient(createSensorDeployment, createSensorReplicaSet bool) *fake.Clientset {
	objects := make([]runtime.Object, 0)
	if createSensorDeployment {
		sensorDeployment := &appsApiv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensorDeploymentName,
				Namespace: sensorNamespace,
			},
		}
		objects = append(objects, sensorDeployment)
		if createSensorReplicaSet {
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
		}
	}

	k8sClient := fake.NewSimpleClientset(objects...)

	return k8sClient
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

type suppliersMock struct {
	mock.Mock
}

func (m *suppliersMock) supplyCertificateRefresher(requestCertificates requestCertificatesFunc,
	repository serviceCertificatesRepo, timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker {
	args := m.Called(requestCertificates, repository, timeout, backoff)
	return args.Get(0).(concurrency.RetryTicker)
}

func (m *suppliersMock) supplyServiceCertificatesRepoSupplier(ownerReference metav1.OwnerReference, namespace string,
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
