package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
)

func TestTLSChallenge(t *testing.T) {
	s := "stackrox"
	const proxyServiceName = "nginx-loadbalancer"
	const proxyNs = "test-tls-challenge"
	const proxyEndpoint = proxyServiceName + "." + proxyNs + ":443"

	ctx, cancel := testContext(t, "TestTLSChallenge", 10*time.Minute)
	defer cancel()
	defer waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)

	originalCentralEndpoint := getDeploymentEnvVal(t, ctx, s, "sensor", "ROX_CENTRAL_ENDPOINT")
	defer setDeploymentEnvVal(t, ctx, s, "sensor", "ROX_CENTRAL_ENDPOINT", originalCentralEndpoint)

	setupProxy(t, ctx, proxyNs)
	defer cleanupProxy(t, ctx, proxyNs)

	t.Logf("Pointing sensor at the proxy...")
	setDeploymentEnvVal(t, ctx, s, "sensor", "ROX_CENTRAL_ENDPOINT", proxyEndpoint)
	t.Logf("Waiting for sensor log to mention... WHAT?")
	
}

func cleanupProxy(t *testing.T, ctx context.Context, proxyNs string) {
	t.Logf("Cleaning up nginx proxy...")
	panic("unimplemented")
}

func setupProxy(t *testing.T, ctx context.Context, proxyNs string) {
	t.Logf("Setting up nginx proxy...")
	panic("unimplemented")
}

func waitUntilCentralSensorConnectionIs(t *testing.T, ctx context.Context, statusHealthy storage.ClusterHealthStatus_HealthStatusLabel) {
	panic("unimplemented")
}

func setDeploymentEnvVal(t *testing.T, ctx context.Context, namespace string, deployment string, envVar string, value string) {
	panic("unimplemented")
}

func getDeploymentEnvVal(t *testing.T, ctx context.Context, namespace string, deployment string, envVar string) string {
	panic("unimplemented")
}
