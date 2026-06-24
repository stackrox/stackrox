//go:build test_e2e_vm

package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

// ensureComplianceMetricsExposed patches the collector DaemonSet so the
// compliance container serves Prometheus metrics on port 9091 and installs
// dedicated test-owned NetworkPolicies that allow ingress to collector and
// sensor metrics ports.
//
// The StackRox Helm chart sets ROX_METRICS_PORT=disabled and deploys a
// "collector-no-ingress" NetworkPolicy (deny-all) when exposeMonitoring
// is false (the operator default). There is no SecuredCluster CR field to
// override this for collector, so the test patches the DaemonSet directly
// and creates focused, test-owned monitoring policies.
//
// Scraping uses the Kubernetes pods/proxy subresource, so no Service is
// needed; only the NetworkPolicies must allow ingress on the metrics ports.
func (s *VMScanningSuite) ensureComplianceMetricsExposed() {
	const (
		ns                         = namespaces.StackRox
		dsName                     = "collector"
		containerName              = "compliance"
		envName                    = "ROX_METRICS_PORT"
		envValue                   = ":9091"
		collectorMetricsPolicyName = "collector-monitoring-vm-scanning-e2e"
		sensorMetricsPolicyName    = "sensor-monitoring-vm-scanning-e2e"
	)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	s.ensureComplianceMetricsEnv(ctx, ns, dsName, containerName, envName, envValue)
	s.ensureMonitoringNetworkPolicy(ctx, ns, collectorMetricsPolicyName, "collector", []int32{9090, 9091})
	s.ensureMonitoringNetworkPolicy(ctx, ns, sensorMetricsPolicyName, "sensor", []int32{9090})
}

// ensureComplianceMetricsEnv patches the collector DaemonSet to set the
// metrics port env var and waits for the rollout to complete.
func (s *VMScanningSuite) ensureComplianceMetricsEnv(ctx context.Context, ns, dsName, containerName, envName, envValue string) {
	t := s.T()

	ds, err := s.k8sClient.AppsV1().DaemonSets(ns).Get(ctx, dsName, metaV1.GetOptions{})
	require.NoError(t, err, "getting DaemonSet %s/%s", ns, dsName)

	changed, err := setContainerEnv(ds, containerName, envName, envValue)
	require.NoError(t, err)
	if !changed {
		s.logf("VM scanning setup: %s/%s container %q already has %s=%s", ns, dsName, containerName, envName, envValue)
		return
	}

	s.logf("VM scanning setup: patching %s/%s container %q: %s=%s", ns, dsName, containerName, envName, envValue)
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, getErr := s.k8sClient.AppsV1().DaemonSets(ns).Get(ctx, dsName, metaV1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		needsUpdate, setErr := setContainerEnv(current, containerName, envName, envValue)
		if setErr != nil || !needsUpdate {
			return setErr
		}
		_, updateErr := s.k8sClient.AppsV1().DaemonSets(ns).Update(ctx, current, metaV1.UpdateOptions{})
		return updateErr
	})
	require.NoError(t, err, "updating DaemonSet %s/%s", ns, dsName)

	s.logf("VM scanning setup: waiting for %s/%s rollout", ns, dsName)
	err = wait.PollUntilContextCancel(ctx, 10*time.Second, false, func(pollCtx context.Context) (bool, error) {
		current, getErr := s.k8sClient.AppsV1().DaemonSets(ns).Get(pollCtx, dsName, metaV1.GetOptions{})
		if getErr != nil {
			return false, getErr
		}
		ready := current.Status.DesiredNumberScheduled > 0 &&
			current.Status.UpdatedNumberScheduled == current.Status.DesiredNumberScheduled &&
			current.Status.NumberReady == current.Status.DesiredNumberScheduled &&
			current.Status.ObservedGeneration >= current.Generation
		if !ready {
			s.logf("VM scanning setup: %s/%s rollout in progress (desired=%d updated=%d ready=%d)",
				ns, dsName, current.Status.DesiredNumberScheduled, current.Status.UpdatedNumberScheduled, current.Status.NumberReady)
		}
		return ready, nil
	})
	require.NoError(t, err, "waiting for %s/%s rollout", ns, dsName)
	s.logf("VM scanning setup: %s/%s rollout complete", ns, dsName)
}

// ensureMonitoringNetworkPolicy creates or updates a dedicated test-owned
// ingress-allow NetworkPolicy for the given app label and TCP ports.
func (s *VMScanningSuite) ensureMonitoringNetworkPolicy(ctx context.Context, ns, name, appLabel string, ports []int32) {
	t := s.T()
	tcp := coreV1.ProtocolTCP
	var ingressPorts []networkingV1.NetworkPolicyPort
	for _, p := range ports {
		ingressPorts = append(ingressPorts, networkingV1.NetworkPolicyPort{
			Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: p},
			Protocol: &tcp,
		})
	}

	desired := &networkingV1.NetworkPolicy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "vm-scanning-e2e",
			},
		},
		Spec: networkingV1.NetworkPolicySpec{
			PodSelector: metaV1.LabelSelector{
				MatchLabels: map[string]string{"app": appLabel},
			},
			Ingress: []networkingV1.NetworkPolicyIngressRule{{
				Ports: ingressPorts,
			}},
			PolicyTypes: []networkingV1.PolicyType{networkingV1.PolicyTypeIngress},
		},
	}

	existing, err := s.k8sClient.NetworkingV1().NetworkPolicies(ns).Get(ctx, name, metaV1.GetOptions{})
	switch {
	case err == nil:
		s.logf("VM scanning setup: updating NetworkPolicy %s/%s (app=%s, ports=%v)", ns, name, appLabel, ports)
		desired.ResourceVersion = existing.ResourceVersion
		_, err = s.k8sClient.NetworkingV1().NetworkPolicies(ns).Update(ctx, desired, metaV1.UpdateOptions{})
		require.NoError(t, err, "updating NetworkPolicy %s/%s", ns, name)
	case apierrors.IsNotFound(err):
		s.logf("VM scanning setup: creating NetworkPolicy %s/%s (app=%s, ports=%v)", ns, name, appLabel, ports)
		_, err = s.k8sClient.NetworkingV1().NetworkPolicies(ns).Create(ctx, desired, metaV1.CreateOptions{})
		require.NoError(t, err, "creating NetworkPolicy %s/%s", ns, name)
	default:
		require.NoError(t, err, "getting NetworkPolicy %s/%s", ns, name)
	}
}

// setContainerEnv ensures ds has envName=envValue on the named container.
// Returns (true, nil) if the DaemonSet was modified.
func setContainerEnv(ds *appsV1.DaemonSet, containerName, envName, envValue string) (bool, error) {
	for i := range ds.Spec.Template.Spec.Containers {
		c := &ds.Spec.Template.Spec.Containers[i]
		if c.Name != containerName {
			continue
		}
		for j := range c.Env {
			if c.Env[j].Name != envName {
				continue
			}
			if c.Env[j].Value == envValue && c.Env[j].ValueFrom == nil {
				return false, nil
			}
			c.Env[j].Value = envValue
			c.Env[j].ValueFrom = nil
			return true, nil
		}
		c.Env = append(c.Env, coreV1.EnvVar{Name: envName, Value: envValue})
		return true, nil
	}
	return false, fmt.Errorf("container %q not found in DaemonSet %s/%s", containerName, ds.Namespace, ds.Name)
}
