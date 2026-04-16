//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	toxiproxyImage       = "ghcr.io/shopify/toxiproxy:2.5.0"
	toxiproxyAPIPort     = 8474
	toxiproxyProxyPort   = 8989
	sensorNamespace      = "stackrox"
	sensorDeploymentName = "sensor"

	testTimeout  = 120 * time.Second
	testInterval = 5 * time.Second
)

// TestSensorKubernetesPipeline_ConnectionResilience verifies that the sensor Kubernetes
// event pipeline does not lose messages when sensor reconnects to Central after a network
// disruption. This test simulates a network outage using toxiproxy and verifies that
// deployments remain visible in Central after reconnection.
func TestSensorKubernetesPipeline_ConnectionResilience(t *testing.T) {
	ctx := context.Background()
	k8sClient := createK8sClient(t)

	// Step 1: Get sensor deployment using Eventually
	var sensorDeploy *appsv1.Deployment
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		deploy, err := k8sClient.AppsV1().Deployments(sensorNamespace).Get(ctx, sensorDeploymentName, metav1.GetOptions{})
		require.NoErrorf(c, err, "failed to get sensor deployment")
		sensorDeploy = deploy
	}, testTimeout, testInterval)

	// Get original central endpoint
	var originalCentralEndpoint string
	for _, container := range sensorDeploy.Spec.Template.Spec.Containers {
		if container.Name == "sensor" {
			for _, env := range container.Env {
				if env.Name == "ROX_CENTRAL_ENDPOINT" {
					originalCentralEndpoint = env.Value
					break
				}
			}
		}
	}
	require.NotEmpty(t, originalCentralEndpoint, "ROX_CENTRAL_ENDPOINT not found")

	// Step 2: Configure sensor with toxiproxy sidecar and wait for pod to be ready
	sensorPod := configureSensorWithToxiproxy(ctx, t, k8sClient, originalCentralEndpoint)

	// Cleanup: restore original deployment on test completion
	t.Cleanup(func() {
		cleanupSensorToxiproxyConfig(ctx, t, k8sClient, originalCentralEndpoint)
	})

	// Step 3: Set up port-forward to toxiproxy API
	localPort, cleanupPortForward := setupPortForward(t, sensorPod, toxiproxyAPIPort)
	t.Cleanup(cleanupPortForward)

	// Step 4: Connect to toxiproxy API and get "central" proxy
	toxiproxyEndpoint := fmt.Sprintf("localhost:%d", localPort)
	centralProxy := getToxiproxyCentralProxy(t, toxiproxyEndpoint)

	// Step 5: Wait for sensor to be healthy
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
	t.Log("Sensor is healthy (baseline)")

	// Step 6: Create test namespace and deployments
	testNamespace := "pipeline-test"
	_, err := k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}, metav1.CreateOptions{})
	require.NoError(t, err, "failed to create test namespace")

	t.Cleanup(func() {
		_ = k8sClient.CoreV1().Namespaces().Delete(ctx, testNamespace, metav1.DeleteOptions{})
	})

	// Step 7: Create docker config secrets to trigger ResolveAllDeployments
	// This simulates the customer case where docker secrets caused amplification
	numSecrets := 5
	for i := 0; i < numSecrets; i++ {
		secretName := fmt.Sprintf("docker-secret-%d", i)
		createDockerConfigSecret(ctx, t, k8sClient, testNamespace, secretName)
	}
	t.Logf("Created %d docker config secrets in namespace %s", numSecrets, testNamespace)

	// Step 8: Create deployment BEFORE disconnection
	deployment1 := "nginx-before"
	require.NoError(t, createDeploymentViaAPI(t, "nginx:1.27", deployment1, 1, testNamespace))
	waitForDeploymentInCentral(t, deployment1)
	t.Logf("Deployment '%s' created and visible in Central (before disconnection)", deployment1)

	// Step 9: Update one docker secret to trigger ResolveAllDeployments before disconnection
	updateDockerConfigSecret(ctx, t, k8sClient, testNamespace, "docker-secret-0")
	t.Log("Updated docker secret to trigger ResolveAllDeployments (baseline)")

	// Step 10: Disable proxy to simulate connection loss
	centralProxy.Enabled = false
	err = centralProxy.Save()
	require.NoError(t, err, "failed to disable central proxy")

	t.Log("Disabled toxiproxy - connection to Central severed")

	// Step 11: Wait for sensor to become degraded
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_DEGRADED)
	t.Log("Sensor is degraded (connection disrupted)")

	// Step 12: Create deployment DURING disconnection (while offline)
	deployment2 := "redis-during"
	require.NoError(t, createDeploymentViaAPI(t, "redis:7.4", deployment2, 1, testNamespace))
	t.Logf("Deployment '%s' created while sensor is offline", deployment2)

	// Step 13: Update docker secret DURING disconnection to trigger ResolveAllDeployments while offline
	updateDockerConfigSecret(ctx, t, k8sClient, testNamespace, "docker-secret-1")
	t.Log("Updated docker secret while sensor is offline")

	// Step 14: Sleep for disconnect duration (simulate sustained outage)
	disconnectDuration := 10 * time.Second
	t.Logf("Sleeping for %s to simulate sustained connection loss", disconnectDuration)
	time.Sleep(disconnectDuration)

	// Step 15: Re-enable proxy to restore connection
	centralProxy.Enabled = true
	err = centralProxy.Save()
	require.NoError(t, err, "failed to re-enable central proxy")

	t.Log("Re-enabled toxiproxy - connection to Central restored")

	// Step 16: Wait for sensor to become healthy again
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
	t.Log("Sensor is healthy again (reconnected)")

	// Step 17: Create deployment AFTER reconnection
	deployment3 := "busybox-after"
	require.NoError(t, createDeploymentViaAPI(t, "busybox:1.36", deployment3, 1, testNamespace))
	waitForDeploymentInCentral(t, deployment3)
	t.Logf("Deployment '%s' created and visible in Central (after reconnection)", deployment3)

	// Step 18: Update docker secret AFTER reconnection to trigger ResolveAllDeployments after reconnection
	updateDockerConfigSecret(ctx, t, k8sClient, testNamespace, "docker-secret-2")
	t.Log("Updated docker secret after reconnection")

	// Step 19: Verify ALL three deployments are visible in Central (critical validation)
	t.Log("Verifying all deployments are visible in Central...")

	// Wait for deployment created during offline to sync
	waitForDeploymentInCentral(t, deployment2)
	t.Logf("Deployment '%s' (created during offline) now visible in Central", deployment2)

	// Verify deployment created before disconnection is still there
	waitForDeploymentInCentral(t, deployment1)
	t.Logf("Deployment '%s' (created before disconnection) still visible in Central", deployment1)

	// Verify all three deployments have scanned images
	conn := centralgrpc.GRPCConnectionToCentral(t)
	deploymentService := v1.NewDeploymentServiceClient(conn)
	for _, deploymentName := range []string{deployment1, deployment2, deployment3} {
		verifyDeploymentHasScannedImage(t, ctx, deploymentService, deploymentName)
	}

	t.Log("SUCCESS: All deployments visible in Central with scanned images after connection resilience test")
}

