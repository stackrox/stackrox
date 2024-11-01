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
	replicaSetName = "sensor-replica-set"
	deploymentName = "sensor-deployment"
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

func TestFetchSensorDeploymentOwnerRef(t *testing.T) {
	tests := []struct {
		name           string
		podName        string
		namespace      string
		k8sClient      *fake.Clientset
		expectedErr    bool
		expectedErrMsg string
		expectedOwner  *metav1.OwnerReference
	}{
		{
			name:      "Success",
			podName:   sensorPodName,
			namespace: sensorNamespace,
			k8sClient: fake.NewSimpleClientset(
				createTestPod(sensorPodName, sensorNamespace, replicaSetName),
				createTestReplicaSet(replicaSetName, sensorNamespace, deploymentName),
			),
			expectedErr:   false,
			expectedOwner: &metav1.OwnerReference{APIVersion: "apps/v1", Kind: "Deployment", Name: deploymentName, UID: types.UID("deployment-uid")},
		},
		{
			name:          "EmptyPodName",
			podName:       "",
			namespace:     sensorNamespace,
			k8sClient:     fake.NewSimpleClientset(),
			expectedErr:   true,
			expectedOwner: nil,
		},
		{
			name:          "PodNotFound",
			podName:       sensorPodName,
			namespace:     sensorNamespace,
			k8sClient:     fake.NewSimpleClientset(),
			expectedErr:   true,
			expectedOwner: nil,
		},
		{
			name:      "ReplicaSetNotFound",
			podName:   sensorPodName,
			namespace: sensorNamespace,
			k8sClient: fake.NewSimpleClientset(
				createTestPod(sensorPodName, sensorNamespace, replicaSetName),
			),
			expectedErr:   true,
			expectedOwner: nil,
		},
		{
			name:      "UnexpectedPodOwners",
			podName:   sensorPodName,
			namespace: sensorNamespace,
			k8sClient: func() *fake.Clientset {
				pod := createTestPod(sensorPodName, sensorNamespace, replicaSetName)
				pod.OwnerReferences = append(pod.OwnerReferences, metav1.OwnerReference{
					APIVersion: "v1",
					Kind:       "Kind",
					Name:       "second-owner",
				})
				return fake.NewSimpleClientset(pod)
			}(),
			expectedErr:    true,
			expectedErrMsg: "pod \"" + sensorPodName + "\" has unexpected owners",
			expectedOwner:  nil,
		},
		{
			name:      "UnexpectedReplicaSetOwners",
			podName:   sensorPodName,
			namespace: sensorNamespace,
			k8sClient: func() *fake.Clientset {
				pod := createTestPod(sensorPodName, sensorNamespace, replicaSetName)
				replicaSet := createTestReplicaSet(replicaSetName, sensorNamespace, deploymentName)
				replicaSet.OwnerReferences = append(replicaSet.OwnerReferences, metav1.OwnerReference{
					APIVersion: "v1",
					Kind:       "Kind",
					Name:       "second-owner",
				})
				return fake.NewSimpleClientset(pod, replicaSet)
			}(),
			expectedErr:    true,
			expectedErrMsg: "replica set \"" + replicaSetName + "\" has unexpected owners",
			expectedOwner:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ownerRef, err := FetchSensorDeploymentOwnerRef(context.Background(), tt.podName, tt.namespace, tt.k8sClient, backoff)

			if tt.expectedErr {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
				assert.Nil(t, ownerRef)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ownerRef)
				assert.Equal(t, tt.expectedOwner.APIVersion, ownerRef.APIVersion)
				assert.Equal(t, tt.expectedOwner.Kind, ownerRef.Kind)
				assert.Equal(t, tt.expectedOwner.Name, ownerRef.Name)
				assert.Equal(t, tt.expectedOwner.UID, ownerRef.UID)
			}
		})
	}
}
