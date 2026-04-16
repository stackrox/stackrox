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

	// Step 6: Verify sensor deployment is visible in Central (baseline)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	deploymentService := v1.NewDeploymentServiceClient(conn)

	deployments, err := deploymentService.ListDeployments(ctx, &v1.RawQuery{Query: "Deployment:sensor"})
	require.NoError(t, err, "failed to list deployments")
	require.NotEmpty(t, deployments.GetDeployments(), "sensor deployment not found in Central")

	t.Logf("Baseline: sensor deployment visible in Central (found %d deployments)", len(deployments.GetDeployments()))

	// Step 7: Disable proxy to simulate connection loss
	centralProxy.Enabled = false
	err = centralProxy.Save()
	require.NoError(t, err, "failed to disable central proxy")

	t.Log("Disabled toxiproxy - connection to Central severed")

	// Step 8: Wait for sensor to become degraded
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_DEGRADED)
	t.Log("Sensor is degraded (connection disrupted)")

	// Step 9: Sleep for disconnect duration (simulate sustained outage)
	disconnectDuration := 10 * time.Second
	t.Logf("Sleeping for %s to simulate sustained connection loss", disconnectDuration)
	time.Sleep(disconnectDuration)

	// Step 10: Re-enable proxy to restore connection
	centralProxy.Enabled = true
	err = centralProxy.Save()
	require.NoError(t, err, "failed to re-enable central proxy")

	t.Log("Re-enabled toxiproxy - connection to Central restored")

	// Step 11: Wait for sensor to become healthy again
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
	t.Log("Sensor is healthy again (reconnected)")

	// Step 12: Verify sensor deployment is STILL visible in Central (critical validation)
	var finalDeployments []*storage.ListDeployment
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		deployments, err := deploymentService.ListDeployments(ctx, &v1.RawQuery{Query: "Deployment:sensor"})
		require.NoErrorf(c, err, "failed to list deployments")
		require.NotEmptyf(c, deployments.GetDeployments(), "sensor deployment lost after reconnection")
		finalDeployments = deployments.GetDeployments()
	}, testTimeout, testInterval)

	t.Logf("SUCCESS: sensor deployment still visible in Central after reconnection (found %d deployments)", len(finalDeployments))
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
