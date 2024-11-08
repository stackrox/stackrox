package certrefresh

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certificates"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	"github.com/stretchr/testify/mock"
	appsApiv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	sensorNamespace      = "stackrox-ns"
	sensorReplicasetName = "sensor-replicaset"
	sensorPodName        = "sensor-pod"

	errForced        = errors.New("forced error")
	sensorDeployment = &appsApiv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensor-deployment",
			Namespace: sensorNamespace,
		},
	}
)

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

type certificateRequesterMock struct {
	mock.Mock
}

func (m *certificateRequesterMock) Start() {
	m.Called()
}
func (m *certificateRequesterMock) Stop() {
	m.Called()
}
func (m *certificateRequesterMock) RequestCertificates(ctx context.Context) (*certificates.Response, error) {
	args := m.Called(ctx)
	return args.Get(0).(*certificates.Response), args.Error(1)
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

func (m *componentGetterMock) getCertificateRefresher(certsDescription string, requestCertificates requestCertificatesFunc,
	repository certrepo.ServiceCertificatesRepo, timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker {
	args := m.Called(certsDescription, requestCertificates, repository, timeout, backoff)
	return args.Get(0).(concurrency.RetryTicker)
}

func (m *componentGetterMock) getServiceCertificatesRepo(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) certrepo.ServiceCertificatesRepo {
	args := m.Called(ownerReference, namespace, secretsClient)
	return args.Get(0).(certrepo.ServiceCertificatesRepo)
}

type certsRepoMock struct {
	mock.Mock
}

func (m *certsRepoMock) GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
	args := m.Called(ctx)
	return args.Get(0).(*storage.TypedServiceCertificateSet), args.Error(1)
}

func (m *certsRepoMock) EnsureServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) ([]*storage.TypedServiceCertificate, error) {
	args := m.Called(ctx, certificates)
	return certificates.ServiceCerts, args.Error(0)
}