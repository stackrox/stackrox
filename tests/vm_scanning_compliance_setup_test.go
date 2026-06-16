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
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

// ensureComplianceMetricsExposed patches the collector DaemonSet so the
// compliance container serves Prometheus metrics on port 9091.
//
// The StackRox Helm chart sets ROX_METRICS_PORT=disabled when
// exposeMonitoring is false (the operator default). There is no
// SecuredCluster CR field to override this for collector, so the test
// patches the DaemonSet directly.
//
// Scraping uses the Kubernetes pods/proxy subresource which bypasses
// Services and NetworkPolicies, so no additional resources are needed.
func (s *VMScanningSuite) ensureComplianceMetricsExposed() {
	const (
		ns            = namespaces.StackRox
		dsName        = "collector"
		containerName = "compliance"
		envName       = "ROX_METRICS_PORT"
		envValue      = ":9091"
	)
	t := s.T()

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

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