// Helper functions

// configureSensorWithToxiproxy patches the sensor deployment to add toxiproxy sidecar
// and configure sensor to proxy Central connection through toxiproxy.
// Waits for the sensor pod to be ready with both containers.
func configureSensorWithToxiproxy(ctx context.Context, t *testing.T, k8sClient kubernetes.Interface, originalCentralEndpoint string) *corev1.Pod {
	var deploy *appsv1.Deployment
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		d, err := k8sClient.AppsV1().Deployments(sensorNamespace).Get(ctx, sensorDeploymentName, metav1.GetOptions{})
		require.NoErrorf(c, err, "failed to get sensor deployment")
		deploy = d
	}, testTimeout, testInterval)

	// Add toxiproxy sidecar container
	toxiproxyContainer := corev1.Container{
		Name:  "toxiproxy",
		Image: toxiproxyImage,
		Ports: []corev1.ContainerPort{
			{ContainerPort: toxiproxyAPIPort, Name: "toxiproxy-api", Protocol: corev1.ProtocolTCP},
			{ContainerPort: toxiproxyProxyPort, Name: "toxiproxy-proxy", Protocol: corev1.ProtocolTCP},
		},
	}

	// Find sensor container and update env vars
	for i := range deploy.Spec.Template.Spec.Containers {
		if deploy.Spec.Template.Spec.Containers[i].Name == "sensor" {
			// Replace or add env vars
			envVars := []corev1.EnvVar{}
			for _, env := range deploy.Spec.Template.Spec.Containers[i].Env {
				if env.Name == "ROX_CENTRAL_ENDPOINT" {
					// Replace with localhost endpoint
					envVars = append(envVars, corev1.EnvVar{Name: "ROX_CENTRAL_ENDPOINT", Value: fmt.Sprintf("localhost:%d", toxiproxyProxyPort)})
				} else {
					envVars = append(envVars, env)
				}
			}
			// Add new env vars for toxiproxy and fast reconnection
			envVars = append(envVars,
				corev1.EnvVar{Name: "ROX_CENTRAL_ENDPOINT_NO_PROXY", Value: originalCentralEndpoint},
				corev1.EnvVar{Name: "ROX_CHAOS_PROFILE", Value: "none"},
				corev1.EnvVar{Name: "ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL", Value: "1s"},
				corev1.EnvVar{Name: "ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL", Value: "2s"},
			)
			deploy.Spec.Template.Spec.Containers[i].Env = envVars
			break
		}
	}

	// Add toxiproxy container
	deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, toxiproxyContainer)

	_, err := k8sClient.AppsV1().Deployments(sensorNamespace).Update(ctx, deploy, metav1.UpdateOptions{})
	require.NoError(t, err, "failed to update sensor deployment")

	// Wait for sensor pod to be ready with both containers
	var sensorPod *corev1.Pod
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		pods, err := k8sClient.CoreV1().Pods(sensorNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=sensor",
		})
		require.NoErrorf(c, err, "failed to list sensor pods")
		require.NotEmptyf(c, pods.Items, "no sensor pods found")

		// Check if pod has both containers ready
		pod := &pods.Items[0]
		readyContainers := 0
		for _, status := range pod.Status.ContainerStatuses {
			if status.Ready {
				readyContainers++
			}
		}
		require.Equalf(c, 2, readyContainers, "expected 2 ready containers (sensor + toxiproxy), got %d", readyContainers)
		sensorPod = pod
	}, testTimeout, testInterval)

	t.Logf("Sensor pod %s is ready with toxiproxy", sensorPod.Name)
	return sensorPod
}

