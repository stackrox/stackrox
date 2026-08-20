//go:build test_e2e_vm

package tests

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/tests/vmhelpers"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	err := vmhelpers.EnsureComplianceMetricsEnv(ctx, s.k8sClient, s.logf, ns, dsName, containerName, envName, envValue)
	require.NoError(s.T(), err)
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
