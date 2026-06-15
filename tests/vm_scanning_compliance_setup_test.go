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

// ensureComplianceMetricsExposed guarantees that the collector compliance container
// serves Prometheus metrics on port 9091, that a headless Service routes to it, and
// that dedicated test-owned NetworkPolicies allow ingress to the collector and
// sensor metrics ports used by the VM scanning E2E assertions.
//
// The StackRox Helm chart sets ROX_METRICS_PORT=disabled and deploys a
// "collector-no-ingress" NetworkPolicy (deny-all) when exposeMonitoring is false
// (the operator default). There is no SecuredCluster CR field to set
// exposeMonitoring for collector/sensor, so the test enables the compliance
// metrics port and creates focused, test-owned monitoring policies instead of
// mutating any chart-managed monitoring policies that may already exist.
func (s *VMScanningSuite) ensureComplianceMetricsExposed() {
	ns := namespaces.StackRox
	const (
		svcName                    = "compliance-metrics"
		dsName                     = "collector"
		containerName              = "compliance"
		metricsEnv                 = "ROX_METRICS_PORT"
		metricsValue               = ":9091"
		collectorMetricsPolicyName = "collector-monitoring-vm-scanning-e2e"
		sensorMetricsPolicyName    = "sensor-monitoring-vm-scanning-e2e"
	)
	metricsPort := int32(9091)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	s.ensureComplianceMetricsEnv(ctx, ns, dsName, containerName, metricsEnv, metricsValue)
	s.ensureComplianceMetricsService(ctx, ns, svcName, metricsPort)
	s.ensureMonitoringNetworkPolicy(ctx, ns, collectorMetricsPolicyName, "collector", []int32{9090, 9091})
	s.ensureMonitoringNetworkPolicy(ctx, ns, sensorMetricsPolicyName, "sensor", []int32{9090})
}

func (s *VMScanningSuite) ensureComplianceMetricsEnv(ctx context.Context, ns, dsName, containerName, envName, envValue string) {
	t := s.T()
	ds, err := s.k8sClient.AppsV1().DaemonSets(ns).Get(ctx, dsName, metaV1.GetOptions{})
	require.NoError(t, err, "getting DaemonSet %s/%s", ns, dsName)

	changed, err := ensureDaemonSetContainerEnv(ds, containerName, envName, envValue)
	require.NoError(t, err, "preparing DaemonSet %s/%s container %q for %s=%s", ns, dsName, containerName, envName, envValue)
	if !changed {
		s.logf("VM scanning setup: DaemonSet %s/%s container %q already has %s=%s", ns, dsName, containerName, envName, envValue)
		return
	}

	s.logf("VM scanning setup: patching DaemonSet %s/%s container %q: setting %s=%s", ns, dsName, containerName, envName, envValue)
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, getErr := s.k8sClient.AppsV1().DaemonSets(ns).Get(ctx, dsName, metaV1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		needsUpdate, setErr := ensureDaemonSetContainerEnv(current, containerName, envName, envValue)
		if setErr != nil {
			return setErr
		}
		if !needsUpdate {
			return nil
		}
		_, updateErr := s.k8sClient.AppsV1().DaemonSets(ns).Update(ctx, current, metaV1.UpdateOptions{})
		return updateErr
	})
	require.NoError(t, err, "updating DaemonSet %s/%s to set %s=%s", ns, dsName, envName, envValue)

	s.logf("VM scanning setup: waiting for DaemonSet %s/%s rollout", ns, dsName)
	err = wait.PollUntilContextCancel(ctx, 10*time.Second, false, func(pollCtx context.Context) (bool, error) {
		current, getErr := s.k8sClient.AppsV1().DaemonSets(ns).Get(pollCtx, dsName, metaV1.GetOptions{})
		if getErr != nil {
			s.logf("VM scanning setup: transient error checking DaemonSet rollout: %v", getErr)
			return false, nil
		}
		ready := current.Status.DesiredNumberScheduled > 0 &&
			current.Status.UpdatedNumberScheduled == current.Status.DesiredNumberScheduled &&
			current.Status.NumberReady == current.Status.DesiredNumberScheduled &&
			current.Status.ObservedGeneration >= current.Generation
		if !ready {
			s.logf("VM scanning setup: DaemonSet %s/%s rollout in progress (desired=%d updated=%d ready=%d)",
				ns, dsName, current.Status.DesiredNumberScheduled, current.Status.UpdatedNumberScheduled, current.Status.NumberReady)
		}
		return ready, nil
	})
	require.NoError(t, err, "waiting for DaemonSet %s/%s rollout after setting %s=%s", ns, dsName, envName, envValue)
	s.logf("VM scanning setup: DaemonSet %s/%s rollout complete", ns, dsName)
}