// cleanupSensorToxiproxyConfig removes toxiproxy sidecar and restores original sensor configuration.
// Waits for the sensor pod to be ready with restored configuration.
func cleanupSensorToxiproxyConfig(ctx context.Context, t *testing.T, k8sClient kubernetes.Interface, originalCentralEndpoint string) {
	var deploy *appsv1.Deployment
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		d, err := k8sClient.AppsV1().Deployments(sensorNamespace).Get(ctx, sensorDeploymentName, metav1.GetOptions{})
		require.NoErrorf(c, err, "failed to get sensor deployment")
		deploy = d
	}, testTimeout, testInterval)

	// Remove toxiproxy container
	containers := []corev1.Container{}
	for _, c := range deploy.Spec.Template.Spec.Containers {
		if c.Name != "toxiproxy" {
			containers = append(containers, c)
		}
	}
	deploy.Spec.Template.Spec.Containers = containers

	// Restore original env vars
	for i := range deploy.Spec.Template.Spec.Containers {
		if deploy.Spec.Template.Spec.Containers[i].Name == "sensor" {
			envVars := []corev1.EnvVar{}
			for _, env := range deploy.Spec.Template.Spec.Containers[i].Env {
				if env.Name == "ROX_CENTRAL_ENDPOINT" && env.Value == fmt.Sprintf("localhost:%d", toxiproxyProxyPort) {
					// Restore original endpoint
					envVars = append(envVars, corev1.EnvVar{Name: "ROX_CENTRAL_ENDPOINT", Value: originalCentralEndpoint})
				} else if env.Name != "ROX_CENTRAL_ENDPOINT_NO_PROXY" &&
					env.Name != "ROX_CHAOS_PROFILE" &&
					env.Name != "ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL" &&
					env.Name != "ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL" {
					envVars = append(envVars, env)
				}
			}
			deploy.Spec.Template.Spec.Containers[i].Env = envVars
			break
		}
	}

	_, err := k8sClient.AppsV1().Deployments(sensorNamespace).Update(ctx, deploy, metav1.UpdateOptions{})
	require.NoError(t, err, "failed to restore sensor deployment")

	// Wait for sensor pod to be ready with only sensor container
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		pods, err := k8sClient.CoreV1().Pods(sensorNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=sensor",
		})
		require.NoErrorf(c, err, "failed to list sensor pods")
		require.NotEmptyf(c, pods.Items, "no sensor pods found")

		// Check if pod has only sensor container ready
		pod := &pods.Items[0]
		require.Equalf(c, 1, len(pod.Status.ContainerStatuses), "expected 1 container, got %d", len(pod.Status.ContainerStatuses))
		require.Truef(c, pod.Status.ContainerStatuses[0].Ready, "sensor container not ready")
	}, testTimeout, testInterval)

	t.Log("Sensor pod restored to original configuration")
}

