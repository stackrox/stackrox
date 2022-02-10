package localscanner

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsApiv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	sensorNamespace      = "stackrox-ns"
	sensorReplicasetName = "sensor-replicaset"
	sensorDeploymentName = "sensor-deployment"
	errForced            = errors.New("forced")
)

type localScannerTLSIssuerFixture struct {
	k8sClient *fake.Clientset
	requester *certificateRequesterMock
	refresher *certificateRefresherMock
	supplier  *suppliersMock
	issuer    *localScannerTLSIssuerImpl
}

func newLocalScannerTLSIssuerFixture(withSensorDeployment, withReplicaSet bool) *localScannerTLSIssuerFixture {
	fixture := &localScannerTLSIssuerFixture{}
	fixture.requester = &certificateRequesterMock{}
	fixture.refresher = &certificateRefresherMock{}
	fixture.supplier = &suppliersMock{}
	fixture.k8sClient = testK8sClient(withSensorDeployment, withReplicaSet)
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

func (f *localScannerTLSIssuerFixture) mockForStart(refresherStartReturn error) {
	f.requester.On("Start").Once()
	f.refresher.On("Start").Once().Return(refresherStartReturn)
	repo := struct{}{} // TODO ROX-9128 replace by nil casting to impl pointer
	f.supplier.On("supplyServiceCertificatesRepoSupplier", mock.Anything, scannerSpec, scannerDBSpec,
		mock.Anything, mock.Anything, mock.Anything).Once().Return(repo, nil)
	f.supplier.On("supplyCertificateRefresher", mock.Anything,
		certRefreshTimeout, certRefreshBackoff, repo).Once().Return(f.refresher)
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

// TODO fetch sensor deployment failures
// TODO sensor component interface methods

// if withSensorDeployment is false then withReplicaSet is not included
func testK8sClient(withSensorDeployment, withReplicaSet bool) *fake.Clientset {
	objects := make([]runtime.Object, 0)
	if withSensorDeployment {
		sensorDeployment := &appsApiv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensorDeploymentName,
				Namespace: sensorNamespace,
			},
		}
		objects = append(objects, sensorDeployment)
		if withReplicaSet {
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

func (m *suppliersMock) supplyCertificateRefresher(requestCertificates requestCertificatesFunc, timeout time.Duration, backoff wait.Backoff, repository serviceCertificatesRepo) CertificateRefresher {
	args := m.Called(requestCertificates, timeout, backoff, repository)
	return args.Get(0).(CertificateRefresher)
}

func (m *suppliersMock) supplyServiceCertificatesRepoSupplier(ctx context.Context, scannerSpec, scannerDBSpec ServiceCertSecretSpec, sensorDeployment *appsApiv1.Deployment, initialCertsSupplier func(context.Context) (*storage.TypedServiceCertificateSet, error), secretsClient v1.SecretInterface) (serviceCertificatesRepo, error) {
	args := m.Called(ctx, scannerSpec, scannerDBSpec, sensorDeployment, initialCertsSupplier, secretsClient)
	return args.Get(0).(serviceCertificatesRepo), args.Error(1)
}