func (s *VMScanningSuite) ensureComplianceMetricsService(ctx context.Context, ns, svcName string, metricsPort int32) {
	t := s.T()
	desired := &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      svcName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "vm-scanning-e2e",
			},
		},
		Spec: coreV1.ServiceSpec{
			ClusterIP: "None",
			Selector:  map[string]string{"app": "collector"},
			Ports: []coreV1.ServicePort{{
				Name:       "monitoring",
				Port:       metricsPort,
				TargetPort: intstr.FromInt32(metricsPort),
				Protocol:   coreV1.ProtocolTCP,
			}},
		},
	}

	err := wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(pollCtx context.Context) (bool, error) {
		existing, getErr := s.k8sClient.CoreV1().Services(ns).Get(pollCtx, svcName, metaV1.GetOptions{})
		if getErr == nil {
			if serviceExposesPort(existing, metricsPort) {
				s.logf("VM scanning setup: service %s/%s verified (port %d)", ns, svcName, metricsPort)
				return true, nil
			}
			s.logf("VM scanning setup: service %s/%s exists but missing port %d, deleting and re-creating", ns, svcName, metricsPort)
			_ = s.k8sClient.CoreV1().Services(ns).Delete(pollCtx, svcName, metaV1.DeleteOptions{})
			return false, nil
		}
		if !apierrors.IsNotFound(getErr) {
			s.logf("VM scanning setup: transient error checking service %s/%s: %v (retrying)", ns, svcName, getErr)
			return false, nil
		}

		s.logf("VM scanning setup: creating service %s/%s (compliance metrics port %d)", ns, svcName, metricsPort)
		_, createErr := s.k8sClient.CoreV1().Services(ns).Create(pollCtx, desired, metaV1.CreateOptions{})
		if createErr != nil {
			if apierrors.IsAlreadyExists(createErr) {
				return false, nil
			}
			s.logf("VM scanning setup: create service %s/%s failed: %v (retrying)", ns, svcName, createErr)
			return false, nil
		}
		return false, nil
	})
	require.NoError(t, err, "ensuring compliance-metrics service %s/%s with port %d", ns, svcName, metricsPort)
}

func serviceExposesPort(svc *coreV1.Service, port int32) bool {
	for _, p := range svc.Spec.Ports {
		if p.Port == port || p.TargetPort.IntValue() == int(port) {
			return true
		}
	}
	return false
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
		s.logf("VM scanning setup: updating dedicated NetworkPolicy %s/%s (app=%s, ports=%v)", ns, name, appLabel, ports)
		desired.ResourceVersion = existing.ResourceVersion
		_, err = s.k8sClient.NetworkingV1().NetworkPolicies(ns).Update(ctx, desired, metaV1.UpdateOptions{})
		require.NoError(t, err, "updating NetworkPolicy %s/%s", ns, name)
	case apierrors.IsNotFound(err):
		s.logf("VM scanning setup: creating dedicated NetworkPolicy %s/%s (app=%s, ports=%v)", ns, name, appLabel, ports)
		_, err = s.k8sClient.NetworkingV1().NetworkPolicies(ns).Create(ctx, desired, metaV1.CreateOptions{})
		require.NoError(t, err, "creating NetworkPolicy %s/%s", ns, name)
	default:
		require.NoError(t, err, "getting NetworkPolicy %s/%s", ns, name)
	}
}

func ensureDaemonSetContainerEnv(ds *appsV1.DaemonSet, containerName, envName, envValue string) (bool, error) {
	for i := range ds.Spec.Template.Spec.Containers {
		container := &ds.Spec.Template.Spec.Containers[i]
		if container.Name != containerName {
			continue
		}
		for j := range container.Env {
			if container.Env[j].Name != envName {
				continue
			}
			if container.Env[j].Value == envValue && container.Env[j].ValueFrom == nil {
				return false, nil
			}
			container.Env[j].Value = envValue
			container.Env[j].ValueFrom = nil
			return true, nil
		}
		container.Env = append(container.Env, coreV1.EnvVar{Name: envName, Value: envValue})
		return true, nil
	}
	return false, fmt.Errorf("container %q not found in DaemonSet %s/%s", containerName, ds.Namespace, ds.Name)
}