// setupPortForward creates a port-forward to the specified pod and port.
// Returns the local port and a cleanup function.
func setupPortForward(t *testing.T, pod *corev1.Pod, remotePort int32) (uint16, func()) {
	restConfig := getConfig(t)

	// Set Kubernetes defaults required for REST client
	restConfig.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	if restConfig.APIPath == "" {
		restConfig.APIPath = "/api"
	}
	if restConfig.NegotiatedSerializer == nil {
		restConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	}
	if len(restConfig.UserAgent) == 0 {
		restConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	restClient, err := rest.RESTClientFor(restConfig)
	require.NoError(t, err, "failed to create REST client")

	req := restClient.Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	require.NoError(t, err, "failed to create SPDY transport")

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())

	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{})

	// Request port 0 (any available local port) -> remotePort
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("0:%d", remotePort)}, stopChannel, readyChannel, nil, nil)
	require.NoError(t, err, "failed to create port forwarder")

	// Start port forwarding in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- forwarder.ForwardPorts()
	}()

	// Wait for port-forward to be ready
	select {
	case <-readyChannel:
		t.Log("Port-forward is ready")
	case err := <-errChan:
		require.NoError(t, err, "port-forward failed to start")
	case <-time.After(testTimeout):
		require.Fail(t, "timeout waiting for port-forward to be ready")
	}

	// Get the actual local port assigned
	ports, err := forwarder.GetPorts()
	require.NoError(t, err, "failed to get forwarded ports")
	require.NotEmpty(t, ports, "no ports forwarded")

	localPort := ports[0].Local
	t.Logf("Port-forward established: localhost:%d -> %s:%d", localPort, pod.Name, remotePort)

	cleanup := func() {
		close(stopChannel)
		<-errChan // Wait for forwarder to stop
	}

	return localPort, cleanup
}

// getToxiproxyCentralProxy connects to toxiproxy API and returns the "central" proxy
func getToxiproxyCentralProxy(t *testing.T, toxiproxyEndpoint string) *toxiproxy.Proxy {
	toxiproxyClient := toxiproxy.NewClient(fmt.Sprintf("http://%s", toxiproxyEndpoint))

	var centralProxy *toxiproxy.Proxy
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		// First check if toxiproxy is reachable
		_, err := toxiproxyClient.Proxies()
		if err != nil {
			t.Logf("Toxiproxy API not ready yet: %v", err)
			require.NoErrorf(c, err, "toxiproxy API not reachable")
			return
		}

		// Get the "central" proxy
		proxy, err := toxiproxyClient.Proxy("central")
		require.NoErrorf(c, err, "failed to get central proxy")
		centralProxy = proxy
	}, testTimeout, testInterval)

	t.Logf("Got toxiproxy 'central' proxy: %s -> %s", centralProxy.Listen, centralProxy.Upstream)
	return centralProxy
}

// verifyDeploymentHasScannedImage verifies that a deployment has a scanned image
func verifyDeploymentHasScannedImage(t *testing.T, ctx context.Context, deploymentService v1.DeploymentServiceClient, deploymentName string) {
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		listDeployments, err := deploymentService.ListDeployments(ctx, &v1.RawQuery{
			Query: search.NewQueryBuilder().AddExactMatches(search.DeploymentName, deploymentName).Query(),
		})
		require.NoErrorf(c, err, "failed to list deployments")
		require.NotEmptyf(c, listDeployments.GetDeployments(), "deployment %s not found", deploymentName)

		deployments, err := retrieveDeployments(deploymentService, listDeployments.GetDeployments())
		require.NoErrorf(c, err, "failed to retrieve full deployment")
		require.NotEmptyf(c, deployments, "no deployments retrieved")

		deployment := deployments[0]
		require.NotEmptyf(c, deployment.GetContainers(), "deployment has no containers")
		require.NotNilf(c, deployment.GetContainers()[0].GetImage(), "container has no image")
		require.NotEmptyf(c, deployment.GetContainers()[0].GetImage().GetId(), "image has not been scanned (no image ID)")
	}, 3*time.Minute, testInterval)

	t.Logf("Deployment %s has scanned image", deploymentName)
}

// createDockerConfigSecret creates a docker config secret in the specified namespace
func createDockerConfigSecret(ctx context.Context, t *testing.T, k8sClient kubernetes.Interface, namespace, name string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{"https://index.docker.io/v1/":{"username":"test","password":"test","auth":"dGVzdDp0ZXN0"}}}`),
		},
	}

	_, err := k8sClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	require.NoError(t, err, "failed to create docker config secret %s", name)
}

// updateDockerConfigSecret updates a docker config secret to trigger ResolveAllDeployments
func updateDockerConfigSecret(ctx context.Context, t *testing.T, k8sClient kubernetes.Interface, namespace, name string) {
	secret, err := k8sClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	require.NoError(t, err, "failed to get secret %s", name)

	// Update the annotation to trigger a change event
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}
	secret.Annotations["updated"] = time.Now().Format(time.RFC3339)

	_, err = k8sClient.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	require.NoError(t, err, "failed to update docker config secret %s", name)
}
