package certrefresh

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	backoff = wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   1.2,
		Jitter:   0.1,
		Steps:    3,
	}
	sensorPodName   = "sensor-pod"
	sensorNamespace = "stackrox-ns"
	replicaSetName  = "sensor-replica-set"
	deploymentName  = "sensor-deployment"
)

func createTestPod(podName, namespace, ownerName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "ReplicaSet",
					Name:       ownerName,
					UID:        "replica-set-uid",
				},
			},
		},
	}
}

func createTestReplicaSet(rsName, namespace, ownerName string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rsName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       ownerName,
					UID:        "deployment-uid",
				},
			},
		},
	}
}

func TestFetchSensorDeploymentOwnerRef_Success(t *testing.T) {
	k8sClient := fake.NewSimpleClientset(
		createTestPod(sensorPodName, sensorNamespace, replicaSetName),
		createTestReplicaSet(replicaSetName, sensorNamespace, deploymentName),
	)

	ownerRef, err := FetchSensorDeploymentOwnerRef(context.TODO(), sensorPodName, sensorNamespace, k8sClient, backoff)

	assert.NoError(t, err)
	assert.NotNil(t, ownerRef)
	assert.Equal(t, "apps/v1", ownerRef.APIVersion)
	assert.Equal(t, "Deployment", ownerRef.Kind)
	assert.Equal(t, deploymentName, ownerRef.Name)
	assert.Equal(t, types.UID("deployment-uid"), ownerRef.UID)
}

func TestFetchSensorDeploymentOwnerRef_EmptyPodName(t *testing.T) {
	ownerRef, err := FetchSensorDeploymentOwnerRef(context.TODO(), "", sensorNamespace,
		fake.NewSimpleClientset(), backoff)

	assert.Error(t, err)
	assert.Nil(t, ownerRef)
}

func TestFetchSensorDeploymentOwnerRef_PodNotFound(t *testing.T) {

	ownerRef, err := FetchSensorDeploymentOwnerRef(context.TODO(), sensorPodName, sensorNamespace,
		fake.NewSimpleClientset(), backoff)

	assert.Error(t, err)
	assert.Nil(t, ownerRef)
}

func TestFetchSensorDeploymentOwnerRef_ReplicaSetNotFound(t *testing.T) {
	ownerRef, err := FetchSensorDeploymentOwnerRef(context.TODO(), sensorPodName, sensorNamespace,
		fake.NewSimpleClientset(createTestPod(sensorPodName, sensorNamespace, replicaSetName)), backoff)

	assert.Error(t, err)
	assert.Nil(t, ownerRef)
}

func TestFetchSensorDeploymentOwnerRef_UnexpectedPodOwners(t *testing.T) {
	// create a pod with multiple owners
	pod := createTestPod(sensorPodName, sensorNamespace, replicaSetName)
	pod.OwnerReferences = append(pod.OwnerReferences, metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Kind",
		Name:       "second-owner",
	})

	ownerRef, err := FetchSensorDeploymentOwnerRef(context.TODO(), sensorPodName, sensorNamespace,
		fake.NewSimpleClientset(pod), backoff)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has unexpected owners")
	assert.Nil(t, ownerRef)
}

func TestFetchSensorDeploymentOwnerRef_UnexpectedReplicaSetOwners(t *testing.T) {
	// create a ReplicaSet with multiple owners
	pod := createTestPod(sensorPodName, sensorNamespace, replicaSetName)
	replicaSet := createTestReplicaSet(replicaSetName, sensorNamespace, deploymentName)
	replicaSet.OwnerReferences = append(pod.OwnerReferences, metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Kind",
		Name:       "second-owner",
	})

	ownerRef, err := FetchSensorDeploymentOwnerRef(context.TODO(), sensorPodName, sensorNamespace,
		fake.NewSimpleClientset(pod, replicaSet), backoff)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has unexpected owners")
	assert.Nil(t, ownerRef)
}
