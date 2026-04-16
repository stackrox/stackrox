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

// TestResolverOptimization_ConnectionResilience verifies that the resolver optimization
// does not cause message loss when the sensor reconnects to Central after a network disruption.
// This test simulates a network outage using toxiproxy and verifies that deployments remain
// visible in Central after reconnection.
func TestResolverOptimization_ConnectionResilience(t *testing.T) {
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

	// Step 2: Patch sensor to add toxiproxy sidecar
	toxiproxyContainer := corev1.Container{
		Name:  "toxiproxy",
		Image: toxiproxyImage,
		Ports: []corev1.ContainerPort{
			{ContainerPort: toxiproxyAPIPort, Name: "toxiproxy-api", Protocol: corev1.ProtocolTCP},
			{ContainerPort: toxiproxyProxyPort, Name: "toxiproxy-proxy", Protocol: corev1.ProtocolTCP},
		},
	}

	// Find sensor container and update env vars
	for i := range sensorDeploy.Spec.Template.Spec.Containers {
		if sensorDeploy.Spec.Template.Spec.Containers[i].Name == "sensor" {
			// Replace or add env vars
			envVars := []corev1.EnvVar{}
			for _, env := range sensorDeploy.Spec.Template.Spec.Containers[i].Env {
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
			sensorDeploy.Spec.Template.Spec.Containers[i].Env = envVars
			break
		}
	}

	// Add toxiproxy container
	sensorDeploy.Spec.Template.Spec.Containers = append(sensorDeploy.Spec.Template.Spec.Containers, toxiproxyContainer)

	_, err := k8sClient.AppsV1().Deployments(sensorNamespace).Update(ctx, sensorDeploy, metav1.UpdateOptions{})
	require.NoError(t, err, "failed to update sensor deployment")

	// Cleanup: restore original deployment on test completion
	t.Cleanup(func() {
		// Remove toxiproxy env vars and container
		deploy, err := k8sClient.AppsV1().Deployments(sensorNamespace).Get(ctx, sensorDeploymentName, metav1.GetOptions{})
		if err != nil {
			return
		}

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

		_, _ = k8sClient.AppsV1().Deployments(sensorNamespace).Update(ctx, deploy, metav1.UpdateOptions{})
	})

	// Step 3: Wait for sensor pod to be ready using Eventually
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

	t.Logf("Sensor pod %s is ready", sensorPod.Name)

	// Step 4: Set up port-forward to toxiproxy API
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
		Namespace(sensorNamespace).
		Name(sensorPod.Name).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	require.NoError(t, err, "failed to create SPDY transport")

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())

	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{})

	// Request port 0 (any available local port) -> toxiproxyAPIPort
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("0:%d", toxiproxyAPIPort)}, stopChannel, readyChannel, nil, nil)
	require.NoError(t, err, "failed to create port forwarder")

	// Start port forwarding in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- forwarder.ForwardPorts()
	}()

	// Cleanup: stop port-forward on test completion
	t.Cleanup(func() {
		close(stopChannel)
		<-errChan // Wait for forwarder to stop
	})

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
	toxiproxyEndpoint := fmt.Sprintf("localhost:%d", localPort)
	t.Logf("Port-forward established: localhost:%d -> %s:%d", localPort, sensorPod.Name, toxiproxyAPIPort)

	// Step 5: Connect to toxiproxy API and wait for it to be ready
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

	// Step 7: Wait for sensor to be healthy (waitUntilCentralSensorConnectionIs already uses Eventually internally)
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
	t.Log("Sensor is healthy (baseline)")

	// Step 8: Verify sensor deployment is visible in Central (baseline)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	deploymentService := v1.NewDeploymentServiceClient(conn)

	deployments, err := deploymentService.ListDeployments(ctx, &v1.RawQuery{Query: "Deployment:sensor"})
	require.NoError(t, err, "failed to list deployments")
	require.NotEmpty(t, deployments.GetDeployments(), "sensor deployment not found in Central")

	t.Logf("Baseline: sensor deployment visible in Central (found %d deployments)", len(deployments.GetDeployments()))

	// Step 9: Disable proxy to simulate connection loss
	centralProxy.Enabled = false
	err = centralProxy.Save()
	require.NoError(t, err, "failed to disable central proxy")

	t.Log("Disabled toxiproxy - connection to Central severed")

	// Step 10: Wait for sensor to become degraded (waitUntilCentralSensorConnectionIs already uses Eventually internally)
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_DEGRADED)
	t.Log("Sensor is degraded (connection disrupted)")

	// Step 11: Sleep for disconnect duration (simulate sustained outage)
	disconnectDuration := 10 * time.Second
	t.Logf("Sleeping for %s to simulate sustained connection loss", disconnectDuration)
	time.Sleep(disconnectDuration)

	// Step 12: Re-enable proxy to restore connection
	centralProxy.Enabled = true
	err = centralProxy.Save()
	require.NoError(t, err, "failed to re-enable central proxy")

	t.Log("Re-enabled toxiproxy - connection to Central restored")

	// Step 13: Wait for sensor to become healthy again (waitUntilCentralSensorConnectionIs already uses Eventually internally)
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
	t.Log("Sensor is healthy again (reconnected)")

	// Step 14: Verify sensor deployment is STILL visible in Central (critical validation)
	var finalDeployments []*storage.ListDeployment
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		deployments, err := deploymentService.ListDeployments(ctx, &v1.RawQuery{Query: "Deployment:sensor"})
		require.NoErrorf(c, err, "failed to list deployments")
		require.NotEmptyf(c, deployments.GetDeployments(), "sensor deployment lost after reconnection")
		finalDeployments = deployments.GetDeployments()
	}, testTimeout, testInterval)

	t.Logf("SUCCESS: sensor deployment still visible in Central after reconnection (found %d deployments)", len(finalDeployments))
}
