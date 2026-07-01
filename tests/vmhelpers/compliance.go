package vmhelpers

import (
	"context"
	"fmt"
	"time"

	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

// SetContainerEnv ensures ds has envName=envValue on the named container.
// It returns true when the DaemonSet spec was modified.
func SetContainerEnv(ds *appsV1.DaemonSet, containerName, envName, envValue string) (bool, error) {
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

// EnsureComplianceMetricsEnv patches the collector DaemonSet to set the metrics env var and waits for rollout.
func EnsureComplianceMetricsEnv(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	logf func(string, ...any),
	ns, dsName, containerName, envName, envValue string,
) error {
	if logf == nil {
		logf = func(string, ...any) {}
	}

	ds, err := k8sClient.AppsV1().DaemonSets(ns).Get(ctx, dsName, metaV1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting DaemonSet %s/%s: %w", ns, dsName, err)
	}

	changed, err := SetContainerEnv(ds, containerName, envName, envValue)
	if err != nil {
		return err
	}
	if !changed {
		logf("VM scanning setup: %s/%s container %q already has %s=%s", ns, dsName, containerName, envName, envValue)
		return nil
	}

	logf("VM scanning setup: patching %s/%s container %q: %s=%s", ns, dsName, containerName, envName, envValue)
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, getErr := k8sClient.AppsV1().DaemonSets(ns).Get(ctx, dsName, metaV1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		needsUpdate, setErr := SetContainerEnv(current, containerName, envName, envValue)
		if setErr != nil || !needsUpdate {
			return setErr
		}
		_, updateErr := k8sClient.AppsV1().DaemonSets(ns).Update(ctx, current, metaV1.UpdateOptions{})
		return updateErr
	})
	if err != nil {
		return fmt.Errorf("updating DaemonSet %s/%s: %w", ns, dsName, err)
	}

	logf("VM scanning setup: waiting for %s/%s rollout", ns, dsName)
	err = wait.PollUntilContextCancel(ctx, 10*time.Second, false, func(pollCtx context.Context) (bool, error) {
		current, getErr := k8sClient.AppsV1().DaemonSets(ns).Get(pollCtx, dsName, metaV1.GetOptions{})
		if getErr != nil {
			return false, getErr
		}
		ready := current.Status.DesiredNumberScheduled > 0 &&
			current.Status.UpdatedNumberScheduled == current.Status.DesiredNumberScheduled &&
			current.Status.NumberReady == current.Status.DesiredNumberScheduled &&
			current.Status.ObservedGeneration >= current.Generation
		if !ready {
			logf("VM scanning setup: %s/%s rollout in progress (desired=%d updated=%d ready=%d)",
				ns, dsName, current.Status.DesiredNumberScheduled, current.Status.UpdatedNumberScheduled, current.Status.NumberReady)
		}
		return ready, nil
	})
	if err != nil {
		return fmt.Errorf("waiting for %s/%s rollout: %w", ns, dsName, err)
	}
	logf("VM scanning setup: %s/%s rollout complete", ns, dsName)
	return nil
}
