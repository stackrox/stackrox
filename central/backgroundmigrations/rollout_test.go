package backgroundmigrations

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace    = "stackrox"
	testPollInterval = 10 * time.Millisecond
)

func int32Ptr(i int32) *int32 { return &i }

func newTestRolloutChecker(client *fake.Clientset) *k8sRolloutChecker {
	return &k8sRolloutChecker{
		client:       client,
		inCluster:    true,
		pollInterval: testPollInterval,
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

func TestRolloutCompleteImmediately(t *testing.T) {
	client := fake.NewSimpleClientset(readyDeployment())
	checker := newTestRolloutChecker(client)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	err := checker.WaitForRolloutComplete(ctx)
	assert.NoError(t, err)
}

func TestRolloutNotInCluster(t *testing.T) {
	checker := &k8sRolloutChecker{
		inCluster:    false,
		pollInterval: testPollInterval,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	err := checker.WaitForRolloutComplete(ctx)
	assert.NoError(t, err)
}

func TestRolloutWaitsUntilReplicasReady(t *testing.T) {
	dep := readyDeployment()
	dep.Status.UpdatedReplicas = 0
	dep.Status.AvailableReplicas = 0

	client := fake.NewSimpleClientset(dep)
	checker := newTestRolloutChecker(client)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- checker.WaitForRolloutComplete(ctx)
	}()

	// Update the deployment to be ready after a short delay.
	time.Sleep(50 * time.Millisecond)
	dep.Status.UpdatedReplicas = 1
	dep.Status.AvailableReplicas = 1
	_, err := client.AppsV1().Deployments(testNamespace).UpdateStatus(ctx, dep, metav1.UpdateOptions{})
	require.NoError(t, err)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(testTimeout):
		t.Fatal("WaitForRolloutComplete did not return within timeout")
	}
}

func TestRolloutWaitsForObservedGeneration(t *testing.T) {
	dep := readyDeployment()
	dep.Generation = 2
	dep.Status.ObservedGeneration = 1

	client := fake.NewSimpleClientset(dep)
	checker := newTestRolloutChecker(client)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- checker.WaitForRolloutComplete(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	dep.Status.ObservedGeneration = 2
	_, err := client.AppsV1().Deployments(testNamespace).UpdateStatus(ctx, dep, metav1.UpdateOptions{})
	require.NoError(t, err)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(testTimeout):
		t.Fatal("WaitForRolloutComplete did not return within timeout")
	}
}

func TestRolloutWaitsForTerminatingPods(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- checker.WaitForRolloutComplete(ctx)
	}()

	// Remove the terminating pod after a short delay.
	time.Sleep(50 * time.Millisecond)
	err := client.CoreV1().Pods(testNamespace).Delete(ctx, "central-old", metav1.DeleteOptions{})
	require.NoError(t, err)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(testTimeout):
		t.Fatal("WaitForRolloutComplete did not return within timeout")
	}
}

func TestRolloutCancelledByContext(t *testing.T) {
	dep := readyDeployment()
	dep.Status.UpdatedReplicas = 0

	client := fake.NewSimpleClientset(dep)
	checker := newTestRolloutChecker(client)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := checker.WaitForRolloutComplete(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRolloutDefaultsToOneReplica(t *testing.T) {
	dep := readyDeployment()
	dep.Spec.Replicas = nil
	dep.Status.UpdatedReplicas = 1
	dep.Status.AvailableReplicas = 1

	client := fake.NewSimpleClientset(dep)
	checker := newTestRolloutChecker(client)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	err := checker.WaitForRolloutComplete(ctx)
	assert.NoError(t, err)
}
