package backgroundmigrations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace = "stackrox"
)

func int32Ptr(i int32) *int32 { return &i }

func newTestRolloutChecker(client *fake.Clientset) *k8sRolloutChecker {
	return &k8sRolloutChecker{
		client:    client,
		inCluster: true,
	}
}

func readyDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       deploymentName,
			Namespace:  testNamespace,
			Generation: 1,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "central"},
			},
		},
		Status: appsv1.DeploymentStatus{
			UpdatedReplicas:    1,
			AvailableReplicas:  1,
			ObservedGeneration: 1,
		},
	}
}

func TestRolloutDone(t *testing.T) {
	client := fake.NewSimpleClientset(readyDeployment())
	checker := newTestRolloutChecker(client)

	done, err := checker.IsRolloutDone(context.Background())
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestRolloutNotInCluster(t *testing.T) {
	checker := &k8sRolloutChecker{
		inCluster: false,
	}

	done, err := checker.IsRolloutDone(context.Background())
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestRolloutNotDoneWhenReplicasNotReady(t *testing.T) {
	dep := readyDeployment()
	dep.Status.UpdatedReplicas = 0
	dep.Status.AvailableReplicas = 0

	client := fake.NewSimpleClientset(dep)
	checker := newTestRolloutChecker(client)

	done, err := checker.IsRolloutDone(context.Background())
	assert.NoError(t, err)
	assert.False(t, done)
}

func TestRolloutNotDoneWhenObservedGenerationBehind(t *testing.T) {
	dep := readyDeployment()
	dep.Generation = 2
	dep.Status.ObservedGeneration = 1

	client := fake.NewSimpleClientset(dep)
	checker := newTestRolloutChecker(client)

	done, err := checker.IsRolloutDone(context.Background())
	assert.NoError(t, err)
	assert.False(t, done)
}

func TestRolloutNotDoneWithTerminatingPods(t *testing.T) {
	dep := readyDeployment()
	now := metav1.Now()
	terminatingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "central-old",
			Namespace:         testNamespace,
			Labels:            map[string]string{"app": "central"},
			DeletionTimestamp: &now,
		},
	}

	client := fake.NewSimpleClientset(dep, terminatingPod)
	checker := newTestRolloutChecker(client)

	done, err := checker.IsRolloutDone(context.Background())
	assert.NoError(t, err)
	assert.False(t, done)
}

func TestRolloutCheckDefaultsToOneReplica(t *testing.T) {
	dep := readyDeployment()
	dep.Spec.Replicas = nil
	dep.Status.UpdatedReplicas = 1
	dep.Status.AvailableReplicas = 1

	client := fake.NewSimpleClientset(dep)
	checker := newTestRolloutChecker(client)

	done, err := checker.IsRolloutDone(context.Background())
	assert.NoError(t, err)
	assert.True(t, done)
}
