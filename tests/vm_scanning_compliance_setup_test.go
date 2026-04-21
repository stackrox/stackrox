//go:build test_e2e

package tests

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ensureComplianceMetricsExposed guarantees that the collector compliance container
// serves Prometheus metrics on port 9091, that a headless Service routes to it, and
// that NetworkPolicies allow ingress to both collector and sensor metrics ports.
//
// The StackRox Helm chart sets ROX_METRICS_PORT=disabled and deploys a
// "collector-no-ingress" NetworkPolicy (deny-all) when exposeMonitoring is false
// (the operator default). There is no SecuredCluster CR field to set
// exposeMonitoring for collector/sensor, so the test creates the equivalent
// resources: env patch, Service, and permissive NetworkPolicies matching what
// the chart would produce with exposeMonitoring=true.
func (s *VMScanningSuite) ensureComplianceMetricsExposed() {
	ns := namespaces.StackRox
	const (
		svcName       = "compliance-metrics"
		dsName        = "collector"
		containerName = "compliance"
		metricsEnv    = "ROX_METRICS_PORT"
		metricsValue  = ":9091"
	)
	metricsPort := int32(9091)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	s.ensureComplianceMetricsEnv(ctx, ns, dsName, containerName, metricsEnv, metricsValue)
	s.ensureComplianceMetricsService(ctx, ns, svcName, metricsPort)
	s.ensureMonitoringNetworkPolicy(ctx, ns, "collector-monitoring", "collector", []int32{9090, 9091})
	s.ensureMonitoringNetworkPolicy(ctx, ns, "sensor-monitoring", "sensor", []int32{9090})
}

func (s *VMScanningSuite) ensureComplianceMetricsEnv(ctx context.Context, ns, dsName, containerName, envName, envValue string) {
	t := s.T()
	ds, err := s.k8sClient.AppsV1().DaemonSets(ns).Get(ctx, dsName, metaV1.GetOptions{})
	require.NoError(t, err, "getting DaemonSet %s/%s", ns, dsName)

	var container *coreV1.Container
	for i := range ds.Spec.Template.Spec.Containers {
		if ds.Spec.Template.Spec.Containers[i].Name == containerName {
			container = &ds.Spec.Template.Spec.Containers[i]
			break
		}
	}
	require.NotNil(t, container, "container %q not found in DaemonSet %s/%s", containerName, ns, dsName)

	for _, e := range container.Env {
		if e.Name == envName && e.Value == envValue {
			s.logf("VM scanning setup: DaemonSet %s/%s container %q already has %s=%s", ns, dsName, containerName, envName, envValue)
			return
		}
	}

	s.logf("VM scanning setup: patching DaemonSet %s/%s container %q: setting %s=%s", ns, dsName, containerName, envName, envValue)
	updated := false
	for i, e := range container.Env {
		if e.Name == envName {
			container.Env[i].Value = envValue
			container.Env[i].ValueFrom = nil
			updated = true
			break
		}
	}
	if !updated {
		container.Env = append(container.Env, coreV1.EnvVar{Name: envName, Value: envValue})
	}

	_, err = s.k8sClient.AppsV1().DaemonSets(ns).Update(ctx, ds, metaV1.UpdateOptions{})
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

// ensureMonitoringNetworkPolicy creates an ingress-allow NetworkPolicy for the
// given app label and TCP ports. This mirrors what the Helm chart creates when
// exposeMonitoring is true (e.g. "collector-monitoring" or "sensor-monitoring").
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
	if err == nil {
		if networkPolicyMatchesPorts(existing, ports) {
			s.logf("VM scanning setup: NetworkPolicy %s/%s already allows ports %v", ns, name, ports)
			return
		}
		s.logf("VM scanning setup: NetworkPolicy %s/%s exists but ports differ, updating", ns, name)
		desired.ResourceVersion = existing.ResourceVersion
		_, err = s.k8sClient.NetworkingV1().NetworkPolicies(ns).Update(ctx, desired, metaV1.UpdateOptions{})
		require.NoError(t, err, "updating NetworkPolicy %s/%s", ns, name)
		return
	}
	if !apierrors.IsNotFound(err) {
		require.NoError(t, err, "getting NetworkPolicy %s/%s", ns, name)
	}

	s.logf("VM scanning setup: creating NetworkPolicy %s/%s (app=%s, ports=%v)", ns, name, appLabel, ports)
	_, err = s.k8sClient.NetworkingV1().NetworkPolicies(ns).Create(ctx, desired, metaV1.CreateOptions{})
	require.NoError(t, err, "creating NetworkPolicy %s/%s", ns, name)
}

func networkPolicyMatchesPorts(np *networkingV1.NetworkPolicy, wantPorts []int32) bool {
	if len(np.Spec.Ingress) != 1 {
		return false
	}
	have := make(map[int32]bool)
	for _, p := range np.Spec.Ingress[0].Ports {
		if p.Port != nil {
			have[p.Port.IntVal] = true
		}
	}
	for _, p := range wantPorts {
		if !have[p] {
			return false
		}
	}
	return len(have) == len(wantPorts)
}
